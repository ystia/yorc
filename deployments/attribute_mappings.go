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
	"strconv"
)

// ResolveAttributeMapping allows to resolve an attribute mapping
// i.e update the corresponding attribute with the defined value
// parameters can be nested keys and/or attribute name in case of capability
func ResolveAttributeMapping(ctx context.Context, deploymentID, nodeName, instanceName, capabilityOrAttributeName string, attributeValue interface{}, parameters ...string) error {
	// Check if node has an attribute with corresponding name
	attrs, err := GetNodeAttributesNames(ctx, deploymentID, nodeName)
	if err != nil {
		return err
	}

	if collections.ContainsString(attrs, capabilityOrAttributeName) {
		// It's an attribute: get the complex value and update it
		return resolveInstanceAttributeMapping(ctx, deploymentID, nodeName, instanceName, capabilityOrAttributeName, attributeValue, parameters...)
	}

	// At this point, it's a capability name, so we ensure we have the attribute name in parameters
	if len(parameters) < 1 {
		return errors.Errorf("attribute name is missing in parameters for resolving attribute mapping for deploymentId:%q, node name:%q, instance name:%q, capability name:%q", deploymentID, nodeName, instanceName, capabilityOrAttributeName)
	}

	// It's a capability attribute: get the complex value and update it
	return resolveCapabilityAttributeMapping(ctx, deploymentID, nodeName, instanceName, capabilityOrAttributeName, parameters[0], attributeValue, parameters[1:]...)
}

func resolveInstanceAttributeMapping(ctx context.Context, deploymentID, nodeName, instanceName, attributeName string, attributeValue interface{}, nestedKeys ...string) error {
	// The simplest case
	if len(nestedKeys) == 0 {
		return SetInstanceAttributeComplex(ctx, deploymentID, nodeName, instanceName, attributeName, attributeValue)
	}

	var value interface{}
	// Get existing or default value
	res, err := GetInstanceAttributeValue(ctx, deploymentID, nodeName, instanceName, attributeName)
	if err != nil {
		return err
	}

	if res != nil && res.Value == nil {
		value = res.Value
	} else {
		nodeType, err := GetNodeType(ctx, deploymentID, nodeName)
		if err != nil {
			return err
		}

		attrDataType, err := GetTypeAttributeDataType(ctx, deploymentID, nodeType, attributeName)
		if err != nil {
			return err
		}

		// Build complex value from scratch
		value, err = buildComplexValue(ctx, deploymentID, attrDataType, nestedKeys...)
		if err != nil {
			return err
		}
	}

	updated, err := updateComplexValue(value, attributeValue, nestedKeys...)
	if err != nil {
		return err
	}
	return SetInstanceAttributeComplex(ctx, deploymentID, nodeName, instanceName, attributeName, updated)
}

func resolveCapabilityAttributeMapping(ctx context.Context, deploymentID, nodeName, instanceName, capabilityName, attributeName string, attributeValue interface{}, nestedKeys ...string) error {
	// The simplest case
	if len(nestedKeys) == 0 {
		return SetInstanceCapabilityAttributeComplex(ctx, deploymentID, nodeName, instanceName, capabilityName, attributeName, attributeValue)
	}

	var value interface{}
	// Get existing or default value
	res, err := GetInstanceCapabilityAttributeValue(ctx, deploymentID, nodeName, instanceName, capabilityName, attributeName)
	if err != nil {
		return err
	}

	if res != nil && res.Value != nil {
		value = res.Value
	} else {
		// Build value from scratch
		nodeType, err := GetNodeCapabilityType(ctx, deploymentID, nodeName, capabilityName)
		if err != nil {
			return err
		}

		attrDataType, err := GetTypeAttributeDataType(ctx, deploymentID, nodeType, attributeName)
		if err != nil {
			return err
		}

		value, err = buildComplexValue(ctx, deploymentID, attrDataType, nestedKeys...)
		if err != nil {
			return err
		}
	}

	updated, err := updateComplexValue(value, attributeValue, nestedKeys...)
	if err != nil {
		return err
	}
	return SetInstanceCapabilityAttributeComplex(ctx, deploymentID, nodeName, instanceName, capabilityName, attributeName, updated)
}

func buildComplexValue(ctx context.Context, deploymentID, baseDataType string, nestedKeys ...string) (interface{}, error) {
	if len(nestedKeys) == 0 {
		return nil, errors.Errorf("missing nested keys for building complex value for deploymentID:%q, base data type:%q", deploymentID, baseDataType)
	}
	var complexVal interface{}
	dType := getDataTypeComplexType(baseDataType)
	switch dType {
	case "list":
		ind, err := strconv.Atoi(nestedKeys[0])
		if err != nil {
			return nil, errors.Errorf("%q is not a valid array index", nestedKeys[0])
		}
		complexVal = makeSlice("", ind+1)
	default:
		complexVal = make(map[string]interface{}, 0)
	}

	tmp := complexVal
	var ind int
	var nestedValue interface{}
	for i := 0; i < len(nestedKeys); i++ {

		dataType, err := GetNestedDataType(ctx, deploymentID, baseDataType, nestedKeys[:i+1]...)
		if err != nil || dataType == "" {
			return complexVal, err
		}
		if isPrimitiveDataType(dataType) {
			continue
		}

		dType = getDataTypeComplexType(dataType)
		switch dType {
		case "list":
			ind, err = strconv.Atoi(nestedKeys[i+1])
			if err != nil {
				return nil, errors.Errorf("%q is not a valid array index", nestedKeys[i])
			}
			nestedValue = makeSlice("", ind+1)
		default:
			nestedValue = make(map[string]interface{}, 0)
		}

		switch v := tmp.(type) {
		case []interface{}:
			v[ind] = nestedValue
			tmp = nestedValue
		case map[string]interface{}:
			v[nestedKeys[i]] = nestedValue
			tmp = v[nestedKeys[i]]
		}
	}
	return complexVal, nil
}

func makeSlice(def interface{}, size int) []interface{} {
	s := make([]interface{}, size)
	for i := 0; i < len(s); i++ {
		s[i] = def
	}
	return s
}

func updateComplexValue(complexValue, nestedValueToUpdate interface{}, nestedKeys ...string) (interface{}, error) {
	if len(nestedKeys) == 0 {
		return nestedValueToUpdate, nil
	}

	tmp := complexValue
	for i := 0; i < len(nestedKeys); i++ {
		nk := nestedKeys[i]
		switch v := tmp.(type) {
		case []interface{}:
			ind, err := strconv.Atoi(nk)
			// Check the slice index is valid
			if err != nil {
				return nil, errors.Errorf("%q is not a valid array index", nk)
			}
			if ind+1 > len(v) {
				return nil, errors.Errorf("%q: index not found", ind)
			}

			if i != len(nestedKeys)-1 {
				tmp = v[ind]
				continue
			}

			// set the value
			v[ind] = nestedValueToUpdate
		case map[string]interface{}:
			if i != len(nestedKeys)-1 {
				tmp = v[nk]
				continue
			}

			// set the value
			v[nk] = nestedValueToUpdate
		}
	}
	return complexValue, nil
}
