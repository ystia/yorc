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
	"github.com/ystia/yorc/v4/tosca"
	"path"
)

func storeSubstitutionMappings(ctx context.Context, topology tosca.Topology, topologyPrefix string) error {
	substitutionPrefix := path.Join(topologyPrefix, "substitution_mappings")
	substitution := topology.TopologyTemplate.SubstitionMappings
	if substitution != nil {
		return storage.GetStore(types.StoreTypeDeployment).Set(ctx, substitutionPrefix, substitution)
	}
	return nil
}
