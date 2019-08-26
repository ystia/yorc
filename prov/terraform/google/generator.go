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

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/prov/terraform/commons"
	"github.com/ystia/yorc/v3/tosca"
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
	terraformStateKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "terraform-state", nodeName)

	infrastructure := commons.Infrastructure{}

	// Remote Configuration for Terraform State to store it in the Consul KV store
	infrastructure.Terraform = commons.GetBackendConfiguration(terraformStateKey, cfg)

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
		"consul": commons.GetConsulProviderfiguration(cfg),
		"null": map[string]interface{}{
			"version": commons.NullPluginVersionConstraint,
		},
	}

	log.Debugf("inspecting node %s", nodeName)
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, nil, nil, nil, err
	}
	outputs := make(map[string]string)
	instances, err := deployments.GetNodeInstancesIds(kv, deploymentID, nodeName)
	if err != nil {
		return false, nil, nil, nil, err
	}

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

			err := g.generateComputeInstance(ctx, kv, cfg, deploymentID, nodeName, instanceName, instNb, &infrastructure, outputs, &cmdEnv)
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

	jsonInfra, err := json.MarshalIndent(infrastructure, "", "  ")
	if err != nil {
		return false, nil, nil, nil, errors.Wrap(err, "Failed to generate JSON of terraform Infrastructure description")
	}

	if err = ioutil.WriteFile(filepath.Join(infrastructurePath, "infra.tf.json"), jsonInfra, 0664); err != nil {
		return false, nil, nil, nil, errors.Wrapf(err, "Failed to write file %q", filepath.Join(infrastructurePath, "infra.tf.json"))
	}

	log.Debugf("Infrastructure generated for deployment with id %s", deploymentID)
	return true, outputs, cmdEnv, nil, nil
}
