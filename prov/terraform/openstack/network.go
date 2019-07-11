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
	"fmt"
	"strconv"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
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

func (g *osGenerator) generateSubnet(kv *api.KV, cfg config.Configuration, deploymentID,
	nodeName, resourceType string) (Subnet, error) {

	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return Subnet{}, err
	}
	if nodeType != openstackNetworkType {
		return Subnet{}, errors.Errorf("Unsupported node type for %s: %s", nodeName, nodeType)
	}

	subnet := Subnet{}

	subnet.Name, err = getSubnetName(kv, cfg, deploymentID, nodeName)
	if err != nil {
		return Subnet{}, err
	}

	subnet.IPVersion, err = getSubnetIPVersion(kv, deploymentID, nodeName)
	if err != nil {
		return Subnet{}, err
	}

	subnet.NetworkID, err = getSubnetNetworkID(kv, deploymentID, nodeName, resourceType)
	if err != nil {
		return Subnet{}, err
	}

	subnet.CIDR, err = deployments.GetStringNodeProperty(kv, deploymentID, nodeName,
		"cidr", false)
	if err != nil {
		return Subnet{}, err
	}

	subnet.GatewayIP, err = deployments.GetStringNodeProperty(kv, deploymentID, nodeName,
		"gateway_ip", false)
	if err != nil {
		return Subnet{}, err
	}

	startIP, err := deployments.GetStringNodeProperty(kv, deploymentID, nodeName,
		"start_ip", false)
	if err != nil {
		return Subnet{}, err
	}
	if startIP != "" {
		endIP, err := deployments.GetStringNodeProperty(kv, deploymentID, nodeName,
			"end_ip", false)
		if err != nil {
			return Subnet{}, err
		}
		if endIP == "" {
			return Subnet{}, errors.Errorf("A start_ip and a end_ip need to be provided")
		}
		subnet.AllocationPools = &AllocationPool{Start: startIP, End: endIP}
	}

	subnet.EnableDHCP, err = isDHCPEnabled(kv, deploymentID, nodeName)
	if err != nil {
		return Subnet{}, err
	}

	subnet.Region = cfg.Infrastructures[infrastructureName].GetStringOrDefault("region", defaultOSRegion)

	return subnet, nil
}

func getSubnetName(kv *api.KV, cfg config.Configuration, deploymentID, nodeName string) (string, error) {

	var subnetName string
	netName, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "network_name")
	if err != nil {
		return "", err
	}
	if netName != nil && netName.RawString() != "" {
		subnetName = cfg.ResourcesPrefix + netName.RawString() + "_subnet"
	} else {
		subnetName = cfg.ResourcesPrefix + nodeName + "_subnet"
	}
	return subnetName, err
}

func getSubnetIPVersion(kv *api.KV, deploymentID, nodeName string) (int, error) {

	ipVersion := 4
	ipVersionProp, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "ip_version")
	if err != nil {
		return ipVersion, err
	}
	if ipVersionProp != nil && ipVersionProp.RawString() != "" {
		ipVersion, err = strconv.Atoi(ipVersionProp.RawString())
	}

	return ipVersion, err
}

func getSubnetNetworkID(kv *api.KV, deploymentID, nodeName, resourceType string) (string, error) {

	var networkID string
	nodeID, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "network_id")
	if err != nil {
		return networkID, err
	}
	if nodeID != nil && nodeID.RawString() != "" {
		networkID = nodeID.RawString()
	} else {
		networkID = fmt.Sprintf("${%s.%s.id}", resourceType, nodeName)
	}
	return networkID, err
}

func isDHCPEnabled(kv *api.KV, deploymentID, nodeName string) (bool, error) {

	dhcpEnabled := true
	dhcpVal, err := deployments.GetStringNodeProperty(kv, deploymentID, nodeName,
		"dhcp_enabled", false)
	if err != nil {
		return dhcpEnabled, err
	}
	if dhcpVal != "" {
		dhcpEnabled, err = strconv.ParseBool(dhcpVal)
	}

	return dhcpEnabled, err
}
