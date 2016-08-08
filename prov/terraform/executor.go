package terraform

import (
	"context"
	"fmt"
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/executil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/openstack"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/slurm"
	"path"
	"path/filepath"
	"strings"
)

type Executor interface {
	ProvisionNode(ctx context.Context, deploymentId, nodeName string) error
	DestroyNode(ctx context.Context, deploymentId, nodeName string) error
}

type defaultExecutor struct {
	kv  *api.KV
	cfg config.Configuration
}

func NewExecutor(kv *api.KV, cfg config.Configuration) Executor {
	return &defaultExecutor{kv: kv, cfg: cfg}
}

func (e *defaultExecutor) ProvisionNode(ctx context.Context, deploymentId, nodeName string) error {
	kvPair, _, err := e.kv.Get(path.Join(deployments.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeName, "type"), nil)
	if err != nil {
		return err
	}
	if kvPair == nil {
		return fmt.Errorf("Type for node '%s' in deployment '%s' not found", nodeName, deploymentId)
	}
	nodeType := string(kvPair.Value)
	infraGenerated := true
	switch {
	case strings.HasPrefix(nodeType, "janus.nodes.openstack."):
		osGenerator := openstack.NewGenerator(e.kv, e.cfg)
		if infraGenerated, err = osGenerator.GenerateTerraformInfraForNode(deploymentId, nodeName); err != nil {
			return err
		}
	case strings.HasPrefix(nodeType, "janus.nodes.slurm."):
		osGenerator := slurm.NewGenerator(e.kv)
		if err := osGenerator.GenerateTerraformInfraForNode(deploymentId, nodeName); err != nil {
			return err
		}
	default:
		return fmt.Errorf("In executor : Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, deploymentId)
	}
	if infraGenerated {
		if err := e.applyInfrastructure(ctx, deploymentId, nodeName); err != nil {
			return err
		}
	}
	return nil
}

func (e *defaultExecutor) DestroyNode(ctx context.Context, deploymentId, nodeName string) error {
	if err := e.destroyInfrastructure(ctx, deploymentId, nodeName); err != nil {
		return err
	}
	return nil
}

func (e *defaultExecutor) applyInfrastructure(ctx context.Context, depId, nodeName string) error {
	deployments.LogInConsul(e.kv, depId, "Applying the infrastructure")
	infraPath := filepath.Join("work", "deployments", depId, "infra", nodeName)
	cmd := executil.Command(ctx, "terraform", "apply")
	cmd.Dir = infraPath
	errbuf := log.NewWriterSize(e.kv, depId, deployments.DeploymentKVPrefix)
	out := log.NewWriterSize(e.kv, depId, deployments.DeploymentKVPrefix)
	cmd.Stdout = out
	cmd.Stderr = errbuf

	quit := make(chan bool)
	defer close(quit)
	out.Run(quit)
	errbuf.Run(quit)

	if err := cmd.Start(); err != nil {
		log.Print(err)
	}

	err := cmd.Wait()

	return err

}

func (e *defaultExecutor) destroyInfrastructure(ctx context.Context, depId, nodeName string) error {
	nodePath := path.Join(deployments.DeploymentKVPrefix, depId, "topology/nodes", nodeName)
	if kp, _, err := e.kv.Get(nodePath+"/type", nil); err != nil {
		return err
	} else if kp == nil {
		return fmt.Errorf("Can't retrieve node type for node %q, in deployment %q", nodeName, depId)
	} else {
		if string(kp.Value) == "janus.nodes.openstack.BlockStorage" {
			if kp, _, err = e.kv.Get(nodePath+"/properties/deletable", nil); err != nil {
				return err
			} else if kp == nil || strings.ToLower(string(kp.Value)) != "true" {
				// False by default
				log.Printf("Node %q is a BlockStorage without the property 'deletable' do not destroy it...", nodeName)
				return nil
			}
		}
	}

	infraPath := filepath.Join("work", "deployments", depId, "infra", nodeName)
	cmd := executil.Command(ctx, "terraform", "destroy", "-force")
	cmd.Dir = infraPath
	errbuf := log.NewWriterSize(e.kv, depId, deployments.DeploymentKVPrefix)
	out := log.NewWriterSize(e.kv, depId, deployments.DeploymentKVPrefix)
	cmd.Stdout = out
	cmd.Stderr = errbuf

	quit := make(chan bool)
	defer close(quit)
	out.Run(quit)
	errbuf.Run(quit)

	if err := cmd.Start(); err != nil {
		log.Print(err)
	}

	err := cmd.Wait()

	return err

}
