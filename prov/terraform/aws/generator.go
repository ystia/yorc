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

package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
	"github.com/ystia/yorc/v4/tosca"
)

const infrastructureType = "aws"

type awsGenerator struct{}

type nodeParams struct {
	deploymentID   string
	infrastructure *commons.Infrastructure
	nodeName       string
}

func (g *awsGenerator) GenerateTerraformInfraForNode(ctx context.Context, cfg config.Configuration, deploymentID, nodeName, infrastructurePath string) (bool, map[string]string, []string, commons.PostApplyCallback, error) {
	log.Debugf("Generating infrastructure for deployment with id %s", deploymentID)

	terraformStateKey := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "terraform-state", nodeName)
	infrastructure := commons.Infrastructure{}

	nodeParams := &nodeParams{
		deploymentID:   deploymentID,
		nodeName:       nodeName,
		infrastructure: &infrastructure,
	}

	var locationProps config.DynamicMap
	locationMgr, err := locations.GetManager(cfg)
	if err == nil {
		locationProps, err = locationMgr.GetLocationPropertiesForNode(ctx, deploymentID, nodeName, infrastructureType)
	}
	if err != nil {
		return false, nil, nil, nil, err
	}

	// Remote Configuration for Terraform State to store it in the Consul KV store
	infrastructure.Terraform = commons.GetBackendConfiguration(terraformStateKey, cfg)

	cmdEnv := []string{
		fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", locationProps.GetString("access_key")),
		fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", locationProps.GetString("secret_key")),
	}
	// Management of variables for Terraform
	infrastructure.Provider = map[string]interface{}{
		"aws": map[string]interface{}{
			"region":  locationProps.GetString("region"),
			"version": cfg.Terraform.AWSPluginVersionConstraint,
		},
		"consul": commons.GetConsulProviderfiguration(cfg),
		"null": map[string]interface{}{
			"version": commons.NullPluginVersionConstraint,
		},
	}

	log.Debugf("inspecting node %s", nodeName)
	outputs := make(map[string]string)
	instances, err := deployments.GetNodeInstancesIds(ctx, deploymentID, nodeName)
	if err != nil {
		return false, nil, nil, nil, err
	}
	err = g.generateInstances(ctx, cfg, &instances, nodeParams, outputs, &cmdEnv)
	if err != nil {
		return false, nil, nil, nil, errors.Wrap(err, "Failed to generate instances")
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

func (g *awsGenerator) generateInstances(ctx context.Context, cfg config.Configuration, instances *[]string, nodeParams *nodeParams, outputs map[string]string, cmdEnv *[]string) error {
	nodeType, err := deployments.GetNodeType(ctx, nodeParams.deploymentID, nodeParams.nodeName)
	if err != nil {
		return err
	}

	for instNb, instanceName := range *instances {
		instanceState, err := deployments.GetInstanceState(ctx, nodeParams.deploymentID, nodeParams.nodeName, instanceName)
		if err != nil {
			return err
		}
		if instanceState == tosca.NodeStateDeleting || instanceState == tosca.NodeStateDeleted {
			// Do not generate something for this node instance (will be deleted if exists)
			continue
		}

		switch nodeType {
		case "yorc.nodes.aws.Compute":
			err = g.generateAWSInstance(ctx, cfg, nodeParams.deploymentID, nodeParams.nodeName, instanceName, nodeParams.infrastructure, outputs, cmdEnv)
			if err != nil {
				return err
			}
		case "yorc.nodes.aws.PublicNetwork":
			// Nothing to do
		case "yorc.nodes.aws.EBSVolume":
			err = g.generateEBS(ctx, *nodeParams, instanceName, instNb, outputs)
			if err != nil {
				return err
			}
		case "yorc.nodes.aws.VPC":
			err = g.generateVPC(ctx, *nodeParams, instanceName, outputs)
			if err != nil {
				return err
			}

		case "yorc.nodes.aws.Subnet":
			err = g.generateSubnet(ctx, *nodeParams, instanceName, outputs)
			if err != nil {
				return err
			}
		default:
			return errors.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeParams.nodeName, nodeParams.deploymentID)
		}
	}

	return nil
}
