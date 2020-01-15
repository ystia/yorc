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
	"github.com/ystia/yorc/v4/config"
	"golang.org/x/sync/errgroup"
	"path"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
)

// UpgradeTo110 allows to upgrade Consul schema from 1.0.0 to 1.1.0
func UpgradeTo110(cfg config.Configuration, kv *api.KV, leaderch <-chan struct{}) error {
	log.Print("Upgrading to database version 1.1.0...")
	keys, _, err := kv.Keys(consulutil.DeploymentKVPrefix+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	ctx := context.Background()
	errGroup, ctx := errgroup.WithContext(ctx)
	for _, deploymentPrefix := range keys {
		topologyPrefix := path.Join(deploymentPrefix, "topology")
		errGroup.Go(func() error {
			if err = deleteElementsFromConsulPrefix(kv, topologyPrefix, "description", "tosca_version"); err != nil {
				return err
			}
			if err = deleteElementsFromAllSubPathsOfPrefix(kv, path.Join(topologyPrefix, "imports"), "description", "tosca_version"); err != nil {
				return err
			}
			if err = deleteElementsFromAllSubPathsOfPrefix(kv, path.Join(topologyPrefix, "repositories"), "description"); err != nil {
				return err
			}
			if err = deleteElementsFromAllSubPathsOfPrefix(kv, path.Join(topologyPrefix, "outputs"), "description", "name"); err != nil {
				return err
			}
			if err = deleteElementsFromAllSubPathsOfPrefix(kv, path.Join(topologyPrefix, "inputs"), "description", "name"); err != nil {
				return err
			}
			if err = up110CleanupTypes(kv, topologyPrefix); err != nil {
				return err
			}
			if err = up110CleanupNodes(kv, topologyPrefix); err != nil {
				return err
			}
			return nil
		})
	}

	return errGroup.Wait()
}

func up110CleanupTypes(kv *api.KV, topoPrefix string) error {
	types, _, err := kv.Keys(path.Join(topoPrefix, "types")+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, typePrefix := range types {

		if err = deleteElementsFromConsulPrefix(kv, typePrefix, "description", "name", "version"); err != nil {
			return err
		}

		if err = deleteElementsFromAllSubPathsOfPrefix(kv, path.Join(typePrefix, "artifacts"), "description", "name"); err != nil {
			return err
		}

		if err = deleteElementsFromAllSubPathsOfPrefix(kv, path.Join(typePrefix, "attributes"), "description", "name"); err != nil {
			return err
		}
		if err = deleteElementsFromAllSubPathsOfPrefix(kv, path.Join(typePrefix, "properties"), "description", "name"); err != nil {
			return err
		}

		if err = deleteElementsFromAllSubPathsOfPrefix(kv, path.Join(typePrefix, "capabilities"), "description", "name"); err != nil {
			return err
		}

		if err = up110CleanupInterfaces(kv, typePrefix); err != nil {
			return err
		}

	}
	return nil
}

func up110CleanupNodes(kv *api.KV, topoPrefix string) error {
	nodes, _, err := kv.Keys(path.Join(topoPrefix, "nodes")+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, nodePrefix := range nodes {

		if err = deleteElementsFromConsulPrefix(kv, nodePrefix, "description", "name"); err != nil {
			return err
		}
		if err = deleteElementsFromAllSubPathsOfPrefix(kv, path.Join(nodePrefix, "artifacts"), "description", "name"); err != nil {
			return err
		}

	}
	return nil
}

func up110CleanupInterfaces(kv *api.KV, elementPrefix string) error {
	interfaces, _, err := kv.Keys(path.Join(elementPrefix, "interfaces")+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, interfacePrefix := range interfaces {

		// Global inputs
		if err = deleteElementsFromAllSubPathsOfPrefix(kv, path.Join(interfacePrefix, "inputs"), "description", "name"); err != nil {
			return err
		}

		operations, _, err := kv.Keys(interfacePrefix+"/", "/", nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}

		for _, opPrefix := range operations {
			if strings.HasSuffix(opPrefix, "inputs") {
				continue
			}

			// operation inputs
			if err = deleteElementsFromAllSubPathsOfPrefix(kv, path.Join(opPrefix, "inputs"), "description", "name"); err != nil {
				return err
			}

			if err = deleteElementsFromConsulPrefix(kv, opPrefix, "description", "name", "implementation/description"); err != nil {
				return err
			}

		}
	}
	return nil
}
