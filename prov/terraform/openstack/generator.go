package openstack

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

type osGenerator struct {
	kv  *api.KV
	cfg config.Configuration
}

// NewGenerator creates a generator for OpenStack resources
func NewGenerator(kv *api.KV, cfg config.Configuration) commons.Generator {
	return &osGenerator{kv: kv, cfg: cfg}
}

func (g *osGenerator) getStringFormConsul(baseURL, property string) (string, error) {
	getResult, _, err := g.kv.Get(baseURL+"/"+property, nil)
	if err != nil {
		log.Printf("Can't get property %s for node %s", property, baseURL)
		return "", fmt.Errorf("Can't get property %s for node %s: %v", property, baseURL, err)
	}
	if getResult == nil {
		log.Debugf("Can't get property %s for node %s (not found)", property, baseURL)
		return "", nil
	}
	return string(getResult.Value), nil
}

func addResource(infrastructure *commons.Infrastructure, resourceType, resourceName string, resource interface{}) {
	if len(infrastructure.Resource) != 0 {
		if infrastructure.Resource[resourceType] != nil && len(infrastructure.Resource[resourceType].(map[string]interface{})) != 0 {
			resourcesMap := infrastructure.Resource[resourceType].(map[string]interface{})
			resourcesMap[resourceName] = resource
		} else {
			resourcesMap := make(map[string]interface{})
			resourcesMap[resourceName] = resource
			infrastructure.Resource[resourceType] = resourcesMap
		}

	} else {
		resourcesMap := make(map[string]interface{})
		infrastructure.Resource = resourcesMap
		resourcesMap = make(map[string]interface{})
		resourcesMap[resourceName] = resource
		infrastructure.Resource[resourceType] = resourcesMap
	}
}

