package ansible

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"strconv"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

const ansibleConfig = `[defaults]
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

// IsRetriable checks if a given error is an Ansible retriable error
func IsRetriable(err error) bool {
	_, ok := err.(ansibleRetriableError)
	return ok
}

// IsOperationNotImplemented checks if a given error is an error indicating that an operation is not implemented
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
	host       string
	user       string
	instanceID string
}

// An EnvInput represent a TOSCA operation input
//
// This element is exported in order to be used by text.Template but should be consider as internal
type EnvInput struct {
	Name           string
	Value          string
	InstanceName   string
	IsTargetScoped bool
}

func (ei EnvInput) String() string {
	return fmt.Sprintf("EnvInput: [Name: %q, Value: %q, InstanceName: %q, IsTargetScoped: \"%t\"]", ei.Name, ei.Value, ei.InstanceName, ei.IsTargetScoped)
}

type execution interface {
	resolveExecution() error
	execute(ctx context.Context, retry bool) error
}

type ansibleRunner interface {
	runAnsible(ctx context.Context, retry bool, currentInstance, ansibleRecipePath string) error
}
type executionCommon struct {
	kv                       *api.KV
	cfg                      config.Configuration
	deploymentID             string
	taskID                   string
	NodeName                 string
	Operation                string
	NodeType                 string
	Description              string
	OperationRemotePath      string
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
	IsCustomCommand          bool
	relationshipType         string
	relationshipTargetName   string
	requirementIndex         string
	ansibleRunner            ansibleRunner
	sourceNodeInstances      []string
	targetNodeInstances      []string
}

func newExecution(kv *api.KV, cfg config.Configuration, taskID, deploymentID, nodeName, operation string) (execution, error) {
	execCommon := &executionCommon{kv: kv,
		cfg:            cfg,
		deploymentID:   deploymentID,
		NodeName:       nodeName,
		Operation:      operation,
		VarInputsNames: make([]string, 0),
		EnvInputs:      make([]*EnvInput, 0),
		taskID:         taskID,
	}
	if err := execCommon.resolveOperation(); err != nil {
		return nil, err
	}
	// TODO: should use implementation artifacts (tosca.artifacts.Implementation.Bash, tosca.artifacts.Implementation.Python, tosca.artifacts.Implementation.Ansible...) in some way
	var exec execution
	if strings.HasSuffix(execCommon.BasePrimary, ".sh") || strings.HasSuffix(execCommon.BasePrimary, ".py") {
		execScript := &executionScript{executionCommon: execCommon}
		execCommon.ansibleRunner = execScript
		exec = execScript
	} else if strings.HasSuffix(execCommon.BasePrimary, ".yml") || strings.HasSuffix(execCommon.BasePrimary, ".yaml") {
		execAnsible := &executionAnsible{executionCommon: execCommon}
		execCommon.ansibleRunner = execAnsible
		exec = execAnsible
	} else {
		return nil, errors.Errorf("Unsupported artifact implementation for node: %q, operation: %q, primary implementation: %q", nodeName, operation, execCommon.Primary)
	}

	return exec, exec.resolveExecution()
}

func (e *executionCommon) resolveOperation() error {
	e.NodePath = path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "topology/nodes", e.NodeName)
	var err error
	e.NodeType, err = deployments.GetNodeType(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}
	e.NodeTypePath = path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "topology/types", e.NodeType)
	e.isRelationshipOperation, e.Operation, e.requirementIndex, e.relationshipTargetName, err = deployments.DecodeOperation(e.kv, e.deploymentID, e.NodeName, e.Operation)
	if err != nil {
		return err
	}
	if e.isRelationshipOperation {
		e.relationshipType, err = deployments.GetRelationshipForRequirement(e.kv, e.deploymentID, e.NodeName, e.requirementIndex)
		if err != nil {
			return err
		}

		e.isRelationshipTargetNode = isTargetOperation(e.Operation)

		err = e.resolveIsPerInstanceOperation(e.Operation)
		if err != nil {
			return err
		}

	} else if strings.Contains(e.Operation, "custom") {
		e.IsCustomCommand = true
	}

	operationNodeType := e.NodeType
	if e.isRelationshipOperation {
		operationNodeType = e.relationshipType
	}
	e.OperationPath, e.Primary, err = deployments.GetOperationPathAndPrimaryImplementationForNodeType(e.kv, e.deploymentID, operationNodeType, e.Operation)
	if err != nil {
		return err
	}
	if e.OperationPath == "" || e.Primary == "" {
		return operationNotImplemented{msg: fmt.Sprintf("primary implementation missing for operation %q of type %q in deployment %q is missing", e.Operation, e.NodeType, e.deploymentID)}
	}
	e.Primary = strings.TrimSpace(e.Primary)
	log.Debugf("Operation Path: %q, primary implementation: %q", e.OperationPath, e.Primary)
	e.BasePrimary = path.Base(e.Primary)
	kvPair, _, err := e.kv.Get(e.OperationPath+"/implementation/dependencies", nil)
	if err != nil {
		return err
	}

	if kvPair != nil {
		e.Dependencies = strings.Split(string(kvPair.Value), ",")
	} else {
		e.Dependencies = make([]string, 0)
	}
	kvPair, _, err = e.kv.Get(e.OperationPath+"/description", nil)
	if err != nil {
		return errors.Wrap(err, "Consul query failed: ")
	}
	if kvPair != nil && len(kvPair.Value) > 0 {
		e.Description = string(kvPair.Value)
	}

	return e.resolveInstances()
}

func (e *executionCommon) resolveInstances() error {
	var err error
	if e.isRelationshipOperation {
		e.targetNodeInstances, err = tasks.GetInstances(e.kv, e.taskID, e.deploymentID, e.relationshipTargetName)
		if err != nil {
			return err
		}
	}
	e.sourceNodeInstances, err = tasks.GetInstances(e.kv, e.taskID, e.deploymentID, e.NodeName)
	return err
}

func (e *executionCommon) resolveArtifacts() error {
	log.Debugf("Resolving artifacts")
	var err error
	if e.isRelationshipOperation {
		// First get linked node artifacts
		if e.isRelationshipTargetNode {
			e.Artifacts, err = deployments.GetArtifactsForNode(e.kv, e.deploymentID, e.relationshipTargetName)
			if err != nil {
				return err
			}
		} else {
			e.Artifacts, err = deployments.GetArtifactsForNode(e.kv, e.deploymentID, e.NodeName)
			if err != nil {
				return err
			}
		}
		// Then get relationship type artifacts
		var arts map[string]string
		arts, err = deployments.GetArtifactsForType(e.kv, e.deploymentID, e.relationshipType)
		if err != nil {
			return err
		}
		for artName, art := range arts {
			e.Artifacts[artName] = art
		}
	} else {
		e.Artifacts, err = deployments.GetArtifactsForNode(e.kv, e.deploymentID, e.NodeName)
		if err != nil {
			return err
		}
	}
	log.Debugf("Resolved artifacts: %v", e.Artifacts)
	return nil
}

func (e *executionCommon) resolveInputs() error {
	log.Debug("resolving inputs")
	resolver := deployments.NewResolver(e.kv, e.deploymentID)

	var inputKeys []string
	var err error

	inputKeys, _, err = e.kv.Keys(e.OperationPath+"/inputs/", "/", nil)

	if err != nil {
		return err
	}
	for _, input := range inputKeys {
		kvPair, _, err := e.kv.Get(input+"/name", nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvPair == nil {
			return errors.Errorf("%s/name missing", input)
		}
		inputName := string(kvPair.Value)

		kvPair, _, err = e.kv.Get(input+"/is_property_definition", nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		isPropDef, err := strconv.ParseBool(string(kvPair.Value))
		if err != nil {
			return err
		}

		va := tosca.ValueAssignment{}
		var targetContext bool
		if !isPropDef {
			kvPair, _, err = e.kv.Get(input+"/expression", nil)
			if err != nil {
				return err
			}
			if kvPair == nil {
				return errors.Errorf("%s/expression missing", input)
			}

			err = yaml.Unmarshal(kvPair.Value, &va)
			if err != nil {
				return errors.Wrap(err, "Failed to resolve operation inputs, unable to unmarshal yaml expression: ")
			}
			targetContext = va.Expression.IsTargetContext()
		}

		var instancesIds []string
		if targetContext {
			instancesIds = e.targetNodeInstances
		} else {
			instancesIds = e.sourceNodeInstances
		}
		var inputValue string
		for i, instanceID := range instancesIds {
			envI := &EnvInput{Name: inputName, IsTargetScoped: targetContext}
			if e.isRelationshipOperation && targetContext {
				envI.InstanceName = getInstanceName(e.relationshipTargetName, instanceID)
			} else {
				envI.InstanceName = getInstanceName(e.NodeName, instanceID)
			}
			if e.isRelationshipOperation {
				inputValue, err = resolver.ResolveExpressionForRelationship(va.Expression, e.NodeName, e.relationshipTargetName, e.requirementIndex, instanceID)
			} else if isPropDef {
				inputValue, err = tasks.GetTaskInput(e.kv, e.taskID, inputName)
			} else {
				inputValue, err = resolver.ResolveExpressionForNode(va.Expression, e.NodeName, instanceID)
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
	}

	log.Debugf("Resolved env inputs: %s", e.EnvInputs)
	return nil
}

func (e *executionCommon) resolveHosts(nodeName string) error {
	log.Debugf("Resolving hosts for node %q", nodeName)

	hosts := make(map[string]hostConnection)

	var instances []string
	if e.isRelationshipTargetNode {
		instances = e.targetNodeInstances
	} else {
		instances = e.sourceNodeInstances
	}

	for _, instance := range instances {
		found, ipAddress, err := deployments.GetInstanceCapabilityAttribute(e.kv, e.deploymentID, nodeName, instance, "endpoint", "ip_address")
		if err != nil {
			return err
		}
		if found && ipAddress != "" {
			var instanceName string
			if e.isRelationshipTargetNode {
				instanceName = getInstanceName(e.relationshipTargetName, instance)
			} else {
				instanceName = getInstanceName(e.NodeName, instance)
			}

			hostConn := hostConnection{host: ipAddress, instanceID: instance}
			var user string
			found, user, err = deployments.GetNodeProperty(e.kv, e.deploymentID, nodeName, "user")
			if err != nil {
				return err
			}
			if found && user != "" {
				va := tosca.ValueAssignment{}
				err = yaml.Unmarshal([]byte(user), &va)
				if err != nil {
					return errors.Wrapf(err, "Unable to resolve username to connect to host %q, unmarshaling yaml failed: ", nodeName)
				}
				hostConn.user, err = deployments.NewResolver(e.kv, e.deploymentID).ResolveExpressionForNode(va.Expression, nodeName, instance)
				if err != nil {
					return err
				}
			}
			hosts[instanceName] = hostConn
		}
	}
	if len(hosts) == 0 {
		// So we have to traverse the HostedOn relationships...
		hostedOnNode, err := deployments.GetHostedOnNode(e.kv, e.deploymentID, nodeName)
		if err != nil {
			return err
		}
		if hostedOnNode == "" {
			return errors.Errorf("Can't find an Host with an ip_address in the HostedOn hierarchy for node %q in deployment %q", e.NodeName, e.deploymentID)
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

func (e *executionCommon) resolveContext() error {
	execContext := make(map[string]string)

	newNode := sanitizeForShell(e.NodeName)
	if !e.isRelationshipOperation {
		execContext["NODE"] = newNode
	}
	var instances []string
	if e.isRelationshipTargetNode {
		instances = e.targetNodeInstances
	} else {
		instances = e.sourceNodeInstances
	}

	names := make([]string, len(instances))
	for i := range instances {
		instanceName := getInstanceName(e.NodeName, instances[i])
		names[i] = instanceName
	}
	if !e.isRelationshipOperation {
		e.VarInputsNames = append(e.VarInputsNames, "INSTANCE")
		execContext["INSTANCES"] = strings.Join(names, ",")
		if host, err := deployments.GetHostedOnNode(e.kv, e.deploymentID, e.NodeName); err != nil {
			return err
		} else if host != "" {
			execContext["HOST"] = host
		}
	} else {

		if host, err := deployments.GetHostedOnNode(e.kv, e.deploymentID, e.NodeName); err != nil {
			return err
		} else if host != "" {
			execContext["SOURCE_HOST"] = host
		}
		if host, err := deployments.GetHostedOnNode(e.kv, e.deploymentID, e.relationshipTargetName); err != nil {
			return err
		} else if host != "" {
			execContext["TARGET_HOST"] = host
		}
		execContext["SOURCE_NODE"] = newNode
		if e.isRelationshipTargetNode && !e.isPerInstanceOperation {
			execContext["SOURCE_INSTANCE"] = names[0]
		} else {
			e.VarInputsNames = append(e.VarInputsNames, "SOURCE_INSTANCE")
		}

		sourceNames := make([]string, len(e.sourceNodeInstances))
		for i := range e.sourceNodeInstances {
			sourceNames[i] = getInstanceName(e.NodeName, e.sourceNodeInstances[i])
		}
		execContext["SOURCE_INSTANCES"] = strings.Join(sourceNames, ",")
		execContext["TARGET_NODE"] = sanitizeForShell(e.relationshipTargetName)

		targetNames := make([]string, len(e.targetNodeInstances))
		for i := range e.targetNodeInstances {
			targetNames[i] = getInstanceName(e.relationshipTargetName, e.targetNodeInstances[i])
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

func (e *executionCommon) resolveOperationOutput() error {
	log.Debugf(e.OperationPath)
	log.Debugf(e.Operation)
	//We get all the output of the NodeType
	outputsPathList, _, err := e.kv.Keys(e.NodeTypePath+"/output/", "", nil)

	if err != nil {
		return err
	}

	output := make(map[string]string)

	//For each type we compare if we are in the good lifecycle operation
	for _, outputPath := range outputsPathList {
		tmp := strings.Split(e.Operation, ".")
		outOPPrefix := strings.ToLower(path.Join(e.NodeTypePath, "output", tmp[len(tmp)-2], tmp[len(tmp)-1]))
		if strings.Contains(strings.ToLower(outputPath), outOPPrefix) {
			e.HaveOutput = true
			kvp, _, err := e.kv.Get(outputPath, nil)
			if err != nil {
				return errors.Wrap(err, "Fail to get the attribute name")
			}
			output[path.Base(outputPath)] = "/attributes/" + string(kvp.Value)
		}
	}

	log.Debugf("Resolved outputs: %v", output)
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
func (e *executionCommon) resolveIsPerInstanceOperation(operationName string) error {
	op := strings.ToLower(operationName)
	if strings.Contains(op, "add_target") || strings.Contains(op, "remove_target") || strings.Contains(op, "target_changed") || strings.Contains(op, "add_source") {
		// Do not call the call the operation several time for an HostedOn relationship (makes no sense till we scale at compute level)
		if hostedOn, err := deployments.IsTypeDerivedFrom(e.kv, e.deploymentID, e.relationshipType, "tosca.relationships.HostedOn"); err != nil || hostedOn {
			e.isPerInstanceOperation = false
			return err
		}
		e.isPerInstanceOperation = true
		return nil
	}
	e.isPerInstanceOperation = false
	return nil
}

func (e *executionCommon) resolveExecution() error {
	log.Printf("Preparing execution of operation %q on node %q for deployment %q", e.Operation, e.NodeName, e.deploymentID)
	ovPath, err := filepath.Abs(filepath.Join(e.cfg.WorkingDirectory, "deployments", e.deploymentID, "overlay"))
	if err != nil {
		return err
	}
	e.OverlayPath = ovPath

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

func (e *executionCommon) execute(ctx context.Context, retry bool) error {
	if e.isPerInstanceOperation {
		var nodeName string
		var instances []string
		if !e.isRelationshipTargetNode {
			nodeName = e.relationshipTargetName
			instances = e.targetNodeInstances
		} else {
			nodeName = e.NodeName
			instances = e.sourceNodeInstances
		}

		for _, instanceID := range instances {
			instanceName := getInstanceName(nodeName, instanceID)
			log.Debugf("Executing operation %q, on node %q, with current instance %q", e.Operation, e.NodeName, instanceName)
			err := e.executeWithCurrentInstance(ctx, retry, instanceName)
			if err != nil {
				return err
			}
		}
	} else {
		return e.executeWithCurrentInstance(ctx, retry, "")
	}
	return nil
}

func (e *executionCommon) executeWithCurrentInstance(ctx context.Context, retry bool, currentInstance string) error {
	events.LogEngineMessage(e.kv, e.deploymentID, "Start the ansible execution of : "+e.NodeName+" with operation : "+e.Operation)
	var ansibleRecipePath string
	if e.isRelationshipOperation {
		ansibleRecipePath = filepath.Join(e.cfg.WorkingDirectory, "deployments", e.deploymentID, "ansible", e.NodeName, e.relationshipType, e.Operation, currentInstance)
	} else {
		ansibleRecipePath = filepath.Join(e.cfg.WorkingDirectory, "deployments", e.deploymentID, "ansible", e.NodeName, e.Operation, currentInstance)
	}
	ansibleRecipePath, err := filepath.Abs(ansibleRecipePath)
	if err != nil {
		return err
	}
	if err = os.RemoveAll(ansibleRecipePath); err != nil {
		err = errors.Wrapf(err, "Failed to remove ansible recipe directory %q for node %q operation %q", ansibleRecipePath, e.NodeName, e.Operation)
		log.Print(err)
		log.Debugf("%+v", err)
		events.LogEngineError(e.kv, e.deploymentID, err)
		return err
	}
	ansibleHostVarsPath := filepath.Join(ansibleRecipePath, "host_vars")
	if err = os.MkdirAll(ansibleHostVarsPath, 0775); err != nil {
		log.Printf("%+v", err)
		events.LogEngineError(e.kv, e.deploymentID, err)
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
			return errors.Errorf("DeploymentID: %q, NodeName: %q, Missing ssh user information", e.deploymentID, e.NodeName)
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
					var hostedOn bool
					hostedOn, err = deployments.IsTypeDerivedFrom(e.kv, e.deploymentID, e.relationshipType, "tosca.relationships.HostedOn")
					if err != nil {
						return err
					} else if hostedOn {
						// In case of operation for relationships derived from HostedOn we should match the inputs with the same instanceID
						instanceIDIdx := strings.LastIndex(instanceName, "_")
						// Get index
						if instanceIDIdx > 0 {
							instanceID := instanceName[instanceIDIdx:]
							for _, envInput := range e.EnvInputs {
								if envInput.Name == varInput && strings.HasSuffix(envInput.InstanceName, instanceID) {
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
				return errors.Errorf("Unable to find a suitable input for input name %q and instance %q", varInput, instanceName)
			}
		NEXT:
		}
		if perInstanceInputsBuffer.Len() > 0 {
			if err = ioutil.WriteFile(filepath.Join(ansibleHostVarsPath, host.host+".yml"), perInstanceInputsBuffer.Bytes(), 0664); err != nil {
				log.Printf("Failed to write vars for host %q file: %v", host, err)
				return err
			}
		}
	}

	if err = ioutil.WriteFile(filepath.Join(ansibleRecipePath, "hosts"), buffer.Bytes(), 0664); err != nil {
		log.Print("Failed to write hosts file")
		events.LogEngineMessage(e.kv, e.deploymentID, "Failed to write hosts file")
		return err
	}
	if err = ioutil.WriteFile(filepath.Join(ansibleRecipePath, "ansible.cfg"), []byte(strings.Replace(ansibleConfig, "#PLAY_PATH#", ansibleRecipePath, -1)), 0664); err != nil {
		log.Print("Failed to write ansible.cfg file")
		events.LogEngineMessage(e.kv, e.deploymentID, "Failed to write ansible.cfg file")
		return err
	}
	if e.isRelationshipOperation {
		e.OperationRemotePath = fmt.Sprintf(".janus/%s/%s/%s", e.NodeName, e.relationshipType, e.Operation)
	} else {
		e.OperationRemotePath = fmt.Sprintf(".janus/%s/%s", e.NodeName, e.Operation)
	}
	err = e.ansibleRunner.runAnsible(ctx, retry, currentInstance, ansibleRecipePath)
	if err != nil {
		return err
	}
	if e.HaveOutput {
		outputsFiles, err := filepath.Glob(filepath.Join(ansibleRecipePath, "*-out.csv"))
		if err != nil {
			err = errors.Wrapf(err, "Output retrieving of Ansible execution for node %q failed", e.NodeName)
			events.LogEngineError(e.kv, e.deploymentID, err)
			return err
		}
		for _, outFile := range outputsFiles {
			currentHost := strings.TrimSuffix(filepath.Base(outFile), "-out.csv")
			instanceID, err := e.getInstanceIDFromHost(currentHost)
			if err != nil {
				return err
			}
			fi, err := os.Open(outFile)
			if err != nil {
				err = errors.Wrapf(err, "Output retrieving of Ansible execution for node %q failed", e.NodeName)
				events.LogEngineError(e.kv, e.deploymentID, err)
				return err
			}
			r := csv.NewReader(fi)
			records, err := r.ReadAll()
			if err != nil {
				err = errors.Wrapf(err, "Output retrieving of Ansible execution for node %q failed", e.NodeName)
				events.LogEngineError(e.kv, e.deploymentID, err)
				return err
			}
			for _, line := range records {
				if err = consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "topology/instances", e.NodeName, instanceID, e.Output[line[0]]), line[1]); err != nil {
					return err
				}

			}
		}
	}
	return nil

}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func getInstanceName(nodeName, instanceID string) string {
	return sanitizeForShell(nodeName + "_" + instanceID)
}

func (e *executionCommon) checkAnsibleRetriableError(err error) error {
	events.LogEngineError(e.kv, e.deploymentID, errors.Wrapf(err, "Ansible execution for operation %q on node %q failed", e.Operation, e.NodeName))
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

func (e *executionCommon) getInstanceIDFromHost(host string) (string, error) {
	for _, hostConn := range e.hosts {
		if hostConn.host == host {
			return hostConn.instanceID, nil
		}
	}
	return "", errors.Errorf("Unknown host %q", host)
}
