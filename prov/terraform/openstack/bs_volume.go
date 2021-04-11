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

package openstack

import (
	"context"
	"strconv"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/sizeutil"
	"github.com/ystia/yorc/v4/log"
)

func (g *osGenerator) generateOSBSVolume(ctx context.Context, cfg config.Configuration, locationProps config.DynamicMap,
	deploymentID, nodeName, instanceName string) (BlockStorageVolume, error) {

	volume := BlockStorageVolume{}
	nodeType, err := deployments.GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return volume, err
	}
	if nodeType != "yorc.nodes.openstack.BlockStorage" {
		return volume, errors.Errorf("Unsupported node type for %s: %s", nodeName, nodeType)
	}
	volume.Name = cfg.ResourcesPrefix + nodeName + "-" + instanceName

	size, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "size")
	if err != nil {
		return volume, err
	}
	if size == nil || size.RawString() == "" {
		return volume, errors.Errorf("Missing mandatory property 'size' for %s", nodeName)
	}
	// Default size unit is MB
	log.Debugf("Size form consul is %q", size)
	volume.Size, err = sizeutil.ConvertToGB(size.RawString())
	if err != nil {
		return volume, err
	}
	log.Debugf("Computed size rounded in GB: %d", volume.Size)

	region, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "region")
	if err != nil {
		return volume, err
	} else if region != nil && region.RawString() != "" {
		volume.Region = region.RawString()
	} else {
		volume.Region = locationProps.GetStringOrDefault("region", defaultOSRegion)
	}
	az, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "availability_zone")
	if err != nil {
		return volume, err
	}
	if az != nil {
		volume.AvailabilityZone = az.RawString()
	}
	return volume, nil
}

// computeBootVolume gets value of a boot volume if any is defined
// If none, nil is returned
func computeBootVolume(ctx context.Context, deploymentID, nodeName string) (*BootVolume, error) {
	var vol BootVolume
	var err error
	// If a boot volume is defined, its source definition is mandatory
	vol.Source, err = deployments.GetStringNodePropertyValue(ctx, deploymentID, nodeName, bootVolumeTOSCAAttr, sourceTOSCAKey)
	if err != nil || vol.Source == "" {
		return nil, err
	}

	keys := []string{uuidTOSCAKey, destinationTOSCAKey, sizeTOSCAKey, volumeTypeTOSCAKey, deleteOnTerminationTOSCAKey}
	strValues := make(map[string]string, len(keys))
	for _, key := range keys {
		strValues[key], err = deployments.GetStringNodePropertyValue(ctx, deploymentID, nodeName, bootVolumeTOSCAAttr, key)
		if err != nil {
			return nil, err
		}
	}

	vol.UUID = strValues[uuidTOSCAKey]
	vol.Destination = strValues[destinationTOSCAKey]
	vol.VolumeType = strValues[volumeTypeTOSCAKey]

	if strValues[sizeTOSCAKey] != "" {
		vol.Size, err = sizeutil.ConvertToGB(strValues[sizeTOSCAKey])
		if err != nil {
			log.Printf("Failed to convert %s %s boot volume size %q : %s", deploymentID,
				nodeName, strValues[sizeTOSCAKey], err.Error())
			return nil, err
		}
	}
	if strValues[deleteOnTerminationTOSCAKey] != "" {
		vol.DeleteOnTermination, err = strconv.ParseBool(strValues[deleteOnTerminationTOSCAKey])
		if err != nil {
			log.Printf("Failed to convert %s %s %s boolean value %q : %s", deploymentID,
				nodeName, deleteOnTerminationTOSCAKey, strValues[sizeTOSCAKey], err.Error())
			return nil, err
		}
	}

	return &vol, err
}
