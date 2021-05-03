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

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/tosca"
)

type maasCompute struct {
	host hostCapabilities
	os   osCapabilities
}

type hostCapabilities struct {
	num_cpus  string
	mem_size  string
	disk_size string
}

type osCapabilities struct {
	architecture string
	distribution string
	version      string
}

func (c *maasCompute) deploy(ctx context.Context, operationParams *operationParameters, instance string) error {
	deploymentID := operationParams.deploymentID
	nodeName := operationParams.nodeName
	deployments.SetInstanceStateWithContextualLogs(ctx, deploymentID, nodeName, instance, tosca.NodeStateCreating)

	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(fmt.Sprintf("Creating node allocation for: deploymentID:%q, node name:%q", deploymentID, nodeName))

	maasClient, err := getMaasClient(operationParams.locationProps)
	if err != nil {
		return err
	}

	allocateParams, err := c.buildAllocateParams()
	if err != nil {
		return err
	}

	deployRes, err := allocateAndDeploy(maasClient, allocateParams, newDeployParams(c.os.distribution))
	if err != nil {
		return err
	}

	err = deployments.SetInstanceAttribute(ctx, deploymentID, nodeName, instance, "system_id", deployRes.system_id)
	if err != nil {
		return errors.Wrapf(err, "Failed to set attribute (system_id) for node name:%q, instance name:%q", nodeName, instance)
	}

	err = deployments.SetInstanceCapabilityAttribute(ctx, deploymentID, nodeName, instance, "endpoint", "ip_address", deployRes.ips[0])
	if err != nil {
		return errors.Wrapf(err, "Failed to set capability attribute (ip_address) for node name:%s, instance name:%q", nodeName, instance)
	}

	deployments.SetInstanceStateWithContextualLogs(ctx, deploymentID, nodeName, instance, tosca.NodeStateCreated)
	return nil
}

func getComputeFromDeployment(ctx context.Context, operationParams *operationParameters) (*maasCompute, error) {
	maasCompute := &maasCompute{}

	err := maasCompute.getAndsetPropertiesFromHostCapabilities(ctx, operationParams)
	if err != nil {
		return nil, err
	}

	err = maasCompute.getAndsetPropertiesFromOSCapabilities(ctx, operationParams)
	if err != nil {
		return nil, err
	}

	return maasCompute, nil
}

func (*maasCompute) undeploy(ctx context.Context, operationParams *operationParameters, instance string) error {
	deploymentID := operationParams.deploymentID
	nodeName := operationParams.nodeName
	deployments.SetInstanceStateWithContextualLogs(ctx, deploymentID, nodeName, instance, tosca.NodeStateDeleting)

	maasClient, err := getMaasClient(operationParams.locationProps)
	if err != nil {
		return err
	}

	system_id, err := deployments.GetInstanceAttributeValue(ctx, deploymentID, nodeName, instance, "system_id")
	if err != nil || system_id == nil {
		return errors.Wrapf(err, "can't find instance attribute system id for nodename:%s deployementId: %s \n Maybe last deployement was not successful", nodeName, deploymentID)
	}

	err = release(maasClient, system_id.RawString())
	if err != nil {
		return errors.Wrapf(err, "Release API call error for nodename:%s deployementId: %s", nodeName, deploymentID)
	}

	deployments.SetInstanceStateWithContextualLogs(ctx, deploymentID, nodeName, instance, tosca.NodeStateDeleted)
	return nil
}

// Set host capabilities using deployments values
func (c *maasCompute) getAndsetPropertiesFromHostCapabilities(ctx context.Context, operationParams *operationParameters) error {
	deploymentID := operationParams.deploymentID
	nodeName := operationParams.nodeName
	hostCapabilities := &c.host

	p, err := deployments.GetCapabilityPropertyValue(ctx, deploymentID, nodeName, "host", "num_cpus")
	if err != nil {
		return err
	}
	if p != nil && p.RawString() != "" {
		hostCapabilities.num_cpus = p.RawString()
	}

	p, err = deployments.GetCapabilityPropertyValue(ctx, deploymentID, nodeName, "host", "mem_size")
	if err != nil {
		return err
	}
	if p != nil && p.RawString() != "" {
		hostCapabilities.mem_size = p.RawString()
	}

	p, err = deployments.GetCapabilityPropertyValue(ctx, deploymentID, nodeName, "host", "disk_size")
	if err != nil {
		return err
	}
	if p != nil && p.RawString() != "" {
		hostCapabilities.disk_size = p.RawString()
	}
	return nil
}

// Set os capabilities using deployments values
func (c *maasCompute) getAndsetPropertiesFromOSCapabilities(ctx context.Context, operationParams *operationParameters) error {
	deploymentID := operationParams.deploymentID
	nodeName := operationParams.nodeName
	os := &c.os

	p, err := deployments.GetCapabilityPropertyValue(ctx, deploymentID, nodeName, "os", "architecture")
	if err != nil {
		return err
	}
	if p != nil && p.RawString() != "" {
		os.architecture = p.RawString()
	}

	p, err = deployments.GetCapabilityPropertyValue(ctx, deploymentID, nodeName, "os", "distribution")
	if err != nil {
		return err
	}
	if p != nil && p.RawString() != "" {
		os.distribution = p.RawString()
	}

	// p, err = deployments.GetCapabilityPropertyValue(ctx, deploymentID, nodeName, "os", "version")
	// if err != nil {
	// 	return err
	// }
	// if p != nil && p.RawString() != "" {
	// 	os.version = p.RawString()
	// }
	return nil
}

func (c *maasCompute) buildAllocateParams() (*allocateParams, error) {
	// Convert mem into MB without text
	mem := ""
	if c.host.mem_size != "" {
		memInt, err := humanize.ParseBytes(c.host.mem_size)
		if err != nil {
			return nil, err
		}
		memInt = memInt / 1000000
		mem = fmt.Sprint(memInt)
	}

	return newAllocateParams(c.host.num_cpus, mem), nil
}
