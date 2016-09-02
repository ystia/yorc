package ansible

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const ANSIBLE_OUTPUT_JSON_LOCATION = "plays.#.tasks.#.hosts.*.stdout"

const output_custom_wrapper = `
[[[printf ". $HOME/.janus/%s/%s/%s" .NodeName .Operation .BasePrimary]]]
[[[range $artName, $art := .Output -]]]
[[[printf "echo %s,$%s >> $HOME/out.csv" $artName $artName]]]
[[[printf "chmod 777 $HOME/out.csv" ]]]
[[[printf "echo $%s" $artName]]]
[[[end]]]
`

const ansible_playbook = `
- name: Executing script {{ script_to_run }}
  hosts: all
  tasks:
    - file: path="{{ ansible_env.HOME}}/.janus/[[[.NodeName]]]/[[[.Operation]]]" state=directory mode=0755
    [[[if .HaveOutput]]]
    [[[printf  "- copy: src=\"{{ wrapper_location }}\" dest=\"{{ ansible_env.HOME}}/.janus/wrapper.sh\" mode=0744" ]]]
    [[[end]]]
    - copy: src="{{ script_to_run }}" dest="{{ ansible_env.HOME}}/.janus/[[[.NodeName]]]/[[[.Operation]]]" mode=0744
    [[[ range $artName, $art := .Artifacts -]]]
    [[[printf "- copy: src=\"%s/%s\" dest=\"{{ ansible_env.HOME}}/.janus/%s/%s/%s\"" $.OverlayPath $art $.NodeName $.Operation (path $art)]]]
    [[[end]]]
    [[[if not .HaveOutput]]]
    [[[printf "- shell: \"{{ ansible_env.HOME}}/.janus/%s/%s/%s\"" .NodeName .Operation .BasePrimary]]][[[else]]]
    [[[printf "- shell: \"/bin/bash -l {{ ansible_env.HOME}}/.janus/wrapper.sh\""]]][[[end]]]
      environment:
        [[[ range $key, $input := .Inputs -]]]
        [[[ if (len $input) gt 0]]][[[printf  "%s: %s" $key $input]]][[[else]]]
        [[[printf  "%s: \"\"" $key]]]
        [[[end]]]
        [[[end]]][[[ range $artName, $art := .Artifacts -]]]
        [[[printf "%s: \"{{ ansible_env.HOME}}/.janus/%v/%v/%s\"" $artName $.NodeName $.Operation $art]]]
        [[[end]]][[[ range $contextK, $contextV := .Context -]]]
        [[[printf "%s: %s" $contextK $contextV]]]
        [[[end]]]
    [[[if .HaveOutput]]]
    [[[printf "- fetch: src={{ ansible_env.HOME}}/out.csv dest={{dest_folder}} flat=yes" ]]]
    [[[end]]]
`

const ansible_config = `[defaults]
host_key_checking=False
timeout=600
stdout_callback = json
`

type execution struct {
	kv            *api.KV
	DeploymentId  string
	NodeName      string
	Operation     string
	NodeType      string
	Inputs        map[string]string
	Primary       string
	BasePrimary   string
	Dependencies  []string
	Hosts         []string
	OperationPath string
	NodePath      string
	NodeTypePath  string
	Artifacts     map[string]string
	OverlayPath   string
	Context       map[string]string
	Output        map[string]string
	HaveOutput    bool
}

func newExecution(kv *api.KV, deploymentId, nodeName, operation string) (*execution, error) {
	exec := &execution{kv: kv,
		DeploymentId: deploymentId,
		NodeName:     nodeName,
		Operation:    operation}
	return exec, exec.resolveExecution()
}

type BufferedConsulWriter struct {
	kv    *api.KV
	depId string
	buf   []byte
	io.Writer
}

func NewWriterSize(api *api.KV, depId string) *BufferedConsulWriter {
	return &BufferedConsulWriter{
		buf:   make([]byte, 1),
		kv:    api,
		depId: depId,
	}
}

func (b *BufferedConsulWriter) Write(p []byte) (nn int, err error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *BufferedConsulWriter) Flush() error {
	//fmt.Printf(string(p))
	if gjson.Get(string(b.buf), ANSIBLE_OUTPUT_JSON_LOCATION).Exists() {
		out := gjson.Get(string(b.buf), ANSIBLE_OUTPUT_JSON_LOCATION).String()
		out = strings.TrimPrefix(out, "[[,")
		out = strings.TrimSuffix(out, "]]")
		out, err := strconv.Unquote(out)
		if err != nil {
			return err
		}
		kv := &api.KVPair{Key: filepath.Join(deployments.DeploymentKVPrefix, b.depId, "logs", "ansible", time.Now().Format(time.RFC3339Nano)), Value: []byte(out)}
		_, err = b.kv.Put(kv, nil)
		if err != nil {
			return err
		}
	}
	return nil

}

