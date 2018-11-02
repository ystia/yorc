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

package google

import (
	"context"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov/terraform/commons"
	"path"
	"strconv"
	"strings"
)

func (g *googleGenerator) generatePrivateNetwork(ctx context.Context, kv *api.KV,
	cfg config.Configuration, deploymentID, nodeName, instanceName string, instanceID int,
	infrastructure *commons.Infrastructure,
	outputs map[string]string) error {

	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return errors.Wrapf(err, "failed to generate private network for deploymentID:%q, nodeName:%q, instanceName:%q", deploymentID, nodeName, instanceName)
	}
	if nodeType != "yorc.nodes.google.PrivateNetwork" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}

	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID,
		"topology", "instances")
	instancesKey := path.Join(instancesPrefix, nodeName)

	privateNetwork := &PrivateNetwork{}

	stringParams := []struct {
		pAttr        *string
		propertyName string
		mandatory    bool
	}{
		{&privateNetwork.Description, "description", false},
		{&privateNetwork.RoutingMode, "routing_mode", false},
		{&privateNetwork.Project, "project", false},
		{&privateNetwork.Name, "network_name", false},
	}

	for _, stringParam := range stringParams {
		if *stringParam.pAttr, err = deployments.GetStringNodeProperty(kv, deploymentID, nodeName,
			stringParam.propertyName, stringParam.mandatory); err != nil {
			return errors.Wrapf(err, "failed to generate private network for deploymentID:%q, nodeName:%q, instanceName:%q", deploymentID, nodeName, instanceName)
		}
	}

	// Use existing private network if defined with network_name
	if privateNetwork.Name != "" {
		// Just provide network_name attribute
		consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{}}
		consulKeyNetwork := commons.ConsulKey{
			Path:  path.Join(instancesKey, instanceName, "/attributes/network_name"),
			Value: privateNetwork.Name,
		}

		consulKeys.Keys = append(consulKeys.Keys, consulKeyNetwork)
		commons.AddResource(infrastructure, "consul_keys", privateNetwork.Name, &consulKeys)
		return nil
	}

	name := strings.ToLower(cfg.ResourcesPrefix + nodeName + "-" + instanceName)
	privateNetwork.Name = strings.Replace(name, "_", "-", -1)

	autoCreateSubNets := true // Default is sub-networks auto-creation
	s, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "auto_create_subnetworks")
	if err != nil {
		return errors.Wrapf(err, "failed to generate private network for deploymentID:%q, nodeName:%q, instanceName:%q", deploymentID, nodeName, instanceName)
	}
	if s != nil && s.RawString() != "" {
		autoCreateSubNets, err = strconv.ParseBool(s.RawString())
		if err != nil {
			return errors.Wrapf(err, "failed to generate private network for deploymentID:%q, nodeName:%q, instanceName:%q", deploymentID, nodeName, instanceName)
		}
	}
	privateNetwork.AutoCreateSubNetworks = autoCreateSubNets

	// Check if sub-networks have to be created
	customSubnets, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "custom_subnetworks")
	if err != nil {
		return errors.Wrapf(err, "failed to generate private network for deploymentID:%q, nodeName:%q, instanceName:%q", deploymentID, nodeName, instanceName)
	}
	if customSubnets != nil && customSubnets.RawString() != "" {
		subnets, ok := customSubnets.Value.([]interface{})
		if !ok {
			return errors.New("failed to retrieve yorc.datatypes.google.Subnetwork Tosca Value: not expected type")
		}
		for i := range subnets {
			if err = g.generateSubNetwork(ctx, kv, cfg, deploymentID, nodeName, instanceName, instanceID, infrastructure, i, privateNetwork.Name); err != nil {
				return errors.Wrapf(err, "failed to generate private network for deploymentID:%q, nodeName:%q, instanceName:%q", deploymentID, nodeName, instanceName)
			}
		}
	} else if !autoCreateSubNets {
		return errors.New("at least one custom sub-network must be provided if sub-networks auto-creation mode is false")
	}
	log.Debugf("Add network:%+v", privateNetwork)
	commons.AddResource(infrastructure, "google_compute_network", privateNetwork.Name, privateNetwork)

	// Add default firewall
	externalFw := &Firewall{
		Name:         fmt.Sprintf("%s-default-external-fw", privateNetwork.Name),
		Network:      fmt.Sprintf("${google_compute_network.%s.name}", privateNetwork.Name),
		SourceRanges: []string{"0.0.0.0/0"},
		Allow: []AllowRule{
			{Protocol: "icmp"},
			{Protocol: "TCP", Ports: []string{"3389"}}, // RDP
			{Protocol: "TCP", Ports: []string{"22"}},   // SSH
		}}
	commons.AddResource(infrastructure, "google_compute_firewall", externalFw.Name, externalFw)

	// Provide Consul Key for network_name
	consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{}}
	consulKeyNetwork := commons.ConsulKey{
		Path:  path.Join(instancesKey, instanceName, "/attributes/network_name"),
		Value: fmt.Sprintf("${google_compute_network.%s.name}", privateNetwork.Name)}

	consulKeys.Keys = append(consulKeys.Keys, consulKeyNetwork)
	commons.AddResource(infrastructure, "consul_keys", privateNetwork.Name, &consulKeys)
	return nil
}

