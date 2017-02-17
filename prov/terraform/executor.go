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
	"novaforge.bull.com/starlings-janus/janus/helper/executil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/openstack"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/slurm"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

type defaultExecutor struct {
}

// NewExecutor returns an Executor
func NewExecutor() prov.DelegateExecutor {
	return &defaultExecutor{}
}

func (e *defaultExecutor) ExecDelegate(ctx context.Context, kv *api.KV, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	var generator commons.Generator
	switch {
	case strings.HasPrefix(nodeType, "janus.nodes.openstack."):
		generator = openstack.NewGenerator(kv, cfg)
	case strings.HasPrefix(nodeType, "janus.nodes.slurm."):
		generator = slurm.NewGenerator(kv, cfg)
	default:
		return errors.Errorf("Unsupported node type '%s' for node '%s'", nodeType, nodeName)
	}

	instances, err := tasks.GetInstances(kv, taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}

	op := strings.ToLower(delegateOperation)
	switch {
	case op == "install":
		err = e.installNode(ctx, kv, cfg, deploymentID, nodeName, instances, generator)
	case op == "uninstall":
		err = e.uninstallNode(ctx, kv, cfg, deploymentID, nodeName, instances, generator)
	default:
		return errors.Errorf("Unsupported operation %q", delegateOperation)
	}
	return err
}

func (e *defaultExecutor) installNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, instances []string, generator commons.Generator) error {
	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateCreating)
		if err != nil {
			return err
		}
	}
	infraGenerated, err := generator.GenerateTerraformInfraForNode(deploymentID, nodeName)
	if err != nil {
		return err
	}
	if infraGenerated {
		if err = e.applyInfrastructure(ctx, kv, cfg, deploymentID, nodeName); err != nil {
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

func (e *defaultExecutor) uninstallNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, instances []string, generator commons.Generator) error {
	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateDeleting)
		if err != nil {
			return err
		}
	}
	infraGenerated, err := generator.GenerateTerraformInfraForNode(deploymentID, nodeName)
	if err != nil {
		return err
	}
	if infraGenerated {
		if err = e.destroyInfrastructure(ctx, kv, cfg, deploymentID, nodeName); err != nil {
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

func (e *defaultExecutor) applyInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string) error {
	events.LogEngineMessage(kv, deploymentID, "Applying the infrastructure")
	infraPath := filepath.Join(cfg.WorkingDirectory, "deployments", deploymentID, "infra", nodeName)
	cmd := executil.Command(ctx, "terraform", "apply")
	cmd.Dir = infraPath
	errbuf := events.NewBufferedLogEventWriter(kv, deploymentID, events.InfraLogPrefix)
	out := events.NewBufferedLogEventWriter(kv, deploymentID, events.InfraLogPrefix)
	cmd.Stdout = out
	cmd.Stderr = errbuf

	quit := make(chan bool)
	defer close(quit)
	out.Run(quit)
	errbuf.Run(quit)

	if err := cmd.Start(); err != nil {
		log.Print(err)
	}

	return cmd.Wait()

}

func (e *defaultExecutor) destroyInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string) error {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType == "janus.nodes.openstack.BlockStorage" {
		var deletable string
		var found bool
		found, deletable, err = deployments.GetNodeProperty(kv, deploymentID, nodeName, "deletable")
		if err != nil {
			return err
		}
		if !found || strings.ToLower(deletable) != "true" {
			// False by default
			log.Printf("Node %q is a BlockStorage without the property 'deletable' do not destroy it...", nodeName)
			return nil
		}
	}
	return e.applyInfrastructure(ctx, kv, cfg, deploymentID, nodeName)

}
