// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/helper/collections"
)

// ResolveAttributeMapping allows to resolve an attribute mapping
// i.e update the corresponding attribute with the defined value
// parameters can be nested keys and/or attribute name in case of capability
func ResolveAttributeMapping(ctx context.Context, deploymentID, nodeName, instanceName, capabilityOrAttributeName string, attributeValue interface{}, parameters ...string) error {
	// Check if node has an attribute with name
	attrs, err := GetNodeAttributesNames(ctx, deploymentID, nodeName)
	if err != nil {
		return err
	}

	if collections.ContainsString(attrs, capabilityOrAttributeName) {
		// It's an attribute: get the complex value and update it
		res, err := GetInstanceAttributeValue(ctx, deploymentID, nodeName, instanceName, capabilityOrAttributeName, parameters...)
		if err != nil {
			return err
		}
		if res != nil && res.Value != nil {
			updated, err := updateNestedValue(res.Value, attributeValue, parameters...)
			if err != nil {
				return err
			}
			return SetInstanceAttributeComplex(ctx, deploymentID, nodeName, instanceName, capabilityOrAttributeName, updated)
		}

		// If it's a literal, just set the instance attribute
		if len(parameters) == 0 {
			return SetInstanceAttributeComplex(ctx, deploymentID, nodeName, instanceName, capabilityOrAttributeName, attributeValue)
		}
		return errors.Errorf("failed to set instance attribute %q with deploymentID:%q, nodeName:%q, instance:%q, nested keys:%v", capabilityOrAttributeName, deploymentID, nodeName, instanceName, parameters)
	}

	// At this point, it's a capability name, so we ensure we have the attribute name in parameters
	if len(parameters) < 1 {
		return errors.Errorf("an attribute name is missing in corresponding nested keys:%v", parameters)
	}

	// It's a capability attribute: get the complex value and update it
	res, err := GetInstanceCapabilityAttributeValue(ctx, deploymentID, nodeName, instanceName, capabilityOrAttributeName, parameters[0], parameters[1:]...)
	if err != nil {
		return err
	}
	if res != nil && res.Value != nil {
		updated, err := updateNestedValue(res.Value, attributeValue, parameters[1:]...)
		if err != nil {
			return err
		}
		return SetInstanceCapabilityAttributeComplex(ctx, deploymentID, nodeName, instanceName, capabilityOrAttributeName, parameters[0], updated)
	}

	// If it's a literal, just set the instance attribute
	if len(parameters) == 1 {
		return SetInstanceCapabilityAttributeComplex(ctx, deploymentID, nodeName, instanceName, capabilityOrAttributeName, parameters[0], attributeValue)
	}
	return errors.Errorf("failed to set instance capability %q attribute %q with deploymentID:%q, nodeName:%q, instance:%q, nested keys:%v", capabilityOrAttributeName, parameters[0], deploymentID, nodeName, instanceName, parameters)
}
