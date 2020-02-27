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
	"github.com/stretchr/testify/require"
	"reflect"
	"strings"
	"testing"
)

func testResolveAttributeMapping(t *testing.T) {
	ctx := context.Background()
	// t.Parallel()
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), deploymentID, "testdata/value_assignments.yaml")
	require.Nil(t, err)

	t.Run("groupResolveAttributeMapping", func(t *testing.T) {
		t.Run("testBuildNestedValue", func(t *testing.T) {
			testBuildNestedValue(t, ctx, deploymentID)
		})
		t.Run("testResolveAttributeMappingWithCapabilityAttribute", func(t *testing.T) {
			testResolveAttributeMappingWithCapabilityAttribute(t, ctx, deploymentID)
		})
		t.Run("testResolveAttributeMappingWithInstanceAttribute", func(t *testing.T) {
			testResolveAttributeMappingWithInstanceAttribute(t, ctx, deploymentID)
		})
	})
}

func testResolveAttributeMappingWithInstanceAttribute(t *testing.T, ctx context.Context, deploymentID string) {
	// Then test node attributes
	err := ResolveAttributeMapping(ctx, deploymentID, "VANode1", "0", "lit", "myLiteral")
	require.NoError(t, err)
	err = ResolveAttributeMapping(ctx, deploymentID, "VANode1", "0", "listAttr", 42, "0")
	require.NoError(t, err)
	err = ResolveAttributeMapping(ctx, deploymentID, "VANode1", "0", "mapAttr", "v2", "map2")
	require.NoError(t, err)
	err = ResolveAttributeMapping(ctx, deploymentID, "VANode1", "0", "complexAttr", "11", "literal")
	require.NoError(t, err)
	err = ResolveAttributeMapping(ctx, deploymentID, "VANode2", "0", "baseComplexAttr", 3, "nestedType", "listofcomplex", "3", "mymap")
	require.NoError(t, err)
	type nodeAttrArgs struct {
		nodeName      string
		instanceName  string
		attributeName string
		nestedKeys    []string
	}
	nodeAttrTests := []struct {
		name      string
		args      nodeAttrArgs
		wantErr   bool
		wantFound bool
		want      string
	}{
		{"TestNodeAttrListDef0", nodeAttrArgs{"VANode1", "0", "listDef", []string{"0"}}, false, true, `1`},
		{"TestNodeAttrListDef1", nodeAttrArgs{"VANode1", "0", "listDef", []string{"1"}}, false, true, `2`},
		{"TestNodeAttrListDefAll", nodeAttrArgs{"VANode1", "0", "listDef", nil}, false, true, `["1","2","3"]`},
		{"TestNodeAttrMapDefT2", nodeAttrArgs{"VANode1", "0", "mapDef", []string{"T2"}}, false, true, `1 TiB`},
		{"TestNodeAttrMapDefT3", nodeAttrArgs{"VANode1", "0", "mapDef", []string{"T3"}}, false, true, `3 GB`},
		{"TestNodeAttrMapDefAll", nodeAttrArgs{"VANode1", "0", "mapDef", nil}, false, true, `{"T1":"4 GiB","T2":"1 TiB","T3":"3 GB"}`},
		{"TestNodeAttrLiteral", nodeAttrArgs{"VANode1", "0", "lit", nil}, false, true, `myLiteral`},
		{"TestNodeAttrListAll", nodeAttrArgs{"VANode1", "0", "listAttr", nil}, false, true, `["42"]`},
		{"TestNodeAttrListIndex0", nodeAttrArgs{"VANode1", "0", "listAttr", []string{"0"}}, false, true, `42`},
		{"TestNodeAttrMapAll", nodeAttrArgs{"VANode1", "0", "mapAttr", nil}, false, true, `{"map2":"v2"}`},
		{"TestNodeAttrMapKey2", nodeAttrArgs{"VANode1", "0", "mapAttr", []string{"map2"}}, false, true, `v2`},
		{"TestAttrComplexTypeLit", nodeAttrArgs{"VANode1", "0", "complexAttr", []string{"literal"}}, false, true, `11`},
		{"TestAttrComplexTypeLitDef", nodeAttrArgs{"VANode1", "0", "complexAttr", []string{"literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		{"TestAttrComplexTypeAll", nodeAttrArgs{"VANode1", "0", "complexAttr", nil}, false, true, `{"literal":"11","literalDefault":"ComplexDataTypeDefault"}`},
		{"TestAttrComplexDefaultAll", nodeAttrArgs{"VANode2", "0", "complexDefAttr", nil}, false, true, `{"literal":"1","literalDefault":"ComplexDataTypeDefault"}`},
		{"TestAttrComplexDefaultFromDT", nodeAttrArgs{"VANode2", "0", "complexDefAttr", []string{"literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		{"TestAttrComplexDTNestedListOfComplex1Literal", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex", "3"}}, false, true, `{"literalDefault":"ComplexDataTypeDefault","mymap":"3"}`},
		{"TestAttrComplexDTNestedListOfComplex1MyMap", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex", "3", "mymap"}}, false, true, `3`},
		{"TestAttrComplexDTNestedAll", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType"}}, false, true, `{"listofcomplex":["","","",{"literalDefault":"ComplexDataTypeDefault","mymap":"3"}]}`},
		{"TestAttrComplexDTAll", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", nil}, false, true, `{"nestedType":{"listofcomplex":["","","",{"literalDefault":"ComplexDataTypeDefault","mymap":"3"}]}}`},
	}
	for _, tt := range nodeAttrTests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetInstanceAttributeValue(ctx, deploymentID, tt.args.nodeName, tt.args.instanceName, tt.args.attributeName, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInstanceAttribute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && (got != nil) != tt.wantFound {
				t.Errorf("GetInstanceAttribute() found result = %v, want %v", got, tt.wantFound)
			}
			if got != nil && got.RawString() != tt.want {
				t.Errorf("GetInstanceAttribute() = %q, want %q", got.RawString(), tt.want)
			}
		})
	}
}

func testResolveAttributeMappingWithCapabilityAttribute(t *testing.T, ctx context.Context, deploymentID string) {
	err := ResolveAttributeMapping(ctx, deploymentID, "VANode1", "0", "host", "user cap literal attr", "literalAttr")
	require.NoError(t, err)
	err = ResolveAttributeMapping(ctx, deploymentID, "VANode1", "0", "host", "V2", "mapAttr", "U2")
	require.NoError(t, err)
	err = ResolveAttributeMapping(ctx, deploymentID, "VANode1", "0", "host", "UV3", "listAttr", "2")
	require.NoError(t, err)
	err = ResolveAttributeMapping(ctx, deploymentID, "VANode1", "0", "host", 5, "complexAttr", "literal")
	require.NoError(t, err)
	err = ResolveAttributeMapping(ctx, deploymentID, "VANode1", "0", "host", 2, "baseComplexAttr", "nestedType", "subcomplex", "literal")
	require.NoError(t, err)

	type capAttrArgs struct {
		nodeName      string
		instanceName  string
		capability    string
		attributeName string
		nestedKeys    []string
	}
	capAttrTests := []struct {
		name      string
		args      capAttrArgs
		wantErr   bool
		wantFound bool
		want      string
	}{
		{"TestCapabilityAttrLitDefaultNode1", capAttrArgs{"VANode1", "0", "host", "literalDefault", nil}, false, true, `capDefault`},
		{"TestCapabilityAttrLitDefaultNode2", capAttrArgs{"VANode2", "0", "host", "literalDefault", nil}, false, true, `capDefault`},
		{"TestCapabilityAttrMapDefaultAllNode1", capAttrArgs{"VANode1", "0", "host", "mapAttrDefault", nil}, false, true, `{"capAttr1":"capAttrVal1","capAttr2":"capAttrVal2"}`},
		{"TestCapabilityAttrMapDefaultAllNode2", capAttrArgs{"VANode2", "0", "host", "mapAttrDefault", nil}, false, true, `{"capAttr1":"capAttrVal1","capAttr2":"capAttrVal2"}`},
		{"TestCapabilityAttrMapDefaultKey1Node1", capAttrArgs{"VANode1", "0", "host", "mapAttrDefault", []string{"capAttr1"}}, false, true, `capAttrVal1`},
		{"TestCapabilityAttrMapDefaultKey2Node1", capAttrArgs{"VANode1", "0", "host", "mapAttrDefault", []string{"capAttr2"}}, false, true, `capAttrVal2`},
		{"TestCapabilityAttrListDefaultAllNode1", capAttrArgs{"VANode1", "0", "host", "listAttrDefault", nil}, false, true, `["capAttrI1","capAttrI2","capAttrI3"]`},
		{"TestCapabilityAttrListDefaultAllNode2", capAttrArgs{"VANode2", "0", "host", "listAttrDefault", nil}, false, true, `["capAttrI1","capAttrI2","capAttrI3"]`},
		{"TestCapabilityAttrListDefaultKey1Node1", capAttrArgs{"VANode1", "0", "host", "listAttrDefault", []string{"0"}}, false, true, `capAttrI1`},
		{"TestCapabilityAttrListDefaultKey2Node1", capAttrArgs{"VANode1", "0", "host", "listAttrDefault", []string{"1"}}, false, true, `capAttrI2`},
		{"TestCapabilityAttrListDefaultKey3Node1", capAttrArgs{"VANode1", "0", "host", "listAttrDefault", []string{"2"}}, false, true, `capAttrI3`},
		{"TestCapabilityAttrAttrNotFound", capAttrArgs{"VANode1", "0", "host", "doesnotexist", nil}, false, false, ``},
		{"TestCapabilityAttrListIndexNotFound", capAttrArgs{"VANode1", "0", "host", "listAttrDefault", []string{"42"}}, false, false, ``},
		{"TestCapabilityAttrMapKeyNotFound", capAttrArgs{"VANode1", "0", "host", "mapAttrDefault", []string{"42"}}, false, false, ``},
		{"TestCapabilityAttrliteralAttr", capAttrArgs{"VANode1", "0", "host", "literalAttr", nil}, false, true, `user cap literal attr`},
		{"TestCapabilityAttrMapAll", capAttrArgs{"VANode1", "0", "host", "mapAttr", nil}, false, true, `{"U2":"V2"}`},
		{"TestCapabilityAttrMapKey1", capAttrArgs{"VANode1", "0", "host", "mapAttr", []string{"U1"}}, false, false, ``},
		{"TestCapabilityAttrMapKey2", capAttrArgs{"VANode1", "0", "host", "mapAttr", []string{"U2"}}, false, true, `V2`},

		{"TestCapabilityAttrListAll", capAttrArgs{"VANode1", "0", "host", "listAttr", nil}, false, true, `["","","UV3"]`},
		{"TestCapabilityAttrListIndex0", capAttrArgs{"VANode1", "0", "host", "listAttr", []string{"0"}}, false, true, ``},
		{"TestCapabilityAttrListIndex1", capAttrArgs{"VANode1", "0", "host", "listAttr", []string{"1"}}, false, true, ``},
		{"TestCapabilityAttrListIndex2", capAttrArgs{"VANode1", "0", "host", "listAttr", []string{"2"}}, false, true, `UV3`},

		{"TestCapabilityPropComplexTypeLit", capAttrArgs{"VANode1", "0", "host", "complexAttr", []string{"literal"}}, false, true, `5`},
		{"TestCapabilityAttrComplexTypeLitDef", capAttrArgs{"VANode1", "0", "host", "complexAttr", []string{"literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		{"TestCapabilityAttrComplexTypeAll", capAttrArgs{"VANode1", "0", "host", "complexAttr", nil}, false, true, `{"literal":"5","literalDefault":"ComplexDataTypeDefault"}`},
		{"TestCapabilityAttrComplexDefaultAll", capAttrArgs{"VANode1", "0", "host", "complexDefAttr", nil}, false, true, `{"literal":"1","literalDefault":"ComplexDataTypeDefault"}`},
		{"TestCapabilityAttrComplexDefaultFromDT", capAttrArgs{"VANode1", "0", "host", "complexDefAttr", []string{"literalDefault"}}, false, true, `ComplexDataTypeDefault`},

		{"TestCapabilityAttrComplexDTNestedListOfString", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "listofstring"}}, false, false, ``},
		{"TestCapabilityAttrComplexDTNestedSubComplexLiteral", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "subcomplex", "literal"}}, false, true, `2`},
		{"TestCapabilityAttrComplexDTNestedSubComplexLiteralDefault", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "subcomplex", "literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		{"TestCapabilityAttrComplexDTNestedSubComplexAll", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "subcomplex"}}, false, true, `{"literal":"2","literalDefault":"ComplexDataTypeDefault"}`},
	}
	for _, tt := range capAttrTests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetInstanceCapabilityAttributeValue(ctx, deploymentID,
				tt.args.nodeName, tt.args.instanceName, tt.args.capability,
				tt.args.attributeName, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInstanceCapabilityAttribute(%s, %s, %s, %s) error = %v, wantErr %v",
					tt.args.nodeName, tt.args.instanceName, tt.args.capability, tt.args.attributeName,
					err, tt.wantErr)
				return
			}
			if err == nil && (got != nil) != tt.wantFound {
				t.Errorf("GetInstanceCapabilityAttribute(%s, %s, %s, %s) found result = %v, want %v",
					tt.args.nodeName, tt.args.instanceName, tt.args.capability, tt.args.attributeName,
					got, tt.wantFound)
			}
			if got != nil && got.RawString() != tt.want {
				t.Errorf("GetInstanceCapabilityAttribute(%s, %s, %s, %s) = %q, want %q",
					tt.args.nodeName, tt.args.instanceName, tt.args.capability, tt.args.attributeName,
					got.RawString(), tt.want)
			}
		})
	}
}

