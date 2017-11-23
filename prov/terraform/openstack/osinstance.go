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

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
)

func (g *osGenerator) generateOSInstance(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName, instanceName string, infrastructure *commons.Infrastructure, outputs map[string]string) error {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType != "janus.nodes.openstack.Compute" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}
	instance := ComputeInstance{}
	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances")
	instancesKey := path.Join(instancesPrefix, nodeName)

	instance.Name = cfg.ResourcesPrefix + nodeName + "-" + instanceName

	_, image, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "image")
	if err != nil {
		return err
	}
	instance.ImageID = image
	_, image, err = deployments.GetNodeProperty(kv, deploymentID, nodeName, "imageName")
	if err != nil {
		return err
	}
	instance.ImageName = image
	_, flavor, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "flavor")
	if err != nil {
		return err
	}
	instance.FlavorID = flavor
	_, flavor, err = deployments.GetNodeProperty(kv, deploymentID, nodeName, "flavorName")
	if err != nil {
		return err
	}
	instance.FlavorName = flavor

	_, az, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "availability_zone")
	if err != nil {
		return err
	}
	instance.AvailabilityZone = az
	_, region, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "region")
	if err != nil {
		return err
	} else if region != "" {
		instance.Region = region
	} else {
		instance.Region = cfg.Infrastructures[infrastructureName].GetStringOrDefault("region", defaultOSRegion)
	}

	_, keyPair, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "key_pair")
	if err != nil {
		return err
	}
	// TODO if empty use a default one or fail ?
	instance.KeyPair = keyPair

	instance.SecurityGroups = cfg.Infrastructures[infrastructureName].GetStringSlice("default_security_groups")
	_, secGroups, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "security_groups")
	if err != nil {
		return err
	} else if secGroups != "" {
		for _, secGroup := range strings.Split(strings.NewReplacer("\"", "", "'", "").Replace(secGroups), ",") {
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

	_, networkName, err := deployments.GetCapabilityProperty(kv, deploymentID, nodeName, "endpoint", "network_name")
	if err != nil {
		return err
	}
	defaultPrivateNetName := cfg.Infrastructures[infrastructureName].GetString("private_network_name")
	if networkName != "" {
		// TODO Deal with networks aliases (PUBLIC)
		var networkSlice []ComputeNetwork
		if strings.EqualFold(networkName, "private") {
			if defaultPrivateNetName == "" {
				return errors.Errorf(`You should either specify a default private network name using the "private_network_name" configuration parameter for the "openstack" infrastructure or specify a "network_name" property in the "endpoint" capability of node %q`, nodeName)
			}
			networkSlice = append(networkSlice, ComputeNetwork{Name: defaultPrivateNetName, AccessNetwork: true})
		} else if strings.EqualFold(networkName, "public") {
			//TODO
			return errors.Errorf("Public Network aliases currently not supported")
		} else {
			networkSlice = append(networkSlice, ComputeNetwork{Name: networkName, AccessNetwork: true})
		}
		instance.Networks = networkSlice
	} else {
		// Use a default
		if defaultPrivateNetName == "" {
			return errors.Errorf(`You should either specify a default private network name using the "private_network_name" configuration parameter for the "openstack" infrastructure or specify a "network_name" property in the "endpoint" capability of node %q`, nodeName)
		}
		instance.Networks = append(instance.Networks, ComputeNetwork{Name: defaultPrivateNetName, AccessNetwork: true})
	}

	var user string
	if _, user, err = deployments.GetNodeProperty(kv, deploymentID, nodeName, "user"); err != nil {
		return err
	} else if user == "" {
		return errors.Errorf("Missing mandatory parameter 'user' node type for %s", nodeName)
	}

	consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{}}

	storageKeys, err := deployments.GetRequirementsKeysByNameForNode(kv, deploymentID, nodeName, "local_storage")
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

			_, device, err := deployments.GetRelationshipPropertyFromRequirement(kv, deploymentID, nodeName, requirementIndex, "device")
			if err != nil {
				return err
			}
			log.Debugf("Looking for volume_id")
			_, volumeID, err := deployments.GetNodeProperty(kv, deploymentID, volumeNodeName, "volume_id")
			if err != nil {
				return err
			}

			if volumeID == "" {
				resultChan := make(chan string, 1)
				go func() {
					for {
						// ignore errors and retry
						found, volID, _ := deployments.GetInstanceAttribute(kv, deploymentID, volumeNodeName, instanceName, "volume_id")
						if found {
							resultChan <- volID
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
			}
			volumeAttach := ComputeVolumeAttach{
				Region:     region,
				VolumeID:   volumeID,
				InstanceID: fmt.Sprintf("${openstack_compute_instance_v2.%s.id}", instance.Name),
				Device:     device,
			}
			attachName := "Vol" + volumeNodeName + "to" + instance.Name
			commons.AddResource(infrastructure, "openstack_compute_volume_attach_v2", attachName, &volumeAttach)
			// retrieve the actual used device as depending on the hypervisor it may not be the one we provided, and if there was no devices provided
			// then we can get it back

			// Bellow code lead to an issue in terraform (https://github.com/hashicorp/terraform/issues/15284) so as a workaround we use a output variable
			// volumeDevConsulKey := commons.ConsulKey{Path: path.Join(instancesPrefix, volumeNodeName, instanceName, "attributes/device"), Value: fmt.Sprintf("${openstack_compute_volume_attach_v2.%s.device}", attachName)} // to be backward compatible with Alien stuff
			// relDevConsulKey := commons.ConsulKey{Path: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "relationship_instances", nodeName, requirementIndex, instanceName, "attributes/device"), Value: fmt.Sprintf("${openstack_compute_volume_attach_v2.%s.device}", attachName)}
			// relVolDevConsulKey := commons.ConsulKey{Path: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "relationship_instances", volumeNodeName, requirementIndex, instanceName, "attributes/device"), Value: fmt.Sprintf("${openstack_compute_volume_attach_v2.%s.device}", attachName)}
			// consulKeys.Keys = append(consulKeys.Keys, volumeDevConsulKey, relDevConsulKey, relVolDevConsulKey)
			key1 := attachName + "ActualDevkey"
			commons.AddOutput(infrastructure, key1, &commons.Output{Value: fmt.Sprintf("${openstack_compute_volume_attach_v2.%s.device}", attachName)})
			outputs[path.Join(instancesPrefix, volumeNodeName, instanceName, "attributes/device")] = key1
			outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "relationship_instances", nodeName, requirementIndex, instanceName, "attributes/device")] = key1
			outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "relationship_instances", volumeNodeName, requirementIndex, instanceName, "attributes/device")] = key1
		}
	}

	networkKeys, err := deployments.GetRequirementsKeysByNameForNode(kv, deploymentID, nodeName, "network")
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
			isFip, err = deployments.IsTypeDerivedFrom(kv, deploymentID, capability, "janus.capabilities.openstack.FIPConnectivity")
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
					if _, fip, _ := deployments.GetInstanceCapabilityAttribute(kv, deploymentID, networkNodeName, instanceName, "endpoint", "floating_ip_address"); fip != "" {
						resultChan <- fip
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
				Region:     region,
				FloatingIP: floatingIP,
				InstanceID: fmt.Sprintf("${openstack_compute_instance_v2.%s.id}", instance.Name),
			}
			fipAssociateName = "FIP" + instance.Name
			commons.AddResource(infrastructure, "openstack_compute_floatingip_associate_v2", fipAssociateName, &floatingIPAssociate)
			consulKeyFloatingIP := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/public_address"), Value: floatingIP}
			// In order to be backward compatible to components developed for Alien (only the above is standard)
			consulKeyFloatingIPBak := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/public_ip_address"), Value: floatingIP}
			consulKeys.Keys = append(consulKeys.Keys, consulKeyFloatingIP, consulKeyFloatingIPBak)
		} else {
			log.Debugf("Looking for Network id for %q", networkNodeName)
			var networkID string
			resultChan := make(chan string, 1)
			go func() {
				for {
					found, nID, err := deployments.GetInstanceAttribute(kv, deploymentID, networkNodeName, instanceName, "network_id")
					if err != nil {
						log.Printf("[Warning] bypassing error while waiting for a network id: %v", err)
					}
					if found {
						resultChan <- nID
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
			consulKetNetName := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "attributes/networks", strconv.Itoa(i), "network_name"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.name}", instance.Name, i)}
			consulKetNetID := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "attributes/networks", strconv.Itoa(i), "network_id"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.uuid}", instance.Name, i)}
			consulKetNetAddresses := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "attributes/networks", strconv.Itoa(i), "addresses"), Value: fmt.Sprintf("[ ${openstack_compute_instance_v2.%s.network.%d.fixed_ip_v4} ]", instance.Name, i)}
			consulKeys.Keys = append(consulKeys.Keys, consulKetNetName, consulKetNetID, consulKetNetAddresses)
		}
	}

	commons.AddResource(infrastructure, "openstack_compute_instance_v2", instance.Name, &instance)

	nullResource := commons.Resource{}
	// Do this in order to be sure that ansible will be able to log on the instance
	// TODO private key should not be hard-coded
	re := commons.RemoteExec{Inline: []string{`echo "connected"`}, Connection: &commons.Connection{User: user, PrivateKey: `${file("~/.ssh/janus.pem")}`}}
	var accessIP string
	if fipAssociateName != "" && cfg.Infrastructures[infrastructureName].GetBool("provisioning_over_fip_allowed") {
		// Use Floating IP for provisioning
		accessIP = "${openstack_compute_floatingip_associate_v2." + fipAssociateName + ".floating_ip}"
	} else {
		accessIP = "${openstack_compute_instance_v2." + instance.Name + ".network.0.fixed_ip_v4}"
	}
	re.Connection.Host = accessIP
	consulKeys.Keys = append(consulKeys.Keys, commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address"), Value: accessIP}) // Use access ip here
	nullResource.Provisioners = make([]map[string]interface{}, 0)
	provMap := make(map[string]interface{})
	provMap["remote-exec"] = re
	nullResource.Provisioners = append(nullResource.Provisioners, provMap)

	commons.AddResource(infrastructure, "null_resource", instance.Name+"-ConnectionCheck", &nullResource)

	consulKeyAttrib := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/ip_address"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.fixed_ip_v4}", instance.Name, len(instance.Networks)-1)} // Use latest provisioned network for private access
	consulKeyFixedIP := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/private_address"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.fixed_ip_v4}", instance.Name, len(instance.Networks)-1)}
	consulKeys.Keys = append(consulKeys.Keys, consulKeyAttrib, consulKeyFixedIP)

	commons.AddResource(infrastructure, "consul_keys", instance.Name, &consulKeys)

	return nil
}
