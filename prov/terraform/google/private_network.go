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
		return err
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
	}

	for _, stringParam := range stringParams {
		if *stringParam.pAttr, err = deployments.GetStringNodeProperty(kv, deploymentID, nodeName,
			stringParam.propertyName, stringParam.mandatory); err != nil {
			return err
		}
	}

	name := strings.ToLower(cfg.ResourcesPrefix + nodeName + "-" + instanceName)
	privateNetwork.Name = strings.Replace(name, "_", "-", -1)

	autoCreateSubNets := true // Default is sub-networks auto-creation
	s, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "auto_create_subnetworks")
	if err != nil {
		return err
	}
	if s != nil && s.RawString() != "" {
		autoCreateSubNets, err = strconv.ParseBool(s.RawString())
		if err != nil {
			return err
		}
	}
	privateNetwork.AutoCreateSubNetworks = autoCreateSubNets

	// Check if sub-networks have to be created
	customSubnets, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "custom_subnetworks")
	if err != nil {
		return err
	}
	if customSubnets != nil && customSubnets.RawString() != "" {
		subnets, ok := customSubnets.Value.([]interface{})
		if !ok {
			return errors.New("failed to retrieve yorc.datatypes.google.Subnetwork Tosca Value: not expected type")
		}
		for i := range subnets {
			if err = g.generateSubNetwork(ctx, kv, cfg, deploymentID, nodeName, instanceName, instanceID, infrastructure, i, privateNetwork.Name); err != nil {
				return err
			}
		}
	} else if !autoCreateSubNets {
		return errors.New("at least one custom sub-network must be provided is sub-networks auto-creation is false")
	}
	log.Debugf("network=%+v", privateNetwork)
	commons.AddResource(infrastructure, "google_compute_network", privateNetwork.Name, privateNetwork)

	// Provide Consul Key for attribute gateway_ip and network_name
	consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{}}
	consulKeyGateway := commons.ConsulKey{
		Path:  path.Join(instancesKey, instanceName, "/attributes/gateway_ip"),
		Value: fmt.Sprintf("${google_compute_network.%s.gateway_ipv4}", privateNetwork.Name)}
	consulKeyNetwork := commons.ConsulKey{
		Path:  path.Join(instancesKey, instanceName, "/attributes/network_name"),
		Value: fmt.Sprintf("${google_compute_network.%s.name}", privateNetwork.Name)}

	consulKeys.Keys = append(consulKeys.Keys, consulKeyGateway, consulKeyNetwork)
	commons.AddResource(infrastructure, "consul_keys", privateNetwork.Name, &consulKeys)
	return nil
}

func (g *googleGenerator) generateSubNetwork(ctx context.Context, kv *api.KV,
	cfg config.Configuration, deploymentID, nodeName, instanceName string, instanceID int,
	infrastructure *commons.Infrastructure, subNetIndex int, networkName string) error {

	subnet := &SubNetwork{}
	// Name is a mandatory property
	nameRaw, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "custom_subnetworks", strconv.Itoa(subNetIndex), "name")
	if err != nil {
		return errors.Wrapf(err, "failed to generate sub-network %d for network:%q", subNetIndex, networkName)
	} else if nameRaw == nil || nameRaw.RawString() == "" {
		return errors.Errorf("Name is a mandatory property for sub-network with index:%d and network:%q", subNetIndex, networkName)
	}
	subnet.Name = strings.Replace(strings.ToLower(nameRaw.RawString()), "_", "-", -1)

	// IP CIDR range is a mandatory property
	cidrRaw, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "custom_subnetworks", strconv.Itoa(subNetIndex), "ip_cidr_range")
	if err != nil {
		return errors.Wrapf(err, "failed to generate sub-network %d for network:%q", subNetIndex, networkName)
	} else if cidrRaw == nil || cidrRaw.RawString() == "" {
		return errors.Errorf("IP CIDR Range is a mandatory property for sub-network with index:%d and network:%q", subNetIndex, networkName)
	}
	subnet.IPCIDRRange = cidrRaw.RawString()

	regionRaw, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "custom_subnetworks", strconv.Itoa(subNetIndex), "region")
	if err != nil {
		return errors.Wrapf(err, "failed to generate sub-network %d for network:%q", subNetIndex, networkName)
	} else if cidrRaw != nil && cidrRaw.RawString() != "" {
		subnet.Region = regionRaw.RawString()
	}
	if subnet.Region == "" {
		if cfg.Infrastructures[infrastructureName].GetString("region") == "" {
			return errors.New("Region must be set for ComputeAddress node type or in google infrastructure config")
		}
		subnet.Region = cfg.Infrastructures[infrastructureName].GetString("region")
	}

	subnet.Network = fmt.Sprintf("${google_compute_network.%s.name}", networkName)
	log.Debugf("subnet is:%+v", subnet)

	commons.AddResource(infrastructure, "google_compute_subnetwork", subnet.Name, subnet)
	return nil
}
