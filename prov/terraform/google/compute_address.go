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
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/prov/terraform/commons"
)

func (g *googleGenerator) generateComputeAddress(ctx context.Context, kv *api.KV,
	cfg config.Configuration, deploymentID, nodeName, instanceName string, instanceID int,
	infrastructure *commons.Infrastructure,
	outputs map[string]string) error {

	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType != "yorc.nodes.google.Address" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}

	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID,
		"topology", "instances")
	instancesKey := path.Join(instancesPrefix, nodeName)

	computeAddress := &ComputeAddress{}
	var addresses string
	stringParams := []struct {
		pAttr        *string
		propertyName string
		mandatory    bool
	}{
		{&computeAddress.Region, "region", false},
		{&computeAddress.Description, "description", false},
		{&addresses, "addresses", false},
		{&computeAddress.SubNetwork, "subnetwork", false},
		{&computeAddress.Project, "project", false},
		{&computeAddress.SubNetwork, "subnetwork", false},
		{&computeAddress.NetworkTier, "network_tier", false},
		{&computeAddress.AddressType, "address_type", false},
	}

	for _, stringParam := range stringParams {
		if *stringParam.pAttr, err = deployments.GetStringNodeProperty(kv, deploymentID, nodeName,
			stringParam.propertyName, stringParam.mandatory); err != nil {
			return err
		}
	}

	computeAddress.Labels, err = deployments.GetKeyValuePairsNodeProperty(kv, deploymentID, nodeName, "labels")
	if err != nil {
		return err
	}

	name := strings.ToLower(getResourcesPrefix(cfg, deploymentID) + nodeName + "-" + instanceName)
	computeAddress.Name = strings.Replace(name, "_", "-", -1)

	if computeAddress.Region == "" {
		if cfg.Infrastructures[infrastructureName].GetString("region") == "" {
			return errors.New("Region must be set for ComputeAddress node type or in google infrastructure config")
		}
		computeAddress.Region = cfg.Infrastructures[infrastructureName].GetString("region")
	}

	var ipAddress string
	if addresses != "" {
		tabAdd := strings.Split(addresses, ",")
		if len(tabAdd) > instanceID {
			ipAddress = strings.TrimSpace(tabAdd[instanceID])
		}
	}

	// Add google compute computeAddress resource if not any IP computeAddress is provided for this instance
	if ipAddress == "" {
		commons.AddResource(infrastructure, "google_compute_address", computeAddress.Name, computeAddress)
		ipAddress = fmt.Sprintf("${google_compute_address.%s.address}", computeAddress.Name)
	}

	// Provide Consul Key for attribute ip_address
	consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{}}
	consulKeyIPAddr := commons.ConsulKey{
		Path:  path.Join(instancesKey, instanceName, "/attributes/ip_address"),
		Value: ipAddress}

	consulKeys.Keys = append(consulKeys.Keys, consulKeyIPAddr)
	commons.AddResource(infrastructure, "consul_keys", computeAddress.Name, &consulKeys)
	return nil
}
