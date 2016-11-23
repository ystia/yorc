package ansible

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/executil"
	"novaforge.bull.com/starlings-janus/janus/helper/logsutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

type executionAnsible struct {
	*executionCommon
}

func (e *executionAnsible) runAnsible(ctx context.Context, retry bool, currentInstance, ansibleRecipePath string) error {
	playbookPath, err := filepath.Abs(filepath.Join(e.OverlayPath, e.Primary))
	if err != nil {
		return err
	}

	ansibleGroupsVarsPath := filepath.Join(ansibleRecipePath, "group_vars")
	if err := os.MkdirAll(ansibleGroupsVarsPath, 0775); err != nil {
		log.Printf("%+v", err)
		deployments.LogErrorInConsul(e.kv, e.DeploymentId, err)
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
	if err := ioutil.WriteFile(filepath.Join(ansibleGroupsVarsPath, "all.yml"), buffer.Bytes(), 0664); err != nil {
		err = errors.Wrap(err, "Failed to write global group vars file: ")
		log.Printf("%v", err)
		log.Debugf("%+v", err)
		return err
	}

	log.Printf("Ansible recipe for deployment with id %q and node %q: executing %q on remote host(s)", e.DeploymentId, e.NodeName, playbookPath)
	deployments.LogInConsul(e.kv, e.DeploymentId, fmt.Sprintf("Ansible recipe for node %q: executing %q on remote host(s)", e.NodeName, filepath.Base(playbookPath)))
	cmd := executil.Command(ctx, "ansible-playbook", "-v", "-i", "hosts", playbookPath)
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

	return nil
}
