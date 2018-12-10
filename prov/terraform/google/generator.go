// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/helper/sshutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov/terraform/commons"
	"github.com/ystia/yorc/tosca"
)

const infrastructureName = "google"

type googleGenerator struct {
}

func (g *googleGenerator) GenerateTerraformInfraForNode(ctx context.Context, cfg config.Configuration, deploymentID, nodeName, infrastructurePath string) (bool, map[string]string, []string, commons.PostApplyCallback, error) {
	log.Debugf("Generating infrastructure for deployment with id %s", deploymentID)

	cClient, err := cfg.GetConsulClient()
	if err != nil {
		return false, nil, nil, nil, err
	}
	kv := cClient.KV()
	nodeKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	terraformStateKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "terraform-state", nodeName)

	infrastructure := commons.Infrastructure{}

	consulAddress := "127.0.0.1:8500"
	if cfg.Consul.Address != "" {
		consulAddress = cfg.Consul.Address
	}
	consulScheme := "http"
	if cfg.Consul.SSL {
		consulScheme = "https"
	}
	consulCA := ""
	if cfg.Consul.CA != "" {
		consulCA = cfg.Consul.CA
	}
	consulKey := ""
	if cfg.Consul.Key != "" {
		consulKey = cfg.Consul.Key
	}
	consulCert := ""
	if cfg.Consul.Cert != "" {
		consulCert = cfg.Consul.Cert
	}

	// Remote Configuration for Terraform State to store it in the Consul KV store
	infrastructure.Terraform = map[string]interface{}{
		"backend": map[string]interface{}{
			"consul": map[string]interface{}{
				"path": terraformStateKey,
			},
		},
	}

	// Define Terraform provider environment variables
	var cmdEnv []string
	configParams := []string{"application_credentials", "credentials", "project", "region"}
	for _, configParam := range configParams {
		value := cfg.Infrastructures[infrastructureName].GetString(configParam)
		if value != "" {
			cmdEnv = append(cmdEnv,
				fmt.Sprintf("%s=%s",
					"GOOGLE_"+strings.ToUpper(configParam),
					value))
		}
	}

	// Management of variables for Terraform
	infrastructure.Provider = map[string]interface{}{
		"google": map[string]interface{}{
			"version": cfg.Terraform.GooglePluginVersionConstraint,
		},
		"consul": map[string]interface{}{
			"version":   cfg.Terraform.ConsulPluginVersionConstraint,
			"address":   consulAddress,
			"scheme":    consulScheme,
			"ca_file":   consulCA,
			"cert_file": consulCert,
			"key_file":  consulKey,
		},
		"null": map[string]interface{}{
			"version": commons.NullPluginVersionConstraint,
		},
	}

	log.Debugf("inspecting node %s", nodeKey)
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, nil, nil, nil, err
	}
	outputs := make(map[string]string)
	instances, err := deployments.GetNodeInstancesIds(kv, deploymentID, nodeName)
	if err != nil {
		return false, nil, nil, nil, err
	}

	var sshAgent *sshutil.SSHAgent

	for instNb, instanceName := range instances {
		instanceState, err := deployments.GetInstanceState(kv, deploymentID, nodeName, instanceName)
		if err != nil {
			return false, nil, nil, nil, err
		}
		if instanceState == tosca.NodeStateDeleting || instanceState == tosca.NodeStateDeleted {
			// Do not generate something for this node instance (will be deleted if exists)
			continue
		}

		switch nodeType {
		case "yorc.nodes.google.Compute":
			instances, err = deployments.GetNodeInstancesIds(kv, deploymentID, nodeName)
			if err != nil {
				return false, nil, nil, nil, err
			}

			err = g.generateComputeInstance(ctx, kv, cfg, deploymentID, nodeName, instanceName, instNb, &infrastructure, outputs, &cmdEnv, sshAgent)
			if err != nil {
				return false, nil, nil, nil, err
			}
		case "yorc.nodes.google.Address":
			err = g.generateComputeAddress(ctx, kv, cfg, deploymentID, nodeName, instanceName, instNb, &infrastructure, outputs)
			if err != nil {
				return false, nil, nil, nil, err
			}
		case "yorc.nodes.google.PersistentDisk":
			err = g.generatePersistentDisk(ctx, kv, cfg, deploymentID, nodeName, instanceName, instNb, &infrastructure, outputs)
			if err != nil {
				return false, nil, nil, nil, err
			}
		case "yorc.nodes.google.PrivateNetwork":
			err = g.generatePrivateNetwork(ctx, kv, cfg, deploymentID, nodeName, &infrastructure, outputs)
			if err != nil {
				return false, nil, nil, nil, err
			}
		case "yorc.nodes.google.Subnetwork":
			err = g.generateSubNetwork(ctx, kv, cfg, deploymentID, nodeName, &infrastructure, outputs)
			if err != nil {
				return false, nil, nil, nil, err
			}
		default:
			return false, nil, nil, nil, errors.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, deploymentID)
		}

	}

	// If ssh-agent has been created, it needs to be stopped after the infrastructure creation
	// This is done with this callback
	var postInstallCb func()
	if sshAgent != nil {
		postInstallCb = func() {
			// Stop the sshAgent if used during provisioning
			// Do not return any error if failure occured during this
			err := sshAgent.RemoveAllKeys()
			if err != nil {
				log.Debugf("Warning: failed to remove all SSH agents keys due to error:%+v", err)
			}
			err = sshAgent.Stop()
			if err != nil {
				log.Debugf("Warning: failed to stop SSH agent due to error:%+v", err)
			}
		}
	}

	jsonInfra, err := json.MarshalIndent(infrastructure, "", "  ")
	if err != nil {
		return false, nil, nil, nil, errors.Wrap(err, "Failed to generate JSON of terraform Infrastructure description")
	}

	if err = ioutil.WriteFile(filepath.Join(infrastructurePath, "infra.tf.json"), jsonInfra, 0664); err != nil {
		return false, nil, nil, nil, errors.Wrapf(err, "Failed to write file %q", filepath.Join(infrastructurePath, "infra.tf.json"))
	}

	log.Debugf("Infrastructure generated for deployment with id %s", deploymentID)
	return true, outputs, cmdEnv, postInstallCb, nil
}
