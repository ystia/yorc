// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package slurm

import (
	"context"
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tosca"
)

const infrastructureName = "slurm"

func generateInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName, operation string) (*infrastructure, error) {
	log.Debugf("Generating infrastructure for deployment with id %s", deploymentID)
	nodeKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	infra := &infrastructure{}
	log.Debugf("inspecting node %s", nodeKey)
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}

	switch nodeType {
	case "yorc.nodes.slurm.Compute":
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

			if err := generateNodeAllocation(ctx, kv, cfg, deploymentID, nodeName, instanceName, infra); err != nil {
				return nil, err
			}
		}
	default:
		return nil, errors.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, deploymentID)
	}

	return infra, nil
}
