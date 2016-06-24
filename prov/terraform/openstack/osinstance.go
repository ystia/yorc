package openstack

import (
	"fmt"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
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

	if keyPair, err := g.getStringFormConsul(url, "properties/key_pair"); err != nil {
		return ComputeInstance{}, err
	} else {
		// TODO if empty use a default one or fail ?
		instance.KeyPair = keyPair
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

	var user string
	if user, err = g.getStringFormConsul(url, "properties/user"); err != nil {
		return ComputeInstance{}, err
	} else if user == "" {
		return ComputeInstance{}, fmt.Errorf("Missing mandatory parameter 'user' node type for %s", url)
	}

	// Do this in order to be sure that ansible will be able to log on the instance
	re := commons.RemoteExec{Inline: []string{`echo "connected"`}, Connection: commons.Connection{User: user, PrivateKey: `${file("~/.ssh/cloudify.pem")}`}}
	instance.Provisioners = make(map[string]interface{})
	instance.Provisioners["remote-exec"] = re
	return instance, nil
}
