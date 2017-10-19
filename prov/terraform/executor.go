package terraform

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/helper/executil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

type defaultExecutor struct {
	generator commons.Generator
}

// NewExecutor returns an Executor
func NewExecutor(generator commons.Generator) prov.DelegateExecutor {
	return &defaultExecutor{generator}
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

	op := strings.ToLower(delegateOperation)
	switch {
	case op == "install":
		err = e.installNode(ctx, kv, cfg, deploymentID, nodeName, instances)
	case op == "uninstall":
		err = e.uninstallNode(ctx, kv, cfg, deploymentID, nodeName, instances)
	default:
		return errors.Errorf("Unsupported operation %q", delegateOperation)
	}
	return err
}

func (e *defaultExecutor) installNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, instances []string) error {
	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateCreating)
		if err != nil {
			return err
		}
	}
	infraGenerated, outputs, err := e.generator.GenerateTerraformInfraForNode(ctx, cfg, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if infraGenerated {
		if err = e.applyInfrastructure(ctx, kv, cfg, deploymentID, nodeName, outputs); err != nil {
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

func (e *defaultExecutor) uninstallNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, instances []string) error {
	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateDeleting)
		if err != nil {
			return err
		}
	}
	infraGenerated, outputs, err := e.generator.GenerateTerraformInfraForNode(ctx, cfg, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if infraGenerated {
		if err = e.destroyInfrastructure(ctx, kv, cfg, deploymentID, nodeName, outputs); err != nil {
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

func (e *defaultExecutor) remoteConfigInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string) error {
	// Fill log optional fields for log registration
	logOptFields := events.OptionalFields{
		events.NodeID:      nodeName,
		events.WorkFlowID:  "TODO",
		events.InterfaceID: "delegate",
		events.OperationID: "TODO",
	}

	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString("Remote configuring the infrastructure")
	infraPath := filepath.Join(cfg.WorkingDirectory, "deployments", deploymentID, "infra", nodeName)
	cmd := executil.Command(ctx, "terraform", "init")
	cmd.Dir = infraPath
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
	// This could be optimized by generating a json output for all outputs and then look at it for given outputs
	for outPath, outName := range outputs {
		cmd := executil.Command(ctx, "terraform", "output", outName)
		cmd.Dir = infraPath
		result, err := cmd.Output()
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve the infrastructure outputs via terraform")
		}

		_, err = kv.Put(&api.KVPair{Key: outPath, Value: result}, nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
	}

	return nil
}

func (e *defaultExecutor) applyInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, outputs map[string]string) error {

	// Remote Configuration for Terraform State to store it in the Consul KV store
	if err := e.remoteConfigInfrastructure(ctx, kv, cfg, deploymentID, nodeName); err != nil {
		return err
	}

	// Register log entry via error buffer
	logOptFields := events.OptionalFields{
		events.NodeID: nodeName,
	}

	events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString("Applying the infrastructure")
	infraPath := filepath.Join(cfg.WorkingDirectory, "deployments", deploymentID, "infra", nodeName)
	cmd := executil.Command(ctx, "terraform", "apply")
	cmd.Dir = infraPath
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

func (e *defaultExecutor) destroyInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, outputs map[string]string) error {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	// TODO  consider making this generic: references to OpenStack should not be found here.
	if nodeType == "janus.nodes.openstack.BlockStorage" {
		var deletable string
		var found bool
		found, deletable, err = deployments.GetNodeProperty(kv, deploymentID, nodeName, "deletable")
		if err != nil {
			return err
		}
		if !found || strings.ToLower(deletable) != "true" {
			// False by default
			log.Debugf("Node %q is a BlockStorage without the property 'deletable' do not destroy it...", nodeName)
			return nil
		}
	}
	return e.applyInfrastructure(ctx, kv, cfg, deploymentID, nodeName, outputs)
}
