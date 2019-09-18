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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
	"github.com/ystia/yorc/v4/tosca"
)

const (
	floatingIPEndpointCapAttribute = "/capabilities/endpoint/attributes/floating_ip_address"
	consulKeysResource             = "consul_keys"
)

type osGenerator struct {
}

type generateInfraOptions struct {
	kv             *api.KV
	cfg            config.Configuration
	infrastructure *commons.Infrastructure
	locationProps  config.DynamicMap
	instancesKey   string
	deploymentID   string
	nodeName       string
	nodeType       string
	instanceName   string
	instanceIndex  int
	resourceTypes  map[string]string
}

// func (g *osGenerator) getStringFormConsul(kv *api.KV, baseURL, property string) (string, error) {
// 	getResult, _, err := kv.Get(baseURL+"/"+property, nil)
// 	if err != nil {
// 		return "", errors.Errorf("Can't get property %s for node %s: %v", property, baseURL, err)
// 	}
// 	if getResult == nil {
// 		log.Debugf("Can't get property %s for node %s (not found)", property, baseURL)
// 		return "", nil
// 	}
// 	return string(getResult.Value), nil
// }

func (g *osGenerator) GenerateTerraformInfraForNode(ctx context.Context, cfg config.Configuration, deploymentID, nodeName, infrastructurePath string) (bool, map[string]string, []string, commons.PostApplyCallback, error) {
	log.Debugf("Generating infrastructure for deployment with id %s", deploymentID)
	cClient, err := cfg.GetConsulClient()
	if err != nil {
		return false, nil, nil, nil, err
	}
	kv := cClient.KV()

	return g.generateTerraformInfraForNode(ctx, kv, cfg, deploymentID, nodeName, infrastructurePath)
}

func (g *osGenerator) generateTerraformInfraForNode(ctx context.Context, kv *api.KV,
	cfg config.Configuration, deploymentID,
	nodeName, infrastructurePath string) (bool, map[string]string, []string, commons.PostApplyCallback, error) {

	instancesKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName)
	terraformStateKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "terraform-state", nodeName)

	infrastructure := commons.Infrastructure{}

	log.Debugf("Generating infrastructure for deployment with node %s", nodeName)

	// Remote Configuration for Terraform State to store it in the Consul KV store
	infrastructure.Terraform = commons.GetBackendConfiguration(terraformStateKey, cfg)

	locationProps, err := locations.GetLocationPropertiesForNode(deploymentID, nodeName, infrastructureType)
	if err != nil {
		return false, nil, nil, nil, err
	}
	var cmdEnv []string
	infrastructure.Provider, cmdEnv = getOpenStackProviderEnv(cfg, locationProps)

	log.Debugf("inspecting node %s", nodeName)
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, nil, nil, nil, err
	}
	outputs := make(map[string]string)

	instances, err := deployments.GetNodeInstancesIds(kv, deploymentID, nodeName)
	if err != nil {
		return false, nil, nil, nil, err
	}

	resourceTypes := getOpenstackResourceTypes(locationProps)

	for instIdx, instanceName := range instances {
		infraOpts := generateInfraOptions{
			kv:             kv,
			cfg:            cfg,
			infrastructure: &infrastructure,
			locationProps:  locationProps,
			instancesKey:   instancesKey,
			deploymentID:   deploymentID,
			nodeName:       nodeName,
			nodeType:       nodeType,
			instanceName:   instanceName,
			instanceIndex:  instIdx,
			resourceTypes:  resourceTypes,
		}
		err := g.generateInstanceInfra(ctx, infraOpts, outputs, &cmdEnv)
		if err != nil {
			return false, nil, nil, nil, err
		}
	}

	jsonInfra, err := json.MarshalIndent(infrastructure, "", "  ")
	if err != nil {
		return false, nil, nil, nil, errors.Wrap(err, "Failed to generate JSON of terraform Infrastructure description")
	}

	if err = ioutil.WriteFile(filepath.Join(infrastructurePath, "infra.tf.json"), jsonInfra, 0664); err != nil {
		return false, nil, nil, nil, errors.Wrapf(err, "Failed to write file %q", filepath.Join(infrastructurePath, "infra.tf.json"))
	}

	log.Debugf("Infrastructure generated for deployment with id %s", deploymentID)
	return true, outputs, cmdEnv, nil, nil
}