func (g *googleGenerator) generateSubNetwork(ctx context.Context, kv *api.KV,
	cfg config.Configuration, deploymentID, nodeName, instanceName string, instanceID int,
	infrastructure *commons.Infrastructure, subNetIndex int, networkName string) error {

	ind := strconv.Itoa(subNetIndex)
	subnet := &SubNetwork{}

	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID,
		"topology", "instances")
	instancesKey := path.Join(instancesPrefix, nodeName)

	strParams := []struct {
		pAttr        *string
		propertyName string
		mandatory    bool
	}{
		{&subnet.Name, "name", true},
		{&subnet.IPCIDRRange, "ip_cidr_range", true},
		{&subnet.Region, "region", true},
		{&subnet.Description, "description", false},
		{&subnet.Project, "project", false},
	}
	for _, param := range strParams {
		value, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "custom_subnetworks", ind, param.propertyName)
		if err != nil {
			return errors.Wrapf(err, "failed to generate sub-network %d for network:%q", subNetIndex, networkName)
		}
		if value != nil && value.RawString() != "" {
			*param.pAttr = value.RawString()
		} else if param.mandatory {
			return errors.Errorf("%s is a mandatory property for sub-network with index:%d and network:%q", param.propertyName, subNetIndex, networkName)
		}
	}
	boolParams := []struct {
		pAttr        *bool
		propertyName string
		mandatory    bool
	}{
		{&subnet.EnableFlowLogs, "enable_flow_logs", false},
		{&subnet.PrivateIPGoogleAccess, "private_ip_google_access", false},
	}
	for _, param := range boolParams {
		value, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "custom_subnetworks", ind, param.propertyName)
		if err != nil {
			return errors.Wrapf(err, "failed to generate sub-network %d for network:%q", subNetIndex, networkName)
		}
		if value != nil && value.RawString() != "" {
			*param.pAttr, err = strconv.ParseBool(value.RawString())
			if err != nil {
				return errors.Wrapf(err, "failed to generate sub-network %d for network:%q", subNetIndex, networkName)
			}
		} else if param.mandatory {
			return errors.Errorf("%s is a mandatory property for sub-network with index:%d and network:%q", param.propertyName, subNetIndex, networkName)
		}
	}

	// Name must respect regular expression
	subnet.Name = strings.Replace(strings.ToLower(subnet.Name), "_", "-", -1)

	// Handle secondary IP ranges
	var secondarySourceRange []string
	secondaryIPRangesRaws, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "custom_subnetworks", ind, "secondary_ip_ranges")
	if err != nil {
		return errors.Wrapf(err, "failed to generate sub-network %d for network:%q", subNetIndex, networkName)
	} else if secondaryIPRangesRaws != nil && secondaryIPRangesRaws.RawString() != "" {
		list, ok := secondaryIPRangesRaws.Value.([]interface{})
		if !ok {
			return errors.New("failed to retrieve yorc.datatypes.google.IPRange Tosca Value: not expected type")
		}

		ipRanges := make([]IPRange, len(list))
		for i := range list {
			ipRange, err := buildIPRange(kv, deploymentID, nodeName, ind, networkName, i)
			if err != nil {
				return errors.Wrapf(err, "failed to generate private network for deploymentID:%q, nodeName:%q, instanceName:%q", deploymentID, nodeName, instanceName)
			}
			ipRanges = append(ipRanges, *ipRange)
			secondarySourceRange = append(secondarySourceRange, ipRange.IPCIDRRange)
		}
		subnet.SecondaryIPRanges = ipRanges
	}

	subnet.Network = fmt.Sprintf("${google_compute_network.%s.name}", networkName)
	log.Debugf("Add subnet:%+v", subnet)
	commons.AddResource(infrastructure, "google_compute_subnetwork", subnet.Name, subnet)

	// Provide Consul Key for attribute gateway_ip
	consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{}}
	consulKeyGateway := commons.ConsulKey{
		Path:  path.Join(instancesKey, instanceName, fmt.Sprintf("/attributes/subnets/%s/gateway_ip", subnet.Name)),
		Value: fmt.Sprintf("${google_compute_subnetwork.%s.gateway_address}", subnet.Name)}

	consulKeys.Keys = append(consulKeys.Keys, consulKeyGateway)
	commons.AddResource(infrastructure, "consul_keys", subnet.Name, &consulKeys)

	// Add internal firewall rules for subnet
	sourceRanges := append(secondarySourceRange, subnet.IPCIDRRange)
	internalFw := &Firewall{
		Name:         fmt.Sprintf("%s-default-internal-fw", subnet.Name),
		Network:      fmt.Sprintf("${google_compute_network.%s.name}", networkName),
		SourceRanges: sourceRanges,
		Allow: []AllowRule{
			{Protocol: "icmp"},
			{Protocol: "TCP", Ports: []string{"0-65535"}}, // RDP
			{Protocol: "UDP", Ports: []string{"0-65535"}}, // SSH
		}}
	commons.AddResource(infrastructure, "google_compute_firewall", internalFw.Name, internalFw)
	return nil
}

