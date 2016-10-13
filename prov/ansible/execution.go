package ansible

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"github.com/hashicorp/consul/api"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/executil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"
)

const output_custom_wrapper = `
[[[printf ". $HOME/%s/%s" $.OperationRemotePath .BasePrimary]]]
[[[range $artName, $art := .Output -]]]
[[[printf "echo %s,$%s >> $HOME/%s/out.csv" $artName $artName $.OperationRemotePath]]]
[[[printf "echo $%s" $artName]]]
[[[end]]]
[[[printf "chmod 777 $HOME/%s/out.csv" $.OperationRemotePath]]]
`

const ansible_playbook = `
- name: Executing script {{ script_to_run }}
  hosts: all
  strategy: free
  tasks:
    - file: path="{{ ansible_env.HOME}}/[[[.OperationRemotePath]]]" state=directory mode=0755
    [[[if .HaveOutput]]]
    [[[printf  "- copy: src=\"{{ wrapper_location }}\" dest=\"{{ ansible_env.HOME}}/%s/wrapper.sh\" mode=0744" $.OperationRemotePath]]]
    [[[end]]]
    - copy: src="{{ script_to_run }}" dest="{{ ansible_env.HOME}}/[[[.OperationRemotePath]]]" mode=0744
    [[[ range $artName, $art := .Artifacts -]]]
    [[[printf "- copy: src=\"%s/%s\" dest=\"{{ ansible_env.HOME}}/%s/%s\"" $.OverlayPath $art $.OperationRemotePath (path $art)]]]
    [[[end]]]
    [[[if not .HaveOutput]]]
    [[[printf "- shell: \"{{ ansible_env.HOME}}/%s/%s\"" $.OperationRemotePath .BasePrimary]]][[[else]]]
    [[[printf "- shell: \"/bin/bash -l {{ ansible_env.HOME}}/%s/wrapper.sh\"" $.OperationRemotePath]]][[[end]]]
      environment:
        [[[ range $key, $envInput := .EnvInputs -]]]
        [[[ if (len $envInput.InstanceName) gt 0]]][[[ if (len $envInput.Value) gt 0]]][[[printf  "%s_%s: %s" $envInput.InstanceName $envInput.Name $envInput.Value]]][[[else]]][[[printf  "%s_%s: \"\"" $envInput.InstanceName $envInput.Name]]]
        [[[end]]][[[else]]][[[ if (len $envInput.Value) gt 0]]][[[printf  "%s: %s" $envInput.Name $envInput.Value]]][[[else]]]
        [[[printf  "%s: \"\"" $envInput.Name]]]
        [[[end]]][[[end]]]
        [[[end]]][[[ range $artName, $art := .Artifacts -]]]
        [[[printf "%s: \"{{ ansible_env.HOME}}/%s/%s\"" $artName $.OperationRemotePath $art]]]
        [[[end]]][[[ range $contextK, $contextV := .Context -]]]
        [[[printf "%s: %s" $contextK $contextV]]]
        [[[end]]][[[ range $hostVarIndex, $hostVarValue := .VarInputsNames -]]]
        [[[printf "%s: \"{{%s}}\"" $hostVarValue $hostVarValue]]]
        [[[end]]]
    [[[if .HaveOutput]]]
    [[[printf "- fetch: src={{ ansible_env.HOME}}/%s/out.csv dest={{dest_folder}} flat=yes" $.OperationRemotePath]]]
    [[[end]]]
`

const ansible_config = `[defaults]
host_key_checking=False
timeout=600
stdout_callback = json
retry_files_save_path = #PLAY_PATH#
`

type ansibleRetriableError struct {
	root error
}

func (are ansibleRetriableError) Error() string {
	return are.root.Error()
}

func IsRetriable(err error) bool {
	_, ok := err.(ansibleRetriableError)
	return ok
}

func IsOperationNotImplemented(err error) bool {
	_, ok := err.(operationNotImplemented)
	return ok
}

type operationNotImplemented struct {
	msg string
}

func (oni operationNotImplemented) Error() string {
	return oni.msg
}

type hostConnection struct {
	host string
	user string
}

type EnvInput struct {
	Name           string
	Value          string
	InstanceName   string
	IsTargetScoped bool
}