func getOpenStackProviderEnv(cfg config.Configuration, locationProps config.DynamicMap) (map[string]interface{}, []string) {
	cmdEnv := []string{
		fmt.Sprintf("OS_USERNAME=%s", locationProps.GetString("user_name")),
		fmt.Sprintf("OS_PASSWORD=%s", locationProps.GetString("password")),
		fmt.Sprintf("OS_PROJECT_NAME=%s", locationProps.GetString("project_name")),
		fmt.Sprintf("OS_PROJECT_ID=%s", locationProps.GetString("project_id")),
		fmt.Sprintf("OS_USER_DOMAIN_NAME=%s", locationProps.GetString("user_domain_name")),
		fmt.Sprintf("OS_AUTH_URL=%s", locationProps.GetString("auth_url")),
	}

	// Management of variables for Terraform
	provider := map[string]interface{}{
		"openstack": map[string]interface{}{
			"version":     cfg.Terraform.OpenStackPluginVersionConstraint,
			"tenant_name": locationProps.GetString("tenant_name"),
			"insecure":    locationProps.GetString("insecure"),
			"cacert_file": locationProps.GetString("cacert_file"),
			"cert":        locationProps.GetString("cert"),
			"key":         locationProps.GetString("key"),
		},
		"consul": commons.GetConsulProviderfiguration(cfg),
		"null": map[string]interface{}{
			"version": commons.NullPluginVersionConstraint,
		},
	}

	return provider, cmdEnv
}

func (g *osGenerator) generateInstanceInfra(ctx context.Context, opts generateInfraOptions,
	outputs map[string]string, cmdEnv *[]string) error {

	instanceState, err := deployments.GetInstanceState(opts.kv, opts.deploymentID,
		opts.nodeName, opts.instanceName)
	if err != nil {
		return err
	}
	if instanceState == tosca.NodeStateDeleting || instanceState == tosca.NodeStateDeleted {
		// Do not generate something for this node instance (will be deleted if exists)
		return err
	}

	switch opts.nodeType {
	case "yorc.nodes.openstack.Compute":
		err = g.generateOSInstance(ctx,
			osInstanceOptions{
				kv:             opts.kv,
				cfg:            opts.cfg,
				infrastructure: opts.infrastructure,
				locationProps:  opts.locationProps,
				deploymentID:   opts.deploymentID,
				nodeName:       opts.nodeName,
				instanceName:   opts.instanceName,
				resourceTypes:  opts.resourceTypes,
			},
			outputs, cmdEnv)

	case "yorc.nodes.openstack.BlockStorage":
		err = g.generateBlockStorageInfra(opts)

	case "yorc.nodes.openstack.FloatingIP":
		err = g.generateFloatingIPInfra(opts)

	case "yorc.nodes.openstack.Network":
		err = g.generateNetworkInfra(opts)

	case "yorc.nodes.openstack.ServerGroup":
		err = g.generateServerGroup(ctx,
			serverGroupOptions{
				kv:            opts.kv,
				deploymentID:  opts.deploymentID,
				nodeName:      opts.nodeName,
				resourceTypes: opts.resourceTypes,
			},
			opts.infrastructure, outputs, cmdEnv)
	default:
		err = errors.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'",
			opts.nodeType, opts.nodeName, opts.deploymentID)
	}
	return err
}

func (g *osGenerator) generateBlockStorageInfra(opts generateInfraOptions) error {

	var bsIds []string
	volumeID, err := deployments.GetNodePropertyValue(opts.kv, opts.deploymentID, opts.nodeName, "volume_id")
	if err != nil {
		return err
	}

	if volumeID != nil && volumeID.RawString() != "" {
		log.Debugf("Reusing existing volume with id %q for node %q", volumeID, opts.nodeName)
		bsIds = strings.Split(volumeID.RawString(), ",")
	}

	var bsVolume BlockStorageVolume
	bsVolume, err = g.generateOSBSVolume(opts.kv, opts.cfg, opts.locationProps,
		opts.deploymentID, opts.nodeName, opts.instanceName)
	if err != nil {
		return err
	}

	if len(bsIds)-1 < opts.instanceIndex {
		commons.AddResource(opts.infrastructure, opts.resourceTypes[blockStorageVolume], bsVolume.Name, &bsVolume)
		consulKey := commons.ConsulKey{
			Path:  path.Join(opts.instancesKey, opts.instanceName, "/attributes/volume_id"),
			Value: fmt.Sprintf("${%s.%s.id}", opts.resourceTypes[blockStorageVolume], bsVolume.Name)}
		consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
		commons.AddResource(opts.infrastructure, consulKeysResource, bsVolume.Name, &consulKeys)
	} else {
		name := opts.cfg.ResourcesPrefix + opts.nodeName + "-" + opts.instanceName
		consulKey := commons.ConsulKey{
			Path:  path.Join(opts.instancesKey, opts.instanceName, "/properties/volume_id"),
			Value: strings.TrimSpace(bsIds[opts.instanceIndex])}
		consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
		commons.AddResource(opts.infrastructure, consulKeysResource, name, &consulKeys)
	}

	return err
}

