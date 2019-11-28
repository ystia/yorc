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
func StoreAllTypes(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix, importPath string) error {
	storeDataTypes(ctx, consulStore, topology, topologyPrefix, importPath)
	if err := storeNodeTypes(ctx, consulStore, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	if err := storeRelationshipTypes(ctx, consulStore, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	if err := storeCapabilityTypes(ctx, consulStore, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	if err := storeArtifactTypes(ctx, consulStore, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	if err := storePolicyTypes(ctx, consulStore, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	return nil
}

// storePolicyTypes stores topology policy types
func storePolicyTypes(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix, importPath string) error {
	for policyName, policyType := range topology.PolicyTypes {
		key := path.Join(topologyPrefix, "types", policyName)
		policyType.ImportPath = importPath
		policyType.Base = "policy"
		err := storage.GetStore(types.StoreTypeDeployment).Set(key, policyType)
		if err != nil {
			return err
		}
	}
	return nil
}

// storeDataTypes store data types
func storeDataTypes(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix, importPath string) error {
	dataTypesPrefix := path.Join(topologyPrefix, "types")
	for dataTypeName, dataType := range topology.DataTypes {
		dtPrefix := path.Join(dataTypesPrefix, dataTypeName)
		dataType.ImportPath = importPath
		dataType.Base = "data"
		err := storage.GetStore(types.StoreTypeDeployment).Set(dtPrefix, dataType)
		if err != nil {
			return err
		}
	}

	return nil
}

// storeNodeTypes stores topology types
func storeNodeTypes(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix, importPath string) error {
	typesPrefix := path.Join(topologyPrefix, "types")
	for nodeTypeName, nodeType := range topology.NodeTypes {
		nodeTypePrefix := typesPrefix + "/" + nodeTypeName
		nodeType.ImportPath = importPath
		nodeType.Base = "node"
		err := storage.GetStore(types.StoreTypeDeployment).Set(nodeTypePrefix, nodeType)
		if err != nil {
			return err
		}
	}
	return nil
}

// storeRelationshipTypes stores topology relationships types
func storeRelationshipTypes(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix, importPath string) error {
	for relationName, relationType := range topology.RelationshipTypes {
		relationTypePrefix := path.Join(topologyPrefix, "types", relationName)
		relationType.ImportPath = importPath
		relationType.Base = "relationship"
		err := storage.GetStore(types.StoreTypeDeployment).Set(relationTypePrefix, relationType)
		if err != nil {
			return err
		}
	}
	return nil
}

// storeCapabilityTypes stores topology capabilities types
func storeCapabilityTypes(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix, importPath string) error {
	for capabilityTypeName, capabilityType := range topology.CapabilityTypes {
		capabilityTypePrefix := path.Join(topologyPrefix, "types", capabilityTypeName)
		capabilityType.ImportPath = importPath
		capabilityType.Base = "capability"
		err := storage.GetStore(types.StoreTypeDeployment).Set(capabilityTypePrefix, capabilityType)
		if err != nil {
			return err
		}
	}
	return nil
}

// storeArtifactTypes stores topology artifacts types
func storeArtifactTypes(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix, importPath string) error {
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
		err := storage.GetStore(types.StoreTypeDeployment).Set(artTypePrefix, artType)
		if err != nil {
			return err
		}
	}
	return nil
}
