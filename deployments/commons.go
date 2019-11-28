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
	"path"
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

func readNestedValue(value interface{}, nestedKeys ...string) interface{} {
	for _, nk := range nestedKeys {
		switch v := value.(type) {
		case []interface{}:
			ind, err := strconv.Atoi(nk)
			// Check the slice index is valid
			if err != nil {
				log.Printf("[ERROR] %q is not a valid array index", nk)
				return nil
			}
			value = v[ind]
		case map[string]interface{}:
			value = v[nk]
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
				// Check the value is a va and needs to be retrieved
				v, ok := m[nestedKeys[0]].(tosca.ValueAssignment)
				if ok {
					return readComplexVA(ctx, deploymentID, &v, baseDataType, nestedKeys[1:]...)
				}
				// result is a  nested value
				result = readNestedValue(m[nestedKeys[0]], nestedKeys[1:]...)
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
				// Check the value is a va and needs to be retrieved
				v, ok := l[ind].(tosca.ValueAssignment)
				if ok {
					return readComplexVA(ctx, deploymentID, &v, baseDataType, nestedKeys[1:]...)
				}
				// result is a  nested value
				result = readNestedValue(l[ind], nestedKeys[1:]...)
			}
		}
	}

	if result != nil && baseDataType != "" && baseDataType != "map:string" && va.Type == tosca.ValueAssignmentMap {
		currentDatatype, err := GetNestedDataType(ctx, deploymentID, baseDataType, nestedKeys...)
		if err != nil {
			return nil, err
		}
		castedResult, ok := result.(map[string]interface{})
		if ok {
			for currentDatatype != "" {
				dtProps, err := GetTypeProperties(ctx, deploymentID, currentDatatype, false)
				if err != nil {
					return nil, err
				}
				for _, prop := range dtProps {
					if _, ok := castedResult[prop]; !ok {
						// not found check default
						result, _, err := getTypeDefaultProperty(ctx, deploymentID, currentDatatype, prop)
						if err != nil {
							return nil, err
						}
						if result != nil {
							// TODO: should we get the /fmt.Stringer/ wrapped value or the original value?????
							castedResult[prop] = result.Value
						}
					}
				}
				currentDatatype, err = GetParentType(ctx, deploymentID, currentDatatype)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return result, nil
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
			if err != nil {
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
	keyPath := path.Join(vaPath, path.Join(urlEscapeAll(nestedKeys)...))
	kvp, _, err := consulutil.GetKV().Get(keyPath, nil)
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
		var js json.RawMessage
		err := json.Unmarshal([]byte(s), &js)
		if err != nil {
			// Not a valid JSON, lets return string
			return &TOSCAValue{Value: s}, nil
		}
		return &TOSCAValue{Value: js}, nil
	}
	return nil, nil
}
