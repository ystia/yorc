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

	"github.com/ystia/yorc/v4/tosca"
)

func getValueAssignment(ctx context.Context, deploymentID, nodeName, instanceName, requirementIndex string, va, vaDef *tosca.ValueAssignment, nestedKeys ...string) (*TOSCAValue, error) {
	value, isFunction, err := getValueAssignmentWithoutResolve(ctx, va, vaDef, nestedKeys...)
	if err != nil || value == nil || !isFunction {
		return value, err
	}
	return resolveValueAssignment(ctx, deploymentID, nodeName, instanceName, requirementIndex, value, nestedKeys...)

}

func resolveVA(ctx context.Context, va *tosca.ValueAssignment, nestedKeys ...string) *TOSCAValue {
	if len(nestedKeys) == 0 {
		return &TOSCAValue{Value: va.Value}
	}
	if va.Type == tosca.ValueAssignmentMap {
		m, ok := va.Value.(map[string]interface{})
		if ok {
			v, ok := m[nestedKeys[0]].(tosca.ValueAssignment)
			if ok {
				return resolveVA(ctx, &v, nestedKeys[1:]...)
			}
			return &TOSCAValue{Value: m[nestedKeys[0]]}
		}
	}
	return nil
}

func getValueAssignmentWithoutResolve(ctx context.Context, va, vaDef *tosca.ValueAssignment, nestedKeys ...string) (*TOSCAValue, bool, error) {
	// Get value if not nil
	if va != nil && va.Value != nil {
		switch va.Type {
		case tosca.ValueAssignmentFunction:
			return &TOSCAValue{Value: va.Value}, true, nil
		case tosca.ValueAssignmentLiteral:
			return &TOSCAValue{Value: va.Value}, false, nil
		case tosca.ValueAssignmentList, tosca.ValueAssignmentMap:
			if len(nestedKeys) > 0 {
				return resolveVA(ctx, va, nestedKeys...), false, nil
			}
			return &TOSCAValue{Value: va.Value}, false, nil
		}
	}

	// Get default property otherwise
	if vaDef != nil {
		return &TOSCAValue{Value: vaDef.Value}, vaDef.Type == tosca.ValueAssignmentFunction, nil
	}
	// not found
	return nil, false, nil
}

func readComplexVA(ctx context.Context, vaType tosca.ValueAssignmentType, deploymentID, keyPath, baseDatatype string, nestedKeys ...string) (interface{}, error) {
	return nil, nil
}
func getValueAssignmentWithDataType(ctx context.Context, deploymentID, vaPath, nodeName, instanceName, requirementIndex, baseDatatype string, nestedKeys ...string) (*TOSCAValue, error) {
	return nil, nil
}
func getValueAssignmentWithoutResolveDeprecated(ctx context.Context, deploymentID, vaPath, baseDatatype string, nestedKeys ...string) (*TOSCAValue, bool, error) {
	return nil, false, nil
}
