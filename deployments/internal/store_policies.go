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
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
	"path"
)

// storePolicies stores topology policies
func storePolicies(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, topologyPrefix string) {
	nodesPrefix := path.Join(topologyPrefix, "policies")
	for _, policyMap := range topology.TopologyTemplate.Policies {
		for policyName, policy := range policyMap {
			nodePrefix := nodesPrefix + "/" + policyName

			storage.GetStore(types.StoreTypeDeployment).Set(nodePrefix, policy)

			//consulStore.StoreConsulKeyAsString(nodePrefix+"/type", policy.Type)

			//if policy.Targets != nil {
			//	targetPrefix := nodePrefix + "/targets"
			//	consulStore.StoreConsulKeyAsString(targetPrefix, strings.Join(policy.Targets, ","))
			//}
			storeMapValueAssignment(consulStore, path.Join(nodePrefix, "properties"), policy.Properties)
			//metadataPrefix := nodePrefix + "/metadata/"
			//for metaName, metaValue := range policy.Metadata {
			//	consulStore.StoreConsulKeyAsString(metadataPrefix+metaName, metaValue)
			//}
		}
	}
}
