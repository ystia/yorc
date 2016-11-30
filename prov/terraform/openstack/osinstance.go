package openstack

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"path"
	"strings"
	"time"
)

func (g *Generator) generateOSInstance(url, deploymentId, instanceName string) (ComputeInstance, error) {
	var nodeType string
	var err error

	var OS_prefix string = g.cfg.OS_PREFIX
	var OS_region string = g.cfg.OS_REGION

	if nodeType, err = g.getStringFormConsul(url, "type"); err != nil {
		return ComputeInstance{}, err
	}
	if nodeType != "janus.nodes.openstack.Compute" {
		return ComputeInstance{}, fmt.Errorf("Unsupported node type for %s: %s", url, nodeType)
	}
	instance := ComputeInstance{}
	var nodeName string
	if nodeName, err = g.getStringFormConsul(url, "name"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.Name = OS_prefix + nodeName + "-" + instanceName
	}
	if image, err := g.getStringFormConsul(url, "properties/image"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.ImageId = image
	}
	if image, err := g.getStringFormConsul(url, "properties/imageName"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.ImageName = image
	}
	if flavor, err := g.getStringFormConsul(url, "properties/flavor"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.FlavorId = flavor
	}
	if flavor, err := g.getStringFormConsul(url, "properties/flavorName"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.FlavorName = flavor
	}

	if az, err := g.getStringFormConsul(url, "properties/availability_zone"); err != nil {
		return ComputeInstance{}, err
	} else {
		instance.AvailabilityZone = az
	}
	if region, err := g.getStringFormConsul(url, "properties/region"); err != nil {
		return ComputeInstance{}, err
	} else if region != "" {
		instance.Region = region
	} else {
		instance.Region = OS_region
	}

	if keyPair, err := g.getStringFormConsul(url, "properties/key_pair"); err != nil {
		return ComputeInstance{}, err
	} else {
		// TODO if empty use a default one or fail ?
		instance.KeyPair = keyPair
	}

	instance.SecurityGroups = g.cfg.OS_DEFAULT_SECURITY_GROUPS
	if secGroups, err := g.getStringFormConsul(url, "properties/security_groups"); err != nil {
		return ComputeInstance{}, err
	} else if secGroups != "" {
		for _, secGroup := range strings.Split(strings.NewReplacer("\"", "", "'", "").Replace(secGroups), ",") {
			secGroup = strings.TrimSpace(secGroup)
			instance.SecurityGroups = append(instance.SecurityGroups, secGroup)
		}
	}

	if instance.ImageId == "" && instance.ImageName == "" {
		return ComputeInstance{}, fmt.Errorf("Missing mandatory parameter 'image' or 'imageName' node type for %s", url)
	}
	if instance.FlavorId == "" && instance.FlavorName == "" {
		return ComputeInstance{}, fmt.Errorf("Missing mandatory parameter 'flavor' or 'flavorName' node type for %s", url)
	}

	if networkName, err := g.getStringFormConsul(url, "capabilities/endpoint/properties/network_name"); err != nil {
		return ComputeInstance{}, err
	} else {
		if networkName != "" {
			// TODO Deal with networks aliases (PUBLIC/PRIVATE)
			var networkSlice []ComputeNetwork
			if strings.EqualFold(networkName, "private") {
				networkSlice = append(networkSlice, ComputeNetwork{Name: g.cfg.OS_PRIVATE_NETWORK_NAME, AccessNetwork: true})
			} else if strings.EqualFold(networkName, "public") {
				//TODO
				return ComputeInstance{}, fmt.Errorf("Public Network aliases currently not supported")
			} else {
				networkSlice = append(networkSlice, ComputeNetwork{Name: networkName, AccessNetwork: true})
			}
			instance.Networks = networkSlice
		} else {
			// Use a default
			instance.Networks = append(instance.Networks, ComputeNetwork{Name: g.cfg.OS_PRIVATE_NETWORK_NAME, AccessNetwork: true})
		}
	}

	var user string
	if user, err = g.getStringFormConsul(url, "properties/user"); err != nil {
		return ComputeInstance{}, err
	} else if user == "" {
		return ComputeInstance{}, fmt.Errorf("Missing mandatory parameter 'user' node type for %s", url)
	}

	// TODO deal with multi-instances
	if storageKeys, err := deployments.GetRequirementsKeysByNameForNode(g.kv, deploymentId, nodeName, "local_storage"); err != nil {
		return ComputeInstance{}, err
	} else {

		for _, storagePrefix := range storageKeys {
			if instance.Volumes == nil {
				instance.Volumes = make([]Volume, 0)
			}
			if volumeNodeName, err := g.getStringFormConsul(storagePrefix, "node"); err != nil {
				return ComputeInstance{}, err
			} else if volumeNodeName != "" {
				log.Debugf("Volume attachment required form Volume named %s", volumeNodeName)
				var device string
				if device, err = g.getStringFormConsul(storagePrefix, "properties/location"); err != nil {
					return ComputeInstance{}, err
				}
				if device != "" {
					resolver := deployments.NewResolver(g.kv, deploymentId)
					expr := tosca.ValueAssignment{}
					if err := yaml.Unmarshal([]byte(device), &expr); err != nil {
						return ComputeInstance{}, err
					}
					// TODO check if instanceName is correct in all cases maybe we should check if we are in target context
					if device, err = resolver.ResolveExpressionForRelationship(expr.Expression, nodeName, volumeNodeName, path.Base(storagePrefix), instanceName); err != nil {
						return ComputeInstance{}, err
					}
				}
				var volumeId string
				log.Debugf("Looking for volume_id")
				if kp, _, _ := g.kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", volumeNodeName, "properties/volume_id"), nil); kp != nil {
					if dId := string(kp.Value); dId != "" {
						volumeId = dId
					}
				} else {
					resultChan := make(chan string, 1)
					go func() {
						for {
							// ignore errors and retry
							if kp, _, _ := g.kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", volumeNodeName, "attributes/volume_id"), nil); kp != nil {
								if dId := string(kp.Value); dId != "" {
									resultChan <- dId
									return
								}
							}
							time.Sleep(1 * time.Second)
						}
					}()
					// TODO add a cancellation signal
					select {
					case volumeId = <-resultChan:
					}
				}

				vol := Volume{VolumeId: volumeId, Device: device}
				instance.Volumes = append(instance.Volumes, vol)
			}
		}
	}
	// Do this in order to be sure that ansible will be able to log on the instance
	// TODO private key should not be hard-coded
	re := commons.RemoteExec{Inline: []string{`echo "connected"`}, Connection: commons.Connection{User: user, PrivateKey: `${file("~/.ssh/janus.pem")}`}}
	instance.Provisioners = make(map[string]interface{})
	instance.Provisioners["remote-exec"] = re

	// TODO deal with multi-instances
	if networkKeys, err := deployments.GetRequirementsKeysByNameForNode(g.kv, deploymentId, nodeName, "network"); err != nil {
		return ComputeInstance{}, err
	} else {
		for _, networkReqPrefix := range networkKeys {
			capability, err := g.getStringFormConsul(networkReqPrefix, "capability")
			if err != nil {
				return ComputeInstance{}, err
			}
			isFip, err := deployments.IsNodeTypeDerivedFrom(g.kv, deploymentId, capability, "janus.capabilities.openstack.FIPConnectivity")
			if err != nil {
				return ComputeInstance{}, err
			}
			networkNodeName, err := g.getStringFormConsul(networkReqPrefix, "node")
			if err != nil {
				return ComputeInstance{}, err
			}
			if isFip {
				log.Debugf("Looking for Floating IP")
				var floatingIP string
				resultChan := make(chan string, 1)
				go func() {
					for {
						if kp, _, _ := g.kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances", networkNodeName, instanceName, "capabilities/endpoint/attributes/floating_ip_address"), nil); kp != nil {
							if dId := string(kp.Value); dId != "" {
								resultChan <- dId
								return
							}
						}
						time.Sleep(1 * time.Second)
					}
				}()
				select {
				case floatingIP = <-resultChan:
				}
				instance.Networks[0].FloatingIp = floatingIP
			} else {
				log.Debugf("Looking for Network id for %q", networkNodeName)
				var networkId string
				resultChan := make(chan string, 1)
				go func() {
					for {
						if kp, _, _ := g.kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", networkNodeName, "attributes/network_id"), nil); kp != nil {
							if dId := string(kp.Value); dId != "" {
								resultChan <- dId
								return
							}
						}
						time.Sleep(1 * time.Second)
					}
				}()
				select {
				case networkId = <-resultChan:
				}
				cn := ComputeNetwork{UUID: networkId, AccessNetwork: false}
				if instance.Networks == nil {
					instance.Networks = make([]ComputeNetwork, 0)
				}
				instance.Networks = append(instance.Networks, cn)
			}
		}
	}
	return instance, nil
}
