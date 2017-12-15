package terraform

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/helper/executil"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

type defaultExecutor struct {
	generator       commons.Generator
	preDestroyCheck commons.PreDestroyInfraCallback
}

// NewExecutor returns an Executor
func NewExecutor(generator commons.Generator, preDestroyCheck commons.PreDestroyInfraCallback) prov.DelegateExecutor {
	return &defaultExecutor{generator: generator, preDestroyCheck: preDestroyCheck}
}

func (e *defaultExecutor) ExecDelegate(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	consulClient, err := cfg.GetConsulClient()
	if err != nil {
		return err
	}
	kv := consulClient.KV()
	instances, err := tasks.GetInstances(kv, taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}

	// Fill log optional fields for log registration
	wfName, _ := tasks.GetTaskData(kv, taskID, "workflowName")
	logOptFields := events.LogOptionalFields{
		events.NodeID:        nodeName,
		events.WorkFlowID:    wfName,
		events.InterfaceName: "delegate",
		events.OperationName: delegateOperation,
	}

	op := strings.ToLower(delegateOperation)
	switch {
	case op == "install":
		err = e.installNode(ctx, kv, cfg, deploymentID, nodeName, instances, logOptFields)
	case op == "uninstall":
		err = e.uninstallNode(ctx, kv, cfg, deploymentID, nodeName, instances, logOptFields)
	default:
		return errors.Errorf("Unsupported operation %q", delegateOperation)
	}
	return err
}

func (e *defaultExecutor) installNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, instances []string, logOptFields events.LogOptionalFields) error {
	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateCreating)
		if err != nil {
			return err
		}
	}
	infraGenerated, outputs, env, err := e.generator.GenerateTerraformInfraForNode(ctx, cfg, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if infraGenerated {
		if err = e.applyInfrastructure(ctx, kv, cfg, deploymentID, nodeName, outputs, env, logOptFields); err != nil {
			return err
		}
	}
	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateStarted)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *defaultExecutor) uninstallNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, instances []string, logOptFields events.LogOptionalFields) error {
	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateDeleting)
		if err != nil {
			return err
		}
	}
	infraGenerated, outputs, env, err := e.generator.GenerateTerraformInfraForNode(ctx, cfg, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if infraGenerated {
		if err = e.destroyInfrastructure(ctx, kv, cfg, deploymentID, nodeName, outputs, env, logOptFields); err != nil {
			return err
		}
	}
	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateDeleted)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *defaultExecutor) remoteConfigInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, env []string, logOptFields events.LogOptionalFields) error {
	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString("Remote configuring the infrastructure")
	infraPath := filepath.Join(cfg.WorkingDirectory, "deployments", deploymentID, "infra", nodeName)
	cmd := executil.Command(ctx, "terraform", "init")
	cmd.Dir = infraPath
	cmd.Env = mergeEnvironments(env)
	errbuf := events.NewBufferedLogEntryWriter()
	out := events.NewBufferedLogEntryWriter()
	cmd.Stdout = out
	cmd.Stderr = errbuf

	quit := make(chan bool)
	defer close(quit)

	// Register log entries via stderr/stdout buffers
	events.WithOptionalFields(logOptFields).NewLogEntry(events.ERROR, deploymentID).RunBufferedRegistration(errbuf, quit)
	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RunBufferedRegistration(out, quit)

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "Failed to setup Consul remote backend for terraform")
	}

	return errors.Wrap(cmd.Wait(), "Failed to setup Consul remote backend for terraform")
}

func (e *defaultExecutor) retrieveOutputs(ctx context.Context, kv *api.KV, infraPath string, outputs map[string]string) error {
	if len(outputs) == 0 {
		return nil
	}
	type tfJSONOutput struct {
		Sensitive bool   `json:"sensitive,omitempty"`
		Type      string `json:"type,omitempty"`
		Value     string `json:"value,omitempty"`
	}
	type tfOutputsList map[string]tfJSONOutput

	cmd := executil.Command(ctx, "terraform", "output", "-json")
	cmd.Dir = infraPath
	result, err := cmd.Output()
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve the infrastructure outputs via terraform")
	}
	var outputsList tfOutputsList
	err = json.Unmarshal(result, &outputsList)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve the infrastructure outputs via terraform")
	}
	for outPath, outName := range outputs {
		output, ok := outputsList[outName]
		if !ok {
			return errors.Errorf("failed to retrieve output %q in terraform result", outName)
		}
		_, err = kv.Put(&api.KVPair{Key: outPath, Value: []byte(output.Value)}, nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
	}

	return nil
}

func (e *defaultExecutor) applyInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, outputs map[string]string, env []string, logOptFields events.LogOptionalFields) error {

	// Remote Configuration for Terraform State to store it in the Consul KV store
	if err := e.remoteConfigInfrastructure(ctx, kv, cfg, deploymentID, nodeName, env, logOptFields); err != nil {
		return err
	}

	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString("Applying the infrastructure")
	infraPath := filepath.Join(cfg.WorkingDirectory, "deployments", deploymentID, "infra", nodeName)
	cmd := executil.Command(ctx, "terraform", "apply")
	cmd.Dir = infraPath
	cmd.Env = mergeEnvironments(env)
	errbuf := events.NewBufferedLogEntryWriter()
	out := events.NewBufferedLogEntryWriter()
	cmd.Stdout = out
	cmd.Stderr = errbuf

	quit := make(chan bool)
	defer close(quit)

	// Register log entries via stderr/stdout buffers
	events.WithOptionalFields(logOptFields).NewLogEntry(events.ERROR, deploymentID).RunBufferedRegistration(errbuf, quit)
	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RunBufferedRegistration(out, quit)

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "Failed to apply the infrastructure changes via terraform")
	}

	return e.retrieveOutputs(ctx, kv, infraPath, outputs)

}

func (e *defaultExecutor) destroyInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, outputs map[string]string, env []string, logOptFields events.LogOptionalFields) error {
	if e.preDestroyCheck != nil {

		check, err := e.preDestroyCheck(ctx, kv, cfg, deploymentID, nodeName, logOptFields)
		if err != nil || !check {
			return err
		}
	}

	return e.applyInfrastructure(ctx, kv, cfg, deploymentID, nodeName, outputs, env, logOptFields)
}

// mergeEnvironments merges given env with current process env
// in commands if duplicates only the last one is taken into account
func mergeEnvironments(env []string) []string {
	return append(os.Environ(), env...)
}
