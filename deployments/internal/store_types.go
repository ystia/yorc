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

package internal

import (
	"context"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"path"
	"strings"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/tosca"
)

// StoreAllTypes stores all types of a given topology
func StoreAllTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) error {
	storeDataTypes(ctx, topology, topologyPrefix, importPath)
	if err := storeNodeTypes(ctx, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	if err := storeRelationshipTypes(ctx, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	if err := storeCapabilityTypes(ctx, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	if err := storeArtifactTypes(ctx, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	if err := storePolicyTypes(ctx, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	return nil
}

// storePolicyTypes stores topology policy types
func storePolicyTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) error {
	kv := make([]*types.KeyValue, 0)
	for policyName, policyType := range topology.PolicyTypes {
		key := path.Join(topologyPrefix, "types", policyName)
		policyType.ImportPath = importPath
		policyType.Base = "policy"

		kv = append(kv, &types.KeyValue{
			Key:   key,
			Value: policyType,
		})
	}
	return storage.GetStore(types.StoreTypeDeployment).SetCollection(ctx, kv)
}

// storeDataTypes store data types
func storeDataTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) error {
	dataTypesPrefix := path.Join(topologyPrefix, "types")
	kv := make([]*types.KeyValue, 0)
	for dataTypeName, dataType := range topology.DataTypes {
		dtPrefix := path.Join(dataTypesPrefix, dataTypeName)
		dataType.ImportPath = importPath
		dataType.Base = "data"
		kv = append(kv, &types.KeyValue{
			Key:   dtPrefix,
			Value: dataType,
		})
	}

	return storage.GetStore(types.StoreTypeDeployment).SetCollection(ctx, kv)
}

// storeNodeTypes stores topology types
func storeNodeTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) error {
	kv := make([]*types.KeyValue, 0)
	typesPrefix := path.Join(topologyPrefix, "types")
	for nodeTypeName, nodeType := range topology.NodeTypes {
		nodeTypePrefix := typesPrefix + "/" + nodeTypeName
		nodeType.ImportPath = importPath
		nodeType.Base = "node"
		kv = append(kv, &types.KeyValue{
			Key:   nodeTypePrefix,
			Value: nodeType,
		})
	}
	return storage.GetStore(types.StoreTypeDeployment).SetCollection(ctx, kv)
}

// storeRelationshipTypes stores topology relationships types
func storeRelationshipTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) error {
	kv := make([]*types.KeyValue, 0)
	for relationName, relationType := range topology.RelationshipTypes {
		relationTypePrefix := path.Join(topologyPrefix, "types", relationName)
		relationType.ImportPath = importPath
		relationType.Base = "relationship"
		kv = append(kv, &types.KeyValue{
			Key:   relationTypePrefix,
			Value: relationType,
		})
	}
	return storage.GetStore(types.StoreTypeDeployment).SetCollection(ctx, kv)
}

// storeCapabilityTypes stores topology capabilities types
func storeCapabilityTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) error {
	kv := make([]*types.KeyValue, 0)
	for capabilityTypeName, capabilityType := range topology.CapabilityTypes {
		capabilityTypePrefix := path.Join(topologyPrefix, "types", capabilityTypeName)
		capabilityType.ImportPath = importPath
		capabilityType.Base = "capability"
		kv = append(kv, &types.KeyValue{
			Key:   capabilityTypePrefix,
			Value: capabilityType,
		})
	}
	return storage.GetStore(types.StoreTypeDeployment).SetCollection(ctx, kv)
}

// storeArtifactTypes stores topology artifacts types
func storeArtifactTypes(ctx context.Context, topology tosca.Topology, topologyPrefix, importPath string) error {
	kv := make([]*types.KeyValue, 0)
	typesPrefix := path.Join(topologyPrefix, "types")
	for artTypeName, artType := range topology.ArtifactTypes {
		// TODO(loicalbertin): remove it when migrating to Alien 2.2. Currently alien-base-types has org.alien4cloud.artifacts.AnsiblePlaybook types that do not derives from tosca.artifacts.Implementation
		// as with the change on commons types this types is not overridden by builtin types anymore. This is because we first check on deployments types
		// then on commons types.
		if !strings.HasPrefix(typesPrefix, consulutil.CommonsTypesKVPrefix) && artTypeName == "org.alien4cloud.artifacts.AnsiblePlaybook" {
			continue
		}

		artTypePrefix := path.Join(typesPrefix, artTypeName)
		artType.ImportPath = importPath
		artType.Base = "artifact"
		kv = append(kv, &types.KeyValue{
			Key:   artTypePrefix,
			Value: artType,
		})
	}
	return storage.GetStore(types.StoreTypeDeployment).SetCollection(ctx, kv)
}
