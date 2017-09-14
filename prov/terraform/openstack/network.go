package openstack

import (
	"strconv"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"novaforge.bull.com/starlings-janus/janus/config"
)

const openstackNetworkType = "janus.nodes.openstack.Network"

func (g *osGenerator) generateNetwork(kv *api.KV, cfg config.Configuration, url, deploymentID string) (Network, error) {
	nodeType, err := g.getStringFormConsul(kv, url, "type")
	if err != nil {
		return Network{}, err
	}
	if nodeType != openstackNetworkType {
		return Network{}, errors.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}

	network := Network{}

	if nodeName, err := g.getStringFormConsul(kv, url, "properties/network_name"); err != nil {
		return Network{}, err
	} else if nodeName != "" {
		network.Name = cfg.ResourcesPrefix + nodeName
	}

	network.Region = cfg.Infrastructures[infrastructureName].GetStringOrDefault("region", defaultOSRegion)

	return network, nil

}

func (g *osGenerator) generateSubnet(kv *api.KV, cfg config.Configuration, url, deploymentID, nodeName string) (Subnet, error) {
	nodeType, err := g.getStringFormConsul(kv, url, "type")
	if err != nil {
		return Subnet{}, err
	}
	if nodeType != openstackNetworkType {
		return Subnet{}, errors.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}

	subnet := Subnet{}

	netName, err := g.getStringFormConsul(kv, url, "properties/network_name")
	if err != nil {
		return Subnet{}, err
	} else if netName != "" {
		subnet.Name = cfg.ResourcesPrefix + netName + "_subnet"
	} else {
		subnet.Name = cfg.ResourcesPrefix + nodeName + "_subnet"
	}
	ipVersion, err := g.getStringFormConsul(kv, url, "properties/ip_version")
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
	nodeID, err := g.getStringFormConsul(kv, url, "properties/network_id")
	if err != nil {
		return Subnet{}, err
	} else if nodeID != "" {
		subnet.NetworkID = nodeID
	} else {
		subnet.NetworkID = "${openstack_networking_network_v2." + nodeName + ".id}"
	}
	if nodeCIDR, err := g.getStringFormConsul(kv, url, "properties/cidr"); err != nil {
		return Subnet{}, err
	} else if nodeCIDR != "" {
		subnet.CIDR = nodeCIDR
	}
	if gatewayIP, err := g.getStringFormConsul(kv, url, "properties/gateway_ip"); err != nil {
		return Subnet{}, err
	} else if gatewayIP != "" {
		subnet.GatewayIP = gatewayIP
	}
	if startIP, err := g.getStringFormConsul(kv, url, "properties/start_ip"); err != nil {
		return Subnet{}, err
	} else if startIP != "" {
		endIP, err := g.getStringFormConsul(kv, url, "properties/end_ip")
		if err != nil {
			return Subnet{}, err
		}
		if endIP == "" {
			return Subnet{}, errors.Errorf("A start_ip and a end_ip need to be provided")
		}
		subnet.AllocationPools = &AllocationPool{Start: startIP, End: endIP}
	}
	if dhcp, err := g.getStringFormConsul(kv, url, "properties/dhcp_enabled"); err != nil {
		return Subnet{}, err
	} else if dhcp != "" {
		subnet.EnableDHCP, err = strconv.ParseBool(dhcp)
		if err != nil {
			return Subnet{}, err
		}
	} else {
		subnet.EnableDHCP = true
	}

	subnet.Region = cfg.Infrastructures[infrastructureName].GetStringOrDefault("region", defaultOSRegion)

	return subnet, nil
}
