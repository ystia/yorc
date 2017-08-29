package openstack

import (
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/helper/mathutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func (g *osGenerator) generateOSBSVolume(kv *api.KV, cfg config.Configuration, url, instanceName string) (BlockStorageVolume, error) {
	volume := BlockStorageVolume{}
	var nodeType string
	var err error
	if nodeType, err = g.getStringFormConsul(kv, url, "type"); err != nil {
		return volume, err
	}
	if nodeType != "janus.nodes.openstack.BlockStorage" {
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