func (g *osGenerator) GenerateTerraformInfraForNode(deploymentID, nodeName string) (bool, error) {

	log.Debugf("Generating infrastructure for deployment with id %s", deploymentID)
	nodeKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	instancesKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName)
	terraformStateKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "terraform-state", nodeName)

	infrastructure := commons.Infrastructure{}

	// Remote Configuration for Terraform State to store it in the Consul KV store
	infrastructure.Terraform = map[string]interface{}{
		"backend": map[string]interface{}{
			"consul": map[string]interface{}{
				"path": terraformStateKey,
			},
		},
	}

	// Management of variables for Terraform
	infrastructure.Provider = map[string]interface{}{
		"openstack": map[string]interface{}{
			"user_name":   g.cfg.OSUserName,
			"tenant_name": g.cfg.OSTenantName,
			"password":    g.cfg.OSPassword,
			"auth_url":    g.cfg.OSAuthURL}}

	log.Debugf("inspecting node %s", nodeKey)
	nodeType, err := deployments.GetNodeType(g.kv, deploymentID, nodeName)
	if err != nil {
		return false, err
	}
	var instances []string
	switch nodeType {
	case "janus.nodes.openstack.Compute":
		instances, err = deployments.GetNodeInstancesIds(g.kv, deploymentID, nodeName)
		if err != nil {
			return false, err
		}

		for _, instanceName := range instances {
			var instanceState tosca.NodeState
			instanceState, err = deployments.GetInstanceState(g.kv, deploymentID, nodeName, instanceName)
			if err != nil {
				return false, err
			}
			if instanceState == tosca.NodeStateDeleting || instanceState == tosca.NodeStateDeleted {
				// Do not generate something for this node instance (will be deleted if exists)
				continue
			}
			var compute ComputeInstance
			compute, err = g.generateOSInstance(nodeKey, deploymentID, instanceName)
			if err != nil {
				return false, err
			}
			addResource(&infrastructure, "openstack_compute_instance_v2", compute.Name, &compute)
			consulKey := commons.ConsulKey{Name: compute.Name + "-ip_address-key", Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.access_ip_v4}", compute.Name)}                           // Use access ip here
			consulKeyAttrib := commons.ConsulKey{Name: compute.Name + "-attrib_ip_address-key", Path: path.Join(instancesKey, instanceName, "/attributes/ip_address"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.fixed_ip_v4}", compute.Name, len(compute.Networks)-1)} // Use latest provisioned network for private access
			consulKeyFixedIP := commons.ConsulKey{Name: compute.Name + "-ip_fixed_address-key", Path: path.Join(instancesKey, instanceName, "/attributes/private_address"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.fixed_ip_v4}", compute.Name, len(compute.Networks)-1)}
			var consulKeys commons.ConsulKeys
			if compute.Networks[0].FloatingIP != "" {
				consulKeyFloatingIP := commons.ConsulKey{Name: compute.Name + "-ip_floating_address-key", Path: path.Join(instancesKey, instanceName, "/attributes/public_address"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.0.floating_ip}", compute.Name)}
				//In order to be backward compatible to components developed for Alien/Cloudify (only the above is standard)
				consulKeyFloatingIPBak := commons.ConsulKey{Name: compute.Name + "-ip_floating_address_backward_comp-key", Path: path.Join(instancesKey, instanceName, "/attributes/public_ip_address"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.0.floating_ip}", compute.Name)}
				consulKeys = commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey, consulKeyAttrib, consulKeyFixedIP, consulKeyFloatingIP, consulKeyFloatingIPBak}}

			} else {
				consulKeys = commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey, consulKeyAttrib, consulKeyFixedIP}}
			}

			for i := range compute.Networks {
				consulKetNetName := commons.ConsulKey{Name: fmt.Sprintf("%s-network-%d-name", compute.Name, i), Path: path.Join(instancesKey, instanceName, "attributes/networks", strconv.Itoa(i), "network_name"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.name}", compute.Name, i)}
				consulKetNetID := commons.ConsulKey{Name: fmt.Sprintf("%s-network-%d-id", compute.Name, i), Path: path.Join(instancesKey, instanceName, "attributes/networks", strconv.Itoa(i), "network_id"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.uuid}", compute.Name, i)}
				consulKetNetAddresses := commons.ConsulKey{Name: fmt.Sprintf("%s-network-%d-addresses", compute.Name, i), Path: path.Join(instancesKey, instanceName, "attributes/networks", strconv.Itoa(i), "addresses"), Value: fmt.Sprintf("[ ${openstack_compute_instance_v2.%s.network.%d.fixed_ip_v4} ]", compute.Name, i)}
				consulKeys.Keys = append(consulKeys.Keys, consulKetNetName, consulKetNetID, consulKetNetAddresses)
			}
			addResource(&infrastructure, "consul_keys", compute.Name, &consulKeys)
		}

	case "janus.nodes.openstack.BlockStorage":
		instances, err = deployments.GetNodeInstancesIds(g.kv, deploymentID, nodeName)
		if err != nil {
			return false, err
		}

		var bsIds []string
		var volumeID string
		if volumeID, err = g.getStringFormConsul(nodeKey, "properties/volume_id"); err != nil {
			return false, err
		} else if volumeID != "" {
			log.Debugf("Reusing existing volume with id %q for node %q", volumeID, nodeName)
			bsIds = strings.Split(volumeID, ",")
		}

		for instNb, instanceName := range instances {
			var instanceState tosca.NodeState
			instanceState, err = deployments.GetInstanceState(g.kv, deploymentID, nodeName, instanceName)
			if err != nil {
				return false, err
			}
			if instanceState == tosca.NodeStateDeleting || instanceState == tosca.NodeStateDeleted {
				// Do not generate something for this node instance (will be deleted if exists)
				continue
			}
			var bsVolume BlockStorageVolume
			bsVolume, err = g.generateOSBSVolume(nodeKey, instanceName)
			if err != nil {
				return false, err
			}

			if len(bsIds)-1 < instNb {
				addResource(&infrastructure, "openstack_blockstorage_volume_v1", bsVolume.Name, &bsVolume)
				consulKey := commons.ConsulKey{Name: bsVolume.Name + "-bsVolumeID", Path: path.Join(instancesKey, instanceName, "/attributes/volume_id"), Value: fmt.Sprintf("${openstack_blockstorage_volume_v1.%s.id}", bsVolume.Name)}
				consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
				addResource(&infrastructure, "consul_keys", bsVolume.Name, &consulKeys)
			} else {
				name := g.cfg.ResourcesPrefix + nodeName + "-" + instanceName
				consulKey := commons.ConsulKey{Name: name + "-bsVolumeID", Path: path.Join(instancesKey, instanceName, "/properties/volume_id"), Value: strings.TrimSpace(bsIds[instNb])}
				consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
				addResource(&infrastructure, "consul_keys", name, &consulKeys)
			}

		}

	case "janus.nodes.openstack.FloatingIP":
		instances, err = deployments.GetNodeInstancesIds(g.kv, deploymentID, nodeName)
		if err != nil {
			return false, err
		}

		for _, instanceName := range instances {
			var instanceState tosca.NodeState
			instanceState, err = deployments.GetInstanceState(g.kv, deploymentID, nodeName, instanceName)
			if err != nil {
				return false, err
			}
			if instanceState == tosca.NodeStateDeleting || instanceState == tosca.NodeStateDeleted {
				// Do not generate something for this node instance (will be deleted if exists)
				continue
			}
			var ip IP
			ip, err = g.generateFloatingIP(nodeKey, instanceName)

			if err != nil {
				return false, err
			}

			var consulKey commons.ConsulKey
			if !ip.IsIP {
				floatingIP := FloatingIP{Pool: ip.Pool}
				addResource(&infrastructure, "openstack_compute_floatingip_v2", ip.Name, &floatingIP)
				consulKey = commons.ConsulKey{Name: ip.Name + "-floating_ip_address-key", Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/floating_ip_address"), Value: fmt.Sprintf("${openstack_compute_floatingip_v2.%s.address}", ip.Name)}
			} else {
				ips := strings.Split(ip.Pool, ",")
				// TODO we should change this. instance name should not be considered as an int
				var instName int
				instName, err = strconv.Atoi(instanceName)
				if err != nil {
					return false, err
				}
				if (len(ips) - 1) < instName {
					var networkName string
					networkName, err = g.getStringFormConsul(nodeKey, "properties/floating_network_name")
					if err != nil {
						return false, err
					} else if networkName == "" {
						return false, fmt.Errorf("You need to provide enough IP address or a Pool to generate missing IP address")
					}

					floatingIP := FloatingIP{Pool: networkName}
					addResource(&infrastructure, "openstack_compute_floatingip_v2", ip.Name, &floatingIP)
					consulKey = commons.ConsulKey{Name: ip.Name + "-floating_ip_address-key", Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/floating_ip_address"), Value: fmt.Sprintf("${openstack_compute_floatingip_v2.%s.address}", ip.Name)}

				} else {
					// TODO we should change this. instance name should not be considered as an int
					instName, err = strconv.Atoi(instanceName)
					if err != nil {
						return false, err
					}
					consulKey = commons.ConsulKey{Name: ip.Name + "-floating_ip_address-key", Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/floating_ip_address"), Value: ips[instName]}
				}

			}
			consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
			addResource(&infrastructure, "consul_keys", ip.Name, &consulKeys)
		}

	case "janus.nodes.openstack.Network":
		var networkID string
		networkID, err = g.getStringFormConsul(nodeKey, "properties/network_id")
		if err != nil {
			return false, err
		} else if networkID != "" {
			log.Debugf("Reusing existing volume with id %q for node %q", networkID, nodeName)
			return false, nil
		}
		var network Network
		network, err = g.generateNetwork(nodeKey, deploymentID)

		if err != nil {
			return false, err
		}
		var subnet Subnet
		subnet, err = g.generateSubnet(nodeKey, deploymentID, nodeName)

		if err != nil {
			return false, err
		}

		addResource(&infrastructure, "openstack_networking_network_v2", nodeName, &network)
		addResource(&infrastructure, "openstack_networking_subnet_v2", nodeName+"_subnet", &subnet)
		consulKey := commons.ConsulKey{Name: nodeName + "-NetworkID", Path: nodeKey + "/attributes/network_id", Value: fmt.Sprintf("${openstack_networking_network_v2.%s.id}", nodeName)}
		consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
		addResource(&infrastructure, "consul_keys", nodeName, &consulKeys)

	default:
		return false, fmt.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, deploymentID)
	}

	jsonInfra, err := json.MarshalIndent(infrastructure, "", "  ")
	if err != nil {
		return false, errors.Wrap(err, "Failed to generate JSON of terraform Infrastructure description")
	}
	infraPath := filepath.Join(g.cfg.WorkingDirectory, "deployments", fmt.Sprint(deploymentID), "infra", nodeName)
	if err = os.MkdirAll(infraPath, 0775); err != nil {
		log.Printf("%+v", err)
		return false, err
	}

	if err = ioutil.WriteFile(filepath.Join(infraPath, "infra.tf.json"), jsonInfra, 0664); err != nil {
		log.Print("Failed to write file")
		return false, err
	}

	log.Printf("Infrastructure generated for deployment with id %s", deploymentID)
	return true, nil
}
