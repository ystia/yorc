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

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/deployments"
)

const openstackNetworkType = "yorc.nodes.openstack.Network"

func (g *osGenerator) generateNetwork(kv *api.KV, cfg config.Configuration, deploymentID, nodeName string) (Network, error) {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return Network{}, err
	}
	if nodeType != openstackNetworkType {
		return Network{}, errors.Errorf("Unsupported node type for %s: %s", nodeName, nodeType)
	}

	network := Network{Name: cfg.ResourcesPrefix + nodeName + "Net"}

	if netName, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "network_name"); err != nil {
		return Network{}, err
	} else if netName != nil && netName.RawString() != "" {
		network.Name = cfg.ResourcesPrefix + netName.RawString()
	}

	network.Region = cfg.Infrastructures[infrastructureName].GetStringOrDefault("region", defaultOSRegion)

	return network, nil

}

func (g *osGenerator) generateSubnet(kv *api.KV, cfg config.Configuration, deploymentID, nodeName string) (Subnet, error) {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return Subnet{}, err
	}
	if nodeType != openstackNetworkType {
		return Subnet{}, errors.Errorf("Unsupported node type for %s: %s", nodeName, nodeType)
	}

	subnet := Subnet{}

	netName, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "network_name")
	if err != nil {
		return Subnet{}, err
	} else if netName != nil && netName.RawString() != "" {
		subnet.Name = cfg.ResourcesPrefix + netName.RawString() + "_subnet"
	} else {
		subnet.Name = cfg.ResourcesPrefix + nodeName + "_subnet"
	}
	ipVersion, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "ip_version")
	if err != nil {
		return Subnet{}, err
	} else if ipVersion != nil && ipVersion.RawString() != "" {
		subnet.IPVersion, err = strconv.Atoi(ipVersion.RawString())
		if err != nil {
			return Subnet{}, err
		}
	} else {
		subnet.IPVersion = 4
	}
	nodeID, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "network_id")
	if err != nil {
		return Subnet{}, err
	} else if nodeID != nil && nodeID.RawString() != "" {
		subnet.NetworkID = nodeID.RawString()
	} else {
		subnet.NetworkID = "${openstack_networking_network_v2." + nodeName + ".id}"
	}
	if nodeCIDR, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "cidr"); err != nil {
		return Subnet{}, err
	} else if nodeCIDR != nil && nodeCIDR.RawString() != "" {
		subnet.CIDR = nodeCIDR.RawString()
	}
	if gatewayIP, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "gateway_ip"); err != nil {
		return Subnet{}, err
	} else if gatewayIP != nil && gatewayIP.RawString() != "" {
		subnet.GatewayIP = gatewayIP.RawString()
	}
	if startIP, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "start_ip"); err != nil {
		return Subnet{}, err
	} else if startIP != nil && startIP.RawString() != "" {
		endIP, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "end_ip")
		if err != nil {
			return Subnet{}, err
		}
		if endIP == nil || endIP.RawString() == "" {
			return Subnet{}, errors.Errorf("A start_ip and a end_ip need to be provided")
		}
		subnet.AllocationPools = &AllocationPool{Start: startIP.RawString(), End: endIP.RawString()}
	}
	if dhcp, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "dhcp_enabled"); err != nil {
		return Subnet{}, err
	} else if dhcp != nil && dhcp.RawString() != "" {
		subnet.EnableDHCP, err = strconv.ParseBool(dhcp.RawString())
		if err != nil {
			return Subnet{}, err
		}
	} else {
		subnet.EnableDHCP = true
	}

	subnet.Region = cfg.Infrastructures[infrastructureName].GetStringOrDefault("region", defaultOSRegion)

	return subnet, nil
}
