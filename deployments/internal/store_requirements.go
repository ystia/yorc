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
	"path"
	"strconv"

	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/tosca"
)

// StoreRequirementAssignment stores a TOSCA RequirementAssignment under a given prefix
func StoreRequirementAssignment(consulStore consulutil.ConsulStore, requirement tosca.RequirementAssignment, requirementPrefix, requirementName string) {
	consulStore.StoreConsulKeyAsString(requirementPrefix+"/name", requirementName)
	consulStore.StoreConsulKeyAsString(requirementPrefix+"/node", requirement.Node)
	consulStore.StoreConsulKeyAsString(requirementPrefix+"/relationship", requirement.Relationship)
	consulStore.StoreConsulKeyAsString(requirementPrefix+"/capability", requirement.Capability)
	consulStore.StoreConsulKeyAsString(requirementPrefix+"/type_requirement", requirement.TypeRequirement)
	storeMapValueAssignment(consulStore, path.Join(requirementPrefix, "properties"), requirement.RelationshipProps)
}

func storeRequirementDefinition(consulStore consulutil.ConsulStore, reqDefinition tosca.RequirementDefinition, reqName, reqPrefix string) {
	consulStore.StoreConsulKeyAsString(reqPrefix+"/name", reqName)
	consulStore.StoreConsulKeyAsString(reqPrefix+"/node", reqDefinition.Node)
	consulStore.StoreConsulKeyAsString(reqPrefix+"/occurrences/lower_bound", strconv.FormatUint(reqDefinition.Occurrences.LowerBound, 10))
	consulStore.StoreConsulKeyAsString(reqPrefix+"/occurrences/upper_bound", strconv.FormatUint(reqDefinition.Occurrences.UpperBound, 10))
	consulStore.StoreConsulKeyAsString(reqPrefix+"/relationship", reqDefinition.Relationship)
	consulStore.StoreConsulKeyAsString(reqPrefix+"/capability", reqDefinition.Capability)
	consulStore.StoreConsulKeyAsString(reqPrefix+"/capability_name", reqDefinition.CapabilityName)
}

func storeRequirementDefinitionMap(consulStore consulutil.ConsulStore, requirements []tosca.RequirementDefinitionMap, requirementsPrefix string) {
	for reqIndex, reqMap := range requirements {
		for reqName, reqDefinition := range reqMap {
			storeRequirementDefinition(consulStore, reqDefinition, reqName, path.Join(requirementsPrefix, strconv.Itoa(reqIndex)))
		}
	}
}
