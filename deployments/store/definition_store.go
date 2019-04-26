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

package store

import (
	"context"
	"path"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v3/deployments/internal"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/tosca"
)

// BuiltinDefinition stores a TOSCA definition to the common place
func BuiltinDefinition(ctx context.Context, definitionName string, definitionContent []byte) error {
	topology := tosca.Topology{}
	err := yaml.Unmarshal(definitionContent, &topology)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal TOSCA definition %q", definitionName)
	}
	ctx, errGroup, consulStore := consulutil.WithContext(ctx)
	kv := consulutil.GetKV()
	name := topology.Metadata["template_name"]
	if name == "" {
		return errors.Errorf("Can't store builtin TOSCA definition %q, template_name is missing", definitionName)
	}
	version := topology.Metadata["template_version"]
	if version == "" {
		return errors.Errorf("Can't store builtin TOSCA definition %q, template_version is missing", definitionName)
	}
	topologyPrefix := path.Join(consulutil.BuiltinTypesKVPrefix, name, version)

	keys, _, err := kv.Keys(topologyPrefix+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if len(keys) > 0 {
		log.Printf("Do not storing existing topology definition: %q version: %q", name, version)
		return nil
	}
	errGroup.Go(func() error {
		internal.StoreTopologyTopLevelKeyNames(ctx, consulStore, topology, topologyPrefix)
		return nil
	})
	errGroup.Go(func() error {
		return internal.StoreRepositories(ctx, consulStore, topology, topologyPrefix)
	})
	errGroup.Go(func() error {
		return internal.StoreDataTypes(ctx, consulStore, topology, topologyPrefix, "")
	})
	errGroup.Go(func() error {
		return internal.StoreNodeTypes(ctx, consulStore, topology, topologyPrefix, "")
	})
	errGroup.Go(func() error {
		return internal.StoreRelationshipTypes(ctx, consulStore, topology, topologyPrefix, "")
	})
	errGroup.Go(func() error {
		internal.StoreCapabilityTypes(ctx, consulStore, topology, topologyPrefix, "")
		return nil
	})
	errGroup.Go(func() error {
		internal.StoreArtifactTypes(ctx, consulStore, topology, topologyPrefix, "")
		return nil
	})
	return errGroup.Wait()
}

// Deployment stores a whole deployment.
func Deployment(ctx context.Context, topology tosca.Topology, deploymentID, rootDefPath string) error {
	ctx, errGroup, consulStore := consulutil.WithContext(ctx)

	errGroup.Go(func() error {
		return internal.StoreTopology(ctx, consulStore, errGroup, topology, deploymentID, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology"), "", "", rootDefPath)
	})

	return errGroup.Wait()
}
