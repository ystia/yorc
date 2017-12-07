package slurm

import (
	"context"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"path"
)

const infrastructureName = "slurm"

type defaultGenerator interface {
	generateInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName, operation string) (*infrastructure, error)
}

type slurmGenerator struct {
}

func (g *slurmGenerator) generateInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName, operation string) (*infrastructure, error) {
	log.Debugf("Generating infrastructure for deployment with id %s", deploymentID)
	nodeKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	infra := &infrastructure{}
	log.Debugf("inspecting node %s", nodeKey)
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}

	switch nodeType {
	case "janus.nodes.slurm.Compute":
		var instances []string
		instances, err = deployments.GetNodeInstancesIds(kv, deploymentID, nodeName)
		if err != nil {
			return nil, err
		}

		for _, instanceName := range instances {
			var instanceState tosca.NodeState
			instanceState, err = deployments.GetInstanceState(kv, deploymentID, nodeName, instanceName)
			if err != nil {
				return nil, err
			}

			if operation == "install" && instanceState != tosca.NodeStateCreating {
				continue
			} else if operation == "uninstall" && instanceState != tosca.NodeStateDeleting {
				continue
			}

			if err := g.generateNodeAllocation(ctx, kv, cfg, deploymentID, nodeName, instanceName, infra); err != nil {
				return nil, err
			}
		}
	default:
		return nil, errors.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, deploymentID)
	}

	return infra, nil
}
