package slurm

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"io/ioutil"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
	"os"
	"path"
	"path/filepath"
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

func (g *Generator) GenerateTerraformInfraForNode(depId, nodeName string) (bool, error) {

	log.Debugf("Generating infrastructure for deployment with id %s", depId)
	nodeKey := path.Join(deployments.DeploymentKVPrefix, depId, "topology", "nodes", nodeName)
	instancesKey := path.Join(deployments.DeploymentKVPrefix, depId, "topology", "instances", nodeName)

	// Add Provider
	log.Debugf("Addding provider Slurm")
	infrastructure := commons.Infrastructure{}
	infrastructure.Provider = map[string]interface{}{
		"slurm": map[string]interface{}{
			"username": "root",
			"password": "root",
			"name":     "name",
			"url":      "172.17.0.1",
			"port":     "22"}} // TODO : hardcoded variable

	log.Debugf("inspecting node %s", nodeKey)
	kvPair, _, err := g.kv.Get(nodeKey+"/type", nil)
	if err != nil {
		log.Print(err)
		return false, err
	}

	nodeType := string(kvPair.Value)
	log.Printf("GenerateTerraformInfraForNode switch begin")
	switch nodeType {
	case "janus.nodes.slurm.Compute":
		instances, _, err := g.kv.Keys(instancesKey+"/", "/", nil)
		if err != nil {
			return false, err
		}
		log.Debugf("-----------------------")
		log.Debugf("%+v\n", instances)
		log.Debugf("-----------------------")

		for _, instanceName := range instances {
			instanceName = path.Base(instanceName)
			compute, err := g.generateSlurmNode(nodeKey, depId)
			var computeName = nodeName + "-" + instanceName
			log.Debugf("XBD computeName : IN FOR  %s", computeName)
			log.Debugf("XBD instanceName: IN FOR  %s", instanceName)
			if err != nil {
				return false, err
			}

			addResource(&infrastructure, "slurm_node", computeName, &compute)

			consulKey := commons.ConsulKey{Name: computeName + "-node_name", Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address"), Value: fmt.Sprintf("${slurm_node.%s.node_name}", computeName)}
			consulKey2 := commons.ConsulKey{Name: computeName + "-ip_address-key", Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address"), Value: fmt.Sprintf("${slurm_node.%s.node_name}", computeName)}
			consulKeyAttrib := commons.ConsulKey{Name: computeName + "-attrib_ip_address-key", Path: path.Join(instancesKey, instanceName, "/attributes/ip_address"), Value: fmt.Sprintf("${slurm_node.%s.node_name}", computeName)}

			consulKeyJobId := commons.ConsulKey{Name: computeName + "-job_id", Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/job_id"), Value: fmt.Sprintf("${slurm_node.%s.job_id}", computeName)}

			var consulKeys commons.ConsulKeys
			consulKeys = commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey, consulKey2, consulKeyAttrib, consulKeyJobId}}
			addResource(&infrastructure, "consul_keys", computeName, &consulKeys)

		} //End instances loop

	case "janus.nodes.slurm.Cntk":
		instances, _, err := g.kv.Keys(instancesKey+"/", "/", nil)
		if err != nil {
			return false, err
		}
		for _, instanceName := range instances {
			instanceName = path.Base(instanceName)
			job, err := g.generateSlurmCntk(nodeKey, depId)
			var jobName = nodeName + "-" + instanceName
			log.Debugf("XBD JobName : IN FOR  %s", jobName)
			log.Debugf("XBD instanceName: IN FOR  %s", instanceName)
			if err != nil {
				return false, err
			}

			addResource(&infrastructure, "slurm_cntk", jobName, &job)

			consulKeyFinalError := commons.ConsulKey{Name: jobName + "-final_error", Path: path.Join(instancesKey, instanceName, "/attributes/final_error"), Value: fmt.Sprintf("${slurm_cntk.%s.finalError}", jobName)}
			var consulKeys commons.ConsulKeys
			consulKeys = commons.ConsulKeys{Keys: []commons.ConsulKey{consulKeyFinalError}}
			addResource(&infrastructure, "consul_keys", jobName, &consulKeys)

		} //End instances loop

	default:
		return false, fmt.Errorf("In Slurm : Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, depId)
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
