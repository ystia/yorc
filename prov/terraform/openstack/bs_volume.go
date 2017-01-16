package openstack

import (
	"fmt"
	"strconv"

	"github.com/dustin/go-humanize"
	"novaforge.bull.com/starlings-janus/janus/helper/mathutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func (g *Generator) generateOSBSVolume(url, instanceName string) (BlockStorageVolume, error) {
	volume := BlockStorageVolume{}
	var nodeType string
	var err error
	if nodeType, err = g.getStringFormConsul(url, "type"); err != nil {
		return volume, err
	}
	if nodeType != "janus.nodes.openstack.BlockStorage" {
		return volume, fmt.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}
	var nodeName string
	if nodeName, err = g.getStringFormConsul(url, "name"); err != nil {
		return volume, err
	}
	volume.Name = g.cfg.ResourcesPrefix + nodeName + "-" + instanceName
	size, err := g.getStringFormConsul(url, "properties/size")
	if err != nil {
		return volume, err
	}
	if size == "" {
		return volume, fmt.Errorf("Missing mandatory property 'size' for %s", url)
	}
	// Default size unit is MB
	log.Debugf("Size form consul is %q", size)
	mSize, err := strconv.Atoi(size)
	if err != nil {
		var bsize uint64
		bsize, err = humanize.ParseBytes(size)
		if err != nil {
			return volume, fmt.Errorf("Can't convert size to bytes value: %v", err)
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

	region, err := g.getStringFormConsul(url, "properties/region")
	if err != nil {
		return volume, err
	} else if region != "" {
		volume.Region = region
	} else {
		volume.Region = g.cfg.OSRegion
	}
	az, err := g.getStringFormConsul(url, "properties/availability_zone")
	if err != nil {
		return volume, err
	}
	volume.AvailabilityZone = az

	return volume, nil
}
