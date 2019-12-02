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
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tosca"
	"net/url"
	"strconv"
	"strings"
)

func urlEscapeAll(keys []string) []string {
	t := make([]string, len(keys))
	for i, k := range keys {
		t[i] = url.QueryEscape(k)
	}
	return t
}

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
				var ok bool
				value, ok = v[nk]
				if !ok {
					// Not found
					log.Printf("[ERROR] %q: index not found", nestedKeys[0])
					return nil
				}
			}
		}
	}

	return value
}

func readComplexVA(ctx context.Context, deploymentID string, va *tosca.ValueAssignment, baseDataType string, nestedKeys ...string) (interface{}, error) {
	var result interface{}
	if len(nestedKeys) == 0 {
		result = va.Value
	} else {
		switch va.Type {
		case tosca.ValueAssignmentMap:
			m, ok := va.Value.(map[string]interface{})
			if ok {
				// Check the key exists
				_, ok := m[nestedKeys[0]]
				if !ok {
					// Not found
					log.Printf("[ERROR] %q: index not found", nestedKeys[0])
					return nil, nil
				}
				// result is a  nested value
				result = getNestedValue(m[nestedKeys[0]], nestedKeys[1:]...)
			}
		case tosca.ValueAssignmentList:
			l, ok := va.Value.([]interface{})
			if ok {
				ind, err := strconv.Atoi(nestedKeys[0])
				// Check the slice index is valid
				if err != nil {
					log.Printf("[ERROR] %q is not a valid array index", nestedKeys[0])
					return nil, nil
				}
				if ind+1 > len(l) {
					log.Printf("[ERROR] %q: index not found", nestedKeys[0])
					return nil, nil
				}
				// result is a  nested value
				result = getNestedValue(l[ind], nestedKeys[1:]...)
			}
		case tosca.ValueAssignmentLiteral:
			result = va.Value.(string)
		}
	}


	//result = getNestedValue(va.Value, nestedKeys...)
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
	var entrySchema string
	if strings.HasPrefix(currentDataType, "list:") {
		entrySchema = "list"
	} else if strings.HasPrefix(currentDataType, "map:") {
		entrySchema = "map"
	}

	switch entrySchema {
	case "list":
		propResult, ok := values.([]interface{})
		if ok {
			for _,v := range propResult {
				item, ok := v.(map[string]interface{})
				if ok {
					addTypeDefault(ctx, deploymentID, baseDataType, currentDataType, item, nestedKeys...)
				}
			}
		}
	case "map":
		propResult, ok := values.(map[string]interface{})
		if ok {
			for _,v := range propResult {
				item, ok := v.(map[string]interface{})
				if ok {
					addTypeDefault(ctx, deploymentID, baseDataType,currentDataType,  item, nestedKeys...)
				}
			}

		}
	default:
		// Check if result can be casted in map
		propResult, ok := values.(map[string]interface{})
		if ok {
			addTypeDefault(ctx, deploymentID, baseDataType, currentDataType, propResult, nestedKeys...)
		}
	}
	return nil
}


func addTypeDefault(ctx context.Context, deploymentID string, baseDataType, currentDataType string, values map[string]interface{}, nestedKeys ...string) error {
	for currentDataType != "" {
		if strings.HasPrefix(currentDataType, "list:") {
			currentDataType = currentDataType[5:]
		} else if strings.HasPrefix(currentDataType, "map:") {
			currentDataType = currentDataType[4:]
		}
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
			return &TOSCAValue{Value: va.Value}, false, nil
		case tosca.ValueAssignmentList, tosca.ValueAssignmentMap:
			res, err := readComplexVA(ctx, deploymentID, va, baseDataType, nestedKeys...)
			if err != nil || res == nil  {
				return nil, false, err
			}
			return &TOSCAValue{Value: res}, false, nil
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

func getInstanceValueAssignment(ctx context.Context, vaPath string, nestedKeys ...string) (*TOSCAValue, error) {
	kvp, _, err := consulutil.GetKV().Get(vaPath, nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil {
		s := string(kvp.Value)
		if isQuoted(s) {
			s, err = strconv.Unquote(s)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to unquote value assignment:%q", s)
			}
		}

		// let's try to unmarshall value into list, map or simply return string
		var m map[string]interface{}
		err := json.Unmarshal([]byte(s), &m)
		if err == nil {
			return &TOSCAValue{Value: getNestedValue(m, nestedKeys...)}, nil
		}

		var l []interface{}
		err = json.Unmarshal([]byte(s), &l)
		if err == nil {
			return &TOSCAValue{Value: getNestedValue(l, nestedKeys...)}, nil
		}

		return &TOSCAValue{Value: s}, nil
	}
	return nil, nil
}
