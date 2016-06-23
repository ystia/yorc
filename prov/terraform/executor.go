package terraform

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/openstack"
	"strings"
	"path"
	"os/exec"
	"os"
	"log"
)

type Executor interface {
	ProvisionNode(deploymentId, nodeName string) error
	DestroyNode(deploymentId, nodeName string) error
}

type defaultExecutor struct {
	kv *api.KV
}

func NewExecutor(kv *api.KV) Executor {
	return &defaultExecutor{kv: kv}
}

func (e *defaultExecutor) ProvisionNode(deploymentId, nodeName string) error {

	kvPair, _, err := e.kv.Get(strings.Join([]string{deployments.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeName, "type"}, "/"), nil)
	if err != nil {
		return err
	}
	if kvPair == nil {
		return fmt.Errorf("Type for node '%s' in deployment '%s' not found", nodeName, deploymentId)
	}
	nodeType := string(kvPair.Value)

	switch {
	case strings.HasPrefix(nodeType, "janus.nodes.openstack."):
		osGenerator := openstack.NewGenerator(e.kv)
		if err := osGenerator.GenerateTerraformInfraForNode(deploymentId, nodeName); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, deploymentId)
	}

	if err := e.applyInfrastructure(deploymentId, nodeName); err != nil {
		return err
	}
	return nil
}

func (e *defaultExecutor) DestroyNode(deploymentId, nodeName string) error {
	if err := e.destroyInfrastructure(deploymentId, nodeName); err != nil {
		return err
	}
	return nil
}

func (e *defaultExecutor) applyInfrastructure(depId, nodeName string) error {
	infraPath := path.Join("work", "deployments", depId, "infra", nodeName)
	cmd := exec.Command("terraform", "apply")
	cmd.Dir = infraPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Print(err)
		return err
	}

	return cmd.Wait()

}
func (e *defaultExecutor) destroyInfrastructure(depId, nodeName string) error {
	infraPath := path.Join("work", "deployments", depId, "infra", nodeName)
	cmd := exec.Command("terraform", "destroy", "-force")
	cmd.Dir = infraPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Print(err)
		return err
	}

	return cmd.Wait()

}