func (ei EnvInput) String() string {
	return fmt.Sprintf("EnvInput: [Name: %q, Value: %q, InstanceName: %q, IsTargetScoped: \"%t\"]", ei.Name, ei.Value, ei.InstanceName, ei.IsTargetScoped)
}

type execution struct {
	kv                       *api.KV
	DeploymentId             string
	NodeName                 string
	Operation                string
	OperationRemotePath      string
	NodeType                 string
	EnvInputs                []*EnvInput
	VarInputsNames           []string
	Primary                  string
	BasePrimary              string
	Dependencies             []string
	hosts                    map[string]hostConnection
	OperationPath            string
	NodePath                 string
	NodeTypePath             string
	Artifacts                map[string]string
	OverlayPath              string
	Context                  map[string]string
	Output                   map[string]string
	HaveOutput               bool
	isRelationshipOperation  bool
	isRelationshipTargetNode bool
	isPerInstanceOperation   bool
	relationshipType         string
	relationshipTargetName   string
	requirementIndex         string
}

func newExecution(kv *api.KV, deploymentId, nodeName, operation string) (*execution, error) {
	execution := &execution{kv: kv,
		DeploymentId:   deploymentId,
		NodeName:       nodeName,
		Operation:      operation,
		VarInputsNames: make([]string, 0),
		EnvInputs:      make([]*EnvInput, 0)}
	return execution, execution.resolveExecution()
}

func (e *execution) resolveArtifacts() error {
	log.Debugf("Resolving artifacts")
	artifacts := make(map[string]string)
	// First resolve node type artifacts then node template artifact if the is a conflict then node template will have the precedence
	// TODO deal with type inheritance
	// TODO deal with relationships
	paths := []string{path.Join(e.NodePath, "artifacts"), path.Join(e.NodeTypePath, "artifacts")}
	for _, apath := range paths {
		artsPaths, _, err := e.kv.Keys(apath+"/", "/", nil)
		if err != nil {
			return err
		}
		for _, artPath := range artsPaths {
			kvp, _, err := e.kv.Get(path.Join(artPath, "name"), nil)
			if err != nil {
				return err
			}
			if kvp == nil {
				return fmt.Errorf("Missing mandatory key in consul %q", path.Join(artPath, "name"))
			}
			artName := string(kvp.Value)
			kvp, _, err = e.kv.Get(path.Join(artPath, "file"), nil)
			if err != nil {
				return err
			}
			if kvp == nil {
				return fmt.Errorf("Missing mandatory key in consul %q", path.Join(artPath, "file"))
			}
			artifacts[artName] = string(kvp.Value)
		}
	}

	e.Artifacts = artifacts
	log.Debugf("Resolved artifacts: %v", e.Artifacts)
	return nil
}

func (e *execution) resolveInputs() error {
	log.Debug("resolving inputs")
	resolver := deployments.NewResolver(e.kv, e.DeploymentId)
	inputKeys, _, err := e.kv.Keys(e.OperationPath+"/inputs/", "/", nil)
	if err != nil {
		return err
	}
	for _, input := range inputKeys {
		kvPair, _, err := e.kv.Get(input+"/name", nil)
		if err != nil {
			return err
		}
		if kvPair == nil {
			return fmt.Errorf("%s/name missing", input)
		}
		inputName := string(kvPair.Value)
		kvPair, _, err = e.kv.Get(input+"/expression", nil)
		if err != nil {
			return err
		}
		if kvPair == nil {
			return fmt.Errorf("%s/expression missing", input)
		}
		va := tosca.ValueAssignment{}
		yaml.Unmarshal(kvPair.Value, &va)
		targetContext := va.Expression.IsTargetContext()
		var instancesIds []string
		if e.isRelationshipOperation && targetContext {
			instancesIds, err = deployments.GetNodeInstancesIds(e.kv, e.DeploymentId, e.relationshipTargetName)
		} else {
			instancesIds, err = deployments.GetNodeInstancesIds(e.kv, e.DeploymentId, e.NodeName)
		}
		if err != nil {
			return err
		}
		var inputValue string
		if len(instancesIds) > 0 {
			for i, instanceId := range instancesIds {
				envI := &EnvInput{Name: inputName, IsTargetScoped: targetContext}
				if e.isRelationshipOperation && targetContext {
					envI.InstanceName = getInstanceName(e.relationshipTargetName, instanceId)
				} else {
					envI.InstanceName = getInstanceName(e.NodeName, instanceId)
				}
				if e.isRelationshipOperation {
					inputValue, err = resolver.ResolveExpressionForRelationship(va.Expression, e.NodeName, e.relationshipTargetName, e.requirementIndex, instanceId)
				} else {
					inputValue, err = resolver.ResolveExpressionForNode(va.Expression, e.NodeName, instanceId)
				}
				if err != nil {
					return err
				}
				envI.Value = inputValue
				e.EnvInputs = append(e.EnvInputs, envI)
				if i == 0 {
					e.VarInputsNames = append(e.VarInputsNames, sanitizeForShell(inputName))
				}
			}
		} else {
			envI := &EnvInput{Name: inputName, IsTargetScoped: targetContext}
			if e.isRelationshipOperation {
				inputValue, err = resolver.ResolveExpressionForRelationship(va.Expression, e.NodeName, e.relationshipTargetName, e.requirementIndex, "")
			} else {
				inputValue, err = resolver.ResolveExpressionForNode(va.Expression, e.NodeName, "")
			}
			if err != nil {
				return err
			}
			envI.Value = inputValue
			e.EnvInputs = append(e.EnvInputs, envI)
		}
	}

	log.Debugf("Resolved env inputs: %s", e.EnvInputs)
	return nil
}