func buildIPRange(kv *api.KV, deploymentID, nodeName, subNetIndex, networkName string, ipRangeIndex int) (*IPRange, error) {

	ind := strconv.Itoa(ipRangeIndex)
	ipRange := &IPRange{}
	// Name is a mandatory property
	nameRaw, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "custom_subnetworks", subNetIndex, "secondary_ip_ranges", ind, "name")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate sub-network secondary ip range with sub-net index:%q, network:%q and range index:%q", subNetIndex, networkName, ind)
	} else if nameRaw == nil || nameRaw.RawString() == "" {
		return nil, errors.Errorf("Name is a mandatory property for sub-network secondary ip range with sub-net index:%q, network:%q and range index:%q", subNetIndex, networkName, ind)
	}
	ipRange.Name = strings.Replace(strings.ToLower(nameRaw.RawString()), "_", "-", -1)

	// IPCIDRRange is a mandatory property
	cidrRaw, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "custom_subnetworks", subNetIndex, "secondary_ip_ranges", ind, "ip_cidr_range")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate sub-network secondary ip range with sub-net index:%q, network:%q and range index:%q", subNetIndex, networkName, ind)
	} else if cidrRaw == nil || cidrRaw.RawString() == "" {
		return nil, errors.Errorf("Name is a mandatory property for sub-network secondary ip range with sub-net index:%q, network:%q and range index:%q", subNetIndex, networkName, ind)
	}
	ipRange.IPCIDRRange = cidrRaw.RawString()
	return ipRange, nil
}

func getFirstMatchingSubnetByRegion(kv *api.KV, deploymentID, nodeName, region string) (string, error) {
	subnets, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "custom_subnetworks")
	if err != nil {
		return "", errors.Wrapf(err, "failed to retrieve sub-networks for deploymentID:%q, nodeName:%q", deploymentID, nodeName)
	}
	if subnets != nil && subnets.RawString() != "" {
		subnets, ok := subnets.Value.([]interface{})
		if !ok {
			return "", errors.New("failed to retrieve yorc.datatypes.google.Subnetwork Tosca Value: not expected type")
		}
		for i := range subnets {
			ind := strconv.Itoa(i)
			regionRaw, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "custom_subnetworks", ind, "region")
			if err != nil {
				return "", errors.Wrapf(err, "failed to retrieve sub-network region for deploymentID:%q, nodeName:%q, sub-network index:%q", deploymentID, nodeName, ind)
			} else if regionRaw != nil && regionRaw.RawString() != "" {
				if region == regionRaw.RawString() {
					nameRaw, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "custom_subnetworks", ind, "name")
					if err != nil || nameRaw == nil || nameRaw.RawString() == "" {
						return "", errors.Wrapf(err, "failed to retrieve sub-network name for deploymentID:%q, nodeName:%q, sub-network index:%q", deploymentID, nodeName, ind)
					}
					return nameRaw.RawString(), nil
				}
			}
		}
	}
	return "", nil
}
