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
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
)

func (g *googleGenerator) generatePrivateNetwork(ctx context.Context, cfg config.Configuration, deploymentID, nodeName string, infrastructure *commons.Infrastructure, outputs map[string]string) error {

	nodeType, err := deployments.GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return errors.Wrapf(err, "failed to generate private network for deploymentID:%q, nodeName:%q", deploymentID, nodeName)
	}
	if nodeType != "yorc.nodes.google.PrivateNetwork" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}
	nodeInstanceKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName, "0")
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
		if *stringParam.pAttr, err = deployments.GetStringNodeProperty(ctx, deploymentID, nodeName,
			stringParam.propertyName, stringParam.mandatory); err != nil {
			return errors.Wrapf(err, "failed to generate private network for deploymentID:%q, nodeName:%q", deploymentID, nodeName)
		}
	}

	// Use existing private network if defined with network_name
	if privateNetwork.Name != "" {
		// Provide output for network_name
		networkKey := nodeName + "-network"
		commons.AddOutput(infrastructure, networkKey, &commons.Output{Value: privateNetwork.Name})
		outputs[path.Join(nodeInstanceKey, "/attributes/network_name")] = networkKey
		return nil
	}

	name := strings.ToLower(getResourcesPrefix(cfg, deploymentID) + nodeName)
	privateNetwork.Name = strings.Replace(name, "_", "-", -1)

	var autoCreateSubNets bool
	s, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "auto_create_subnetworks")
	if err != nil {
		return errors.Wrapf(err, "failed to generate private network for deploymentID:%q, nodeName:%q", deploymentID, nodeName)
	}
	if s != nil && s.RawString() != "" {
		autoCreateSubNets, err = strconv.ParseBool(s.RawString())
		if err != nil {
			return errors.Wrapf(err, "failed to generate private network for deploymentID:%q, nodeName:%q", deploymentID, nodeName)
		}
	}
	privateNetwork.AutoCreateSubNetworks = autoCreateSubNets
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

	// Provide output for network_name
	networkKey := nodeName + "-network"
	commons.AddOutput(infrastructure, networkKey, &commons.Output{Value: fmt.Sprintf("${google_compute_network.%s.name}", privateNetwork.Name)})
	outputs[path.Join(nodeInstanceKey, "/attributes/network_name")] = networkKey
	return nil
}

