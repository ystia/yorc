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
)

const ansible_playbook = `
- name: Executing script {{ script_to_run }}
  hosts: all
  tasks:
    - script: "{{ script_to_run }}"
      environment:
`

const ansible_config = `[defaults]
host_key_checking=False
timeout=600
`

type execution struct {
	kv            *api.KV
	deploymentId  string
	nodeName      string
	operation     string
	nodeType      string
	inputs        []string
	primary       string
	dependencies  []string
	hosts         []string
	operationPath string
	nodePath      string
	nodeTypePath  string
}

func newExecution(kv *api.KV, deploymentId, nodeName, operation string) (*execution, error) {
	exec := &execution{kv: kv,
		deploymentId: deploymentId,
		nodeName:     nodeName,
		operation:    operation}
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
		kvPair, _, err = e.kv.Get(e.nodeTypePath+"/"+function+"/"+params[1]+"/default", nil)
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
			return e.resolveToscaFunction("properties", e.nodePath, e.nodeTypePath, params)
		case "SOURCE", "TARGET", "HOST":
			return "", fmt.Errorf("get_property on %q is not yet supported", params[0])
		default:
			nodePath := path.Join(deployments.DeploymentKVPrefix, e.deploymentId, "topology/nodes", params[0])
			kvPair, _, err := e.kv.Get(nodePath+"/type", nil)
			if err != nil {
				return "", err
			}
			if kvPair == nil {
				return "", fmt.Errorf("type for node %s in deployment %s is missing", params[0], e.deploymentId)
			}
			nodeType := string(kvPair.Value)
			nodeTypePath := path.Join(deployments.DeploymentKVPrefix, e.deploymentId, "topology/types", nodeType)
			return e.resolveToscaFunction("properties", nodePath, nodeTypePath, params)
		}
	case "get_attribute":
		if len(params) != 2 {
			return "", fmt.Errorf("get_attribute on requirement or capabability or in nested property is not yet supported")
		}
		switch params[0] {
		case "SELF":
			return e.resolveToscaFunction("attributes", e.nodePath, e.nodeTypePath, params)
		case "SOURCE", "TARGET", "HOST":
			return "", fmt.Errorf("get_attribute on %q is not yet supported", params[0])
		default:
			nodePath := path.Join(deployments.DeploymentKVPrefix, e.deploymentId, "topology/nodes", params[0])
			kvPair, _, err := e.kv.Get(nodePath+"/type", nil)
			if err != nil {
				return "", err
			}
			if kvPair == nil {
				return "", fmt.Errorf("type for node %s in deployment %s is missing", params[0], e.deploymentId)
			}
			nodeType := string(kvPair.Value)
			nodeTypePath := path.Join(deployments.DeploymentKVPrefix, e.deploymentId, "topology/types", nodeType)
			return e.resolveToscaFunction("attributes", nodePath, nodeTypePath, params)
		}
	case "concat":
		return strings.Join(params, ""), nil
	}
	return "", fmt.Errorf("Can't resolve expression %q", node.Value)
}

func (e *execution) resolveInputs() error {
	log.Printf("resolving inputs")
	inputs := make([]string, 0)
	inputKeys, _, err := e.kv.Keys(e.operationPath+"/inputs/", "/", nil)
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
	e.inputs = inputs

	log.Printf("Resolved inputs: %v", e.inputs)
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
		return e.resolveHosts(path.Join(deployments.DeploymentKVPrefix, e.deploymentId, "topology/nodes", string(kvPair.Value)))
	}
	e.hosts = append(hosts, string(kvPair.Value))
	log.Printf("Resolved hosts: %v", e.hosts)
	return nil
}

func (e *execution) resolveExecution() error {
	log.Printf("Resolving execution for deployment %q / node %q / operation %q", e.deploymentId, e.nodeName, e.operation)
	e.nodePath = path.Join(deployments.DeploymentKVPrefix, e.deploymentId, "topology/nodes", e.nodeName)

	kvPair, _, err := e.kv.Get(e.nodePath+"/type", nil)
	if err != nil {
		return err
	}
	if kvPair == nil {
		return fmt.Errorf("type for node %s in deployment %s is missing", e.nodeName, e.deploymentId)
	}

	e.nodeType = string(kvPair.Value)
	//TODO deal with inheritance operation may be not in the direct node type
	e.nodeTypePath = path.Join(deployments.DeploymentKVPrefix, e.deploymentId, "topology/types", e.nodeType)

	e.operationPath = e.nodeTypePath + "/interfaces/" + strings.Replace(strings.TrimPrefix(e.operation, "tosca.interfaces.node.lifecycle."), ".", "/", -1)
	log.Printf("Operation Path: %q", e.operationPath)
	kvPair, _, err = e.kv.Get(e.operationPath+"/implementation/primary", nil)
	if err != nil {
		return err
	}
	if kvPair == nil {
		return fmt.Errorf("primary implementation missing for type %s in deployment %s is missing", e.nodeType, e.deploymentId)
	}
	e.primary = string(kvPair.Value)
	kvPair, _, err = e.kv.Get(e.operationPath+"/implementation/dependencies", nil)
	if err != nil {
		return err
	}
	if kvPair == nil {
		return fmt.Errorf("dependencies implementation missing for type %s in deployment %s is missing", e.nodeType, e.deploymentId)
	}
	e.dependencies = strings.Split(string(kvPair.Value), ",")

	if err = e.resolveInputs(); err != nil {
		return err
	}

	return e.resolveHosts(e.nodePath)
}

func (e *execution) execute() error {

	ansibleRecipePath := filepath.Join("work", "deployments", e.deploymentId, "ansible", e.nodeName, e.operation)
	overlayPath := filepath.Join("work", "deployments", e.deploymentId, "overlay")
	if err := os.MkdirAll(ansibleRecipePath, 0775); err != nil {
		log.Printf("%+v", err)
		return err
	}
	var buffer bytes.Buffer
	buffer.WriteString("[all]\n")
	for _, host := range e.hosts {
		buffer.WriteString(host)
		// TODO should not be hard-coded
		buffer.WriteString(" ansible_ssh_user=cloud-user ansible_ssh_private_key_file=~/.ssh/janus.pem\n")
	}
	if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "hosts"), buffer.Bytes(), 0664); err != nil {
		log.Printf("Failed to write hosts file")
		return err
	}
	buffer.Reset()
	buffer.WriteString(ansible_playbook)
	for _, input := range e.inputs {
		buffer.WriteString("        ")
		buffer.WriteString(input)
	}

	if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "run.ansible.yml"), buffer.Bytes(), 0664); err != nil {
		log.Printf("Failed to write playbook file")
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "ansible.cfg"), []byte(ansible_config), 0664); err != nil {
		log.Printf("Failed to write ansible.cfg file")
		return err
	}
	scriptPath, err := filepath.Abs(filepath.Join(overlayPath, e.primary))
	if err != nil {
		return err
	}
	log.Printf("Ansible recipe for deployment with id %s: executing %q on remote host", e.deploymentId, scriptPath)
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
