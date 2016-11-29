package ansible

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/helper/executil"
	"novaforge.bull.com/starlings-janus/janus/helper/logsutil"
	"novaforge.bull.com/starlings-janus/janus/log"
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
    [[[printf "- fetch: src={{ ansible_env.HOME}}/%s/out.csv dest={{dest_folder}} flat=yes" $.OperationRemotePath]]]
    [[[end]]]
`

type executionScript struct {
	OperationRemotePath string
	*executionCommon
}

func (e *executionScript) setOperationRemotePath(opPath string) {
	e.OperationRemotePath = opPath
}

func (e *executionScript) runAnsible(ctx context.Context, retry bool, currentInstance, ansibleRecipePath string) error {
	if e.isRelationshipOperation {
		e.OperationRemotePath = fmt.Sprintf(".janus/%s/%s/%s", e.NodeName, e.relationshipType, e.Operation)
	} else {
		e.OperationRemotePath = fmt.Sprintf(".janus/%s/%s", e.NodeName, e.Operation)
	}

	var buffer bytes.Buffer
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
	buffer.Reset()
	tmpl, err := tmpl.Parse(ansible_playbook)
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

	scriptPath, err := filepath.Abs(filepath.Join(e.OverlayPath, e.Primary))
	if err != nil {
		return err
	}
	log.Printf("Ansible recipe for deployment with id %q and node %q: executing %q on remote host(s)", e.DeploymentId, e.NodeName, scriptPath)
	deployments.LogInConsul(e.kv, e.DeploymentId, fmt.Sprintf("Ansible recipe for node %q: executing %q on remote host(s)", e.NodeName, filepath.Base(scriptPath)))
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
	errbuf := logsutil.NewBufferedConsulWriter(e.kv, e.DeploymentId, deployments.SOFTWARE_LOG_PREFIX)
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

	if e.HaveOutput {
		fi, err := os.Open(filepath.Join(wrapperPath, "out.csv"))
		if err != nil {
			err = errors.Wrapf(err, "Output retrieving of Ansible execution for node %q failed", e.NodeName)
			deployments.LogErrorInConsul(e.kv, e.DeploymentId, err)
			return err
		}
		r := csv.NewReader(fi)
		records, err := r.ReadAll()
		if err != nil {
			err = errors.Wrapf(err, "Output retrieving of Ansible execution for node %q failed", e.NodeName)
			deployments.LogErrorInConsul(e.kv, e.DeploymentId, err)
			return err
		}
		for _, line := range records {
			if err = consulutil.StoreConsulKeyAsString(e.Output[line[0]], line[1]); err != nil {
				return err
			}

		}
	}
	return nil
}
