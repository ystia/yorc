package slurm

import (
	"context"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"regexp"
	"strings"
)

func (g *slurmGenerator) generateNodeAllocation(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID string, instanceName, nodeName string, infra *infrastructure) error {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType != "janus.nodes.slurm.Compute" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}
	node := nodeAllocation{instanceName: instanceName}

	// Set the node CPU and memory property from Tosca Compute 'host' capability property
	_, cpu, err := deployments.GetCapabilityProperty(kv, deploymentID, nodeName, "host", "num_cpus")
	if err != nil {
		return err
	}
	node.cpu = cpu

	_, memory, err := deployments.GetCapabilityProperty(kv, deploymentID, nodeName, "host", "mem_size")
	if err != nil {
		return err
	}
	// Only take one letter for defining memory unit and remove the blank space between number and unit.
	re := regexp.MustCompile("[[:digit:]]*[[:upper:]]{1}")
	node.memory = re.FindString(strings.Replace(memory, " ", "", -1))

	// Set the node gres property from Tosca slurm.Compute property
	_, gres, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "gres")
	if err != nil {
		return err
	}
	node.gres = gres

	// Set the node partition property from Tosca slurm.Compute property
	_, partition, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "partition")
	if err != nil {
		return err
	}
	node.partition = partition
	infra.nodes = append(infra.nodes, node)
	return nil
}
