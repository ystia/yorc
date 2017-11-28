package slurm

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

type slurmGenerator struct {
}

// NewGenerator creates a generator for Slurm resources
func NewGenerator() commons.Generator {
	return &slurmGenerator{}
}

func (g *slurmGenerator) getStringFormConsul(kv *api.KV, baseURL, property string) (string, error) {
	getResult, _, err := kv.Get(baseURL+"/"+property, nil)
	if err != nil {
		log.Printf("Can't get property %s for node %s", property, baseURL)
		return "", errors.Errorf("Can't get property %s for node %s: %v", property, baseURL, err)
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

func (g *slurmGenerator) GenerateTerraformInfraForNode(ctx context.Context, cfg config.Configuration, deploymentID, nodeName string) (bool, map[string]string, error) {
	log.Debugf("Generating infrastructure for deployment with id %s", deploymentID)
	cClient, err := cfg.GetConsulClient()
	if err != nil {
		return false, nil, err
	}
	kv := cClient.KV()
	nodeKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	instancesKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName)
	terraformStateKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "terraform-state", nodeName)

	consulAddress := "127.0.0.1:8500"
	if cfg.ConsulAddress != "" {
		consulAddress = cfg.ConsulAddress
	}
	consulScheme := "http"
	if cfg.ConsulSSL {
		consulScheme = "https"
	}
	consulCA := ""
	if cfg.ConsulCA != "" {
		consulCA = cfg.ConsulCA
	}
	consulKey := ""
	if cfg.ConsulKey != "" {
		consulKey = cfg.ConsulKey
	}
	consulCert := ""
	if cfg.ConsulCert != "" {
		consulCert = cfg.ConsulCert
	}

	infrastructure := commons.Infrastructure{}

	// Remote Configuration for Terraform State to store it in the Consul KV store
	infrastructure.Terraform = map[string]interface{}{
		"backend": map[string]interface{}{
			"consul": map[string]interface{}{
				"path":    terraformStateKey,
				"address": consulAddress,
				"scheme":  consulScheme,
			},
		},
	}

	// Management of variables for Terraform
	infrastructure.Provider = map[string]interface{}{
		"slurm": map[string]interface{}{
			"username": cfg.Infrastructures["slurm"].GetString("user_name"),
			"password": cfg.Infrastructures["slurm"].GetString("password"),
			"name":     cfg.Infrastructures["slurm"].GetString("name"),
			"url":      cfg.Infrastructures["slurm"].GetString("url"),
			"port":     cfg.Infrastructures["slurm"].GetString("port"),
		},
		"consul": map[string]interface{}{
			"address":   consulAddress,
			"scheme":    consulScheme,
			"ca_file":   consulCA,
			"cert_file": consulCert,
			"key_file":  consulKey,
		},
	}

	log.Debugf("inspecting node %s", nodeKey)
	kvPair, _, err := kv.Get(nodeKey+"/type", nil)
	if err != nil {
		log.Print(err)
		return false, nil, err
	}

	nodeType := string(kvPair.Value)
	log.Printf("GenerateTerraformInfraForNode switch begin")
	switch nodeType {
	case "janus.nodes.slurm.Compute":
		var instances []string
		instances, err = deployments.GetNodeInstancesIds(kv, deploymentID, nodeName)
		if err != nil {
			return false, nil, err
		}
		log.Debugf("-----------------------")
		log.Debugf("%+v\n", instances)
		log.Debugf("-----------------------")

		for _, instanceName := range instances {
			var instanceState tosca.NodeState
			instanceState, err = deployments.GetInstanceState(kv, deploymentID, nodeName, instanceName)
			if err != nil {
				return false, nil, err
			}
			if instanceState == tosca.NodeStateDeleting || instanceState == tosca.NodeStateDeleted {
				// Do not generate something for this node instance (will be deleted if exists)
				continue
			}
			instanceName = path.Base(instanceName)
			var compute ComputeInstance
			compute, err = g.generateSlurmNode(kv, nodeKey, deploymentID)
			var computeName = nodeName + "-" + instanceName
			log.Debugf("XBD computeName : IN FOR  %s", computeName)
			log.Debugf("XBD instanceName: IN FOR  %s", instanceName)
			if err != nil {
				return false, nil, err
			}

			addResource(&infrastructure, "slurm_node", computeName, &compute)

			consulKey := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address"), Value: fmt.Sprintf("${slurm_node.%s.node_name}", computeName)}
			consulKey2 := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/node_name"), Value: fmt.Sprintf("${slurm_node.%s.node_name}", computeName)}
			consulKeyAttrib := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/ip_address"), Value: fmt.Sprintf("${slurm_node.%s.node_name}", computeName)}
			consulKeyJobID := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/job_id"), Value: fmt.Sprintf("${slurm_node.%s.job_id}", computeName)}
			consulKeyCudaVisibleDevices := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/cuda_visible_devices"), Value: fmt.Sprintf("${slurm_node.%s.cuda_visible_devices}", computeName)}

			var consulKeys commons.ConsulKeys
			consulKeys = commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey, consulKey2, consulKeyAttrib, consulKeyJobID, consulKeyCudaVisibleDevices}}
			addResource(&infrastructure, "consul_keys", computeName, &consulKeys)

		} //End instances loop

	case "janus.nodes.slurm.Cntk":
		var instances []string
		instances, err = deployments.GetNodeInstancesIds(kv, deploymentID, nodeName)
		if err != nil {
			return false, nil, err
		}
		log.Debugf("-----------------------")
		log.Debugf("%+v\n", instances)
		log.Debugf("-----------------------")

		for _, instanceName := range instances {
			var instanceState tosca.NodeState
			instanceState, err = deployments.GetInstanceState(kv, deploymentID, nodeName, instanceName)
			if err != nil {
				return false, nil, err
			}
			if instanceState == tosca.NodeStateDeleting || instanceState == tosca.NodeStateDeleted {
				// Do not generate something for this node instance (will be deleted if exists)
				continue
			}
			job, err := g.generateSlurmCntk(kv, nodeKey, deploymentID)
			var jobName = nodeName + "-" + instanceName
			log.Debugf("XBD JobName : IN FOR  %s", jobName)
			log.Debugf("XBD instanceName: IN FOR  %s", instanceName)
			if err != nil {
				return false, nil, err
			}

			addResource(&infrastructure, "slurm_cntk", jobName, &job)

			consulKeyFinalError := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/final_error"), Value: fmt.Sprintf("${slurm_cntk.%s.finalError}", jobName)}
			consulKeyJobID := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/job_id"), Value: fmt.Sprintf("${slurm_cntk.%s.job_id}", jobName)}
			var consulKeys commons.ConsulKeys
			consulKeys = commons.ConsulKeys{Keys: []commons.ConsulKey{consulKeyFinalError, consulKeyJobID}}
			addResource(&infrastructure, "consul_keys", jobName, &consulKeys)

		} //End instances loop

	default:
		return false, nil, errors.Errorf("In Slurm : Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, deploymentID)
	}

	jsonInfra, err := json.MarshalIndent(infrastructure, "", "  ")
	if err != nil {
		return false, nil, errors.Wrap(err, "Failed to generate JSON of terraform Infrastructure description")
	}
	infraPath := filepath.Join(cfg.WorkingDirectory, "deployments", fmt.Sprint(deploymentID), "infra", nodeName)
	if err = os.MkdirAll(infraPath, 0775); err != nil {
		log.Printf("%+v", err)
		return false, nil, err
	}

	if err = ioutil.WriteFile(filepath.Join(infraPath, "infra.tf.json"), jsonInfra, 0664); err != nil {
		log.Print("Failed to write file")
		return false, nil, err
	}

	log.Printf("Infrastructure generated for deployment with id %s", deploymentID)
	return true, nil, nil
}
