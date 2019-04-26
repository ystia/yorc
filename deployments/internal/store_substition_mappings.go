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
	"strings"

	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/tosca"
)

func storeSubstitutionMappings(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix string) {
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
				StoreValueAssignment(consulStore, path.Join(propAttrPrefix, "value"), propAttrMapping.Value)
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
					StoreValueAssignment(
						consulStore, path.Join(propPrefix, name),
						value)
				}
			}

			if capReqMapping.Attributes != nil {
				attrPrefix := path.Join(capReqPrefix, "attributes")
				for name, value := range capReqMapping.Attributes {
					StoreValueAssignment(
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
	mappings map[string]string) {

	if mappings != nil {
		for operationName, workflowName := range mappings {
			consulStore.StoreConsulKeyAsString(path.Join(prefix, operationName), workflowName)
		}
	}
}
