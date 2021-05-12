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
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tosca"
	"golang.org/x/sync/errgroup"
)

const infrastructureType = "maas"

type defaultExecutor struct {
}

type operationParameters struct {
	locationProps     config.DynamicMap
	taskID            string
	deploymentID      string
	nodeName          string
	delegateOperation string
	instances         []string
}

// ExecDelegate : Get required operation params and call the operation
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

	// Get instances names
	instances, err := tasks.GetInstances(ctx, taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}

	// Apply operation
	operationParams := operationParameters{
		locationProps:     locationProps,
		taskID:            taskID,
		deploymentID:      deploymentID,
		nodeName:          nodeName,
		delegateOperation: delegateOperation,
		instances:         instances,
	}

	operation := strings.ToLower(delegateOperation)
	switch {
	case operation == "install":
		err = e.installNode(ctx, &operationParams)
	case operation == "uninstall":
		err = e.uninstallNode(ctx, &operationParams)
	default:
		return errors.Errorf("Unsupported operation %q", delegateOperation)
	}
	return err
}

func (e *defaultExecutor) installNode(ctx context.Context, operationParams *operationParameters) error {
	log.Debugf("Installing node %s for deployment %s", operationParams.nodeName, operationParams.deploymentID)

	// Verify node Type
	nodeType, err := deployments.GetNodeType(ctx, operationParams.deploymentID, operationParams.nodeName)
	if err != nil {
		return err
	}
	if nodeType != "yorc.nodes.maas.Compute" {
		return errors.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, operationParams.nodeName, operationParams.deploymentID)
	}

	compute, err := getComputeFromDeployment(ctx, operationParams)

	if err != nil {
		return errors.Wrapf(err, "Failed to get Compute properties for deploymentID:%q, node name:%s", operationParams.deploymentID, operationParams.nodeName)
	}

	var g errgroup.Group
	for _, instance := range operationParams.instances {
		instanceState, err := deployments.GetInstanceState(ctx, operationParams.deploymentID, operationParams.nodeName, instance)
		if err != nil {
			return err
		}

		// If instance creating or deleting, ignore it
		if instanceState == tosca.NodeStateCreating || instanceState == tosca.NodeStateDeleting {
			continue
		}

		func(ctx context.Context, comp *maasCompute) {
			g.Go(func() error {
				return compute.deploy(ctx, operationParams, instance)
			})
		}(events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.InstanceID: instance}), compute)
	}

	if err := g.Wait(); err != nil {
		err = errors.Wrapf(err, "Failed install node deploymentID:%q, node name:%s", operationParams.deploymentID, operationParams.nodeName)
		log.Debugf("%+v", err)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, operationParams.deploymentID).RegisterAsString(err.Error())
		return err
	}

	return nil
}

func (e *defaultExecutor) uninstallNode(ctx context.Context, operationParams *operationParameters) error {
	log.Debugf("Uninstalling node %s for deployment %s", operationParams.nodeName, operationParams.deploymentID)

	// Verify node Type
	nodeType, err := deployments.GetNodeType(ctx, operationParams.deploymentID, operationParams.nodeName)
	if err != nil {
		return err
	}
	if nodeType != "yorc.nodes.maas.Compute" {
		return errors.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, operationParams.nodeName, operationParams.deploymentID)
	}

	compute, err := getComputeFromDeployment(ctx, operationParams)
	if err != nil {
		return errors.Wrapf(err, "Failed to get Compute properties for deploymentID:%q, node name:%s", operationParams.deploymentID, operationParams.nodeName)
	}

	var g errgroup.Group
	for _, instance := range operationParams.instances {
		instanceState, err := deployments.GetInstanceState(ctx, operationParams.deploymentID, operationParams.nodeName, instance)
		if err != nil {
			return err
		}

		// If instance deleting, ignore it
		if instanceState == tosca.NodeStateDeleting {
			continue
		}

		func(ctx context.Context, comp *maasCompute) {
			g.Go(func() error {
				return compute.undeploy(ctx, operationParams, instance)
			})
		}(events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.InstanceID: instance}), compute)
	}

	if err := g.Wait(); err != nil {
		err = errors.Wrapf(err, "Failed install node deploymentID:%q, node name:%s", operationParams.deploymentID, operationParams.nodeName)
		log.Debugf("%+v", err)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, operationParams.deploymentID).RegisterAsString(err.Error())
		return err
	}

	return nil
}
