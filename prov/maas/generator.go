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

package maas

import (
	"context"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tosca"
)

func generateInfrastructure(ctx context.Context, locationProps config.DynamicMap, deploymentID, nodeName, operation string, instances []string) (*infrastructure, error) {
	log.Debugf("Generating infrastructure for deployment with id %s", deploymentID)
	infra := &infrastructure{}
	log.Debugf("inspecting node %s", nodeName)
	nodeType, err := deployments.GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}

	switch nodeType {
	case "yorc.nodes.maas.Compute":
		var instances []string
		instances, err = deployments.GetNodeInstancesIds(ctx, deploymentID, nodeName)
		if err != nil {
			return nil, err
		}

		for _, instanceName := range instances {
			var instanceState tosca.NodeState
			instanceState, err = deployments.GetInstanceState(ctx, deploymentID, nodeName, instanceName)
			if err != nil {
				return nil, err
			}

			if operation == "install" && instanceState != tosca.NodeStateCreating {
				continue
			} else if operation == "uninstall" && instanceState != tosca.NodeStateDeleting {
				continue
			}

			if err := generateNodeAllocation(ctx, locationProps, deploymentID, nodeName, instanceName, infra); err != nil {
				return nil, err
			}
		}
	default:
		return nil, errors.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, deploymentID)
	}

	return infra, nil
}
