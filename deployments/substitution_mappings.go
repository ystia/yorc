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

package deployments

import (
	"context"
	"path"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/tosca"
)

const (
	// DirectiveSubstitutable is a directive to the Orchestrator that a node
	// type is substitutable, ie. if this node type is a reference to another
	// topology template providing substitution mappings
	DirectiveSubstitutable = "substitutable"
)

// IsSubstitutableNode returns true if a node contains an Orchestrator directive
// that it is substitutable
func IsSubstitutableNode(kv *api.KV, deploymentID, nodeName string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "directives"), nil)
	if err != nil {
		return false, errors.Wrapf(err, "Can't get directives for node %q", nodeName)
	}

	substitutable := false
	if kvp != nil && kvp.Value != nil {
		values := strings.Split(string(kvp.Value), ",")
		for _, value := range values {
			if value == DirectiveSubstitutable {
				substitutable = true
				break
			}
		}
	}
	return substitutable, nil
}

func storeSubstitutionMappings(ctx context.Context, topology tosca.Topology, topologyPrefix string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	substitutionPrefix := path.Join(topologyPrefix, "substitution_mappings")
	substitution := topology.TopologyTemplate.SubstitionMappings
	if substitution != nil {
		consulStore.StoreConsulKeyAsString(path.Join(substitutionPrefix, "node_type"),
			substitution.NodeType)
		storePropAttrMappings(consulStore, path.Join(substitutionPrefix, "properties"),
			substitution.Properties)
		storePropAttrMappings(consulStore, path.Join(substitutionPrefix, "attributes"),
			substitution.Attributes)
		storeCapReqMappings(consulStore, path.Join(substitutionPrefix, "capabilities"),
			substitution.Capabilities)
		storeCapReqMappings(consulStore, path.Join(substitutionPrefix, "requirements"),
			substitution.Requirements)
		storeInterfaceMappings(consulStore, path.Join(substitutionPrefix, "interfaces"),
			substitution.Interfaces)
	}
}

func storePropAttrMappings(
	consulStore consulutil.ConsulStore,
	prefix string,
	mappings map[string]tosca.PropAttrMapping) {

	if mappings != nil {

		for name, propAttrMapping := range mappings {

			propAttrPrefix := path.Join(prefix, name)
			if propAttrMapping.Mapping != nil {
				consulStore.StoreConsulKeyAsString(
					path.Join(propAttrPrefix, "mapping"),
					strings.Join(propAttrMapping.Mapping, ","))
			} else {
				storeValueAssignment(consulStore, path.Join(propAttrPrefix, "value"), propAttrMapping.Value)
			}
		}
	}
}

func storeCapReqMappings(
	consulStore consulutil.ConsulStore,
	prefix string,
	mappings map[string]tosca.CapReqMapping) {

	if mappings != nil {

		for name, capReqMapping := range mappings {

			capReqPrefix := path.Join(prefix, name)

			if capReqMapping.Mapping != nil {
				consulStore.StoreConsulKeyAsString(
					path.Join(capReqPrefix, "mapping"),
					strings.Join(capReqMapping.Mapping, ","))
			}

			if capReqMapping.Properties != nil {
				propPrefix := path.Join(capReqPrefix, "properties")
				for name, value := range capReqMapping.Properties {
					storeValueAssignment(
						consulStore, path.Join(propPrefix, name),
						value)
				}
			}

			if capReqMapping.Attributes != nil {
				attrPrefix := path.Join(capReqPrefix, "attributes")
				for name, value := range capReqMapping.Attributes {
					storeValueAssignment(
						consulStore, path.Join(attrPrefix, name),
						value)
				}
			}
		}
	}
}

func storeInterfaceMappings(
	consulStore consulutil.ConsulStore,
	prefix string,
	mappings map[string]tosca.InterfaceMapping) {

	if mappings != nil {

		for name, itfMapping := range mappings {

			itfPrefix := path.Join(prefix, name)
			if itfMapping != nil {
				for k, v := range itfMapping {
					consulStore.StoreConsulKeyAsString(
						path.Join(itfPrefix, k), v)

				}
			}
		}
	}
}

func getSubstitutionMappingFromStore(kv *api.KV, prefix string) (tosca.SubstitutionMapping, error) {
	substitutionPrefix := path.Join(prefix, "substitution_mappings")
	var substitutionMapping tosca.SubstitutionMapping

	kvp, _, err := kv.Get(path.Join(substitutionPrefix, "node_type"), nil)
	if err != nil {
		return substitutionMapping,
			errors.Wrapf(err, "Can't get node type for substitution at %q", substitutionPrefix)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return substitutionMapping,
			errors.Errorf("Missing mandatory value for %q for substitution at %q",
				"node_type", substitutionPrefix)
	}
	substitutionMapping.NodeType = string(kvp.Value)
	substitutionMapping.Capabilities, err = getCapReqMappingFromStore(
		kv, path.Join(substitutionPrefix, "capabilities"))
	if err != nil {
		return substitutionMapping, err
	}

	// TODO: get other values
	return substitutionMapping, nil
}

func getCapReqMappingFromStore(kv *api.KV, prefix string) (map[string]tosca.CapReqMapping, error) {
	var result map[string]tosca.CapReqMapping
	return result, nil
}
