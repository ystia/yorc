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
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/helper/mathutil"
	"github.com/ystia/yorc/log"
)

func (g *osGenerator) generateOSBSVolume(kv *api.KV, cfg config.Configuration, url, instanceName string) (BlockStorageVolume, error) {
	volume := BlockStorageVolume{}
	var nodeType string
	var err error
	if nodeType, err = g.getStringFormConsul(kv, url, "type"); err != nil {
		return volume, err
	}
	if nodeType != "yorc.nodes.openstack.BlockStorage" {
		return volume, errors.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}
	var nodeName string
	if nodeName, err = g.getStringFormConsul(kv, url, "name"); err != nil {
		return volume, err
	}
	volume.Name = cfg.ResourcesPrefix + nodeName + "-" + instanceName
	size, err := g.getStringFormConsul(kv, url, "properties/size")
	if err != nil {
		return volume, err
	}
	if size == "" {
		return volume, errors.Errorf("Missing mandatory property 'size' for %s", url)
	}
	// Default size unit is MB
	log.Debugf("Size form consul is %q", size)
	mSize, err := strconv.Atoi(size)
	if err != nil {
		var bsize uint64
		bsize, err = humanize.ParseBytes(size)
		if err != nil {
			return volume, errors.Errorf("Can't convert size to bytes value: %v", err)
		}
		// OpenStack needs the size in GB so we round it up.
		gSize := float64(bsize) / humanize.GByte
		log.Debugf("Computed size in GB: %f", gSize)
		gSize = mathutil.Round(gSize, 0, 0)
		log.Debugf("Computed size rounded in GB: %d", int(gSize))
		volume.Size = int(gSize)
	} else {
		log.Debugf("Size in MB: %d", mSize)
		// OpenStack needs the size in GB so we round it up.
		gSize := float64(mSize) / 1000
		log.Debugf("Computed size in GB: %f", gSize)
		gSize = mathutil.Round(gSize, 0, 0)
		log.Debugf("Computed size rounded in GB: %d", int(gSize))
		volume.Size = int(gSize)
	}

	region, err := g.getStringFormConsul(kv, url, "properties/region")
	if err != nil {
		return volume, err
	} else if region != "" {
		volume.Region = region
	} else {
		volume.Region = cfg.Infrastructures[infrastructureName].GetStringOrDefault("region", defaultOSRegion)
	}
	az, err := g.getStringFormConsul(kv, url, "properties/availability_zone")
	if err != nil {
		return volume, err
	}
	volume.AvailabilityZone = az

	return volume, nil
}