func (g *osGenerator) generateFloatingIPInfra(opts generateInfraOptions) error {

	ip, err := g.generateFloatingIP(opts.kv, opts.deploymentID, opts.nodeName, opts.instanceName)
	if err != nil {
		return err
	}

	var consulKey commons.ConsulKey
	if !ip.IsIP {
		floatingIP := FloatingIP{Pool: ip.Pool}
		commons.AddResource(opts.infrastructure, opts.resourceTypes[computeFloatingIP], ip.Name, &floatingIP)
		consulKey = commons.ConsulKey{
			Path:  path.Join(opts.instancesKey, opts.instanceName, floatingIPEndpointCapAttribute),
			Value: fmt.Sprintf("${%s.%s.address}", opts.resourceTypes[computeFloatingIP], ip.Name)}
	} else {
		ips := strings.Split(ip.Pool, ",")
		// TODO we should change this. instance name should not be considered as an int
		var instName int
		instName, err = strconv.Atoi(opts.instanceName)
		if err != nil {
			return err
		}
		if (len(ips) - 1) < instName {
			networkName, err := deployments.GetNodePropertyValue(opts.kv, opts.deploymentID,
				opts.nodeName, "floating_network_name")
			if err != nil {
				return err
			}
			if networkName == nil || networkName.RawString() == "" {
				return errors.Errorf("You need to provide enough IP address or a Pool to generate missing IP address")
			}

			floatingIP := FloatingIP{Pool: networkName.RawString()}
			commons.AddResource(opts.infrastructure, opts.resourceTypes[computeFloatingIP], ip.Name, &floatingIP)
			consulKey = commons.ConsulKey{
				Path:  path.Join(opts.instancesKey, opts.instanceName, floatingIPEndpointCapAttribute),
				Value: fmt.Sprintf("${%s.%s.address}", opts.resourceTypes[computeFloatingIP], ip.Name)}

		} else {
			// TODO we should change this. instance name should not be considered as an int
			instName, err = strconv.Atoi(opts.instanceName)
			if err != nil {
				return err
			}
			consulKey = commons.ConsulKey{
				Path:  path.Join(opts.instancesKey, opts.instanceName, floatingIPEndpointCapAttribute),
				Value: ips[instName]}
		}

	}
	consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
	commons.AddResource(opts.infrastructure, consulKeysResource, ip.Name, &consulKeys)
	return err
}

func (g *osGenerator) generateNetworkInfra(opts generateInfraOptions) error {

	networkID, err := deployments.GetNodePropertyValue(opts.kv, opts.deploymentID,
		opts.nodeName, "network_id")
	if err != nil {
		return err
	}
	if networkID != nil && networkID.RawString() != "" {
		log.Debugf("Reusing existing network with id %q for node %q", networkID, opts.nodeName)
		return err
	}

	var network Network
	network, err = g.generateNetwork(opts.kv, opts.cfg, opts.locationProps, opts.deploymentID, opts.nodeName)
	if err != nil {
		return err
	}
	var subnet Subnet
	subnet, err = g.generateSubnet(opts.kv, opts.cfg, opts.locationProps, opts.deploymentID, opts.nodeName,
		opts.resourceTypes[networkingNetwork])
	if err != nil {
		return err
	}

	commons.AddResource(opts.infrastructure, opts.resourceTypes[networkingNetwork],
		opts.nodeName, &network)
	commons.AddResource(opts.infrastructure, opts.resourceTypes[networkingSubnet],
		opts.nodeName+"_subnet", &subnet)
	consulKey := commons.ConsulKey{
		Path: path.Join(consulutil.DeploymentKVPrefix, opts.deploymentID, "topology",
			"nodes", opts.nodeName, "attributes/network_id"),
		Value: fmt.Sprintf("${%s.%s.id}", opts.resourceTypes[networkingNetwork], opts.nodeName)}
	consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
	consulKeys.DependsOn = []string{fmt.Sprintf("%s.%s_subnet", opts.resourceTypes[networkingSubnet],
		opts.nodeName)}
	commons.AddResource(opts.infrastructure, consulKeysResource, opts.nodeName, &consulKeys)

	return err
}
