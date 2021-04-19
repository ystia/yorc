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
	"strings"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/tosca"
)

const infrastructureType = "maas"

type defaultExecutor struct {
}

func (e *defaultExecutor) ExecDelegate(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	// Get locations props
	var locationProps config.DynamicMap
	locationMgr, err := locations.GetManager(cfg)
	if err == nil {
		locationProps, err = locationMgr.GetLocationPropertiesForNode(ctx, deploymentID, nodeName, infrastructureType)
	}
	if err != nil {
		return err
	}

	// Get deployment instances names
	instances, err := e.getAvailableInstances(ctx, deploymentID, nodeName)
	if err != nil {
		return err
	}

	operation := strings.ToLower(delegateOperation)
	switch {
	case operation == "install":
		err = e.installNode(ctx, cfg, locationProps, deploymentID, nodeName, operation, instances)
	// case operation == "uninstall":
	// 	err = e.uninstallNode(ctx, cfg, locationProps, deploymentID, nodeName, instances, operation)
	default:
		return errors.Errorf("Unsupported operation %q", delegateOperation)
	}
	return err
}

func (e *defaultExecutor) getAvailableInstances(ctx context.Context, deploymentID, nodeName string) ([]string, error) {
	instances, err := deployments.GetNodeInstancesIds(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}

	availableIntances := []string{}
	for _, instanceName := range instances {
		instanceState, err := deployments.GetInstanceState(ctx, deploymentID, nodeName, instanceName)
		if err != nil {
			return nil, err
		}
		if instanceState == tosca.NodeStateDeleting || instanceState == tosca.NodeStateDeleted {
			// Do not generate something for this node instance (will be deleted if exists)
			continue
		}
		availableIntances = append(availableIntances, instanceName)
	}

	return availableIntances, nil
}

func (e *defaultExecutor) installNode(ctx context.Context, cfg config.Configuration, locationProps config.DynamicMap, deploymentID, nodeName string, operation string) error {
	infra, err := generateInfrastructure(ctx, locationProps, deploymentID, nodeName, operation)
	if err != nil {
		return err
	}

	// return e.createInfrastructure(ctx, cfg, locationProps, deploymentID, nodeName, infra)
	return nil
}
