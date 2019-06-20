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

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
	"github.com/ystia/yorc/v4/tosca"
)

type osGenerator struct {
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
	instancesKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName)
	terraformStateKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "terraform-state", nodeName)

	infrastructure := commons.Infrastructure{}

	log.Debugf("Generating infrastructure for deployment with node %s", nodeName)

	// Remote Configuration for Terraform State to store it in the Consul KV store
	infrastructure.Terraform = commons.GetBackendConfiguration(terraformStateKey, cfg)

	cmdEnv := []string{
		fmt.Sprintf("OS_USERNAME=%s", cfg.Infrastructures[infrastructureName].GetString("user_name")),
		fmt.Sprintf("OS_PASSWORD=%s", cfg.Infrastructures[infrastructureName].GetString("password")),
		fmt.Sprintf("OS_PROJECT_NAME=%s", cfg.Infrastructures[infrastructureName].GetString("project_name")),
		fmt.Sprintf("OS_PROJECT_ID=%s", cfg.Infrastructures[infrastructureName].GetString("project_id")),
		fmt.Sprintf("OS_USER_DOMAIN_NAME=%s", cfg.Infrastructures[infrastructureName].GetString("user_domain_name")),
		fmt.Sprintf("OS_AUTH_URL=%s", cfg.Infrastructures[infrastructureName].GetString("auth_url")),
	}

	// Management of variables for Terraform
	infrastructure.Provider = map[string]interface{}{
		"openstack": map[string]interface{}{
			"version":     cfg.Terraform.OpenStackPluginVersionConstraint,
			"tenant_name": cfg.Infrastructures[infrastructureName].GetString("tenant_name"),
			"insecure":    cfg.Infrastructures[infrastructureName].GetString("insecure"),
			"cacert_file": cfg.Infrastructures[infrastructureName].GetString("cacert_file"),
			"cert":        cfg.Infrastructures[infrastructureName].GetString("cert"),
			"key":         cfg.Infrastructures[infrastructureName].GetString("key"),
		},
		"consul": commons.GetConsulProviderfiguration(cfg),
		"null": map[string]interface{}{
			"version": commons.NullPluginVersionConstraint,
		},
	}

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

	for instNb, instanceName := range instances {
		instanceState, err := deployments.GetInstanceState(kv, deploymentID, nodeName, instanceName)
		if err != nil {
			return false, nil, nil, nil, err
		}
		if instanceState == tosca.NodeStateDeleting || instanceState == tosca.NodeStateDeleted {
			// Do not generate something for this node instance (will be deleted if exists)
			continue
		}

		switch nodeType {
		case "yorc.nodes.openstack.Compute":
			err = g.generateOSInstance(ctx, kv, cfg, deploymentID, nodeName, instanceName, &infrastructure, outputs, &cmdEnv)
			if err != nil {
				return false, nil, nil, nil, err
			}

		case "yorc.nodes.openstack.BlockStorage":
			var bsIds []string
			volumeID, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "volume_id")
			if err != nil {
				return false, nil, nil, nil, err
			}

			if volumeID != nil && volumeID.RawString() != "" {
				log.Debugf("Reusing existing volume with id %q for node %q", volumeID, nodeName)
				bsIds = strings.Split(volumeID.RawString(), ",")
			}

			var bsVolume BlockStorageVolume
			bsVolume, err = g.generateOSBSVolume(kv, cfg, deploymentID, nodeName, instanceName)
			if err != nil {
				return false, nil, nil, nil, err
			}

			if len(bsIds)-1 < instNb {
				commons.AddResource(&infrastructure, "openstack_blockstorage_volume_v1", bsVolume.Name, &bsVolume)
				consulKey := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/volume_id"), Value: fmt.Sprintf("${openstack_blockstorage_volume_v1.%s.id}", bsVolume.Name)}
				consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
				commons.AddResource(&infrastructure, "consul_keys", bsVolume.Name, &consulKeys)
			} else {
				name := cfg.ResourcesPrefix + nodeName + "-" + instanceName
				consulKey := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/properties/volume_id"), Value: strings.TrimSpace(bsIds[instNb])}
				consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
				commons.AddResource(&infrastructure, "consul_keys", name, &consulKeys)
			}

		case "yorc.nodes.openstack.FloatingIP":
			var ip IP
			ip, err = g.generateFloatingIP(kv, deploymentID, nodeName, instanceName)

			if err != nil {
				return false, nil, nil, nil, err
			}

			var consulKey commons.ConsulKey
			if !ip.IsIP {
				floatingIP := FloatingIP{Pool: ip.Pool}
				commons.AddResource(&infrastructure, "openstack_compute_floatingip_v2", ip.Name, &floatingIP)
				consulKey = commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/floating_ip_address"), Value: fmt.Sprintf("${openstack_compute_floatingip_v2.%s.address}", ip.Name)}
			} else {
				ips := strings.Split(ip.Pool, ",")
				// TODO we should change this. instance name should not be considered as an int
				var instName int
				instName, err = strconv.Atoi(instanceName)
				if err != nil {
					return false, nil, nil, nil, err
				}
				if (len(ips) - 1) < instName {
					networkName, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "floating_network_name")
					if err != nil {
						return false, nil, nil, nil, err
					} else if networkName == nil || networkName.RawString() == "" {
						return false, nil, nil, nil, errors.Errorf("You need to provide enough IP address or a Pool to generate missing IP address")
					}

					floatingIP := FloatingIP{Pool: networkName.RawString()}
					commons.AddResource(&infrastructure, "openstack_compute_floatingip_v2", ip.Name, &floatingIP)
					consulKey = commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/floating_ip_address"), Value: fmt.Sprintf("${openstack_compute_floatingip_v2.%s.address}", ip.Name)}

				} else {
					// TODO we should change this. instance name should not be considered as an int
					instName, err = strconv.Atoi(instanceName)
					if err != nil {
						return false, nil, nil, nil, err
					}
					consulKey = commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/floating_ip_address"), Value: ips[instName]}
				}

			}
			consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
			commons.AddResource(&infrastructure, "consul_keys", ip.Name, &consulKeys)

		case "yorc.nodes.openstack.Network":
			networkID, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "network_id")
			if err != nil {
				return false, nil, nil, nil, err
			} else if networkID != nil && networkID.RawString() != "" {
				log.Debugf("Reusing existing volume with id %q for node %q", networkID, nodeName)
				return false, nil, cmdEnv, nil, nil
			}
			var network Network
			network, err = g.generateNetwork(kv, cfg, deploymentID, nodeName)

			if err != nil {
				return false, nil, nil, nil, err
			}
			var subnet Subnet
			subnet, err = g.generateSubnet(kv, cfg, deploymentID, nodeName)

			if err != nil {
				return false, nil, nil, nil, err
			}

			commons.AddResource(&infrastructure, "openstack_networking_network_v2", nodeName, &network)
			commons.AddResource(&infrastructure, "openstack_networking_subnet_v2", nodeName+"_subnet", &subnet)
			consulKey := commons.ConsulKey{Path: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName, "attributes/network_id"), Value: fmt.Sprintf("${openstack_networking_network_v2.%s.id}", nodeName)}
			consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
			consulKeys.DependsOn = []string{fmt.Sprintf("openstack_networking_subnet_v2.%s_subnet", nodeName)}
			commons.AddResource(&infrastructure, "consul_keys", nodeName, &consulKeys)

		case "yorc.nodes.openstack.ServerGroup":
			err = g.generateServerGroup(ctx, kv, cfg, deploymentID, nodeName, &infrastructure, outputs, &cmdEnv)
			if err != nil {
				return false, nil, nil, nil, err
			}
		default:
			return false, nil, nil, nil, errors.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, deploymentID)
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