func testBuildNestedValue(t *testing.T, ctx context.Context, deploymentID string) {
	type args struct {
		baseType   string
		nestedKeys []string
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{"NestedTypeSubComplexMapOnBaseType", args{"yorc.tests.datatypes.BaseType", []string{"nestedType", "mapofcomplex", "mymap"}}, map[string]interface{}{
			"nestedType": map[string]interface{}{
				"mapofcomplex": map[string]interface{}{
					"mymap": map[string]interface{}{},
				},
			},
		}, false},
		{"NestedTypeSubComplexMapOnBaseType", args{"yorc.tests.datatypes.BaseType", []string{"nestedType", "listofcomplex", "5", "mymap"}}, map[string]interface{}{
			"nestedType": map[string]interface{}{
				"listofcomplex": []interface{}{"", "", "", "", "", map[string]interface{}{"mymap": map[string]interface{}{}}},
			},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildComplexValue(ctx, deploymentID, tt.args.baseType, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildComplexValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildComplexValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateNestedValue(t *testing.T) {

	type args struct {
		value       interface{}
		nestedValue interface{}
		nestedKeys  []string
	}

	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{"ComplexSlice", args{[]interface{}{map[string]interface{}{"literal": 2, "mymap": map[string]interface{}{"VANode2": 1}}, map[string]interface{}{"literal": 3, "mymap": map[string]interface{}{"VANode2": 2}}}, 5, []string{"0", "literal"}}, []interface{}{map[string]interface{}{"literal": 5, "mymap": map[string]interface{}{"VANode2": 1}}, map[string]interface{}{"literal": 3, "mymap": map[string]interface{}{"VANode2": 2}}}, false},
		{"SimpleValue", args{"myvalue", "myvalueUp", nil}, "myvalueUp", false},
		{"SimpleMap", args{map[string]interface{}{"literal": "11", "literalDefault": "VANode1LitDef"}, "VANode1LitDefUp", []string{"literalDefault"}}, map[string]interface{}{"literal": "11", "literalDefault": "VANode1LitDefUp"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := updateComplexValue(tt.args.value, tt.args.nestedValue, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateComplexValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("updateComplexValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
