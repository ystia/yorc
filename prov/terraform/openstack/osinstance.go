package openstack

import (
	"fmt"
)

func (g *Generator) generateOSInstance(url string) (ComputeInstance, error) {
	var nodeType string
	var err error
	if nodeType, err = g.getStringFormConsul(url, "type"); err != nil {
		return ComputeInstance{}, err
	}
	if nodeType != "janus.nodes.openstack.Compute" {
		return ComputeInstance{}, fmt.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}
	instance := ComputeInstance{}
	if nodeName, err := g.getStringFormConsul(url, "name"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.Name = nodeName
	}
	if image, err := g.getStringFormConsul(url, "properties/image"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.ImageId = image
	}
	if image, err := g.getStringFormConsul(url, "properties/imageName"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.ImageName = image
	}
	if flavor, err := g.getStringFormConsul(url, "properties/flavor"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.FlavorId = flavor
	}
	if flavor, err := g.getStringFormConsul(url, "properties/flavorName"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.FlavorName = flavor
	}

	if az, err := g.getStringFormConsul(url, "properties/availability_zone"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.AvailabilityZone = az
	}
	if region, err := g.getStringFormConsul(url, "properties/region"); err != nil {
		return ComputeInstance{}, err
	} else if region != "" {
		instance.Region = region
	} else {
		//TODO make this configurable
		instance.Region = "RegionOne"
	}

	if instance.ImageId == "" && instance.ImageName == "" {
		return ComputeInstance{}, fmt.Errorf("Missing mandatory parameter 'image' or 'imageName' node type for %s", url)
	}
	if instance.FlavorId == "" && instance.FlavorName == "" {
		return ComputeInstance{}, fmt.Errorf("Missing mandatory parameter 'flavor' or 'flavorName' node type for %s", url)
	}

	if networkName, err := g.getStringFormConsul(url, "capabilities/endpoint/properties/network_name"); err != nil {
		return ComputeInstance{}, err
	} else {
		if networkName != "" {
			// TODO Deal with networks aliases (PUBLIC/PRIVATE)
			var networkSlice []ComputeNetwork
			networkSlice = append(networkSlice, ComputeNetwork{Name: networkName})
			instance.Networks = networkSlice
		}
	}
	return instance, nil
}
