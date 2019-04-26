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
	"path"
	"strconv"

	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/tosca"
	"golang.org/x/sync/errgroup"
)

// StoreTopology stores a given topology.
//
// The given topology may be an import in this case importPrefix and importPath should be specified
func StoreTopology(ctx context.Context, consulStore consulutil.ConsulStore, errGroup *errgroup.Group, topology tosca.Topology, deploymentID, topologyPrefix, importPrefix, importPath, rootDefPath string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	log.Debugf("Storing topology with name %q (Import prefix %q)", topology.Metadata[tosca.TemplateName], importPrefix)
	StoreTopologyTopLevelKeyNames(ctx, consulStore, topology, path.Join(topologyPrefix, importPrefix))
	if err := storeImports(ctx, consulStore, errGroup, topology, deploymentID, topologyPrefix, importPath, rootDefPath); err != nil {
		return err
	}
	StoreRepositories(ctx, consulStore, topology, topologyPrefix)
	StoreDataTypes(ctx, consulStore, topology, topologyPrefix, importPath)

	// There is no need to parse a topology template if this topology template
	// is declared in an import.
	// Parsing only the topology template declared in the root topology file
	isRootTopologyTemplate := (importPrefix == "")
	if isRootTopologyTemplate {
		storeInputs(ctx, consulStore, topology, topologyPrefix)
		storeOutputs(ctx, consulStore, topology, topologyPrefix)
		storeSubstitutionMappings(ctx, consulStore, topology, topologyPrefix)
		storeNodes(ctx, consulStore, topology, topologyPrefix, importPath, rootDefPath)
	} else {
		// For imported templates, storing substitution mappings if any
		// as they contain details on service to application/node type mapping
		storeSubstitutionMappings(ctx, consulStore, topology,
			path.Join(topologyPrefix, importPrefix))
	}

	if err := StoreNodeTypes(ctx, consulStore, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	if err := StoreRelationshipTypes(ctx, consulStore, topology, topologyPrefix, importPath); err != nil {
		return err
	}
	StoreCapabilityTypes(ctx, consulStore, topology, topologyPrefix, importPath)
	StoreArtifactTypes(ctx, consulStore, topology, topologyPrefix, importPath)

	// Detect potential cycles in inline workflows
	if err := checkNestedWorkflows(topology); err != nil {
		return err
	}

	if isRootTopologyTemplate {
		storeWorkflows(ctx, consulStore, topology, deploymentID)
	}
	return nil
}

// StoreTopologyTopLevelKeyNames stores top level keynames for a topology.
//
// This may be done under the import path in case of imports.
func StoreTopologyTopLevelKeyNames(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix string) {
	storeStringMap(consulStore, topologyPrefix+"/metadata", topology.Metadata)
}

// storeOutputs stores topology outputs
func storeOutputs(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix string) {
	outputsPrefix := path.Join(topologyPrefix, "outputs")
	for outputName, output := range topology.TopologyTemplate.Outputs {
		outputPrefix := path.Join(outputsPrefix, outputName)
		StoreValueAssignment(consulStore, path.Join(outputPrefix, "default"), output.Default)
		if output.Required == nil {
			// Required by default
			consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "required"), "true")
		} else {
			consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "required"), strconv.FormatBool(*output.Required))
		}
		consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "status"), output.Status)
		consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "type"), output.Type)
		consulStore.StoreConsulKeyAsString(path.Join(outputPrefix, "entry_schema"), output.EntrySchema.Type)
		StoreValueAssignment(consulStore, path.Join(outputPrefix, "value"), output.Value)
	}
}

// storeInputs stores topology outputs
func storeInputs(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix string) {
	inputsPrefix := path.Join(topologyPrefix, "inputs")
	for inputName, input := range topology.TopologyTemplate.Inputs {
		inputPrefix := path.Join(inputsPrefix, inputName)
		StoreValueAssignment(consulStore, path.Join(inputPrefix, "default"), input.Default)
		if input.Required == nil {
			// Required by default
			consulStore.StoreConsulKeyAsString(path.Join(inputPrefix, "required"), "true")
		} else {
			consulStore.StoreConsulKeyAsString(path.Join(inputPrefix, "required"), strconv.FormatBool(*input.Required))
		}
		consulStore.StoreConsulKeyAsString(path.Join(inputPrefix, "status"), input.Status)
		consulStore.StoreConsulKeyAsString(path.Join(inputPrefix, "type"), input.Type)
		consulStore.StoreConsulKeyAsString(path.Join(inputPrefix, "entry_schema"), input.EntrySchema.Type)
		StoreValueAssignment(consulStore, path.Join(inputPrefix, "value"), input.Value)
	}
}
