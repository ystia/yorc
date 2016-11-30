package openstack

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"io/ioutil"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type Generator struct {
	kv  *api.KV
	cfg config.Configuration
}

func NewGenerator(kv *api.KV, cfg config.Configuration) *Generator {
	return &Generator{kv: kv, cfg: cfg}
}

func (g *Generator) getStringFormConsul(baseUrl, property string) (string, error) {
	getResult, _, err := g.kv.Get(baseUrl+"/"+property, nil)
	if err != nil {
		log.Printf("Can't get property %s for node %s", property, baseUrl)
		return "", fmt.Errorf("Can't get property %s for node %s: %v", property, baseUrl, err)
	}
	if getResult == nil {
		log.Debugf("Can't get property %s for node %s (not found)", property, baseUrl)
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

// GenerateTerraformInfraForNode generates the Terraform infrastructure file for the given node.
// It returns 'true' if a file was generated and 'false' otherwise (in case of a infrastructure component
// already exists for this node and should just be reused).
func (g *Generator) GenerateTerraformInfraForNode(depId, nodeName string) (bool, error) {

	log.Debugf("Generating infrastructure for deployment with id %s", depId)
	nodeKey := path.Join(consulutil.DeploymentKVPrefix, depId, "topology", "nodes", nodeName)
	instancesKey := path.Join(consulutil.DeploymentKVPrefix, depId, "topology", "instances", nodeName)

	// Management of variables for Terraform
	infrastructure := commons.Infrastructure{}
	infrastructure.Provider = map[string]interface{}{
		"openstack": map[string]interface{}{
			"user_name":   g.cfg.OS_USER_NAME,
			"tenant_name": g.cfg.OS_TENANT_NAME,
			"password":    g.cfg.OS_PASSWORD,
			"auth_url":    g.cfg.OS_AUTH_URL}}

	log.Debugf("inspecting node %s", nodeKey)
	kvPair, _, err := g.kv.Get(nodeKey+"/type", nil)
	if err != nil {
		log.Print(err)
		return false, err
	}
	nodeType := string(kvPair.Value)
	switch nodeType {
	case "janus.nodes.openstack.Compute":
		instances, _, err := g.kv.Keys(instancesKey+"/", "/", nil)
		if err != nil {
			return false, err
		}

		for _, instanceName := range instances {
			instanceName = path.Base(instanceName)
			compute, err := g.generateOSInstance(nodeKey, depId, instanceName)
			if err != nil {
				return false, err
			}
			addResource(&infrastructure, "openstack_compute_instance_v2", compute.Name, &compute)
			consulKey := commons.ConsulKey{Name: compute.Name + "-ip_address-key", Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.access_ip_v4}", compute.Name)}                           // Use access ip here
			consulKeyAttrib := commons.ConsulKey{Name: compute.Name + "-attrib_ip_address-key", Path: path.Join(instancesKey, instanceName, "/attributes/ip_address"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.fixed_ip_v4}", compute.Name, len(compute.Networks)-1)} // Use latest provisioned network for private access
			consulKeyFixedIP := commons.ConsulKey{Name: compute.Name + "-ip_fixed_address-key", Path: path.Join(instancesKey, instanceName, "/attributes/private_address"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.fixed_ip_v4}", compute.Name, len(compute.Networks)-1)}
			var consulKeys commons.ConsulKeys
			if compute.Networks[0].FloatingIp != "" {
				consulKeyFloatingIP := commons.ConsulKey{Name: compute.Name + "-ip_floating_address-key", Path: path.Join(instancesKey, instanceName, "/attributes/public_address"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.0.floating_ip}", compute.Name)}
				//In order to be backward compatible to components developed for Alien/Cloudify (only the above is standard)
				consulKeyFloatingIPBak := commons.ConsulKey{Name: compute.Name + "-ip_floating_address_backward_comp-key", Path: path.Join(instancesKey, instanceName, "/attributes/public_ip_address"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.0.floating_ip}", compute.Name)}
				consulKeys = commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey, consulKeyAttrib, consulKeyFixedIP, consulKeyFloatingIP, consulKeyFloatingIPBak}}

			} else {
				consulKeys = commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey, consulKeyAttrib, consulKeyFixedIP}}
			}

			for i := range compute.Networks {
				consulKetNetName := commons.ConsulKey{Name: fmt.Sprintf("%s-network-%d-name", compute.Name, i), Path: path.Join(instancesKey, instanceName, "attributes/networks", strconv.Itoa(i), "network_name"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.name}", compute.Name, i)}
				consulKetNetId := commons.ConsulKey{Name: fmt.Sprintf("%s-network-%d-id", compute.Name, i), Path: path.Join(instancesKey, instanceName, "attributes/networks", strconv.Itoa(i), "network_id"), Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.network.%d.uuid}", compute.Name, i)}
				consulKetNetAddresses := commons.ConsulKey{Name: fmt.Sprintf("%s-network-%d-addresses", compute.Name, i), Path: path.Join(instancesKey, instanceName, "attributes/networks", strconv.Itoa(i), "addresses"), Value: fmt.Sprintf("[ ${openstack_compute_instance_v2.%s.network.%d.fixed_ip_v4} ]", compute.Name, i)}
				consulKeys.Keys = append(consulKeys.Keys, consulKetNetName, consulKetNetId, consulKetNetAddresses)
			}
			addResource(&infrastructure, "consul_keys", compute.Name, &consulKeys)
		}

	case "janus.nodes.openstack.BlockStorage":
		if volumeId, err := g.getStringFormConsul(nodeKey, "properties/volume_id"); err != nil {
			return false, err
		} else if volumeId != "" {
			log.Debugf("Reusing existing volume with id %q for node %q", volumeId, nodeName)
			return false, nil
		}
		bsvol, err := g.generateOSBSVolume(nodeKey)
		if err != nil {
			return false, err
		}

		addResource(&infrastructure, "openstack_blockstorage_volume_v1", nodeName, &bsvol)
		consulKey := commons.ConsulKey{Name: nodeName + "-bsVolumeID", Path: nodeKey + "/attributes/volume_id", Value: fmt.Sprintf("${openstack_blockstorage_volume_v1.%s.id}", nodeName)}
		consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
		addResource(&infrastructure, "consul_keys", nodeName, &consulKeys)

	case "janus.nodes.openstack.FloatingIP":
		instances, err := deployments.GetNodeInstancesIds(g.kv, depId, nodeName)
		if err != nil {
			return false, err
		}

		for _, instanceName := range instances {

			ip, err := g.generateFloatingIP(nodeKey, instanceName)

			if err != nil {
				return false, err
			}

			consulKey := commons.ConsulKey{}
			if !ip.IsIp {
				floatingIP := FloatingIP{Pool: ip.Pool}
				addResource(&infrastructure, "openstack_compute_floatingip_v2", ip.Name, &floatingIP)
				consulKey = commons.ConsulKey{Name: ip.Name + "-floating_ip_address-key", Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/floating_ip_address"), Value: fmt.Sprintf("${openstack_compute_floatingip_v2.%s.address}", ip.Name)}
			} else {
				ips := strings.Split(ip.Pool, ",")
				instName, err := strconv.Atoi(instanceName)
				if (len(ips) - 1) < instName {
					networkName, err := g.getStringFormConsul(nodeKey, "properties/floating_network_name")
					if err != nil {
						return false, err
					} else if networkName == "" {
						return false, fmt.Errorf("You need to provide enough IP address or a Pool to generate missing IP address")
					}

					floatingIP := FloatingIP{Pool: networkName}
					addResource(&infrastructure, "openstack_compute_floatingip_v2", ip.Name, &floatingIP)
					consulKey = commons.ConsulKey{Name: ip.Name + "-floating_ip_address-key", Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/floating_ip_address"), Value: fmt.Sprintf("${openstack_compute_floatingip_v2.%s.address}", ip.Name)}

				} else {
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
		if networkId, err := g.getStringFormConsul(nodeKey, "properties/network_id"); err != nil {
			return false, err
		} else if networkId != "" {
			log.Debugf("Reusing existing volume with id %q for node %q", networkId, nodeName)
			return false, nil
		}
		network, err := g.generateNetwork(nodeKey, depId)

		if err != nil {
			return false, err
		}

		subnet, err := g.generateSubnet(nodeKey, depId, nodeName)

		if err != nil {
			return false, err
		}

		addResource(&infrastructure, "openstack_networking_network_v2", nodeName, &network)
		addResource(&infrastructure, "openstack_networking_subnet_v2", nodeName+"_subnet", &subnet)
		consulKey := commons.ConsulKey{Name: nodeName + "-NetworkID", Path: nodeKey + "/attributes/network_id", Value: fmt.Sprintf("${openstack_networking_network_v2.%s.id}", nodeName)}
		consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
		addResource(&infrastructure, "consul_keys", nodeName, &consulKeys)

	default:
		return false, fmt.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, depId)
	}

	jsonInfra, err := json.MarshalIndent(infrastructure, "", "  ")
	infraPath := filepath.Join("work", "deployments", fmt.Sprint(depId), "infra", nodeName)
	if err = os.MkdirAll(infraPath, 0775); err != nil {
		log.Printf("%+v", err)
		return false, err
	}

	if err = ioutil.WriteFile(filepath.Join(infraPath, "infra.tf.json"), jsonInfra, 0664); err != nil {
		log.Print("Failed to write file")
		return false, err
	}

	log.Printf("Infrastructure generated for deployment with id %s", depId)
	return true, nil
}
