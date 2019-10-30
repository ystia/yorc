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
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"vbom.ml/util/sortorder"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/tosca"
)

func urlEscapeAll(keys []string) []string {
	t := make([]string, len(keys))
	for i, k := range keys {
		t[i] = url.QueryEscape(k)
	}
	return t
}

func getValueAssignmentWithoutResolve(ctx context.Context, deploymentID, vaPath, baseDatatype string, nestedKeys ...string) (*TOSCAValue, bool, error) {
	keyPath := path.Join(vaPath, path.Join(urlEscapeAll(nestedKeys)...))
	exist, value, meta, err := consulutil.GetValueWithMetadata(keyPath)
	if err != nil {
		return nil, false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if exist && meta != nil {
		vat := tosca.ValueAssignmentType(meta.Flag)
		switch vat {
		case tosca.ValueAssignmentLiteral, tosca.ValueAssignmentFunction:
			return &TOSCAValue{Value: string(value)}, vat == tosca.ValueAssignmentFunction, nil
		case tosca.ValueAssignmentList, tosca.ValueAssignmentMap:
			res, err := readComplexVA(ctx, vat, deploymentID, keyPath, baseDatatype, nestedKeys...)
			if err != nil {
				return nil, false, err
			}
			return &TOSCAValue{Value: res}, false, nil
		}
	}
	if baseDatatype != "" && !strings.HasPrefix(baseDatatype, "list:") && !strings.HasPrefix(baseDatatype, "map:") && len(nestedKeys) > 0 {
		result, isFunc, err := getTypeDefaultProperty(ctx, deploymentID, baseDatatype, nestedKeys[0], nestedKeys[1:]...)
		if err != nil || result != nil {
			return result, isFunc, err
		}
	}
	// not found
	return nil, false, nil
}

func getValueAssignment(ctx context.Context, deploymentID, vaPath, nodeName, instanceName, requirementIndex string, nestedKeys ...string) (*TOSCAValue, error) {
	return getValueAssignmentWithDataType(ctx, deploymentID, vaPath, nodeName, instanceName, requirementIndex, "", nestedKeys...)
}

func getValueAssignmentWithDataType(ctx context.Context, deploymentID, vaPath, nodeName, instanceName, requirementIndex, baseDatatype string, nestedKeys ...string) (*TOSCAValue, error) {

	value, isFunction, err := getValueAssignmentWithoutResolve(ctx, deploymentID, vaPath, baseDatatype, nestedKeys...)
	if err != nil || value == nil || !isFunction {
		return value, err
	}
	return resolveValueAssignment(ctx, deploymentID, nodeName, instanceName, requirementIndex, value, nestedKeys...)

}

func readComplexVA(ctx context.Context, vaType tosca.ValueAssignmentType, deploymentID, keyPath, baseDatatype string, nestedKeys ...string) (interface{}, error) {
	keys, err := consulutil.GetKeys(keyPath)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	var result interface{}
	if vaType == tosca.ValueAssignmentList {
		result = make([]interface{}, 0)
	} else {
		result = make(map[string]interface{})
	}

	sort.Sort(sortorder.Natural(keys))
	var i int
	for _, k := range keys {
		exist, value, meta, err := consulutil.GetValueWithMetadata(k)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if exist && meta != nil {
			// Sounds weird to be nil as we listed it just before but also sounds a good practice to check it
			kr, err := url.QueryUnescape(path.Base(k))
			if err != nil {
				return nil, errors.Wrapf(err, "failed to unescape key: %q", path.Base(k))
			}
			subKeyType := tosca.ValueAssignmentType(meta.Flag)
			var sr interface{}
			switch subKeyType {
			case tosca.ValueAssignmentList, tosca.ValueAssignmentMap:
				newNestedkeys := append(nestedKeys, kr)
				sr, err = readComplexVA(ctx, subKeyType, deploymentID, k, baseDatatype, newNestedkeys...)
				if err != nil {
					return nil, err
				}
			default:
				sr = string(value)
			}
			switch v := result.(type) {
			case []interface{}:
				result = append(v, sr)
			case map[string]interface{}:
				v[kr] = sr
			}
			i++
		}
	}

	if baseDatatype != "" && vaType == tosca.ValueAssignmentMap {
		currentDatatype, err := GetNestedDataType(ctx, deploymentID, baseDatatype, nestedKeys...)
		if err != nil {
			return nil, err
		}
		castedResult := result.(map[string]interface{})
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
	return result, nil
}
