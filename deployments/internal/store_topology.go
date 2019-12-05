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
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
	"golang.org/x/sync/errgroup"
	"path"
)

// StoreTopology stores a given topology.
//
// The given topology may be an import in this case importPrefix and importPath should be specified
func StoreTopology(ctx context.Context, errGroup *errgroup.Group, topology tosca.Topology, deploymentID, topologyPrefix, importPrefix, importPath, rootDefPath string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	log.Debugf("Storing topology with name %q (Import prefix %q)", topology.Metadata[tosca.TemplateName], importPrefix)

	errGroup.Go(func() error {
		return errors.Wrapf(StoreTopologyTopLevelKeyNames(ctx, topology, path.Join(topologyPrefix, importPrefix)), "failed to store top level topology key names")
	})
	errGroup.Go(func() error {
		return errors.Wrapf(storeImports(ctx, errGroup, topology, deploymentID, topologyPrefix, importPath, rootDefPath), "failed to store topology imports")
	})
	errGroup.Go(func() error {
		return errors.Wrapf(StoreAllTypes(ctx, topology, topologyPrefix, importPath), "failed to store topology types")
	})
	errGroup.Go(func() error {
		return errors.Wrapf(StoreRepositories(ctx, topology, topologyPrefix), "failed to store topology repositories")
	})

	// There is no need to parse a topology template if this topology template
	// is declared in an import.
	// Parsing only the topology template declared in the root topology file
	isRootTopologyTemplate := importPrefix == ""
	if isRootTopologyTemplate {
		errGroup.Go(func() error {
			return errors.Wrapf(storeInputs(ctx, topology, topologyPrefix), "failed to store topology inputs")
		})
		errGroup.Go(func() error {
			return errors.Wrapf(storeOutputs(ctx, topology, topologyPrefix), "failed to store topology outputs")
		})
		errGroup.Go(func() error {
			return errors.Wrapf(storeSubstitutionMappings(ctx, topology, topologyPrefix), "failed to store substitution mapping")
		})
		errGroup.Go(func() error {
			return errors.Wrapf(storeNodes(ctx, topology, topologyPrefix, importPath, rootDefPath), "failed to store topology nodes")
		})
		errGroup.Go(func() error {
			return errors.Wrapf(storePolicies(ctx, topology, topologyPrefix), "failed to store topology policies")
		})
	} else {
		// For imported templates, storing substitution mappings if any
		// as they contain details on service to application/node type mapping
		errGroup.Go(func() error {
			return errors.Wrapf(storeSubstitutionMappings(ctx, topology, path.Join(topologyPrefix, importPrefix)), "failed to store substitution mapping")
		})
	}

	if isRootTopologyTemplate {
		errGroup.Go(func() error {
			// Detect potential cycles in inline workflows
			if err := checkNestedWorkflows(topology); err != nil {
				return err
			}
			return errors.Wrapf(storeWorkflows(ctx, topology, deploymentID), "failed to store workflows")
		})
	}
	return nil
}

// StoreTopologyTopLevelKeyNames stores top level keynames for a topology.
//
// This may be done under the import path in case of imports.
func StoreTopologyTopLevelKeyNames(ctx context.Context, topology tosca.Topology, topologyPrefix string) error {
	return storage.GetStore(types.StoreTypeDeployment).Set(ctx, topologyPrefix+"/metadata", topology.Metadata)
}

// storeOutputs stores topology outputs
func storeOutputs(ctx context.Context, topology tosca.Topology, topologyPrefix string) error {
	return storeParameterDefinition(ctx, path.Join(topologyPrefix, "outputs"), topology.TopologyTemplate.Outputs)
}

// storeInputs stores topology outputs
func storeInputs(ctx context.Context, topology tosca.Topology, topologyPrefix string) error {
	return storeParameterDefinition(ctx, path.Join(topologyPrefix, "inputs"), topology.TopologyTemplate.Inputs)
}

func storeParameterDefinition(ctx context.Context, paramsPrefix string, paramDefsMap map[string]tosca.ParameterDefinition) error {
	for paramName, paramDef := range paramDefsMap {
		paramDefPrefix := path.Join(paramsPrefix, paramName)
		err := storage.GetStore(types.StoreTypeDeployment).Set(ctx, paramDefPrefix, paramDef)
		if err != nil {
			return err
		}
	}
	return nil
}
