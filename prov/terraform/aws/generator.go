package aws

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

const infrastructureName = "aws"

type osGenerator struct {
}

func (g *osGenerator) getStringFormConsul(kv *api.KV, baseURL, property string) (string, error) {
	getResult, _, err := kv.Get(baseURL+"/"+property, nil)
	if err != nil {
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

func addOutput(infrastructure *commons.Infrastructure, outputName string, output *commons.Output) {
	if infrastructure.Output == nil {
		infrastructure.Output = make(map[string]*commons.Output)
	}
	infrastructure.Output[outputName] = output
}

func (g *osGenerator) GenerateTerraformInfraForNode(ctx context.Context, cfg config.Configuration, deploymentID, nodeName string) (bool, map[string]string, error) {
	log.Debugf("Generating infrastructure for deployment with id %s", deploymentID)
	cClient, err := cfg.GetConsulClient()
	if err != nil {
		return false, nil, err
	}
	kv := cClient.KV()
	nodeKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	terraformStateKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "terraform-state", nodeName)

	infrastructure := commons.Infrastructure{}

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
		"aws": map[string]interface{}{
			"access_key": cfg.Infrastructures[infrastructureName].GetString("access_key"),
			"secret_key": cfg.Infrastructures[infrastructureName].GetString("secret_key"),
			"region":     cfg.Infrastructures[infrastructureName].GetString("region"),
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
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, nil, err
	}
	outputs := make(map[string]string)
	var instances []string
	switch nodeType {
	case "janus.nodes.aws.Compute":
		instances, err = deployments.GetNodeInstancesIds(kv, deploymentID, nodeName)
		if err != nil {
			return false, nil, err
		}

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
			err = g.generateOSInstance(ctx, kv, cfg, deploymentID, nodeName, instanceName, &infrastructure, outputs)
			if err != nil {
				return false, nil, err
			}
		}
	default:
		return false, nil, errors.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, deploymentID)
	}

	jsonInfra, err := json.MarshalIndent(infrastructure, "", "  ")
	if err != nil {
		return false, nil, errors.Wrap(err, "Failed to generate JSON of terraform Infrastructure description")
	}
	infraPath := filepath.Join(cfg.WorkingDirectory, "deployments", fmt.Sprint(deploymentID), "infra", nodeName)
	if err = os.MkdirAll(infraPath, 0775); err != nil {
		return false, nil, errors.Wrapf(err, "Failed to create infrastructure working directory %q", infraPath)
	}

	if err = ioutil.WriteFile(filepath.Join(infraPath, "infra.tf.json"), jsonInfra, 0664); err != nil {
		return false, nil, errors.Wrapf(err, "Failed to write file %q", filepath.Join(infraPath, "infra.tf.json"))
	}

	log.Debugf("Infrastructure generated for deployment with id %s", deploymentID)
	return true, outputs, nil
}
