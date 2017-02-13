package openstack

import (
	"fmt"
	"strconv"
)

func (g *osGenerator) generateNetwork(url, deploymentID string) (Network, error) {
	nodeType, err := g.getStringFormConsul(url, "type")
	if err != nil {
		return Network{}, err
	}
	if nodeType != "janus.nodes.openstack.Network" {
		return Network{}, fmt.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}

	network := Network{}

	if nodeName, err := g.getStringFormConsul(url, "properties/network_name"); err != nil {
		return Network{}, err
	} else if nodeName != "" {
		network.Name = g.cfg.ResourcesPrefix + nodeName
	}

	network.Region = g.cfg.OSRegion

	return network, nil

}

func (g *osGenerator) generateSubnet(url, deploymentID, nodeName string) (Subnet, error) {
	nodeType, err := g.getStringFormConsul(url, "type")
	if err != nil {
		return Subnet{}, err
	}
	if nodeType != "janus.nodes.openstack.Network" {
		return Subnet{}, fmt.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}

	subnet := Subnet{}

	netName, err := g.getStringFormConsul(url, "properties/network_name")
	if err != nil {
		return Subnet{}, err
	} else if netName != "" {
		subnet.Name = g.cfg.ResourcesPrefix + netName + "_subnet"
	} else {
		subnet.Name = g.cfg.ResourcesPrefix + nodeName + "_subnet"
	}
	ipVersion, err := g.getStringFormConsul(url, "properties/ip_version")
	if err != nil {
		return Subnet{}, err
	} else if ipVersion != "" {
		subnet.IPVersion, err = strconv.Atoi(ipVersion)
		if err != nil {
			return Subnet{}, err
		}
	} else {
		subnet.IPVersion = 4
	}
	nodeID, err := g.getStringFormConsul(url, "properties/network_id")
	if err != nil {
		return Subnet{}, err
	} else if nodeID != "" {
		subnet.NetworkID = nodeID
	} else {
		subnet.NetworkID = "${openstack_networking_network_v2." + nodeName + ".id}"
	}
	if nodeCIDR, err := g.getStringFormConsul(url, "properties/cidr"); err != nil {
		return Subnet{}, err
	} else if nodeCIDR != "" {
		subnet.CIDR = nodeCIDR
	}
	if gatewayIP, err := g.getStringFormConsul(url, "properties/gateway_ip"); err != nil {
		return Subnet{}, err
	} else if gatewayIP != "" {
		subnet.GatewayIP = gatewayIP
	}
	if startIP, err := g.getStringFormConsul(url, "properties/start_ip"); err != nil {
		return Subnet{}, err
	} else if startIP != "" {
		endIP, err := g.getStringFormConsul(url, "properties/end_ip")
		if err != nil {
			return Subnet{}, err
		}
		if endIP == "" {
			return Subnet{}, fmt.Errorf("A start_ip and a end_ip need to be provided")
		}
		subnet.AllocationPools = &AllocationPool{Start: startIP, End: endIP}
	}
	if dhcp, err := g.getStringFormConsul(url, "properties/dhcp_enabled"); err != nil {
		return Subnet{}, err
	} else if dhcp != "" {
		subnet.EnableDHCP, err = strconv.ParseBool(dhcp)
		if err != nil {
			return Subnet{}, err
		}
	} else {
		subnet.EnableDHCP = true
	}

	subnet.Region = g.cfg.OSRegion

	return subnet, nil
}
