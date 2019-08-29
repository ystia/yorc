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
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/prov/terraform/commons"
)

func (g *osGenerator) generateOSInstance(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName, instanceName string, infrastructure *commons.Infrastructure, outputs map[string]string, env *[]string) error {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType != "yorc.nodes.openstack.Compute" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}
	instance := ComputeInstance{}
	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances")
	instancesKey := path.Join(instancesPrefix, nodeName)

	instance.Name = cfg.ResourcesPrefix + nodeName + "-" + instanceName

	image, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "image")
	if err != nil {
		return err
	}
	if image != nil {
		instance.ImageID = image.RawString()
	}
	image, err = deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "imageName")
	if err != nil {
		return err
	}
	if image != nil {
		instance.ImageName = image.RawString()
	}
	flavor, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "flavor")
	if err != nil {
		return err
	}
	if flavor != nil {
		instance.FlavorID = flavor.RawString()
	}
	flavor, err = deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "flavorName")
	if err != nil {
		return err
	}
	if flavor != nil {
		instance.FlavorName = flavor.RawString()
	}
	az, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "availability_zone")
	if err != nil {
		return err
	}
	if az != nil {
		instance.AvailabilityZone = az.RawString()
	}
	region, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "region")
	if err != nil {
		return err
	} else if region != nil && region.RawString() != "" {
		instance.Region = region.RawString()
	} else {
		instance.Region = cfg.Infrastructures[infrastructureName].GetStringOrDefault("region", defaultOSRegion)
	}

	keyPair, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "key_pair")
	if err != nil {
		return err
	}
	if keyPair != nil {
		// TODO if empty use a default one or fail ?
		instance.KeyPair = keyPair.RawString()
	}
	instance.SecurityGroups = cfg.Infrastructures[infrastructureName].GetStringSlice("default_security_groups")
	secGroups, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "security_groups")
	if err != nil {
		return err
	}
	if secGroups != nil && secGroups.RawString() != "" {
		for _, secGroup := range strings.Split(strings.NewReplacer("\"", "", "'", "").Replace(secGroups.RawString()), ",") {
			secGroup = strings.TrimSpace(secGroup)
			instance.SecurityGroups = append(instance.SecurityGroups, secGroup)
		}
	}

	if instance.ImageID == "" && instance.ImageName == "" {
		return errors.Errorf("Missing mandatory parameter 'image' or 'imageName' node type for %s", nodeName)
	}
	if instance.FlavorID == "" && instance.FlavorName == "" {
		return errors.Errorf("Missing mandatory parameter 'flavor' or 'flavorName' node type for %s", nodeName)
	}

	networkName, err := deployments.GetCapabilityPropertyValue(kv, deploymentID, nodeName, "endpoint", "network_name")
	if err != nil {
		return err
	}
	defaultPrivateNetName := cfg.Infrastructures[infrastructureName].GetString("private_network_name")
	if networkName != nil && networkName.RawString() != "" {
		// TODO Deal with networks aliases (PUBLIC)
		var networkSlice []ComputeNetwork
		if strings.EqualFold(networkName.RawString(), "private") {
			if defaultPrivateNetName == "" {
				return errors.Errorf(`You should either specify a default private network name using the "private_network_name" configuration parameter for the "openstack" infrastructure or specify a "network_name" property in the "endpoint" capability of node %q`, nodeName)
			}
			networkSlice = append(networkSlice, ComputeNetwork{Name: defaultPrivateNetName, AccessNetwork: true})
		} else if strings.EqualFold(networkName.RawString(), "public") {
			//TODO
			return errors.Errorf("Public Network aliases currently not supported")
		} else {
			networkSlice = append(networkSlice, ComputeNetwork{Name: networkName.RawString(), AccessNetwork: true})
		}
		instance.Networks = networkSlice
	} else {
		// Use a default
		if defaultPrivateNetName == "" {
			return errors.Errorf(`You should either specify a default private network name using the "private_network_name" configuration parameter for the "openstack" infrastructure or specify a "network_name" property in the "endpoint" capability of node %q`, nodeName)
		}
		instance.Networks = append(instance.Networks, ComputeNetwork{Name: defaultPrivateNetName, AccessNetwork: true})
	}

	storageKeys, err := deployments.GetRequirementsKeysByTypeForNode(kv, deploymentID, nodeName, "local_storage")
	if err != nil {
		return err
	}
	for _, storagePrefix := range storageKeys {
		requirementIndex := deployments.GetRequirementIndexFromRequirementKey(storagePrefix)
		volumeNodeName, err := deployments.GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return err
		} else if volumeNodeName != "" {
			log.Debugf("Volume attachment required form Volume named %s", volumeNodeName)

			device, err := deployments.GetRelationshipPropertyValueFromRequirement(kv, deploymentID, nodeName, requirementIndex, "device")
			if err != nil {
				return err
			}
			log.Debugf("Looking for volume_id")
			volumeIDValue, err := deployments.GetNodePropertyValue(kv, deploymentID, volumeNodeName, "volume_id")
			if err != nil {
				return err
			}
			var volumeID string
			if volumeIDValue == nil || volumeIDValue.RawString() == "" {
				resultChan := make(chan string, 1)
				go func() {
					for {
						// ignore errors and retry
						volID, _ := deployments.GetInstanceAttributeValue(kv, deploymentID, volumeNodeName, instanceName, "volume_id")
						// As volumeID is an optional property GetInstanceAttribute then GetProperty
						// may return an empty volumeID so keep checking as long as we have it
						if volID != nil && volID.RawString() != "" {
							resultChan <- volID.RawString()
							return
						}
						select {
						case <-time.After(1 * time.Second):
						case <-ctx.Done():
							// context cancelled, give up!
							return
						}

					}
				}()
				select {
				case volumeID = <-resultChan:
				case <-ctx.Done():
					return ctx.Err()

				}
			} else {
				volumeID = volumeIDValue.RawString()
			}
			volumeAttach := ComputeVolumeAttach{
				Region:     instance.Region,
				VolumeID:   volumeID,
				InstanceID: fmt.Sprintf("${openstack_compute_instance_v2.%s.id}", instance.Name),
			}
			if device != nil {
				volumeAttach.Device = device.RawString()
			}
			attachName := "Vol" + volumeNodeName + "to" + instance.Name
			commons.AddResource(infrastructure, "openstack_compute_volume_attach_v2", attachName, &volumeAttach)
			// retrieve the actual used device as depending on the hypervisor it may not be the one we provided, and if there was no devices provided
			// then we can get it back

			key1 := attachName + "-device"
			commons.AddOutput(infrastructure, key1, &commons.Output{Value: fmt.Sprintf("${openstack_compute_volume_attach_v2.%s.device}", attachName)})
			outputs[path.Join(instancesPrefix, volumeNodeName, instanceName, "attributes/device")] = key1
			outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "relationship_instances", nodeName, requirementIndex, instanceName, "attributes/device")] = key1
			outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "relationship_instances", volumeNodeName, requirementIndex, instanceName, "attributes/device")] = key1
		}
	}

	networkKeys, err := deployments.GetRequirementsKeysByTypeForNode(kv, deploymentID, nodeName, "network")
	if err != nil {
		return err
	}
	var fipAssociateName string
	for _, networkReqPrefix := range networkKeys {
		requirementIndex := deployments.GetRequirementIndexFromRequirementKey(networkReqPrefix)

		capability, err := deployments.GetCapabilityForRequirement(kv, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return err
		}

		networkNodeName, err := deployments.GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return err
		}

		var isFip bool
		if capability != "" {
			isFip, err = deployments.IsTypeDerivedFrom(kv, deploymentID, capability, "yorc.capabilities.openstack.FIPConnectivity")
			if err != nil {
				return err
			}
		}

		if isFip {
			log.Debugf("Looking for Floating IP")
			var floatingIP string
			resultChan := make(chan string, 1)
			go func() {
				for {
					if fip, _ := deployments.GetInstanceCapabilityAttributeValue(kv, deploymentID, networkNodeName, instanceName, "endpoint", "floating_ip_address"); fip != nil && fip.RawString() != "" {
						resultChan <- fip.RawString()
						return
					}

					select {
					case <-time.After(1 * time.Second):
					case <-ctx.Done():
						// context cancelled, give up!
						return
					}
				}
			}()
			select {
			case floatingIP = <-resultChan:
			case <-ctx.Done():
				return ctx.Err()
			}
			floatingIPAssociate := ComputeFloatingIPAssociate{
				Region:     instance.Region,
				FloatingIP: floatingIP,
				InstanceID: fmt.Sprintf("${openstack_compute_instance_v2.%s.id}", instance.Name),
			}
			fipAssociateName = "FIP" + instance.Name
			commons.AddResource(infrastructure, "openstack_compute_floatingip_associate_v2", fipAssociateName, &floatingIPAssociate)

			// Provide output for public IP as floating IP
			publicIPKey := nodeName + "-" + instanceName + "-publicIP"
			commons.AddOutput(infrastructure, publicIPKey, &commons.Output{Value: floatingIP})
			outputs[path.Join(instancesKey, instanceName, "/attributes/public_address")] = publicIPKey
			// In order to be backward compatible to components developed for Alien (only the above is standard)
			outputs[path.Join(instancesKey, instanceName, "/attributes/public_ip_address")] = publicIPKey
		} else {
			log.Debugf("Looking for Network id for %q", networkNodeName)
			var networkID string
			resultChan := make(chan string, 1)
			go func() {
				for {
					nID, err := deployments.GetInstanceAttributeValue(kv, deploymentID, networkNodeName, instanceName, "network_id")
					if err != nil {
						log.Printf("[Warning] bypassing error while waiting for a network id: %v", err)
					}
					// As networkID is an optional property GetInstanceAttribute then GetProperty
					// may return an empty networkID so keep checking as long as we have it
					if nID != nil && nID.RawString() != "" {
						resultChan <- nID.RawString()
						return
					}
					select {
					case <-time.After(1 * time.Second):
					case <-ctx.Done():
						// context cancelled, give up!
						return
					}
				}
			}()
			select {
			case networkID = <-resultChan:
			case <-ctx.Done():
				return ctx.Err()
			}
			cn := ComputeNetwork{UUID: networkID, AccessNetwork: false}
			i := len(instance.Networks)
			if instance.Networks == nil {
				instance.Networks = make([]ComputeNetwork, 0)
			}
			instance.Networks = append(instance.Networks, cn)

			// Provide output for network_name, network_id, addresses attributes
			networkIDKey := nodeName + "-" + instanceName + "-networkID"
			networkNameKey := nodeName + "-" + instanceName + "-networkName"
			networkAddressesKey := nodeName + "-" + instanceName + "-addresses"
			commons.AddOutput(infrastructure, networkIDKey, &commons.Output{Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.uuid}", instance.Name, i)})
			commons.AddOutput(infrastructure, networkNameKey, &commons.Output{Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.name}", instance.Name, i)})
			commons.AddOutput(infrastructure, networkAddressesKey, &commons.Output{Value: fmt.Sprintf("[ ${openstack_compute_instance_v2.%s.network.%d.fixed_ip_v4} ]", instance.Name, i)})

			outputs[path.Join(instancesKey, instanceName, "attributes/networks", strconv.Itoa(i), "network_name")] = networkNameKey
			outputs[path.Join(instancesKey, instanceName, "attributes/networks", strconv.Itoa(i), "network_id")] = networkIDKey
			outputs[path.Join(instancesKey, instanceName, "attributes/networks", strconv.Itoa(i), "addresses")] = networkAddressesKey
		}
	}

	commons.AddResource(infrastructure, "openstack_compute_instance_v2", instance.Name, &instance)

	var accessIP string
	if fipAssociateName != "" && cfg.Infrastructures[infrastructureName].GetBool("provisioning_over_fip_allowed") {
		// Use Floating IP for provisioning
		accessIP = "${openstack_compute_floatingip_associate_v2." + fipAssociateName + ".floating_ip}"
	} else {
		accessIP = "${openstack_compute_instance_v2." + instance.Name + ".network.0.fixed_ip_v4}"
	}

	// Provide output for access IP and private IP
	accessIPKey := nodeName + "-" + instanceName + "-IPAddress"
	commons.AddOutput(infrastructure, accessIPKey, &commons.Output{Value: accessIP})
	outputs[path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address")] = accessIPKey
	outputs[path.Join(instancesKey, instanceName, "/attributes/ip_address")] = accessIPKey

	privateIPKey := nodeName + "-" + instanceName + "-privateIP"
	privateIP := fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.fixed_ip_v4}", instance.Name, len(instance.Networks)-1) // Use latest provisioned network for private access
	commons.AddOutput(infrastructure, privateIPKey, &commons.Output{Value: privateIP})
	outputs[path.Join(instancesKey, instanceName, "/attributes/private_address")] = privateIPKey

	err = addServerGroupMembership(ctx, kv, deploymentID, nodeName, &instance)
	if err != nil {
		return errors.Wrapf(err, "failed to add serverGroup membership for deploymentID:%q, nodeName:%q, instance:%q", deploymentID, nodeName, instanceName)
	}

	// Get connection info (user, private key)
	user, privateKey, err := commons.GetConnInfoFromEndpointCredentials(ctx, kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	return commons.AddConnectionCheckResource(infrastructure, user, privateKey, accessIP, instance.Name, env)
}

func addServerGroupMembership(ctx context.Context, kv *api.KV, deploymentID, nodeName string, compute *ComputeInstance) error {
	keys, err := deployments.GetRequirementsKeysByTypeForNode(kv, deploymentID, nodeName, "group")
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}

	if len(keys) == 1 {
		requirementIndex := deployments.GetRequirementIndexFromRequirementKey(keys[0])
		serverGroup, err := deployments.GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return err
		}

		id, err := deployments.LookupInstanceAttributeValue(ctx, kv, deploymentID, serverGroup, "0", "id")
		if err != nil {
			return err
		}

		compute.SchedulerHints.Group = id
		return nil
	}

	return errors.Errorf("Only one group requirement can be accepted for OpenStack compute with name:%q", nodeName)

}