func (e *execution) resolveArtifacts() error {
	log.Debugf("Resolving artifacts")
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
	log.Debugf("Resolved artifacts: %v", e.Artifacts)
	return nil
}

func (e *execution) isRelationship() bool {
	if strings.Contains(e.OperationPath, "relationship") {
		return true
	}
	return false
}

func (e *execution) resolveInputs() error {
	log.Debug("resolving inputs")
	resolver := deployments.NewResolver(e.kv, e.DeploymentId, e.NodePath, e.NodeTypePath)
	inputs := make(map[string]string)
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
		inputValue, err := resolver.ResolveExpression(va.Expression, e.isRelationship())
		if err != nil {
			return err
		}
		inputs[inputName] = inputValue
	}
	e.Inputs = inputs

	log.Debugf("Resolved inputs: %v", e.Inputs)
	return nil
}

func (e *execution) resolveHosts(nodePath string) error {
	// e.nodePath
	log.Debugf("Resolving hosts.")
	log.Debugf("ip_address kv path %s/capabilities/endpoint/attributes/ip_address", nodePath)
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
	log.Debugf("Resolved hosts: %v", e.Hosts)
	return nil
}

func (e *execution) ReplaceMinus(str string) string {
	return strings.Replace(str, "-", "_", -1)
}