func (e *execution) resolveHosts(nodeName string) error {

	// e.nodePath
	instancesPath := path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/instances", nodeName)
	log.Debugf("Resolving hosts for node %q", nodeName)

	hosts := make(map[string]hostConnection)
	instances, err := deployments.GetNodeInstancesIds(e.kv, e.DeploymentId, nodeName)
	if err != nil {
		return err
	}
	for _, instance := range instances {

		kvp, _, err := e.kv.Get(path.Join(instancesPath, instance, "capabilities/endpoint/attributes/ip_address"), nil)
		if err != nil {
			return err
		}
		if kvp != nil && len(kvp.Value) != 0 {
			var instanceName string
			if e.isRelationshipTargetNode {
				instanceName = getInstanceName(e.relationshipTargetName, instance)
			} else {
				instanceName = getInstanceName(e.NodeName, instance)
			}

			hostConn := hostConnection{host: string(kvp.Value)}
			kvp, _, err := e.kv.Get(path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/nodes", nodeName, "properties/user"), nil)
			if err != nil {
				return err
			}
			if kvp != nil && len(kvp.Value) != 0 {
				va := tosca.ValueAssignment{}
				yaml.Unmarshal(kvp.Value, &va)
				hostConn.user, err = deployments.NewResolver(e.kv, e.DeploymentId).ResolveExpressionForNode(va.Expression, nodeName, instance)
				if err != nil {
					return err
				}
			}
			hosts[instanceName] = hostConn
		}
	}
	if len(hosts) == 0 {
		// So we have to traverse the HostedOn relationships...
		hostedOnNode, err := deployments.GetHostedOnNode(e.kv, e.DeploymentId, nodeName)
		if err != nil {
			return err
		}
		if hostedOnNode == "" {
			return fmt.Errorf("Can't find an Host with an ip_address in the HostedOn hierarchy for node %q in deployment %q", e.NodeName, e.DeploymentId)
		}
		return e.resolveHosts(hostedOnNode)
	}
	e.hosts = hosts
	return nil
}

func sanitizeForShell(str string) string {
	return strings.Map(func(r rune) rune {
		// Replace hyphen by underscore
		if r == '-' {
			return '_'
		}
		// Keep underscores
		if r == '_' {
			return r
		}
		// Drop any other non-alphanum rune
		if r < '0' || r > 'z' || r > '9' && r < 'A' || r > 'Z' && r < 'a' {
			return rune(-1)
		}
		return r

	}, str)
}

func (e *execution) resolveContext() error {
	execContext := make(map[string]string)

	new_node := sanitizeForShell(e.NodeName)
	if !e.isRelationshipOperation {
		execContext["NODE"] = new_node
	}
	names, err := deployments.GetNodeInstancesIds(e.kv, e.DeploymentId, e.NodeName)
	if err != nil {
		return err
	}
	for i := range names {
		instanceName := getInstanceName(e.NodeName, names[i])
		names[i] = instanceName
	}
	if len(names) == 0 {
		names = append(names, new_node)
	}
	if !e.isRelationshipOperation {
		e.VarInputsNames = append(e.VarInputsNames, "INSTANCE")
		execContext["INSTANCES"] = strings.Join(names, ",")
		if host, err := deployments.GetHostedOnNode(e.kv, e.DeploymentId, e.NodeName); err != nil {
			return err
		} else if host != "" {
			execContext["HOST"] = host
		}
	} else {

		if host, err := deployments.GetHostedOnNode(e.kv, e.DeploymentId, e.NodeName); err != nil {
			return err
		} else if host != "" {
			execContext["SOURCE_HOST"] = host
		}
		if host, err := deployments.GetHostedOnNode(e.kv, e.DeploymentId, e.relationshipTargetName); err != nil {
			return err
		} else if host != "" {
			execContext["TARGET_HOST"] = host
		}
		execContext["SOURCE_NODE"] = new_node
		if e.isRelationshipTargetNode && !e.isPerInstanceOperation {
			execContext["SOURCE_INSTANCE"] = names[0]
		} else {
			e.VarInputsNames = append(e.VarInputsNames, "SOURCE_INSTANCE")
		}
		execContext["SOURCE_INSTANCES"] = strings.Join(names, ",")
		execContext["TARGET_NODE"] = sanitizeForShell(e.relationshipTargetName)

		targetNames, err := deployments.GetNodeInstancesIds(e.kv, e.DeploymentId, e.relationshipTargetName)
		if err != nil {
			return err
		}
		for i := range targetNames {
			targetNames[i] = getInstanceName(e.relationshipTargetName, targetNames[i])
		}
		if len(targetNames) == 0 {
			targetNames = append(targetNames, execContext["TARGET_NODE"])
		}
		execContext["TARGET_INSTANCES"] = strings.Join(targetNames, ",")

		if !e.isRelationshipTargetNode && !e.isPerInstanceOperation {
			execContext["TARGET_INSTANCE"] = targetNames[0]
		} else {
			e.VarInputsNames = append(e.VarInputsNames, "TARGET_INSTANCE")
		}

	}

	e.Context = execContext

	return nil
}

func (e *execution) resolveOperationOutput() error {
	log.Debugf(e.OperationPath)
	log.Debugf(e.Operation)

	//We get all the output of the NodeType
	pathList, _, err := e.kv.Keys(e.NodeTypePath+"/output/", "", nil)

	if err != nil {
		return err
	}

	output := make(map[string]string)

	//For each type we compare if we are in the good lifecycle operation
	for _, path := range pathList {
		tmp := strings.Split(e.Operation, ".")
		if strings.Contains(path, tmp[len(tmp)-1]) {
			nodeOutPath := filepath.Join(e.NodePath, "attributes", strings.ToLower(filepath.Base(path)))
			e.HaveOutput = true
			output[filepath.Base(path)] = nodeOutPath
		}
	}

	log.Debugf("%v", output)
	e.Output = output
	return nil
}

// isTargetOperation returns true if the given operationName contains one of the following patterns (case doesn't matter):
//	pre_configure_target, post_configure_target, add_source
func isTargetOperation(operationName string) bool {
	op := strings.ToLower(operationName)
	if strings.Contains(op, "pre_configure_target") || strings.Contains(op, "post_configure_target") || strings.Contains(op, "add_source") {
		return true
	}
	return false
}

// resolveIsPerInstanceOperation sets e.isPerInstanceOperation to true if the given operationName contains one of the following patterns (case doesn't matter):
//	add_target, remove_target, add_source, target_changed
// And in case of a relationship operation the relationship does not derive from "tosca.relationships.HostedOn" as it makes no sense till we scale at compute level
func (e *execution) resolveIsPerInstanceOperation(operationName string) error {
	op := strings.ToLower(operationName)
	if strings.Contains(op, "add_target") || strings.Contains(op, "remove_target") || strings.Contains(op, "target_changed") || strings.Contains(op, "add_source") {
		// Do not call the call the operation several time for an HostedOn relationship (makes no sense till we scale at compute level)
		if hostedOn, err := deployments.IsNodeTypeDerivedFrom(e.kv, e.DeploymentId, e.relationshipType, "tosca.relationships.HostedOn"); err != nil || hostedOn {
			e.isPerInstanceOperation = false
			return err
		}
		e.isPerInstanceOperation = true
		return nil
	}
	e.isPerInstanceOperation = false
	return nil
}

func (e *execution) resolveExecution() error {
	log.Printf("Preparing execution of operation %q on node %q for deployment %q", e.Operation, e.NodeName, e.DeploymentId)
	ovPath, err := filepath.Abs(filepath.Join("work", "deployments", e.DeploymentId, "overlay"))
	if err != nil {
		return err
	}
	e.OverlayPath = ovPath
	e.NodePath = path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/nodes", e.NodeName)
	kvPair, _, err := e.kv.Get(e.NodePath+"/type", nil)
	if err != nil {
		return err
	}
	if kvPair == nil {
		return fmt.Errorf("type for node %s in deployment %s is missing", e.NodeName, e.DeploymentId)
	}

	e.NodeType = string(kvPair.Value)
	e.NodeTypePath = path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/types", e.NodeType)
	if strings.Contains(e.Operation, "Standard") {
		e.isRelationshipOperation = false
	} else {
		// In a relationship
		e.isRelationshipOperation = true
		opAndReq := strings.Split(e.Operation, "/")
		e.isRelationshipTargetNode = false
		if isTargetOperation(opAndReq[0]) {
			e.isRelationshipTargetNode = true
		}
		if len(opAndReq) == 2 {
			e.Operation = opAndReq[0]
			e.requirementIndex = opAndReq[1]

			reqPath := path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/nodes", e.NodeName, "requirements", e.requirementIndex)
			kvPair, _, err := e.kv.Get(path.Join(reqPath, "relationship"), nil)
			if err != nil {
				return err
			}
			if kvPair == nil || len(kvPair.Value) == 0 {
				return fmt.Errorf("Missing required parameter \"relationship\" for requirement at index %q for node %q in deployment %q.", e.requirementIndex, e.NodeName, e.DeploymentId)
			}
			e.relationshipType = string(kvPair.Value)
			kvPair, _, err = e.kv.Get(path.Join(reqPath, "node"), nil)
			if err != nil {
				return err
			}
			if kvPair == nil || len(kvPair.Value) == 0 {
				return fmt.Errorf("Missing required parameter \"node\" for requirement at index %q for node %q in deployment %q.", e.requirementIndex, e.NodeName, e.DeploymentId)
			}
			e.relationshipTargetName = string(kvPair.Value)
		}
		err = e.resolveIsPerInstanceOperation(opAndReq[0])
		if err != nil {
			return err
		}
	}

	//TODO deal with inheritance operation may be not in the direct node type
	if e.isRelationshipOperation {
		var op string
		if idx := strings.Index(e.Operation, "Configure."); idx >= 0 {
			op = e.Operation[idx:]
		} else if idx := strings.Index(e.Operation, "configure."); idx >= 0 {
			op = e.Operation[idx:]
		} else {
			op = strings.TrimPrefix(e.Operation, "tosca.interfaces.node.lifecycle.")
			op = strings.TrimPrefix(op, "tosca.interfaces.relationship.")
		}
		e.OperationPath = path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/types", e.relationshipType) + "/interfaces/" + strings.Replace(op, ".", "/", -1)
	} else {
		var op string
		if idx := strings.Index(e.Operation, "Standard."); idx >= 0 {
			op = e.Operation[idx:]
		} else if idx := strings.Index(e.Operation, "standard."); idx >= 0 {
			op = e.Operation[idx:]
		} else {
			op = strings.TrimPrefix(e.Operation, "tosca.interfaces.node.lifecycle.")
			op = strings.TrimPrefix(op, "tosca.interfaces.relationship.")
		}
		e.OperationPath = e.NodeTypePath + "/interfaces/" + strings.Replace(op, ".", "/", -1)
	}
	log.Debugf("Operation Path: %q", e.OperationPath)
	kvPair, _, err = e.kv.Get(e.OperationPath+"/implementation/primary", nil)
	if err != nil {
		return err
	}
	if kvPair == nil {
		return operationNotImplemented{msg: fmt.Sprintf("primary implementation missing for operation %q of type %q in deployment %q is missing", e.Operation, e.NodeType, e.DeploymentId)}
	}
	e.Primary = string(kvPair.Value)
	e.BasePrimary = path.Base(e.Primary)
	kvPair, _, err = e.kv.Get(e.OperationPath+"/implementation/dependencies", nil)
	if err != nil {
		return err
	}
	if kvPair == nil {
		return fmt.Errorf("dependencies implementation missing for type %s in deployment %s is missing", e.NodeType, e.DeploymentId)
	}
	e.Dependencies = strings.Split(string(kvPair.Value), ",")

	if err = e.resolveInputs(); err != nil {
		return err
	}
	if err = e.resolveArtifacts(); err != nil {
		return err
	}
	if e.isRelationshipTargetNode {
		err = e.resolveHosts(e.relationshipTargetName)
	} else {
		err = e.resolveHosts(e.NodeName)
	}
	if err != nil {
		return err
	}
	if err = e.resolveOperationOutput(); err != nil {
		return err
	}

	return e.resolveContext()

}

func (e *execution) execute(ctx context.Context, retry bool) error {
	if e.isPerInstanceOperation {
		var nodeName string
		if !e.isRelationshipTargetNode {
			nodeName = e.relationshipTargetName
		} else {
			nodeName = e.NodeName
		}
		instancesIds, err := deployments.GetNodeInstancesIds(e.kv, e.DeploymentId, nodeName)
		if err != nil {
			return err
		}
		for _, instanceId := range instancesIds {
			instanceName := getInstanceName(nodeName, instanceId)
			log.Debugf("Executing operation %q, on node %q, with current instance %q", e.Operation, e.NodeName, instanceName)
			err = e.executeWithCurrentInstance(ctx, retry, instanceName)
			if err != nil {
				return err
			}
		}
	} else {
		return e.executeWithCurrentInstance(ctx, retry, "")
	}
	return nil
}
func (e *execution) executeWithCurrentInstance(ctx context.Context, retry bool, currentInstance string) error {
	deployments.LogInConsul(e.kv, e.DeploymentId, "Start the ansible execution of : "+e.NodeName+" with operation : "+e.Operation)
	var ansibleRecipePath string
	if e.isRelationshipOperation {
		ansibleRecipePath = filepath.Join("work", "deployments", e.DeploymentId, "ansible", e.NodeName, e.relationshipType, e.Operation, currentInstance)
		e.OperationRemotePath = fmt.Sprintf(".janus/%s/%s/%s", e.NodeName, e.relationshipType, e.Operation)
	} else {
		ansibleRecipePath = filepath.Join("work", "deployments", e.DeploymentId, "ansible", e.NodeName, e.Operation, currentInstance)
		e.OperationRemotePath = fmt.Sprintf(".janus/%s/%s", e.NodeName, e.Operation)
	}
	ansibleRecipePath, err := filepath.Abs(ansibleRecipePath)
	if err != nil {
		return err
	}
	ansibleHostVarsPath := filepath.Join(ansibleRecipePath, "host_vars")
	if err := os.MkdirAll(ansibleHostVarsPath, 0775); err != nil {
		log.Printf("%+v", err)
		deployments.LogInConsul(e.kv, e.DeploymentId, fmt.Sprintf("%+v", err))
		return err
	}
	log.Debugf("Generating hosts files hosts: %+v ", e.hosts)
	var buffer bytes.Buffer
	buffer.WriteString("[all]\n")
	for instanceName, host := range e.hosts {
		buffer.WriteString(host.host)
		sshUser := host.user
		if sshUser == "" {
			// Thinking: should we have a default user
			return fmt.Errorf("DeploymentID: %q, NodeName: %q, Missing ssh user information", e.DeploymentId, e.NodeName)
		}
		buffer.WriteString(fmt.Sprintf(" ansible_ssh_user=%s ansible_ssh_private_key_file=~/.ssh/janus.pem ansible_ssh_common_args=\"-o ConnectionAttempts=20\"\n", sshUser))

		var perInstanceInputsBuffer bytes.Buffer
		for _, varInput := range e.VarInputsNames {
			if varInput == "INSTANCE" {
				perInstanceInputsBuffer.WriteString(fmt.Sprintf("INSTANCE: \"%s\"\n", instanceName))
			} else if varInput == "SOURCE_INSTANCE" {
				if !e.isPerInstanceOperation {
					perInstanceInputsBuffer.WriteString(fmt.Sprintf("SOURCE_INSTANCE: \"%s\"\n", instanceName))
				} else {
					if e.isRelationshipTargetNode {
						perInstanceInputsBuffer.WriteString(fmt.Sprintf("SOURCE_INSTANCE: \"%s\"\n", currentInstance))
					} else {
						perInstanceInputsBuffer.WriteString(fmt.Sprintf("SOURCE_INSTANCE: \"%s\"\n", instanceName))
					}
				}
			} else if varInput == "TARGET_INSTANCE" {
				if !e.isPerInstanceOperation {
					perInstanceInputsBuffer.WriteString(fmt.Sprintf("TARGET_INSTANCE: \"%s\"\n", instanceName))
				} else {
					if e.isRelationshipTargetNode {
						perInstanceInputsBuffer.WriteString(fmt.Sprintf("TARGET_INSTANCE: \"%s\"\n", instanceName))
					} else {
						perInstanceInputsBuffer.WriteString(fmt.Sprintf("TARGET_INSTANCE: \"%s\"\n", currentInstance))
					}
				}
			} else {
				for _, envInput := range e.EnvInputs {
					if envInput.Name == varInput && (envInput.InstanceName == instanceName || e.isPerInstanceOperation && envInput.InstanceName == currentInstance) {
						perInstanceInputsBuffer.WriteString(fmt.Sprintf("%s: \"%s\"\n", varInput, envInput.Value))
						goto NEXT
					}
				}
				if e.isRelationshipOperation {
					if hostedOn, err := deployments.IsNodeTypeDerivedFrom(e.kv, e.DeploymentId, e.relationshipType, "tosca.relationships.HostedOn"); err != nil {
						return err
					} else if hostedOn {
						// In case of operation for relationships derived from HostedOn we should match the inputs with the same instanceID
						instanceIdIdx := strings.LastIndex(instanceName, "_")
						// Get index
						if instanceIdIdx > 0 {
							instanceId := instanceName[instanceIdIdx:]
							for _, envInput := range e.EnvInputs {
								if envInput.Name == varInput && strings.HasSuffix(envInput.InstanceName, instanceId) {
									perInstanceInputsBuffer.WriteString(fmt.Sprintf("%s: \"%s\"\n", varInput, envInput.Value))
									goto NEXT
								}
							}
						}
					}
				}
				// Not found with the combination inputName/instanceName let's use the first that matches the input name
				for _, envInput := range e.EnvInputs {
					if envInput.Name == varInput {
						perInstanceInputsBuffer.WriteString(fmt.Sprintf("%s: \"%s\"\n", varInput, envInput.Value))
						goto NEXT
					}
				}
				return fmt.Errorf("Unable to find a suitable input for input name %q and instance %q", varInput, instanceName)
			}
		NEXT:
		}
		if perInstanceInputsBuffer.Len() > 0 {
			if err := ioutil.WriteFile(filepath.Join(ansibleHostVarsPath, host.host+".yml"), perInstanceInputsBuffer.Bytes(), 0664); err != nil {
				log.Printf("Failed to write vars for host %q file: %v", host, err)
				return err
			}
		}
	}
	if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "hosts"), buffer.Bytes(), 0664); err != nil {
		log.Print("Failed to write hosts file")
		deployments.LogInConsul(e.kv, e.DeploymentId, "Failed to write hosts file")
		return err
	}
	buffer.Reset()
	funcMap := template.FuncMap{
		// The name "path" is what the function will be called in the template text.
		"path": filepath.Dir,
		"abs":  filepath.Abs,
	}
	tmpl := template.New("execTemplate")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(funcMap)
	if e.HaveOutput {
		wrap_template := template.New("execTemplate")
		wrap_template = wrap_template.Delims("[[[", "]]]")
		wrap_template, err := tmpl.Parse(output_custom_wrapper)
		if err != nil {
			return err
		}
		var buffer bytes.Buffer
		if err := wrap_template.Execute(&buffer, e); err != nil {
			log.Print("Failed to Generate wrapper template")
			deployments.LogInConsul(e.kv, e.DeploymentId, "Failed to Generate wrapper template")
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "wrapper.sh"), buffer.Bytes(), 0664); err != nil {
			log.Print("Failed to write playbook file")
			deployments.LogInConsul(e.kv, e.DeploymentId, "Failed to write playbook file")
			return err
		}
	}
	tmpl, err = tmpl.Parse(ansible_playbook)
	if err := tmpl.Execute(&buffer, e); err != nil {
		log.Print("Failed to Generate ansible playbook template")
		deployments.LogInConsul(e.kv, e.DeploymentId, "Failed to Generate ansible playbook template")
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "run.ansible.yml"), buffer.Bytes(), 0664); err != nil {
		log.Print("Failed to write playbook file")
		deployments.LogInConsul(e.kv, e.DeploymentId, "Failed to write playbook file")
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "ansible.cfg"), []byte(strings.Replace(ansible_config, "#PLAY_PATH#", ansibleRecipePath, -1)), 0664); err != nil {
		log.Print("Failed to write ansible.cfg file")
		deployments.LogInConsul(e.kv, e.DeploymentId, "Failed to write ansible.cfg file")
		return err
	}
	scriptPath, err := filepath.Abs(filepath.Join(e.OverlayPath, e.Primary))
	if err != nil {
		return err
	}
	log.Printf("Ansible recipe for deployment with id %s: executing %q on remote host", e.DeploymentId, scriptPath)
	deployments.LogInConsul(e.kv, e.DeploymentId, fmt.Sprintf("Ansible recipe for deployment with id %s: executing %q on remote host", e.DeploymentId, scriptPath))
	var cmd *executil.Cmd
	var wrapperPath string
	if e.HaveOutput {
		wrapperPath, _ = filepath.Abs(ansibleRecipePath)
		cmd = executil.Command(ctx, "ansible-playbook", "-v", "-i", "hosts", "run.ansible.yml", "--extra-vars", fmt.Sprintf("script_to_run=%s , wrapper_location=%s/wrapper.sh , dest_folder=%s", scriptPath, wrapperPath, wrapperPath))
	} else {
		cmd = executil.Command(ctx, "ansible-playbook", "-i", "hosts", "run.ansible.yml", "--extra-vars", fmt.Sprintf("script_to_run=%s", scriptPath))
	}
	if _, err = os.Stat(filepath.Join(ansibleRecipePath, "run.ansible.retry")); retry && (err == nil || !os.IsNotExist(err)) {
		cmd.Args = append(cmd.Args, "--limit", filepath.Join("@", ansibleRecipePath, "run.ansible.retry"))
	}
	cmd.Dir = ansibleRecipePath
	outbuf := log.NewWriterSize(e.kv, e.DeploymentId, deployments.DeploymentKVPrefix)
	errbuf := log.NewWriterSize(e.kv, e.DeploymentId, deployments.DeploymentKVPrefix)
	cmd.Stdout = outbuf
	cmd.Stderr = errbuf

	errCloseCh := make(chan bool)
	defer close(errCloseCh)
	errbuf.Run(errCloseCh)
	defer outbuf.FlushSoftware()
	if err := cmd.Run(); err != nil {
		deployments.LogInConsul(e.kv, e.DeploymentId, err.Error())
		log.Print(err)
		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0

			// This works on both Unix and Windows. Although package
			// syscall is generally platform dependent, WaitStatus is
			// defined for both Unix and Windows and in both cases has
			// an ExitStatus() method with the same signature.
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				// Retry exit statuses 2 and 3
				if status.ExitStatus() == 2 || status.ExitStatus() == 3 {
					return ansibleRetriableError{root: err}
				}
			}

		}
		return err
	}

	if e.HaveOutput {
		fi, err := os.Open(filepath.Join(wrapperPath, "out.csv"))
		if err != nil {
			deployments.LogInConsul(e.kv, e.DeploymentId, err.Error())
			panic(err)
		}
		r := csv.NewReader(fi)
		records, err := r.ReadAll()
		if err != nil {
			deployments.LogInConsul(e.kv, e.DeploymentId, err.Error())
			log.Fatal(err)
		}
		for _, line := range records {
			storeConsulKey(e.kv, e.Output[line[0]], line[1])
		}
	}

	return nil
}

func storeConsulKey(kv *api.KV, key, value string) {
	// PUT a new KV pair
	p := &api.KVPair{Key: key, Value: []byte(value)}
	if _, err := kv.Put(p, nil); err != nil {
		log.Panic(err)
	}
}

func getInstanceName(nodeName, instanceId string) string {
	return sanitizeForShell(nodeName + "_" + instanceId)
}
