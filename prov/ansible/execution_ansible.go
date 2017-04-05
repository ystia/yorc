package ansible

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"

	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/executil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

const ansiblePlaybook = `
- include: [[[.PlaybookPath]]]
[[[if .HaveOutput]]]
- name: Retrieving Operation outputs
  hosts: all
  strategy: free
  tasks:
    [[[printf "- file: path=\"{{ ansible_env.HOME}}/%s\" state=directory mode=0755" $.OperationRemotePath]]]
    [[[printf "- template: src=\"outputs.csv.j2\" dest=\"{{ ansible_env.HOME}}/%s/out.csv\"" $.OperationRemotePath]]]
    [[[printf "- fetch: src=\"{{ ansible_env.HOME}}/%s/out.csv\" dest={{dest_folder}}/{{ansible_host}}-out.csv flat=yes" $.OperationRemotePath]]]
[[[end]]]
`

type executionAnsible struct {
	*executionCommon
	PlaybookPath string
}

func (e *executionAnsible) runAnsible(ctx context.Context, retry bool, currentInstance, ansibleRecipePath string) error {
	var err error
	e.PlaybookPath, err = filepath.Abs(filepath.Join(e.OverlayPath, e.Primary))
	if err != nil {
		return err
	}

	ansibleGroupsVarsPath := filepath.Join(ansibleRecipePath, "group_vars")
	if err = os.MkdirAll(ansibleGroupsVarsPath, 0775); err != nil {
		log.Printf("%+v", err)
		events.LogEngineError(e.kv, e.deploymentID, err)
		return errors.Wrap(err, "Failed to create group_vars directory: ")
	}
	var buffer bytes.Buffer
	for _, envInput := range e.EnvInputs {
		if envInput.InstanceName != "" {
			buffer.WriteString(envInput.InstanceName)
			buffer.WriteString("_")
		}
		buffer.WriteString(envInput.Name)
		buffer.WriteString(": \"")
		buffer.WriteString(envInput.Value)
		buffer.WriteString("\"\n")
	}
	for contextKey, contextValue := range e.Context {
		buffer.WriteString(contextKey)
		buffer.WriteString(": \"")
		buffer.WriteString(contextValue)
		buffer.WriteString("\"\n")
	}
	buffer.WriteString("dest_folder: ")
	buffer.WriteString(ansibleRecipePath)
	buffer.WriteString("\n")

	if err = ioutil.WriteFile(filepath.Join(ansibleGroupsVarsPath, "all.yml"), buffer.Bytes(), 0664); err != nil {
		err = errors.Wrap(err, "Failed to write global group vars file: ")
		log.Printf("%v", err)
		log.Debugf("%+v", err)
		return err
	}

	if e.HaveOutput {
		buffer.Reset()
		for attrName, outputName := range e.Output {
			buffer.WriteString(path.Base(attrName))
			buffer.WriteString(",{{")
			buffer.WriteString(outputName)
			buffer.WriteString("}}\n")
		}
		if err = ioutil.WriteFile(filepath.Join(ansibleRecipePath, "outputs.csv.j2"), buffer.Bytes(), 0664); err != nil {
			err = errors.Wrap(err, "Failed to generate operation outputs file: ")
			log.Printf("%v", err)
			log.Debugf("%+v", err)
			return err
		}
	}

	buffer.Reset()
	tmpl := template.New("execTemplate")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl, err = tmpl.Parse(ansiblePlaybook)
	if err != nil {
		return errors.Wrap(err, "Failed to generate ansible playbook")
	}
	if err = tmpl.Execute(&buffer, e); err != nil {
		log.Print("Failed to Generate ansible playbook template")
		events.LogEngineMessage(e.kv, e.deploymentID, "Failed to Generate ansible playbook template")
		return err
	}
	if err = ioutil.WriteFile(filepath.Join(ansibleRecipePath, "run.ansible.yml"), buffer.Bytes(), 0664); err != nil {
		log.Print("Failed to write playbook file")
		events.LogEngineMessage(e.kv, e.deploymentID, "Failed to write playbook file")
		return err
	}

	log.Printf("Ansible recipe for deployment with id %q and node %q: executing %q on remote host(s)", e.deploymentID, e.NodeName, e.PlaybookPath)
	events.LogEngineMessage(e.kv, e.deploymentID, fmt.Sprintf("Ansible recipe for node %q: executing %q on remote host(s)", e.NodeName, filepath.Base(e.PlaybookPath)))
	cmd := executil.Command(ctx, "ansible-playbook", "-i", "hosts", "run.ansible.yml")

	if _, err = os.Stat(filepath.Join(ansibleRecipePath, "run.ansible.retry")); retry && (err == nil || !os.IsNotExist(err)) {
		cmd.Args = append(cmd.Args, "--limit", filepath.Join("@", ansibleRecipePath, "run.ansible.retry"))
	}
	cmd.Dir = ansibleRecipePath
	var outbuf bytes.Buffer
	errbuf := events.NewBufferedLogEventWriter(e.kv, e.deploymentID, events.SoftwareLogPrefix)
	cmd.Stdout = &outbuf
	cmd.Stderr = errbuf

	errCloseCh := make(chan bool)
	defer close(errCloseCh)
	errbuf.Run(errCloseCh)
	defer func(buffer *bytes.Buffer) {
		if err := e.logAnsibleOutputInConsul(buffer); err != nil {
			log.Printf("Failed to publish Ansible log %v", err)
			log.Debugf("%+v", err)
		}
	}(&outbuf)
	if err := cmd.Run(); err != nil {
		return e.checkAnsibleRetriableError(err)
	}

	return nil
}
