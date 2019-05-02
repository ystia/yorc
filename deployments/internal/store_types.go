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
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/tosca"
)

// TypeExistsFlagName is the name of a Consul key that is used to prove the existence of a TOSCA type
//
// This key doesn't contain any value and allow to detect types even if there is no other values stored for them into consul
const TypeExistsFlagName = ".existFlag"

// StoreAllTypes stores all types of a given topology
func StoreAllTypes(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix, importPath string) error {
	storeDataTypes(ctx, consulStore, topology, topologyPrefix, importPath)
	if err := storeNodeTypes(ctx, consulStore, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	if err := storeRelationshipTypes(ctx, consulStore, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	storeCapabilityTypes(ctx, consulStore, topology, topologyPrefix, importPath)
	storeArtifactTypes(ctx, consulStore, topology, topologyPrefix, importPath)
	storePolicyTypes(ctx, consulStore, topology, topologyPrefix, importPath)
	return nil
}

func storeCommonType(consulStore consulutil.ConsulStore, commonType tosca.Type, typePrefix, importPath string) {
	consulStore.StoreConsulKeyWithFlags(path.Join(typePrefix, TypeExistsFlagName), nil, 0)
	// TODO(loicalbertin) Consul mem footprint optimizations: May be empty consider not always storing it, but be careful at least one key is required for now to check if type exist
	// may be problematic on normative types
	consulStore.StoreConsulKeyAsString(path.Join(typePrefix, "derived_from"), commonType.DerivedFrom)
	// TODO(loicalbertin) Consul mem footprint optimizations: May be empty consider not always storing it
	consulStore.StoreConsulKeyAsString(path.Join(typePrefix, "importPath"), importPath)
	for metaName, metaValue := range commonType.Metadata {
		consulStore.StoreConsulKeyAsString(path.Join(typePrefix, "metadata", metaName), metaValue)
	}
}

// storeDataTypes store data types
func storeDataTypes(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix, importPath string) error {
	dataTypesPrefix := path.Join(topologyPrefix, "types")
	for dataTypeName, dataType := range topology.DataTypes {
		dtPrefix := path.Join(dataTypesPrefix, dataTypeName)
		storeCommonType(consulStore, dataType.Type, dtPrefix, importPath)
		for propName, propDefinition := range dataType.Properties {
			storePropertyDefinition(ctx, consulStore, path.Join(dtPrefix, "properties", propName), propName, propDefinition)
		}
	}

	return nil
}

// storeNodeTypes stores topology types
func storeNodeTypes(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix, importPath string) error {
	typesPrefix := path.Join(topologyPrefix, "types")
	for nodeTypeName, nodeType := range topology.NodeTypes {
		nodeTypePrefix := typesPrefix + "/" + nodeTypeName
		storeCommonType(consulStore, nodeType.Type, nodeTypePrefix, importPath)
		propertiesPrefix := nodeTypePrefix + "/properties"
		for propName, propDefinition := range nodeType.Properties {
			propPrefix := propertiesPrefix + "/" + propName
			storePropertyDefinition(ctx, consulStore, propPrefix, propName, propDefinition)
		}

		requirementsPrefix := nodeTypePrefix + "/requirements"
		for reqIndex, reqMap := range nodeType.Requirements {
			for reqName, reqDefinition := range reqMap {
				reqPrefix := requirementsPrefix + "/" + strconv.Itoa(reqIndex)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/name", reqName)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/node", reqDefinition.Node)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/occurrences/lower_bound", strconv.FormatUint(reqDefinition.Occurrences.LowerBound, 10))
				consulStore.StoreConsulKeyAsString(reqPrefix+"/occurrences/upper_bound", strconv.FormatUint(reqDefinition.Occurrences.UpperBound, 10))
				consulStore.StoreConsulKeyAsString(reqPrefix+"/relationship", reqDefinition.Relationship)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/capability", reqDefinition.Capability)
				consulStore.StoreConsulKeyAsString(reqPrefix+"/capability_name", reqDefinition.CapabilityName)
			}
		}
		capabilitiesPrefix := nodeTypePrefix + "/capabilities"
		for capName, capability := range nodeType.Capabilities {
			capabilityPrefix := capabilitiesPrefix + "/" + capName

			consulStore.StoreConsulKeyAsString(capabilityPrefix+"/type", capability.Type)
			consulStore.StoreConsulKeyAsString(capabilityPrefix+"/occurrences/lower_bound", strconv.FormatUint(capability.Occurrences.LowerBound, 10))
			consulStore.StoreConsulKeyAsString(capabilityPrefix+"/occurrences/upper_bound", strconv.FormatUint(capability.Occurrences.UpperBound, 10))
			consulStore.StoreConsulKeyAsString(capabilityPrefix+"/valid_sources", strings.Join(capability.ValidSourceTypes, ","))

			storeMapValueAssignment(consulStore, path.Join(capabilityPrefix, "properties"), capability.Properties)
			storeMapValueAssignment(consulStore, path.Join(capabilityPrefix, "attributes"), capability.Attributes)
		}

		err := storeInterfaces(consulStore, nodeType.Interfaces, nodeTypePrefix, false)
		if err != nil {
			return err
		}
		attributesPrefix := nodeTypePrefix + "/attributes"
		for attrName, attrDefinition := range nodeType.Attributes {
			attrPrefix := attributesPrefix + "/" + attrName
			storeAttributeDefinition(ctx, consulStore, attrPrefix, attrName, attrDefinition)
			if attrDefinition.Default != nil && attrDefinition.Default.Type == tosca.ValueAssignmentFunction {
				err := storeOperationOutput(consulStore, attrDefinition.Default, path.Join(nodeTypePrefix, "interfaces"))
				if err != nil {
					return err
				}
			}
		}

		storeArtifacts(consulStore, nodeType.Artifacts, nodeTypePrefix+"/artifacts")

	}
	return nil
}

// storeRelationshipTypes stores topology relationships types
func storeRelationshipTypes(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix, importPath string) error {
	for relationName, relationType := range topology.RelationshipTypes {
		relationTypePrefix := path.Join(topologyPrefix, "types", relationName)
		storeCommonType(consulStore, relationType.Type, relationTypePrefix, importPath)
		propertiesPrefix := relationTypePrefix + "/properties"
		for propName, propDefinition := range relationType.Properties {
			propPrefix := propertiesPrefix + "/" + propName
			storePropertyDefinition(ctx, consulStore, propPrefix, propName, propDefinition)
		}
		attributesPrefix := relationTypePrefix + "/attributes"
		for attrName, attrDefinition := range relationType.Attributes {
			attrPrefix := attributesPrefix + "/" + attrName
			storeAttributeDefinition(ctx, consulStore, attrPrefix, attrName, attrDefinition)
			if attrDefinition.Default != nil && attrDefinition.Default.Type == tosca.ValueAssignmentFunction {
				f := attrDefinition.Default.GetFunction()
				opOutputFuncs := f.GetFunctionsByOperator(tosca.GetOperationOutputOperator)
				for _, oof := range opOutputFuncs {
					if len(oof.Operands) != 4 {
						return errors.Errorf("Invalid %q TOSCA function: %v", tosca.GetOperationOutputOperator, oof)
					}
					entityName := url.QueryEscape(oof.Operands[0].String())

					interfaceName := strings.ToLower(url.QueryEscape(oof.Operands[1].String()))
					operationName := strings.ToLower(url.QueryEscape(oof.Operands[2].String()))
					outputVariableName := url.QueryEscape(oof.Operands[3].String())
					consulStore.StoreConsulKeyAsString(relationTypePrefix+"/interfaces/"+interfaceName+"/"+operationName+"/outputs/"+entityName+"/"+outputVariableName+"/expression", oof.String())
				}
			}
		}

		err := storeInterfaces(consulStore, relationType.Interfaces, relationTypePrefix, true)
		if err != nil {
			return err
		}

		storeArtifacts(consulStore, relationType.Artifacts, relationTypePrefix+"/artifacts")

		consulStore.StoreConsulKeyAsString(relationTypePrefix+"/valid_target_type", strings.Join(relationType.ValidTargetTypes, ", "))

	}
	return nil
}

// storeCapabilityTypes stores topology capabilities types
func storeCapabilityTypes(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix, importPath string) {
	for capabilityTypeName, capabilityType := range topology.CapabilityTypes {
		capabilityTypePrefix := path.Join(topologyPrefix, "types", capabilityTypeName)
		storeCommonType(consulStore, capabilityType.Type, capabilityTypePrefix, importPath)
		propertiesPrefix := capabilityTypePrefix + "/properties"
		for propName, propDefinition := range capabilityType.Properties {
			propPrefix := propertiesPrefix + "/" + propName
			storePropertyDefinition(ctx, consulStore, propPrefix, propName, propDefinition)
		}
		attributesPrefix := capabilityTypePrefix + "/attributes"
		for attrName, attrDefinition := range capabilityType.Attributes {
			attrPrefix := attributesPrefix + "/" + attrName
			storeAttributeDefinition(ctx, consulStore, attrPrefix, attrName, attrDefinition)
		}
		consulStore.StoreConsulKeyAsString(capabilityTypePrefix+"/valid_source_types", strings.Join(capabilityType.ValidSourceTypes, ","))
	}
}

// storeArtifactTypes stores topology artifacts types
func storeArtifactTypes(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix, importPath string) {
	typesPrefix := path.Join(topologyPrefix, "types")
	for artTypeName, artType := range topology.ArtifactTypes {
		artTypePrefix := path.Join(typesPrefix, artTypeName)
		storeCommonType(consulStore, artType.Type, artTypePrefix, importPath)
		consulStore.StoreConsulKeyAsString(artTypePrefix+"/mime_type", artType.MimeType)
		consulStore.StoreConsulKeyAsString(artTypePrefix+"/file_ext", strings.Join(artType.FileExt, ","))
		propertiesPrefix := artTypePrefix + "/properties"
		for propName, propDefinition := range artType.Properties {
			propPrefix := propertiesPrefix + "/" + propName
			storePropertyDefinition(ctx, consulStore, propPrefix, propName, propDefinition)
		}
	}
}
