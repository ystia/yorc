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

package operations

import (
	"context"

	"github.com/hashicorp/consul/api"

	"github.com/ystia/yorc/v4/prov"
)

// GetTargetCapabilityPropertiesAndAttributes retrieves properties and attributes of the target capability of the relationship (if this operation is related to a relationship)
//
// It may happen in rare cases that several capabilities match the same requirement.
// Values are stored in this way:
//   * TARGET_CAPABILITY_NAMES: comma-separated list of matching capabilities names. It could be use to loop over the injected variables
//   * TARGET_CAPABILITY_<capabilityName>_TYPE: actual type of the capability
//   * TARGET_CAPABILITY_TYPE: actual type of the capability of the first matching capability
// 	 * TARGET_CAPABILITY_<capabilityName>_PROPERTY_<propertyName>: value of a property
// 	 * TARGET_CAPABILITY_PROPERTY_<propertyName>: value of a property for the first matching capability
// 	 * TARGET_CAPABILITY_<capabilityName>_<instanceName>_ATTRIBUTE_<attributeName>: value of an attribute of a given instance
// 	 * TARGET_CAPABILITY_<instanceName>_ATTRIBUTE_<attributeName>: value of an attribute of a given instance for the first matching capability
//
// Deprecated: use GetTargetCapabilityPropertiesAndAttributesValues instead
//             will be removed in Yorc 4.0
//             This function is unsafe as it loose the context of the value to know if it is a secret or not.
func GetTargetCapabilityPropertiesAndAttributes(ctx context.Context, kv *api.KV, deploymentID, nodeName string, op prov.Operation) (map[string]string, error) {
	m, err := GetTargetCapabilityPropertiesAndAttributesValues(ctx, kv, deploymentID, nodeName, op)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(m))

	for k, v := range m {
		result[k] = v.RawString()
	}

	return result, nil
}
