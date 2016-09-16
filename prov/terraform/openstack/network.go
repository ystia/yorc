package openstack

import (
	"fmt"
	"strconv"
)

func (g *Generator) generateNetwork(url, deploymentId string) (Network, error) {
	var nodeType string
	var err error

	var OS_region string = g.cfg.OS_REGION

	if nodeType, err = g.getStringFormConsul(url, "type"); err != nil {
		return Network{}, err
	}
	if nodeType != "janus.nodes.openstack.Network" {
		return Network{}, fmt.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}

	network := Network{}

	if nodeName, err := g.getStringFormConsul(url, "properties/network_name"); err != nil {
		return Network{}, err
	} else if nodeName != "" {
		network.Name = nodeName
	}

	network.Region = OS_region

	return network, nil

}

func (g *Generator) generateSubnet(url, deploymentId, nodeName string) (Subnet, error) {
	var nodeType string
	var err error

	var OS_region string = g.cfg.OS_REGION

	if nodeType, err = g.getStringFormConsul(url, "type"); err != nil {
		return Subnet{}, err
	}
	if nodeType != "janus.nodes.openstack.Network" {
		return Subnet{}, fmt.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}

	subnet := Subnet{}

	if nodeName, err := g.getStringFormConsul(url, "properties/network_name"); err != nil {
		return Subnet{}, err
	} else if nodeName != "" {
		subnet.Name = nodeName + "_subnet"
	}
	if ipVersion, err := g.getStringFormConsul(url, "properties/ip_version"); err != nil {
		return Subnet{}, err
	} else if ipVersion != "" {
		subnet.IPVersion, err = strconv.Atoi(ipVersion)
		if err != nil {
			return Subnet{}, err
		}
	} else {
		subnet.IPVersion = 4
	}
	if nodeId, err := g.getStringFormConsul(url, "properties/network_id"); err != nil {
		return Subnet{}, err
	} else if nodeId != "" {
		subnet.NetworkID = nodeId
	} else {
		subnet.NetworkID = "${openstack_networking_network_v2." + nodeName + ".id}"
	}
	if nodeCIDR, err := g.getStringFormConsul(url, "properties/cidr"); err != nil {
		return Subnet{}, err
	} else if nodeCIDR != "" {
		subnet.CIDR = nodeCIDR
	}
	if gatewayIp, err := g.getStringFormConsul(url, "properties/gateway_ip"); err != nil {
		return Subnet{}, err
	} else if gatewayIp != "" {
		subnet.GatewayIP = gatewayIp
	}
	if startIp, err := g.getStringFormConsul(url, "properties/start_ip"); err != nil {
		return Subnet{}, err
	} else if startIp != "" {
		endIP, err := g.getStringFormConsul(url, "properties/end_ip")
		if err != nil {
			return Subnet{}, err
		}
		if endIP == "" {
			return Subnet{}, fmt.Errorf("A start_ip and a end_ip need to be provided")
		}
		subnet.AllocationPools = &AllocationPool{Start: startIp, End: endIP}
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

	subnet.Region = OS_region

	return subnet, nil
}
