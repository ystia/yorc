package openstack

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"io/ioutil"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
	"os"
	"path"
	"path/filepath"
	"novaforge.bull.com/starlings-janus/janus/commands"
	
)

type Generator struct {
	kv *api.KV
}

func NewGenerator(kv *api.KV) *Generator {
	return &Generator{kv: kv}
}

func (g *Generator) getStringFormConsul(baseUrl, property string) (string, error) {
	getResult, _, err := g.kv.Get(baseUrl + "/" + property, nil)
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

func (g *Generator) GenerateTerraformInfraForNode(depId, nodeName string) error {

	commands.init()

	log.Debugf("Generating infrastructure for deployment with id %s", depId)
	nodeKey := path.Join(deployments.DeploymentKVPrefix, depId, "topology", "nodes", nodeName)

	// Management of Variables
	infrastructure := commons.Infrastructure{

		provider := map[string]interface{}{
		"openstack": []interface{}{
			"user_name": USER_NAME,
			"tenant_name": TENANT_NAME,
			"password": PASSWORD,
			"auth_url": AUTH_URL,
		} }

		variable := map[string]interface{}{
		"cli_image_id": []interface{}{
			"default": CLI_IMAGE_ID
		} }

		variable := map[string]interface{}{
		"manager_image_id": []interface{}{
			"default":  MANAGER_IMAGE_ID
		} }

		variable := map[string]interface{}{
		"cli_flavor": []interface{}{
			"default": CLI_FLAVOR
		} }

		variable := map[string]interface{}{
		"manager_flavor_id": []interface{}{
			"default": MANAGER_FLAVOR_ID
		} }

		variable := map[string]interface{}{
		"ssh_key_file": []interface{}{
			"default": SSH_KEY_FILE
		} }

		variable := map[string]interface{}{
		"ssh_user_name": []interface{}{
			"default": SSH_USER_NAME
		} }

		variable := map[string]interface{}{
		"public_network_name": []interface{}{
			"default": PUBLIC_NETWORK_NAME
		} }

		variable := map[string]interface{}{
		"external_gateway": []interface{}{
			"default": EXTERNAL_GATEWAY
		} }

		variable := map[string]interface{}{
		"keystone_user": []interface{}{
			"default": KEYSTONE_USER
		} }

		variable := map[string]interface{}{
		"keystone_password": []interface{}{
			"default": KEYSTONE_PASSWORD
		} }

		variable := map[string]interface{}{
		"keystone_tenant": []interface{}{
			"default": KEYSTONE_TENANT 
		} }

		variable := map[string]interface{}{
		"keystone_url": []interface{}{
			"default": KEYSTONE_URL 
		} }

		variable := map[string]interface{}{
		"region": []interface{}{
			"default": REGION 
		} }

		variable := map[string]interface{}{
		"prefix": []interface{}{
			"default": PREFIX 
		} }

		variable := map[string]interface{}{
		"cli_availability_zone": []interface{}{
			"default": CLI_AVAILABILITY_ZONE 
		} }

		variable := map[string]interface{}{
		"manager_availability_zone": []interface{}{
			"default": MANAGER_AVAILABILITY_ZONE 
		} }

	}
	
	log.Debugf("inspecting node %s", nodeKey)
	kvPair, _, err := g.kv.Get(nodeKey + "/type", nil)
	if err != nil {
		log.Print(err)
		return err
	}
	nodeType := string(kvPair.Value)
	switch nodeType {
	case "janus.nodes.openstack.Compute":
		compute, err := g.generateOSInstance(nodeKey, depId)
		if err != nil {
			return err
		}
		addResource(&infrastructure, "openstack_compute_instance_v2", compute.Name, &compute)

		consulKey := commons.ConsulKey{Name: PREFIX+compute.Name + "-ip_address-key", Path: nodeKey + "/capabilities/endpoint/attributes/ip_address", Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.access_ip_v4}", compute.Name)}
		consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
		addResource(&infrastructure, "consul_keys", PREFIX+compute.Name, &consulKeys)

	case "janus.nodes.openstack.BlockStorage":
		if volumeId, err := g.getStringFormConsul(nodeKey, "properties/volume_id"); err != nil {
			return err
		} else if volumeId != "" {
			log.Debugf("Reusing existing volume with id %q for node %q", volumeId, nodeName)
			return nil
		}
		bsvol, err := g.generateOSBSVolume(nodeKey)
		if err != nil {
			return err
		}

		addResource(&infrastructure, "openstack_blockstorage_volume_v1", nodeName, &bsvol)
		consulKey := commons.ConsulKey{Name: nodeName + "-bsVolumeID", Path: nodeKey + "/properties/volume_id", Value: fmt.Sprintf("${openstack_blockstorage_volume_v1.%s.id}", nodeName)}
		consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
		addResource(&infrastructure, "consul_keys", nodeName, &consulKeys)

	case "janus.nodes.openstack.FloatIP":
		floatIPString, err, isIp := g.generateFloatIP(nodeKey)
		if err != nil {
			return  err
		}

		consulKey := commons.ConsulKey{}
		if !isIp {
			floatIP := FloatingIP{Pool: floatIPString}
			addResource(&infrastructure, "openstack_compute_floatingip_v2", nodeName, &floatIP)
			consulKey = commons.ConsulKey{Name: nodeName + "-floating_ip_address-key", Path: nodeKey + "/capabilities/endpoint/attributes/floating_ip_address", Value: fmt.Sprintf("${openstack_compute_floatingip_v2.%s.address}", nodeName)}
		} else {
			consulKey = commons.ConsulKey{Name: nodeName + "-floating_ip_address-key", Path: nodeKey + "/capabilities/endpoint/attributes/floating_ip_address", Value: floatIPString }
		}
		consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}
		addResource(&infrastructure, "consul_keys", nodeName, &consulKeys)


	default:
		return fmt.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, depId)
	}

	jsonInfra, err := json.MarshalIndent(infrastructure, "", "  ")
	infraPath := filepath.Join("work", "deployments", fmt.Sprint(depId), "infra", nodeName)
	if err = os.MkdirAll(infraPath, 0775); err != nil {
		log.Printf("%+v", err)
		return err
	}

	if err = ioutil.WriteFile(filepath.Join(infraPath, "infra.tf.json"), jsonInfra, 0664); err != nil {
		log.Print("Failed to write file")
		return err
	}

	log.Printf("Infrastructure generated for deployment with id %s", depId)
	return nil
}
