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

	"strings"

	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/executil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

const outputCustomWrapper = `
[[[printf ". $HOME/%s/%s" $.OperationRemotePath .BasePrimary]]]
[[[range $artName, $art := .Outputs -]]]
[[[printf "echo %s,$%s >> $HOME/%s/out.csv" $artName (cut $artName) $.OperationRemotePath]]]
[[[printf "echo $%s" $artName]]]
[[[end]]]
[[[printf "chmod 777 $HOME/%s/out.csv" $.OperationRemotePath]]]
`

const shellAnsiblePlaybook = `
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
    [[[printf "- file: path=\"{{ ansible_env.HOME}}/%s/%s\" state=directory mode=0755" $.OperationRemotePath (path $art)]]]
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
    [[[printf "- fetch: src={{ ansible_env.HOME}}/%s/out.csv dest={{dest_folder}}/{{ansible_host}}-out.csv flat=yes" $.OperationRemotePath]]]
    [[[end]]]
`

type executionScript struct {
	*executionCommon
}

func (e *executionScript) setOperationRemotePath(opPath string) {
	e.OperationRemotePath = opPath
}

func cutAfterLastUnderscore(str string) string {
	idx := strings.LastIndex(str, "_")
	return str[:idx]
}

func (e *executionScript) runAnsible(ctx context.Context, retry bool, currentInstance, ansibleRecipePath string) error {
	var buffer bytes.Buffer
	funcMap := template.FuncMap{
		// The name "path" is what the function will be called in the template text.
		"path": filepath.Dir,
		"abs":  filepath.Abs,
		"cut":  cutAfterLastUnderscore,
	}

	tmpl := template.New("execTemplate")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(funcMap)
	if e.HaveOutput {
		wrapTemplate := template.New("execTemplate")
		wrapTemplate = wrapTemplate.Delims("[[[", "]]]")
		wrapTemplate, err := tmpl.Parse(outputCustomWrapper)
		if err != nil {
			return err
		}
		if err := wrapTemplate.Execute(&buffer, e); err != nil {
			events.LogEngineMessage(e.kv, e.deploymentID, "Failed to Generate wrapper template")
			return errors.Wrap(err, "Failed to Generate wrapper template")
		}
		if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "wrapper.sh"), buffer.Bytes(), 0664); err != nil {
			events.LogEngineMessage(e.kv, e.deploymentID, "Failed to write playbook file")
			return errors.Wrap(err, "Failed to write playbook file")
		}
	}
	buffer.Reset()
	tmpl, err := tmpl.Parse(shellAnsiblePlaybook)
	if err != nil {
		return errors.Wrap(err, "Failed to generate ansible playbook")
	}
	if err = tmpl.Execute(&buffer, e); err != nil {
		events.LogEngineMessage(e.kv, e.deploymentID, "Failed to Generate ansible playbook template")
		return errors.Wrap(err, "Failed to Generate ansible playbook template")
	}
	if err = ioutil.WriteFile(filepath.Join(ansibleRecipePath, "run.ansible.yml"), buffer.Bytes(), 0664); err != nil {
		events.LogEngineMessage(e.kv, e.deploymentID, "Failed to write playbook file")
		return errors.Wrap(err, "Failed to write playbook file")
	}

	scriptPath, err := filepath.Abs(filepath.Join(e.OverlayPath, e.Primary))
	if err != nil {
		return err
	}
	log.Debugf("Ansible recipe for deployment with id %q and node %q: executing %q on remote host(s)", e.deploymentID, e.NodeName, scriptPath)
	events.LogEngineMessage(e.kv, e.deploymentID, fmt.Sprintf("Ansible recipe for node %q: executing %q on remote host(s)", e.NodeName, filepath.Base(scriptPath)))
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
