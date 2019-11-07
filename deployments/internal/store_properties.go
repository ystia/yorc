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

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/tosca"
)

// storePropertyDefinition stores a property definition
func storePropertyDefinition(ctx context.Context, consulStore consulutil.ConsulStore, propPrefix, propName string, propDefinition tosca.PropertyDefinition) {
	if propDefinition.Required == nil {
		b := true
		// Required by default
		propDefinition.Required = &b
	}

	storage.GetStore(types.StoreTypeDeployment).Set(propPrefix, propDefinition)

	//consulStore.StoreConsulKeyAsString(propPrefix+"/type", propDefinition.Type)
	//consulStore.StoreConsulKeyAsString(propPrefix+"/entry_schema", propDefinition.EntrySchema.Type)
	//StoreValueAssignment(consulStore, propPrefix+"/default", propDefinition.Default)
	//if propDefinition.Required == nil {
	//	// Required by default
	//	consulStore.StoreConsulKeyAsString(propPrefix+"/required", "true")
	//} else {
	//	consulStore.StoreConsulKeyAsString(propPrefix+"/required", fmt.Sprint(*propDefinition.Required))
	//}
}
