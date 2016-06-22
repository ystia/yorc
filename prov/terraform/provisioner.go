package terraform

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/openstack"
	"strings"
)

type Provisioner interface {
	ProvisionNode(deploymentId, nodeName string) error
	DestroyNode(deploymentId, nodeName string) error
}

type defaultProvisioner struct {
	kv *api.KV
}

func NewProvisioner(kv *api.KV) Provisioner {
	return &defaultProvisioner{kv: kv}
}

func (p *defaultProvisioner) ProvisionNode(deploymentId, nodeName string) error {

	kvPair, _, err := p.kv.Get(strings.Join([]string{deployments.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeName, "type"}, "/"), nil)
	if err != nil {
		return err
	}
	if kvPair == nil {
		return fmt.Errorf("Type for node '%s' in deployment '%s' not found", nodeName, deploymentId)
	}
	nodeType := string(kvPair.Value)

	switch {
	case strings.HasPrefix(nodeType, "janus.nodes.openstack."):
		osGenerator := openstack.NewGenerator(p.kv)
		if err := osGenerator.GenerateTerraformInfraForNode(deploymentId, nodeName); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, deploymentId)
	}

	executor := &Executor{}
	if err := executor.ApplyInfrastructure(deploymentId, nodeName); err != nil {
		return err
	}
	return nil
}

func (p *defaultProvisioner) DestroyNode(deploymentId, nodeName string) error {
	executor := &Executor{}
	if err := executor.DestroyInfrastructure(deploymentId, nodeName); err != nil {
		return err
	}
	return nil
}
