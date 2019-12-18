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
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tosca"
	"strconv"
	"strings"
)

func getValueAssignment(ctx context.Context, deploymentID, nodeName, instanceName, requirementIndex, baseDataType string, va *tosca.ValueAssignment, nestedKeys ...string) (*TOSCAValue, error) {
	value, isFunction, err := getValueAssignmentWithoutResolve(ctx, deploymentID, va, baseDataType, nestedKeys...)
	if err != nil || value == nil || !isFunction {
		return value, err
	}
	return resolveValueAssignment(ctx, deploymentID, nodeName, instanceName, requirementIndex, value, nestedKeys...)

}

func getNestedValue(value interface{}, nestedKeys ...string) interface{} {
	if len(nestedKeys) > 0 {
		for _, nk := range nestedKeys {
			switch v := value.(type) {
			case []interface{}:
				ind, err := strconv.Atoi(nk)
				// Check the slice index is valid
				if err != nil {
					log.Printf("[ERROR] %q is not a valid array index", nk)
					return nil
				}
				if ind+1 > len(v) {
					log.Printf("[ERROR] %q: index not found", nestedKeys[0])
					return nil
				}
				value = v[ind]
			case map[string]interface{}:
				value = v[nk]
			}
		}
	}

	return value
}

func readComplexVA(ctx context.Context, deploymentID string, value interface{}, baseDataType string, nestedKeys ...string) (interface{}, error) {
	result := getNestedValue(value, nestedKeys...)
	err := checkForDefaultValuesInComplexTypes(ctx, deploymentID, baseDataType, "", result, nestedKeys...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func checkForDefaultValuesInComplexTypes(ctx context.Context, deploymentID string, baseDataType, currentDataType string, values interface{}, nestedKeys ...string) error {
	var err error
	if values == nil || baseDataType == "" {
		return nil
	}
	if currentDataType == "" {
		currentDataType, err = GetNestedDataType(ctx, deploymentID, baseDataType, nestedKeys...)
		if err != nil {
			return err
		}
	}

	entrySchema := getDataTypeComplexType(currentDataType)
	switch entrySchema {
	case "list":
		propResult, ok := values.([]interface{})
		if ok {
			for _, v := range propResult {
				item, ok := v.(map[string]interface{})
				if ok {
					err = addTypeDefault(ctx, deploymentID, baseDataType, currentDataType, item, nestedKeys...)
					if err != nil {
						return err
					}
				}
			}
		}
	case "map":
		propResult, ok := values.(map[string]interface{})
		if ok {
			for _, v := range propResult {
				item, ok := v.(map[string]interface{})
				if ok {
					err = addTypeDefault(ctx, deploymentID, baseDataType, currentDataType, item, nestedKeys...)
					if err != nil {
						return err
					}
				}
			}
		}
	default:
		// Check if result can be casted in map
		propResult, ok := values.(map[string]interface{})
		if ok {
			return addTypeDefault(ctx, deploymentID, baseDataType, currentDataType, propResult, nestedKeys...)
		}
	}
	return nil
}

func addTypeDefault(ctx context.Context, deploymentID string, baseDataType, currentDataType string, values map[string]interface{}, nestedKeys ...string) error {
	for currentDataType != "" {
		currentDataType = getDataTypeComplexEntrySchema(currentDataType)
		dtProps, err := GetTypeProperties(ctx, deploymentID, currentDataType, false)
		if err != nil {
			return err
		}
		for _, prop := range dtProps {
			if _, ok := values[prop]; !ok {
				// not found check default
				result, _, err := getTypeDefaultProperty(ctx, deploymentID, currentDataType, prop)
				if err != nil {
					return err
				}
				if result != nil {
					// TODO: should we get the /fmt.Stringer/ wrapped value or the original value?????
					values[prop] = result.Value
				}
			} else {
				// Check recursively through complex types
				typeProp, err := GetTypePropertyDataType(ctx, deploymentID, currentDataType, prop)
				if err != nil {
					return err
				}
				keys := append(nestedKeys, prop)
				err = checkForDefaultValuesInComplexTypes(ctx, deploymentID, baseDataType, typeProp, values[prop], keys...)
				if err != nil {
					return err
				}
			}
		}
		currentDataType, err = GetParentType(ctx, deploymentID, currentDataType)
		if err != nil {
			return err
		}
	}
	return nil
}

func getValueAssignmentWithoutResolve(ctx context.Context, deploymentID string, va *tosca.ValueAssignment, baseDataType string, nestedKeys ...string) (*TOSCAValue, bool, error) {
	if va != nil && va.Value != nil {
		switch va.Type {
		case tosca.ValueAssignmentFunction:
			return &TOSCAValue{Value: va.Value}, true, nil
		case tosca.ValueAssignmentLiteral:
			return &TOSCAValue{Value: fmt.Sprintf("%v", va.Value)}, false, nil
		case tosca.ValueAssignmentList, tosca.ValueAssignmentMap:
			res, err := readComplexVA(ctx, deploymentID, va.Value, baseDataType, nestedKeys...)
			if err != nil || res == nil {
				return nil, false, err
			}
			return &TOSCAValue{Value: stringifyValue(res)}, false, nil
		}
	}

	// Check default value in data type
	if baseDataType != "" && !strings.HasPrefix(baseDataType, "list:") && !strings.HasPrefix(baseDataType, "map:") && len(nestedKeys) > 0 {
		result, isFunc, err := getTypeDefaultProperty(ctx, deploymentID, baseDataType, nestedKeys[0], nestedKeys[1:]...)
		if err != nil || result != nil {
			return result, isFunc, err
		}
	}

	// not found
	return nil, false, nil
}

func getInstanceValueAssignment(ctx context.Context, deploymentID, nodeName, instanceName, requirementIndex, baseDataType, vaPath string, nestedKeys ...string) (*TOSCAValue, error) {
	kvp, _, err := consulutil.GetKV().Get(vaPath, nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	var va *tosca.ValueAssignment
	// Build a Value assignment from JSON and determine if it's literal, map or list
	if kvp != nil {
		s := string(kvp.Value)
		if isQuoted(s) {
			s, err = strconv.Unquote(s)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to unquote value assignment:%q", s)
			}
		}

		// let's try to unmarshall value into list, map, otherwise keep it as a string
		var m map[string]interface{}
		err := json.Unmarshal([]byte(s), &m)
		if err == nil {
			va = &tosca.ValueAssignment{Type: tosca.ValueAssignmentMap, Value: m}
		}

		if va == nil {
			var l []interface{}
			err = json.Unmarshal([]byte(s), &l)
			if err == nil {
				va = &tosca.ValueAssignment{Type: tosca.ValueAssignmentMap, Value: l}
			}
		}

		if va == nil {
			va = &tosca.ValueAssignment{Type: tosca.ValueAssignmentLiteral, Value: s}
		}
	}
	return getValueAssignment(ctx, deploymentID, nodeName, instanceName, requirementIndex, baseDataType, va, nestedKeys...)
}

func stringifyValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = stringifyValue(val)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = stringifyValue(val)
		}
		return result
	default:
		result := fmt.Sprintf("%v", v)
		return result
	}

}
