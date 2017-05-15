package openstack

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"gopkg.in/yaml.v2"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

func (g *osGenerator) generateOSInstance(kv *api.KV, cfg config.Configuration, url, deploymentID, instanceName string) (ComputeInstance, error) {
	nodeType, err := g.getStringFormConsul(kv, url, "type")
	if err != nil {
		return ComputeInstance{}, err
	}
	if nodeType != "janus.nodes.openstack.Compute" {
		return ComputeInstance{}, fmt.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}
	instance := ComputeInstance{}
	var nodeName string
	if nodeName, err = g.getStringFormConsul(kv, url, "name"); err != nil {
		return ComputeInstance{}, err
	}
	instance.Name = cfg.ResourcesPrefix + nodeName + "-" + instanceName
	image, err := g.getStringFormConsul(kv, url, "properties/image")
	if err != nil {
		return ComputeInstance{}, err
	}
	instance.ImageID = image
	image, err = g.getStringFormConsul(kv, url, "properties/imageName")
	if err != nil {
		return ComputeInstance{}, err
	}
	instance.ImageName = image
	flavor, err := g.getStringFormConsul(kv, url, "properties/flavor")
	if err != nil {
		return ComputeInstance{}, err
	}
	instance.FlavorID = flavor
	flavor, err = g.getStringFormConsul(kv, url, "properties/flavorName")
	if err != nil {
		return ComputeInstance{}, err
	}
	instance.FlavorName = flavor

	az, err := g.getStringFormConsul(kv, url, "properties/availability_zone")
	if err != nil {
		return ComputeInstance{}, err
	}
	instance.AvailabilityZone = az
	region, err := g.getStringFormConsul(kv, url, "properties/region")
	if err != nil {
		return ComputeInstance{}, err
	} else if region != "" {
		instance.Region = region
	} else {
		instance.Region = cfg.OSRegion
	}

	keyPair, err := g.getStringFormConsul(kv, url, "properties/key_pair")
	if err != nil {
		return ComputeInstance{}, err
	}
	// TODO if empty use a default one or fail ?
	instance.KeyPair = keyPair

	instance.SecurityGroups = cfg.OSDefaultSecurityGroups
	secGroups, err := g.getStringFormConsul(kv, url, "properties/security_groups")
	if err != nil {
		return ComputeInstance{}, err
	} else if secGroups != "" {
		for _, secGroup := range strings.Split(strings.NewReplacer("\"", "", "'", "").Replace(secGroups), ",") {
			secGroup = strings.TrimSpace(secGroup)
			instance.SecurityGroups = append(instance.SecurityGroups, secGroup)
		}
	}

	if instance.ImageID == "" && instance.ImageName == "" {
		return ComputeInstance{}, fmt.Errorf("Missing mandatory parameter 'image' or 'imageName' node type for %s", url)
	}
	if instance.FlavorID == "" && instance.FlavorName == "" {
		return ComputeInstance{}, fmt.Errorf("Missing mandatory parameter 'flavor' or 'flavorName' node type for %s", url)
	}

	networkName, err := g.getStringFormConsul(kv, url, "capabilities/endpoint/properties/network_name")
	if err != nil {
		return ComputeInstance{}, err
	}
	if networkName != "" {
		// TODO Deal with networks aliases (PUBLIC/PRIVATE)
		var networkSlice []ComputeNetwork
		if strings.EqualFold(networkName, "private") {
			networkSlice = append(networkSlice, ComputeNetwork{Name: cfg.OSPrivateNetworkName, AccessNetwork: true})
		} else if strings.EqualFold(networkName, "public") {
			//TODO
			return ComputeInstance{}, fmt.Errorf("Public Network aliases currently not supported")
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
		return ComputeInstance{}, err
	} else if user == "" {
		return ComputeInstance{}, fmt.Errorf("Missing mandatory parameter 'user' node type for %s", url)
	}

	// TODO deal with multi-instances
	storageKeys, err := deployments.GetRequirementsKeysByNameForNode(kv, deploymentID, nodeName, "local_storage")
	if err != nil {
		return ComputeInstance{}, err
	}
	for _, storagePrefix := range storageKeys {
		if instance.Volumes == nil {
			instance.Volumes = make([]Volume, 0)
		}
		var volumeNodeName string
		if volumeNodeName, err = g.getStringFormConsul(kv, storagePrefix, "node"); err != nil {
			return ComputeInstance{}, err
		} else if volumeNodeName != "" {
			log.Debugf("Volume attachment required form Volume named %s", volumeNodeName)
			var device string
			if device, err = g.getStringFormConsul(kv, storagePrefix, "properties/location"); err != nil {
				return ComputeInstance{}, err
			}
			if device != "" {
				resolver := deployments.NewResolver(kv, deploymentID)
				expr := tosca.ValueAssignment{}
				if err = yaml.Unmarshal([]byte(device), &expr); err != nil {
					return ComputeInstance{}, err
				}
				// TODO check if instanceName is correct in all cases maybe we should check if we are in target context
				if device, err = resolver.ResolveExpressionForRelationship(expr.Expression, nodeName, volumeNodeName, path.Base(storagePrefix), instanceName); err != nil {
					return ComputeInstance{}, err
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

	networkKeys, err := deployments.GetRequirementsKeysByNameForNode(kv, deploymentID, nodeName, "network")
	if err != nil {
		return ComputeInstance{}, err
	}
	for _, networkReqPrefix := range networkKeys {
		capability, err := g.getStringFormConsul(kv, networkReqPrefix, "capability")
		if err != nil {
			return ComputeInstance{}, err
		}
		isFip, err := deployments.IsTypeDerivedFrom(kv, deploymentID, capability, "janus.capabilities.openstack.FIPConnectivity")
		if err != nil {
			return ComputeInstance{}, err
		}
		networkNodeName, err := g.getStringFormConsul(kv, networkReqPrefix, "node")
		if err != nil {
			return ComputeInstance{}, err
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
			instance.Networks[0].FloatingIP = floatingIP
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
			if instance.Networks == nil {
				instance.Networks = make([]ComputeNetwork, 0)
			}
			instance.Networks = append(instance.Networks, cn)
		}
	}
	return instance, nil
}
