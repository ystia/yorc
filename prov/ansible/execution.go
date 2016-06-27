package ansible

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/consul/api"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

const ansible_playbook = `
- name: Executing script {{ script_to_run }}
  hosts: all
  tasks:
    - file: path="{{ ansible_env.HOME}}/.janus/[[[.NodeName]]]/[[[.Operation]]]" state=directory mode=0755
    - copy: src="{{ script_to_run }}" dest="{{ ansible_env.HOME}}/.janus/[[[.NodeName]]]/[[[.Operation]]]" mode=0744
    [[[ range $artName, $art := .Artifacts -]]]
    [[[printf "- copy: src=\"%s/%s\" dest=\"{{ ansible_env.HOME}}/.janus/%s/%s\"" $.OverlayPath $art $.NodeName $.Operation]]]
    [[[end]]]
    - shell: "{{ ansible_env.HOME}}/.janus/[[[.NodeName]]]/[[[.Operation]]]/[[[.BasePrimary]]]"
      environment:
        [[[ range $index, $input := .Inputs -]]]
        [[[print $input]]]
        [[[end]]][[[ range $artName, $art := .Artifacts -]]]
        [[[printf "%s: \"{{ ansible_env.HOME}}/.janus/%v/%v/%s\"" $artName $.NodeName $.Operation $art]]]
        [[[end]]]
`

const ansible_config = `[defaults]
host_key_checking=False
timeout=600
`

type execution struct {
	kv            *api.KV
	DeploymentId  string
	NodeName      string
	Operation     string
	NodeType      string
	Inputs        []string
	Primary       string
	BasePrimary   string
	Dependencies  []string
	Hosts         []string
	OperationPath string
	NodePath      string
	NodeTypePath  string
	Artifacts     map[string]string
	OverlayPath   string
}

func newExecution(kv *api.KV, deploymentId, nodeName, operation string) (*execution, error) {
	exec := &execution{kv: kv,
		DeploymentId: deploymentId,
		NodeName:     nodeName,
		Operation:    operation}
	return exec, exec.resolveExecution()
}

func (e *execution) resolveToscaFunction(function, nodePath, nodeTypePath string, params []string) (string, error) {

	kvPair, _, err := e.kv.Get(nodePath+"/"+function+"/"+params[1], nil)
	if err != nil {
		return "", err
	}
	if kvPair == nil {
		// Look for a default in node type
		// TODO deal with type inheritance
		kvPair, _, err = e.kv.Get(e.NodeTypePath+"/"+function+"/"+params[1]+"/default", nil)
		if err != nil {
			return "", err
		}
		if kvPair == nil || string(kvPair.Value) == "" {
			return "", fmt.Errorf("Can't retrieve %s %q for type %q either in node definition or node type default", function, params[1], params[0])
		}
	}
	return string(kvPair.Value), nil
}

func (e *execution) resolveExpression(node *tosca.TreeNode) (string, error) {
	log.Printf("Resolving expression %q", node.Value)
	if node.IsLiteral() {
		return node.Value, nil
	}
	params := make([]string, 0)
	for _, child := range node.Children() {
		exp, err := e.resolveExpression(child)
		if err != nil {
			return "", err
		}
		params = append(params, exp)
	}
	switch node.Value {
	case "get_property":
		if len(params) != 2 {
			return "", fmt.Errorf("get_property on requirement or capabability or in nested property is not yet supported")
		}
		switch params[0] {
		case "SELF":
			return e.resolveToscaFunction("properties", e.NodePath, e.NodeTypePath, params)
		case "SOURCE", "TARGET", "HOST":
			return "", fmt.Errorf("get_property on %q is not yet supported", params[0])
		default:
			nodePath := path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/nodes", params[0])
			kvPair, _, err := e.kv.Get(nodePath+"/type", nil)
			if err != nil {
				return "", err
			}
			if kvPair == nil {
				return "", fmt.Errorf("type for node %s in deployment %s is missing", params[0], e.DeploymentId)
			}
			nodeType := string(kvPair.Value)
			nodeTypePath := path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/types", nodeType)
			return e.resolveToscaFunction("properties", nodePath, nodeTypePath, params)
		}
	case "get_attribute":
		if len(params) != 2 {
			return "", fmt.Errorf("get_attribute on requirement or capabability or in nested property is not yet supported")
		}
		switch params[0] {
		case "SELF":
			return e.resolveToscaFunction("attributes", e.NodePath, e.NodeTypePath, params)
		case "SOURCE", "TARGET", "HOST":
			return "", fmt.Errorf("get_attribute on %q is not yet supported", params[0])
		default:
			nodePath := path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/nodes", params[0])
			kvPair, _, err := e.kv.Get(nodePath+"/type", nil)
			if err != nil {
				return "", err
			}
			if kvPair == nil {
				return "", fmt.Errorf("type for node %s in deployment %s is missing", params[0], e.DeploymentId)
			}
			nodeType := string(kvPair.Value)
			nodeTypePath := path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/types", nodeType)
			return e.resolveToscaFunction("attributes", nodePath, nodeTypePath, params)
		}
	case "concat":
		return strings.Join(params, ""), nil
	}
	return "", fmt.Errorf("Can't resolve expression %q", node.Value)
}

