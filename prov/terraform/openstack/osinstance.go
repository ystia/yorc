package openstack

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"gopkg.in/yaml.v2"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

func (g *osGenerator) generateOSInstance(kv *api.KV, cfg config.Configuration, url, deploymentID, instanceName string, infrastructure *commons.Infrastructure) error {
	nodeType, err := g.getStringFormConsul(kv, url, "type")
	if err != nil {
		return err
	}
	if nodeType != "janus.nodes.openstack.Compute" {
		return errors.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}
	instance := ComputeInstance{}
	var nodeName string
	if nodeName, err = g.getStringFormConsul(kv, url, "name"); err != nil {
		return err
	}

	instancesKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName)

	instance.Name = cfg.ResourcesPrefix + nodeName + "-" + instanceName
	image, err := g.getStringFormConsul(kv, url, "properties/image")
	if err != nil {
		return err
	}
	instance.ImageID = image
	image, err = g.getStringFormConsul(kv, url, "properties/imageName")
	if err != nil {
		return err
	}
	instance.ImageName = image
	flavor, err := g.getStringFormConsul(kv, url, "properties/flavor")
	if err != nil {
		return err
	}
	instance.FlavorID = flavor
	flavor, err = g.getStringFormConsul(kv, url, "properties/flavorName")
	if err != nil {
		return err
	}
	instance.FlavorName = flavor

	az, err := g.getStringFormConsul(kv, url, "properties/availability_zone")
	if err != nil {
		return err
	}
	instance.AvailabilityZone = az
	region, err := g.getStringFormConsul(kv, url, "properties/region")
	if err != nil {
		return err
	} else if region != "" {
		instance.Region = region
	} else {
		instance.Region = cfg.OSRegion
	}

	keyPair, err := g.getStringFormConsul(kv, url, "properties/key_pair")
	if err != nil {
		return err
	}
	// TODO if empty use a default one or fail ?
	instance.KeyPair = keyPair

	instance.SecurityGroups = cfg.OSDefaultSecurityGroups
	secGroups, err := g.getStringFormConsul(kv, url, "properties/security_groups")
	if err != nil {
		return err
	} else if secGroups != "" {
		for _, secGroup := range strings.Split(strings.NewReplacer("\"", "", "'", "").Replace(secGroups), ",") {
			secGroup = strings.TrimSpace(secGroup)
			instance.SecurityGroups = append(instance.SecurityGroups, secGroup)
		}
	}

	if instance.ImageID == "" && instance.ImageName == "" {
		return errors.Errorf("Missing mandatory parameter 'image' or 'imageName' node type for %s", url)
	}
	if instance.FlavorID == "" && instance.FlavorName == "" {
		return errors.Errorf("Missing mandatory parameter 'flavor' or 'flavorName' node type for %s", url)
	}

	networkName, err := g.getStringFormConsul(kv, url, "capabilities/endpoint/properties/network_name")
	if err != nil {
		return err
	}
	if networkName != "" {
		// TODO Deal with networks aliases (PUBLIC/PRIVATE)
		var networkSlice []ComputeNetwork
		if strings.EqualFold(networkName, "private") {
			networkSlice = append(networkSlice, ComputeNetwork{Name: cfg.OSPrivateNetworkName, AccessNetwork: true})
		} else if strings.EqualFold(networkName, "public") {
			//TODO
			return errors.Errorf("Public Network aliases currently not supported")
		} else {
			networkSlice = append(networkSlice, ComputeNetwork{Name: networkName, AccessNetwork: true})
		}
		instance.Networks = networkSlice
	} else {
		// Use a default
		instance.Networks = append(instance.Networks, ComputeNetwork{Name: cfg.OSPrivateNetworkName, AccessNetwork: true})
	}

	var user string
	if user, err = g.getStringFormConsul(kv, url, "properties/user"); err != nil {
		return err
	} else if user == "" {
		return errors.Errorf("Missing mandatory parameter 'user' node type for %s", url)
	}

	// TODO deal with multi-instances
	storageKeys, err := deployments.GetRequirementsKeysByNameForNode(kv, deploymentID, nodeName, "local_storage")
	if err != nil {
		return err
	}
	for _, storagePrefix := range storageKeys {
		if instance.Volumes == nil {
			instance.Volumes = make([]Volume, 0)
		}
		var volumeNodeName string
		if volumeNodeName, err = g.getStringFormConsul(kv, storagePrefix, "node"); err != nil {
			return err
		} else if volumeNodeName != "" {
			log.Debugf("Volume attachment required form Volume named %s", volumeNodeName)
			var device string
			if device, err = g.getStringFormConsul(kv, storagePrefix, "properties/location"); err != nil {
				return err
			}
			if device != "" {
				resolver := deployments.NewResolver(kv, deploymentID)
				expr := tosca.ValueAssignment{}
				if err = yaml.Unmarshal([]byte(device), &expr); err != nil {
					return err
				}
				// TODO check if instanceName is correct in all cases maybe we should check if we are in target context
				if device, err = resolver.ResolveExpressionForRelationship(expr.Expression, nodeName, volumeNodeName, path.Base(storagePrefix), instanceName); err != nil {
					return err
				}
			}
			var volumeID string
			log.Debugf("Looking for volume_id")
			// TODO consider the use of a method in the deployments package
			if kp, _, _ := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", volumeNodeName, instanceName, "properties/volume_id"), nil); kp != nil {
				if dID := string(kp.Value); dID != "" {
					volumeID = dID
				}
			} else {
				resultChan := make(chan string, 1)
				go func() {
					for {
						// ignore errors and retry
						// TODO consider the use of a method in the deployments package
						if kp, _, _ := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", volumeNodeName, instanceName, "attributes/volume_id"), nil); kp != nil {
							if dID := string(kp.Value); dID != "" {
								resultChan <- dID
								return
							}
						}
						time.Sleep(1 * time.Second)
					}
				}()
				// TODO add a cancellation signal
				volumeID = <-resultChan
			}

			vol := Volume{VolumeID: volumeID, Device: device}
			instance.Volumes = append(instance.Volumes, vol)
		}
	}
	// Do this in order to be sure that ansible will be able to log on the instance
	// TODO private key should not be hard-coded
	re := commons.RemoteExec{Inline: []string{`echo "connected"`}, Connection: commons.Connection{User: user, PrivateKey: `${file("~/.ssh/janus.pem")}`}}
	instance.Provisioners = make(map[string]interface{})
	instance.Provisioners["remote-exec"] = re

	consulKey := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.access_ip_v4}", instance.Name)} // Use access ip here

	consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}

	networkKeys, err := deployments.GetRequirementsKeysByNameForNode(kv, deploymentID, nodeName, "network")
	if err != nil {
		return err
	}
	for _, networkReqPrefix := range networkKeys {
		capability, err := g.getStringFormConsul(kv, networkReqPrefix, "capability")
		if err != nil {
			return err
		}
		isFip, err := deployments.IsTypeDerivedFrom(kv, deploymentID, capability, "janus.capabilities.openstack.FIPConnectivity")
		if err != nil {
			return err
		}
		networkNodeName, err := g.getStringFormConsul(kv, networkReqPrefix, "node")
		if err != nil {
			return err
		}
		if isFip {
			log.Debugf("Looking for Floating IP")
			var floatingIP string
			resultChan := make(chan string, 1)
			go func() {
				for {
					// TODO consider the use of a method in the deployments package
					if kp, _, _ := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", networkNodeName, instanceName, "capabilities/endpoint/attributes/floating_ip_address"), nil); kp != nil {
						if dID := string(kp.Value); dID != "" {
							resultChan <- dID
							return
						}
					}
					time.Sleep(1 * time.Second)
				}
			}()
			floatingIP = <-resultChan
			floatingIPAssociate := ComputeFloatingIPAssociate{FloatingIP: floatingIP, InstanceID: fmt.Sprintf("${openstack_compute_instance_v2.%s.id}", instance.Name)}
			addResource(infrastructure, "openstack_compute_floatingip_associate_v2", "FIP"+instance.Name, &floatingIPAssociate)
			consulKeyFloatingIP := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/public_address"), Value: floatingIP}
			//In order to be backward compatible to components developed for Alien (only the above is standard)
			consulKeyFloatingIPBak := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/public_ip_address"), Value: floatingIP}
			consulKeys.Keys = append(consulKeys.Keys, consulKeyFloatingIP, consulKeyFloatingIPBak)
		} else {
			log.Debugf("Looking for Network id for %q", networkNodeName)
			var networkID string
			resultChan := make(chan string, 1)
			go func() {
				for {
					// TODO consider the use of a method in the deployments package
					if kp, _, _ := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", networkNodeName, "attributes/network_id"), nil); kp != nil {
						if dID := string(kp.Value); dID != "" {
							resultChan <- dID
							return
						}
					}
					time.Sleep(1 * time.Second)
				}
			}()
			networkID = <-resultChan
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

	addResource(infrastructure, "openstack_compute_instance_v2", instance.Name, &instance)

	consulKeyAttrib := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/ip_address"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.fixed_ip_v4}", instance.Name, len(instance.Networks)-1)} // Use latest provisioned network for private access
	consulKeyFixedIP := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/private_address"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.fixed_ip_v4}", instance.Name, len(instance.Networks)-1)}
	consulKeys.Keys = append(consulKeys.Keys, consulKeyAttrib, consulKeyFixedIP)

	addResource(infrastructure, "consul_keys", instance.Name, &consulKeys)

	return nil
}
