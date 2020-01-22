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

package upgradeschema

import (
	"context"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/deployments/store"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/resources"
	"github.com/ystia/yorc/v4/tosca"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

// UpgradeTo120 allows to upgrade Consul schema from 1.1.1 to 1.2.0
func UpgradeTo120(cfg config.Configuration, kv *api.KV, leaderch <-chan struct{}) error {
	log.Print("Upgrading to database version 1.2.0")
	return upgradeDeploymentsRefactoring(cfg, "1.2.0")
}

func upgradeDeploymentsRefactoring(cfg config.Configuration, targetVersion string) error {
	log.Print("Upgrade deployments store refactoring...")

	deps, err := consulutil.GetKeys(consulutil.DeploymentKVPrefix)
	if err != nil {
		return err
	}

	err = upgradeCommonsTypes(cfg, targetVersion)
	if err != nil {
		return err
	}
	log.Debugf("Upgrade %s: Tosca resources commons types successfully upgraded", targetVersion)

	ctx := context.Background()
	sem := make(chan struct{}, cfg.UpgradeConcurrencyLimit)
	errGroup, ctx := errgroup.WithContext(ctx)
	for _, deployment := range deps {
		sem <- struct{}{}
		deploymentID := path.Base(deployment)
		errGroup.Go(func() error {
			defer func() {
				<-sem
			}()
			return upgradeDeployment(ctx, cfg, targetVersion, deploymentID)
		})
	}

	return errGroup.Wait()
}

func upgradeDeployment(ctx context.Context, cfg config.Configuration, targetVersion, deploymentID string) error {
	var err error
	log.Debugf("Upgrade %s: Handling deployment with deploymentID:%q", targetVersion, deploymentID)
	// Remove all previous tree keys for all deployments
	deploymentPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID)
	topologyPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology")
	trees := []string{
		path.Join(topologyPath, "nodes"),
		path.Join(topologyPath, "policies"),
		path.Join(topologyPath, "repositories"),
		path.Join(topologyPath, "substitution_mappings"),
		path.Join(topologyPath, "metadata"),
		path.Join(topologyPath, "outputs"),
		path.Join(topologyPath, "inputs"),
		path.Join(topologyPath, "types"),
		path.Join(topologyPath, "implementation_artifacts_extensions"),
		path.Join(topologyPath, "imports"),
		path.Join(deploymentPath, "workflows"),
	}

	for _, tree := range trees {
		log.Debugf("Upgrade %s:  Delete tree with path:%q", targetVersion, tree)
		err = consulutil.Delete(tree, true)
		if err != nil {
			return errors.Wrapf(err, "failed to delete tree with path:%q for deploymentID:%q", tree, deploymentID)
		}
	}
	log.Debugf("Upgrade %s: removal of existing topology successfully done for deploymentID:%q", targetVersion, deploymentID)
	// Store topology from original file
	err = storeTopologyInNewSchema(ctx, cfg, deploymentID, targetVersion)
	if err != nil {
		return err
	}
	log.Debugf("Upgrade %s: upgrade topology schema successfully done for deploymentID:%q", targetVersion, deploymentID)
	return nil
}

func upgradeCommonsTypes(cfg config.Configuration, targetVersion string) error {
	if cfg.ServerID == "testUpgrade120_skip_common_types" {
		return nil
	}
	log.Debugf("Upgrade %s: Delete and store Tosca resources in new schema", targetVersion)
	err := consulutil.Delete(consulutil.CommonsTypesKVPrefix, true)
	if err != nil {
		return err
	}
	err = resources.StoreBuiltinTOSCAResources()
	if err != nil {
		return errors.Wrapf(err, "Upgrade %s: failed to upgrade builtin Tosca resources", targetVersion)
	}
	return nil
}

func storeTopologyInNewSchema(ctx context.Context, cfg config.Configuration, deploymentID, targetVersion string) error {
	topologyFilePath, err := getTopologyFilePath(cfg.WorkingDirectory, deploymentID)
	if err != nil {
		return err
	}

	log.Debugf("Upgrade %s: Store topology in new schema for deployment:%q from file path:%q", targetVersion, deploymentID, topologyFilePath)
	topology := tosca.Topology{}
	definition, err := os.Open(topologyFilePath)
	if err != nil {
		return err
	}
	defBytes, err := ioutil.ReadAll(definition)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(defBytes, &topology)
	if err != nil {
		return err
	}

	err = store.Deployment(context.Background(), topology, deploymentID, filepath.Dir(topologyFilePath))
	if err != nil {
		return errors.Wrapf(err, "Upgrade %s: failed to store deployment in new schema for deploymentID:%q", targetVersion, deploymentID)
	}

	// Retrieve new stored nodes
	nodes, err := deployments.GetNodes(ctx, deploymentID)
	if err != nil {
		return err
	}

	err = deployments.PostDeploymentDefinitionStorageProcess(ctx, deploymentID, nodes)
	if err != nil {
		return errors.Wrapf(err, "Upgrade %s: failed to execute Post process deployment storage in new schema for deploymentID:%q", targetVersion, deploymentID)
	}
	return nil
}

func getTopologyFilePath(dir, deploymentID string) (string, error) {
	patterns := []struct {
		pattern string
	}{
		{"*.yml"},
		{"*.yaml"},
	}
	var yamlList []string
	var err error
	for _, pattern := range patterns {
		var yamls []string
		if yamls, err = filepath.Glob(filepath.Join(dir, "deployments", deploymentID, "overlay", pattern.pattern)); err != nil {
			return "", err
		}
		yamlList = append(yamlList, yamls...)
	}
	if len(yamlList) != 1 {
		return "", errors.Errorf("one and only one YAML (.yml or .yaml) file should be present at the root of archive for deployment %s", deploymentID)
	}
	return yamlList[0], nil
}