func (e *execution) resolveArtifacts() error {
	log.Printf("Resolving artifacts")
	artifacts := make(map[string]string)
	// First resolve node type artifacts then node template artifact if the is a conflict then node template will have the precedence
	// TODO deal with type inheritance
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
	log.Printf("Resolved artifacts: %v", e.Artifacts)
	return nil
}

func (e *execution) resolveInputs() error {
	log.Printf("resolving inputs")
	inputs := make([]string, 0)
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
		inputValue, err := e.resolveExpression(va.Expression)
		if err != nil {
			return err
		}
		inputs = append(inputs, inputName+": "+inputValue)
	}
	e.Inputs = inputs

	log.Printf("Resolved inputs: %v", e.Inputs)
	return nil
}

func (e *execution) resolveHosts(nodePath string) error {
	// e.nodePath
	log.Printf("Resolving hosts.")
	log.Printf("ip_address kv path %s/capabilities/endpoint/attributes/ip_address", nodePath)
	hosts := make([]string, 0)
	kvPair, _, err := e.kv.Get(nodePath+"/capabilities/endpoint/attributes/ip_address", nil)
	if err != nil {
		return err
	}
	if kvPair == nil {
		// Key not found look at host
		// TODO check TOSCA spec to know if we can rely on a requirement named 'host' in any cases
		kvPair, _, err = e.kv.Get(nodePath+"/requirements/host/node", nil)
		if err != nil {
			return err
		}
		if kvPair == nil {
			return fmt.Errorf("can't resolve attribute ip_address no more host to inspect")
		}
		return e.resolveHosts(path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/nodes", string(kvPair.Value)))
	}
	e.Hosts = append(hosts, string(kvPair.Value))
	log.Printf("Resolved hosts: %v", e.Hosts)
	return nil
}

func (e *execution) resolveExecution() error {
	log.Printf("Resolving execution for deployment %q / node %q / operation %q", e.DeploymentId, e.NodeName, e.Operation)
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
	//TODO deal with inheritance operation may be not in the direct node type
	e.NodeTypePath = path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/types", e.NodeType)

	e.OperationPath = e.NodeTypePath + "/interfaces/" + strings.Replace(strings.TrimPrefix(e.Operation, "tosca.interfaces.node.lifecycle."), ".", "/", -1)
	log.Printf("Operation Path: %q", e.OperationPath)
	kvPair, _, err = e.kv.Get(e.OperationPath+"/implementation/primary", nil)
	if err != nil {
		return err
	}
	if kvPair == nil {
		return fmt.Errorf("primary implementation missing for type %s in deployment %s is missing", e.NodeType, e.DeploymentId)
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

	return e.resolveHosts(e.NodePath)
}

func (e *execution) execute() error {

	ansibleRecipePath := filepath.Join("work", "deployments", e.DeploymentId, "ansible", e.NodeName, e.Operation)
	if err := os.MkdirAll(ansibleRecipePath, 0775); err != nil {
		log.Printf("%+v", err)
		return err
	}
	var buffer bytes.Buffer
	buffer.WriteString("[all]\n")
	for _, host := range e.Hosts {
		buffer.WriteString(host)
		// TODO should not be hard-coded
		buffer.WriteString(" ansible_ssh_user=cloud-user ansible_ssh_private_key_file=~/.ssh/janus.pem\n")
	}
	if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "hosts"), buffer.Bytes(), 0664); err != nil {
		log.Printf("Failed to write hosts file")
		return err
	}
	buffer.Reset()
	tmpl := template.New("execTemplate")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl, err := tmpl.Parse(ansible_playbook)
	if err := tmpl.Execute(&buffer, e); err != nil {
		log.Printf("Failed to Generate ansible playbook template")
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "run.ansible.yml"), buffer.Bytes(), 0664); err != nil {
		log.Printf("Failed to write playbook file")
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "ansible.cfg"), []byte(ansible_config), 0664); err != nil {
		log.Printf("Failed to write ansible.cfg file")
		return err
	}
	scriptPath, err := filepath.Abs(filepath.Join(e.OverlayPath, e.Primary))
	if err != nil {
		return err
	}
	log.Printf("Ansible recipe for deployment with id %s: executing %q on remote host", e.DeploymentId, scriptPath)
	cmd := exec.Command("ansible-playbook", "-v", "-i", "hosts", "run.ansible.yml", "--extra-vars", fmt.Sprintf("script_to_run=%s", scriptPath))
	cmd.Dir = ansibleRecipePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Print(err)
		return err
	}

	return cmd.Wait()
}