func (e *execution) resolveContext() error {
	metatype, _, err := e.kv.Get(path.Join(e.NodeTypePath, "metatype"), nil)
	if err != nil {
		return err
	}
	context := make(map[string]string)

	//TODO: Need to be improved with the runtime (INSTANCE,INSTANCES)
	new_node := e.ReplaceMinus(e.NodeName)
	context["NODE"] = new_node
	context["INSTANCE"] = new_node
	context["INSTANCES"] = new_node
	context["HOST"] = e.Hosts[0]

	if strings.Contains(string(metatype.Value), "Relationship") {

		context["SOURCE_NODE"] = new_node
		context["SOURCE_INSTANCE"] = new_node
		context["SOURCE_INSTANCES"] = new_node

		require, _, err := e.kv.Keys(filepath.Join(e.NodePath, "requirements"), "", nil)
		if err != nil {
			return err
		}
		for _, path := range require {
			if !strings.HasSuffix(path, "node") {
				continue
			}
			splitedPath2 := strings.Split(e.NodePath, "/")
			kvPair, _, _ := e.kv.Get(path, nil)
			log.Debugf(string(kvPair.Value))
			if !strings.Contains(string(kvPair.Value), "Storage") {

				nodePath := strings.Replace(e.NodePath, splitedPath2[len(splitedPath2)-1], string(kvPair.Value), -1)

				context["TARGET_NODE"] = e.ReplaceMinus(filepath.Base(nodePath))
				context["TARGET_INSTANCE"] = e.ReplaceMinus(filepath.Base(nodePath))
				context["TARGET_INSTANCES"] = e.ReplaceMinus(filepath.Base(nodePath))

				resolver := deployments.NewResolver(e.kv, e.DeploymentId, e.NodePath, e.NodeTypePath)
				params := make([]string, 0)
				params = append(params, "", "ip_address")
				nodePath2, err := resolver.FindInHost(nodePath, e.NodeTypePath, "capabilities/endpoint/attributes", params)

				if err != nil {
					return err
				}

				ip_addr, _, err := e.kv.Get(nodePath2+"/capabilities/endpoint/attributes/ip_address", nil)

				if err != nil {
					return err
				}

				context["TARGET_IP"] = string(ip_addr.Value)
				context[e.ReplaceMinus(filepath.Base(nodePath))+"_TARGET_IP"] = string(ip_addr.Value)
			}
		}

	}

	e.Context = context

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

func (e *execution) resolveExecution() error {
	log.Printf("Preparing execution of operation %q on node %q for deployment %q", e.Operation, e.NodeName, e.DeploymentId)
	ovPath, err := filepath.Abs(filepath.Join("work", "deployments", e.DeploymentId, "overlay"))
	if err != nil {
		return err
	}
	e.OverlayPath = ovPath
	e.NodePath = path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/nodes", e.NodeName)
	if strings.Contains(e.Operation, "Standard") {
		kvPair, _, err := e.kv.Get(e.NodePath+"/type", nil)
		if err != nil {
			return err
		}
		if kvPair == nil {
			return fmt.Errorf("type for node %s in deployment %s is missing", e.NodeName, e.DeploymentId)
		}

		e.NodeType = string(kvPair.Value)
		e.NodeTypePath = path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/types", e.NodeType)
	} else {
		kvPair, _, err := e.kv.Keys(path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/nodes", e.NodeName), "", nil)
		if err != nil {
			return err
		}
		for _, key := range kvPair {
			if strings.Contains(key, "relationship") && !strings.Contains(key, "host") {
				kvPair, _, err := e.kv.Get(path.Join(key), nil)
				if err != nil {
					return err
				}
				e.NodeType = string(kvPair.Value)
				e.NodeTypePath = path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "topology/types", string(kvPair.Value))
			}
		}
	}

	//TODO deal with inheritance operation may be not in the direct node type

	e.OperationPath = e.NodeTypePath + "/interfaces/" + strings.Replace(strings.TrimPrefix(e.Operation, "tosca.interfaces.node.lifecycle."), ".", "/", -1)
	log.Debugf("Operation Path: %q", e.OperationPath)
	kvPair, _, err := e.kv.Get(e.OperationPath+"/implementation/primary", nil)
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
	if err = e.resolveHosts(e.NodePath); err != nil {
		return err
	}
	if err = e.resolveOperationOutput(); err != nil {
		return err
	}

	return e.resolveContext()

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
		log.Print("Failed to write hosts file")
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
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "wrapper.sh"), buffer.Bytes(), 0664); err != nil {
			log.Print("Failed to write playbook file")
			return err
		}
	}
	tmpl, err := tmpl.Parse(ansible_playbook)
	if err := tmpl.Execute(&buffer, e); err != nil {
		log.Print("Failed to Generate ansible playbook template")
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "run.ansible.yml"), buffer.Bytes(), 0664); err != nil {
		log.Print("Failed to write playbook file")
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "ansible.cfg"), []byte(ansible_config), 0664); err != nil {
		log.Print("Failed to write ansible.cfg file")
		return err
	}
	scriptPath, err := filepath.Abs(filepath.Join(e.OverlayPath, e.Primary))
	if err != nil {
		return err
	}
	log.Printf("Ansible recipe for deployment with id %s: executing %q on remote host", e.DeploymentId, scriptPath)
	var cmd *exec.Cmd
	var wrapperPath string
	if e.HaveOutput {
		wrapperPath, _ = filepath.Abs(ansibleRecipePath)
		cmd = exec.Command("ansible-playbook", "-v", "-i", "hosts", "run.ansible.yml", "--extra-vars", fmt.Sprintf("script_to_run=%s , wrapper_location=%s/wrapper.sh , dest_folder=%s", scriptPath, wrapperPath, wrapperPath))
	} else {
		cmd = exec.Command("ansible-playbook", "-i", "hosts", "run.ansible.yml", "--extra-vars", fmt.Sprintf("script_to_run=%s", scriptPath))
	}
	cmd.Dir = ansibleRecipePath
	outbuf := NewWriterSize(e.kv, e.DeploymentId)
	errbuf := NewWriterSize(e.kv, e.DeploymentId)
	cmd.Stdout = outbuf
	cmd.Stderr = errbuf

	if e.HaveOutput {
		if err := cmd.Run(); err != nil {
			log.Print(err)
			return err
		}
		fi, err := os.Open(filepath.Join(wrapperPath, "out.csv"))
		if err != nil {
			panic(err)
		}
		r := csv.NewReader(fi)
		records, err := r.ReadAll()
		if err != nil {
			log.Fatal(err)
		}
		for _, line := range records {
			storeConsulKey(e.kv, e.Output[line[0]], line[1])
		}
		return nil

	} else {
		if err := cmd.Start(); err != nil {
			log.Print(err)
			return err
		}
	}

	err = cmd.Wait()

	outbuf.Flush()

	return err
}

func storeConsulKey(kv *api.KV, key, value string) {
	// PUT a new KV pair
	p := &api.KVPair{Key: key, Value: []byte(value)}
	if _, err := kv.Put(p, nil); err != nil {
		log.Panic(err)
	}
}