func (g *googleGenerator) generateSubNetwork(ctx context.Context, cfg config.Configuration, deploymentID, nodeName string, infrastructure *commons.Infrastructure, outputs map[string]string) error {

	subnet := &SubNetwork{}
	nodeInstanceKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName, "0")

	var err error
	strParams := []struct {
		pAttr        *string
		propertyName string
		mandatory    bool
	}{
		{&subnet.Name, "name", true},
		{&subnet.IPCIDRRange, "ip_cidr_range", true},
		{&subnet.Region, "region", true},
		{&subnet.Network, "network", false},
		{&subnet.Description, "description", false},
		{&subnet.Project, "project", false},
	}
	for _, param := range strParams {
		*param.pAttr, err = deployments.GetStringNodeProperty(ctx, deploymentID, nodeName, param.propertyName, param.mandatory)
		if err != nil {
			return errors.Wrapf(err, "failed to generate sub-network for deploymentID:%q, nodeName:%q", deploymentID, nodeName)
		}
	}
	boolParams := []struct {
		pAttr        *bool
		propertyName string
	}{
		{&subnet.EnableFlowLogs, "enable_flow_logs"},
		{&subnet.PrivateIPGoogleAccess, "private_ip_google_access"},
	}
	for _, param := range boolParams {
		*param.pAttr, err = deployments.GetBooleanNodeProperty(ctx, deploymentID, nodeName, param.propertyName)
		if err != nil {
			return errors.Wrapf(err, "failed to generate sub-network for deploymentID:%q, nodeName:%q", deploymentID, nodeName)
		}
	}

	// Network is either set by user or retrieved via dependency relationship with network node
	if subnet.Network == "" {
		hasDep, networkNode, err := deployments.HasAnyRequirementFromNodeType(ctx, deploymentID, nodeName, "dependency", "yorc.nodes.google.PrivateNetwork")
		if err != nil {
			return err
		}
		if !hasDep {
			return errors.Errorf("failed to retrieve dependency btw any network and the subnet with name:%q", subnet.Name)
		}

		subnet.Network, err = deployments.LookupInstanceAttributeValue(ctx, deploymentID, networkNode, "0", "network_name")
		if err != nil {
			return err
		}
	}

	// As subnet name must be unique in Google project, we concat its name with network name
	subnet.Name = subnet.Network + "-" + subnet.Name

	// Name must respect regular expression
	subnet.Name = strings.Replace(strings.ToLower(subnet.Name), "_", "-", -1)

	// Handle secondary IP ranges
	var secondarySourceRange []string
	secondaryIPRangesRaws, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "secondary_ip_ranges")
	if err != nil {
		return errors.Wrapf(err, "failed to generate sub-network for deploymentID:%q, nodeName:%q", deploymentID, nodeName)
	} else if secondaryIPRangesRaws != nil && secondaryIPRangesRaws.RawString() != "" {
		list, ok := secondaryIPRangesRaws.Value.([]interface{})
		if !ok {
			return errors.New("failed to retrieve yorc.datatypes.google.IPRange Tosca Value: not expected type")
		}

		ipRanges := make([]IPRange, 0)
		for i := range list {
			ipRange, err := buildIPRange(ctx, deploymentID, nodeName, i)
			if err != nil {
				return errors.Wrapf(err, "failed to generate sub-network for deploymentID:%q, nodeName:%q", deploymentID, nodeName)
			}
			ipRanges = append(ipRanges, *ipRange)
			secondarySourceRange = append(secondarySourceRange, ipRange.IPCIDRRange)
		}
		subnet.SecondaryIPRanges = ipRanges
	}

	log.Debugf("Add subnet:%+v", subnet)
	commons.AddResource(infrastructure, "google_compute_subnetwork", subnet.Name, subnet)

	// Provide outputs
	gatewayKey := nodeName + "-gateway"
	commons.AddOutput(infrastructure, gatewayKey, &commons.Output{Value: fmt.Sprintf("${google_compute_subnetwork.%s.gateway_address}", subnet.Name)})
	outputs[path.Join(nodeInstanceKey, "/attributes/gateway_ip")] = gatewayKey

	networkKey := nodeName + "-network"
	commons.AddOutput(infrastructure, networkKey, &commons.Output{Value: subnet.Network})
	outputs[path.Join(nodeInstanceKey, "/attributes/network_name")] = networkKey

	subnetKey := nodeName + "-subnet"
	commons.AddOutput(infrastructure, subnetKey, &commons.Output{Value: subnet.Name})
	outputs[path.Join(nodeInstanceKey, "/attributes/subnetwork_name")] = subnetKey

	// Add internal firewall rules for subnet
	sourceRanges := append(secondarySourceRange, subnet.IPCIDRRange)
	internalFw := &Firewall{
		Name:         fmt.Sprintf("%s-default-internal-fw", subnet.Name),
		Network:      fmt.Sprintf(subnet.Network),
		SourceRanges: sourceRanges,
		Allow: []AllowRule{
			{Protocol: "icmp"},
			{Protocol: "TCP", Ports: []string{"0-65535"}}, // RDP
			{Protocol: "UDP", Ports: []string{"0-65535"}}, // SSH
		}}
	commons.AddResource(infrastructure, "google_compute_firewall", internalFw.Name, internalFw)
	return nil
}

func buildIPRange(ctx context.Context, deploymentID, nodeName string, ipRangeIndex int) (*IPRange, error) {
	ind := strconv.Itoa(ipRangeIndex)
	ipRange := &IPRange{}
	// Name is a mandatory property
	nameRaw, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "secondary_ip_ranges", ind, "name")
	if err != nil {
		return nil, err
	} else if nameRaw == nil || nameRaw.RawString() == "" {
		return nil, errors.New("Missing mandatory name for ip range")
	}
	ipRange.Name = strings.Replace(strings.ToLower(nameRaw.RawString()), "_", "-", -1)

	// IPCIDRRange is a mandatory property
	cidrRaw, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "secondary_ip_ranges", ind, "ip_cidr_range")
	if err != nil {
		return nil, err
	} else if cidrRaw == nil || cidrRaw.RawString() == "" {
		return nil, errors.New("Missing mandatory IP CIDR Range for ip range")
	}
	ipRange.IPCIDRRange = cidrRaw.RawString()
	return ipRange, nil
}
