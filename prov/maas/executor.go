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
	"fmt"
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

	operation := strings.ToLower(delegateOperation)
	switch {
	case operation == "install":
		err = e.installNode(ctx, cfg, locationProps, deploymentID, nodeName, operation, &instances)
	// case operation == "uninstall":
	// 	err = e.uninstallNode(ctx, cfg, locationProps, deploymentID, nodeName, instances, operation)
	default:
		return errors.Errorf("Unsupported operation %q", delegateOperation)
	}
	return err
}

func (e *defaultExecutor) installNode(ctx context.Context, cfg config.Configuration, locationProps config.DynamicMap, deploymentID, nodeName, operation string, instances *[]string) error {
	for _, instance := range *instances {
		err := deployments.SetInstanceStateWithContextualLogs(events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.InstanceID: instance}), deploymentID, nodeName, instance, tosca.NodeStateCreating)
		if err != nil {
			return err
		}
	}

	infra, err := generateInfrastructure(ctx, locationProps, deploymentID, nodeName, operation, instances)
	if err != nil {
		return err
	}

	return e.createInfrastructure(ctx, cfg, locationProps, deploymentID, nodeName, infra)
}

func (e *defaultExecutor) createInfrastructure(ctx context.Context, cfg config.Configuration, locationProps config.DynamicMap, deploymentID, nodeName string, infra *infrastructure) error {
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString("Creating the maas infrastructure")
	var g errgroup.Group
	for _, compute := range infra.nodes {
		func(ctx context.Context, comp *nodeAllocation) {
			g.Go(func() error {
				return e.createNodeAllocation(ctx, cfg, locationProps, comp, deploymentID, nodeName)
			})
		}(events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.InstanceID: compute.instanceName}), compute)
	}

	if err := g.Wait(); err != nil {
		err = errors.Wrapf(err, "Failed to create maas infrastructure for deploymentID:%q, node name:%s", deploymentID, nodeName)
		log.Debugf("%+v", err)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, deploymentID).RegisterAsString(err.Error())
		return err
	}

	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString("Successfully creating the maas infrastructure")
	return nil
}

func (e *defaultExecutor) createNodeAllocation(ctx context.Context, cfg config.Configuration, locationProps config.DynamicMap, nodeAlloc *nodeAllocation, deploymentID, nodeName string) error {
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(fmt.Sprintf("Creating node allocation for: deploymentID:%q, node name:%q", deploymentID, nodeName))

	return nil
}
