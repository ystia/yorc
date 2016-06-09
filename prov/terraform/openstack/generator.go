package openstack

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"io/ioutil"
	"log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform"
	"os"
	"path"
	"strings"
)

type Generator struct {
	consulClient *api.Client
	kv           *api.KV
}

func NewGenerator(consulClient *api.Client) *Generator {
	return &Generator{consulClient: consulClient, kv: consulClient.KV()}
}

func (g *Generator) getStringFormConsul(baseUrl, property string) (string, error) {
	getResult, _, err := g.kv.Get(baseUrl+"/"+property, nil)
	if err != nil {
		log.Printf("Can't get property %s for node %s", property, baseUrl)
		return "", fmt.Errorf("Can't get property %s for node %s: %v", property, baseUrl, err)
	}
	if getResult == nil {
		log.Printf("Can't get property %s for node %s (not found)", property, baseUrl)
		return "", nil
	}
	return string(getResult.Value), nil
}

func (g *Generator) GenerateTerraformInfra(id string) error {
	log.Printf("Generating infrastructure for deployment with id %s", id)
	topoUrl := strings.Join([]string{"_janus", id, "topology", "nodes"}, "/")
	infrastructure := terraform.Infrastructure{}
	nodesKeys, _, err := g.kv.Keys(topoUrl+"/", "/", nil)
	if err != nil {
		log.Print(err)
		return err
	}
	for _, nodeKey := range nodesKeys {
		log.Printf("inspecting node %s", nodeKey)
		kvPair, _, err := g.kv.Get(nodeKey+"/type", nil)
		if err != nil {
			log.Print(err)
			return err
		}
		switch string(kvPair.Value) {
		case "janus.nodes.openstack.Compute":
			compute, err := g.generateOSInstance(nodeKey)
			if err != nil {
				return err
			}
			if len(infrastructure.Resource) != 0 {
				if infrastructure.Resource["openstack_compute_instance_v2"] != nil && len(infrastructure.Resource["openstack_compute_instance_v2"].(map[string]interface{})) != 0 {
					osInstancesMap := infrastructure.Resource["openstack_compute_instance_v2"].(map[string]interface{})
					osInstancesMap[compute.Name] = compute
				} else {
					osInstancesMap := make(map[string]interface{})
					osInstancesMap[compute.Name] = compute
					infrastructure.Resource["openstack_compute_instance_v2"] = osInstancesMap
				}

			} else {
				resourceMap := make(map[string]interface{})
				infrastructure.Resource = resourceMap
				osInstancesMap := make(map[string]interface{})
				osInstancesMap[compute.Name] = compute
				infrastructure.Resource["openstack_compute_instance_v2"] = osInstancesMap
			}

			consulKey := terraform.ConsulKey{Name: compute.Name + "-ip_address-key", Path: nodeKey + "capabilities/endpoint/attributes/ip_address", Value: fmt.Sprintf("${openstack_compute_instance_v2.%s.access_ip_v4}", compute.Name)}
			consulKeys := terraform.ConsulKeys{Keys: []terraform.ConsulKey{consulKey}}
			if infrastructure.Resource["consul_keys"] != nil && len(infrastructure.Resource["consul_keys"].(map[string]interface{})) != 0 {
				consulKeysMap := infrastructure.Resource["consul_keys"].(map[string]interface{})
				consulKeysMap[compute.Name] = consulKeys
			} else {
				consulKeysMap := make(map[string]interface{})
				consulKeysMap[compute.Name] = consulKeys
				infrastructure.Resource["consul_keys"] = consulKeysMap
			}
		}
	}

	jsonInfra, err := json.MarshalIndent(infrastructure, "", "  ")
	infraPath := path.Join("work", "deployments", fmt.Sprint(id), "infra")
	if err = os.MkdirAll(infraPath, 0775); err != nil {
		log.Printf("%+v", err)
		return err
	}

	if err = ioutil.WriteFile(path.Join(infraPath, "infra.tf.json"), jsonInfra, 0664); err != nil {
		log.Printf("Failed to write file")
		return err
	}

	log.Printf("Infrastructure generated for deployment with id %s", id)
	return nil
}
