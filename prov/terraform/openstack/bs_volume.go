package openstack

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"novaforge.bull.com/starlings-janus/janus/helper/mathutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"strconv"
)

func (g *Generator) generateOSBSVolume(url string) (BlockStorageVolume, error) {
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
	} else {
		volume.Name = g.cfg.OS_PREFIX + nodeName
	}
	if size, err := g.getStringFormConsul(url, "properties/size"); err != nil {
		return volume, err
	} else if size != "" {
		// Default size unit is MB
		log.Debugf("Size form consul is %q", size)
		if mSize, err := strconv.Atoi(size); err != nil {
			if bsize, err := humanize.ParseBytes(size); err != nil {
				return volume, fmt.Errorf("Can't convert size to bytes value: %v", err)
			} else {
				// OpenStack needs the size in GB so we round it up.
				gSize := float64(bsize) / humanize.GByte
				log.Debugf("Computed size in GB: %f", gSize)
				gSize = mathutil.Round(gSize, 0, 0)
				log.Debugf("Computed size rounded in GB: %d", int(gSize))
				volume.Size = int(gSize)
			}
		} else {
			log.Debugf("Size in MB: %d", mSize)
			// OpenStack needs the size in GB so we round it up.
			gSize := float64(mSize) / 1000
			log.Debugf("Computed size in GB: %f", gSize)
			gSize = mathutil.Round(gSize, 0, 0)
			log.Debugf("Computed size rounded in GB: %d", int(gSize))
			volume.Size = int(gSize)
		}

	} else {
		return volume, fmt.Errorf("Missing mandatory property 'size' for %s", url)
	}
	if region, err := g.getStringFormConsul(url, "properties/region"); err != nil {
		return volume, err
	} else if region != "" {
		volume.Region = region
	} else {
		volume.Region = g.cfg.OS_REGION
	}
	if az, err := g.getStringFormConsul(url, "properties/availability_zone"); err != nil {
		return volume, err
	} else {
		volume.AvailabilityZone = az
	}

	return volume, nil

}
