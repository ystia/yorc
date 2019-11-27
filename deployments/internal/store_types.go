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
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"net/url"
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
		// Check operation outputs on attributes
		for _, attributeDef := range nodeType.Attributes {
			err := addOperationOutput(attributeDef.Default, nodeType.Interfaces)
			if err != nil {
				return err
			}
		}
		// Check operation outputs on interfaces
		for _, interfaceDef := range nodeType.Interfaces {
			for _, operationDef := range interfaceDef.Operations {
				for _, inputDef := range operationDef.Inputs {
					err := addOperationOutput(inputDef.ValueAssign, nodeType.Interfaces)
					if err != nil {
						return err
					}
				}
			}
		}

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

func addOperationOutput(va *tosca.ValueAssignment, interfaces map[string]tosca.InterfaceDefinition) error {
	if va == nil || va.Type != tosca.ValueAssignmentFunction {
		return nil
	}
	f := va.GetFunction()
	if f != nil {
		opOutputFuncs := f.GetFunctionsByOperator(tosca.GetOperationOutputOperator)
		for _, oof := range opOutputFuncs {
			if len(oof.Operands) != 4 {
				return errors.Errorf("Invalid %q TOSCA function: %v", tosca.GetOperationOutputOperator, oof)
			}
			entityName := url.QueryEscape(oof.Operands[0].String())

			// Only SELF operation output are handled here
			// HOST, SOURCE and TARGET operation outputs are managed next in fixGetOperationOutputForHost/fixGetOperationOutputForRelationship
			if entityName == "SELF" {
				interfaceName := oof.Operands[1].String()
				operationName := oof.Operands[2].String()
				outputName := oof.Operands[3].String()

				output := tosca.Output{ValueAssign: &tosca.ValueAssignment{
					Type:  tosca.ValueAssignmentFunction,
					Value: oof.String(),
				}}

				_, exist := interfaces[interfaceName]
				if !exist {
					ops := make(map[string]tosca.OperationDefinition)
					interfaces[interfaceName] = tosca.InterfaceDefinition{Operations: ops}
				}

				if interfaces[interfaceName].Operations == nil {
					//ops := make(map[string]tosca.OperationDefinition)
					//interfaces[interfaceName].Operations = ops
				}

				op, exist := interfaces[interfaceName].Operations[operationName]
				if !exist {

				}
				if op.Outputs == nil {
					op.Outputs = make(map[string]tosca.Output)
				}
				op.Outputs[outputName] = output
				interfaces[interfaceName].Operations[operationName] = op
			}
		}
	}
	return nil
}

func storeSelfOperationOutputsOnInterfaces(ctx context.Context, interfaceDefs map[string]tosca.InterfaceDefinition) error {
	for _, interfaceDef := range interfaceDefs {
		for _, operationDef := range interfaceDef.Operations {
			for _, inputDef := range operationDef.Inputs {
				if inputDef.ValueAssign == nil || inputDef.ValueAssign.Type != tosca.ValueAssignmentFunction {
					continue
				}
				f := inputDef.ValueAssign.GetFunction()
				if f != nil {
					opOutputFuncs := f.GetFunctionsByOperator(tosca.GetOperationOutputOperator)
					for _, oof := range opOutputFuncs {
						if len(oof.Operands) != 4 {
							return errors.Errorf("Invalid %q TOSCA function: %v", tosca.GetOperationOutputOperator, oof)
						}
						entityName := url.QueryEscape(oof.Operands[0].String())

						// Only SELF operation output are handled here
						// HOST, SOURCE and TARGET operation outputs are managed next in fixGetOperationOutputForHost/fixGetOperationOutputForRelationship
						if entityName == "SELF" {
							interfaceName := oof.Operands[1].String()
							operationName := oof.Operands[2].String()
							outputName := oof.Operands[3].String()

							output := tosca.Output{ValueAssign: &tosca.ValueAssignment{
								Type:  tosca.ValueAssignmentFunction,
								Value: oof.String(),
							}}

							op := interfaceDefs[interfaceName].Operations[operationName]
							if op.Outputs == nil {
								op.Outputs = make(map[string]tosca.Output)
							}
							op.Outputs[outputName] = output
							interfaceDefs[interfaceName].Operations[operationName] = op
						}
					}
				}
			}
		}
	}
	return nil
}

// storeRelationshipTypes stores topology relationships types
func storeRelationshipTypes(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix, importPath string) error {
	for relationName, relationType := range topology.RelationshipTypes {
		// Check operation outputs on interfaces
		for _, interfaceDef := range relationType.Interfaces {
			for _, operationDef := range interfaceDef.Operations {
				for _, inputDef := range operationDef.Inputs {
					err := addOperationOutput(inputDef.ValueAssign, relationType.Interfaces)
					if err != nil {
						return err
					}
				}
			}
		}
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
