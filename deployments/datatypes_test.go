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
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
)

func testGetTypePropertyDataType(t *testing.T, kv *api.KV) {
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/value_assignments.yaml")
	require.Nil(t, err)

	type args struct {
		typeName     string
		propertyName string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"ValueAssignmentComplexProp", args{"yorc.tests.nodes.ValueAssignmentNode", "complex"}, "yorc.tests.datatypes.ComplexType", false},
		{"ValueAssignmentListWithDefaultEntrySchema", args{"yorc.tests.nodes.ValueAssignmentNode", "list"}, "list:string", false},
		{"ComplexTypeMyMap", args{"yorc.tests.datatypes.ComplexType", "mymap"}, "map:integer", false},
		{"SubComplexTypeMyMap", args{"yorc.tests.datatypes.SubComplexType", "mymap"}, "map:integer", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTypePropertyDataType(kv, deploymentID, tt.args.typeName, tt.args.propertyName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTypePropertyDataType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetTypePropertyDataType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testGetNestedDataType(t *testing.T, kv *api.KV) {
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/value_assignments.yaml")
	require.Nil(t, err)

	type args struct {
		baseType   string
		nestedKeys []string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"ComplexTypeMap", args{"yorc.tests.datatypes.ComplexType", []string{"mymap"}}, "map:integer", false},
		{"ComplexTypeMapChild", args{"yorc.tests.datatypes.ComplexType", []string{"mymap", "something"}}, "integer", false},
		{"NestedTypeOnBaseType", args{"yorc.tests.datatypes.BaseType", []string{"nestedType"}}, "yorc.tests.datatypes.NestedType", false},
		{"NestedTypeListOnBaseType", args{"yorc.tests.datatypes.BaseType", []string{"nestedType", "listofstring"}}, "list:string", false},
		{"NestedTypeListChildOnBaseType", args{"yorc.tests.datatypes.BaseType", []string{"nestedType", "listofstring", "0"}}, "string", false},
		{"NestedTypeSubComplexOnBaseType", args{"yorc.tests.datatypes.BaseType", []string{"nestedType", "subcomplex"}}, "yorc.tests.datatypes.SubComplexType", false},
		{"NestedTypeSubComplexLiteralOnBaseType", args{"yorc.tests.datatypes.BaseType", []string{"nestedType", "subcomplex", "literal"}}, "integer", false},
		{"NestedTypeSubComplexMapOnBaseType", args{"yorc.tests.datatypes.BaseType", []string{"nestedType", "mapofcomplex"}}, "map:yorc.tests.datatypes.ComplexType", false},
		{"NestedTypeSubComplexMapChildOnBaseType", args{"yorc.tests.datatypes.BaseType", []string{"nestedType", "mapofcomplex", "something"}}, "yorc.tests.datatypes.ComplexType", false},
		{"NestedTypeSubComplexMapOnBaseType", args{"yorc.tests.datatypes.BaseType", []string{"nestedType", "mapofcomplex", "something", "literal"}}, "integer", false},
		{"NestedTypeSubComplexMapDoesntExistOnBaseType", args{"yorc.tests.datatypes.BaseType", []string{"nestedType", "mapofcomplex", "something", "doNotExist"}}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetNestedDataType(kv, deploymentID, tt.args.baseType, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNestedDataType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetNestedDataType() = %v, want %v", got, tt.want)
			}
		})
	}
}
