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
		t.Skip()
		t.Run("testResolveAttributeMappingWithInstanceAttribute", func(t *testing.T) {
			testResolveAttributeMappingWithInstanceAttribute(t, ctx, deploymentID)
		})
		t.Run("testResolveAttributeMappingWithCapabilityAttribute", func(t *testing.T) {
			testResolveAttributeMappingWithCapabilityAttribute(t, ctx, deploymentID)
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
	err = ResolveAttributeMapping(ctx, deploymentID, "VANode2", "0", "baseComplexAttr", 3, "nestedType", "listofcomplex", "0", "mymap")
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
		{"TestAttrComplexDTNestedListOfString", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "listofstring"}}, false, true, `["VANode2L1","VANode2L2"]`},
		{"TestAttrComplexDTNestedSubComplexLiteral", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "subcomplex", "literal"}}, false, true, `2`},
		{"TestAttrComplexDTNestedSubComplexLiteralDefault", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "subcomplex", "literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		{"TestAttrComplexDTNestedSubComplexAll", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "subcomplex"}}, false, true, `{"literal":"2","literalDefault":"ComplexDataTypeDefault"}`},
		{"TestAttrComplexDTNestedListOfComplex0Literal", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex", "0", "literal"}}, false, true, `2`},
		{"TestAttrComplexDTNestedListOfComplex0MyMap", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex", "0", "mymap"}}, false, true, `{"VANode2":"1"}`},
		{"TestAttrComplexDTNestedListOfComplex0All", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex", "0"}}, false, true, `{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"1"}}`},
		{"TestAttrComplexDTNestedListOfComplex1Literal", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex", "1", "literal"}}, false, true, `3`},
		{"TestAttrComplexDTNestedListOfComplex1MyMap", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex", "1", "mymap"}}, false, true, `{"VANode2":"2"}`},
		{"TestAttrComplexDTNestedListOfComplex1All", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex", "1"}}, false, true, `{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"2"}}`},
		{"TestAttrComplexDTNestedListOfComplexAll", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex"}}, false, true, `[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"2"}}]`},
		{"TestAttrComplexDTNestedMapOfComplex1Literal", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "mapofcomplex", "m1", "literal"}}, false, true, `4`},
		{"TestAttrComplexDTNestedMapOfComplex1MyMap", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "mapofcomplex", "m1", "mymap"}}, false, true, `{"VANode2":"3"}`},
		{"TestAttrComplexDTNestedMapOfComplex1All", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "mapofcomplex", "m1"}}, false, true, `{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"3"}}`},
		{"TestAttrComplexDTNestedMapOfComplexAll", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType", "mapofcomplex"}}, false, true, `{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"3"}}}`},
		{"TestAttrComplexDTNestedAll", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", []string{"nestedType"}}, false, true, `{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"2"}}],"listofstring":["VANode2L1","VANode2L2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}`},
		{"TestAttrComplexDTAll", nodeAttrArgs{"VANode2", "0", "baseComplexAttr", nil}, false, true, `{"nestedType":{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"2"}}],"listofstring":["VANode2L1","VANode2L2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}}`},
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
	err = ResolveAttributeMapping(ctx, deploymentID, "VANode1", "0", "host", map[string]string{"U1": "V1", "U2": "V2"}, "mapAttr")
	require.NoError(t, err)
	err = ResolveAttributeMapping(ctx, deploymentID, "VANode1", "0", "host", []interface{}{"UV1", "UV2", "UV3"}, "listAttr")
	require.NoError(t, err)
	err = ResolveAttributeMapping(ctx, deploymentID, "VANode1", "0", "host", map[string]interface{}{"literal": 5, "literalDefault": "CapNode1"}, "complexAttr")
	require.NoError(t, err)
	err = ResolveAttributeMapping(ctx, deploymentID, "VANode1", "0", "host", map[string]interface{}{
		"nestedType": map[string]interface{}{
			"listofstring":  []string{"CapNode1L1", "CapNode1L2"},
			"subcomplex":    map[string]interface{}{"literal": 2},
			"listofcomplex": []interface{}{map[string]interface{}{"literal": 2, "mymap": map[string]interface{}{"CapNode1": 1}}, map[string]interface{}{"literal": 3, "mymap": map[string]interface{}{"CapNode1": 2}}},
			"mapofcomplex":  map[string]interface{}{"m1": map[string]interface{}{"literal": 4, "mymap": map[string]interface{}{"CapNode1": 3}}},
		},
	}, "baseComplexAttr")
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
		{"TestCapabilityAttrMapAll", capAttrArgs{"VANode1", "0", "host", "mapAttr", nil}, false, true, `{"U1":"V1","U2":"V2"}`},
		{"TestCapabilityAttrMapKey1", capAttrArgs{"VANode1", "0", "host", "mapAttr", []string{"U1"}}, false, true, `V1`},
		{"TestCapabilityAttrMapKey2", capAttrArgs{"VANode1", "0", "host", "mapAttr", []string{"U2"}}, false, true, `V2`},
		{"TestCapabilityAttrListAll", capAttrArgs{"VANode1", "0", "host", "listAttr", nil}, false, true, `["UV1","UV2","UV3"]`},
		{"TestCapabilityAttrListIndex0", capAttrArgs{"VANode1", "0", "host", "listAttr", []string{"0"}}, false, true, `UV1`},
		{"TestCapabilityAttrListIndex1", capAttrArgs{"VANode1", "0", "host", "listAttr", []string{"1"}}, false, true, `UV2`},
		{"TestCapabilityAttrListIndex2", capAttrArgs{"VANode1", "0", "host", "listAttr", []string{"2"}}, false, true, `UV3`},
		{"TestCapabilityPropComplexTypeLit", capAttrArgs{"VANode1", "0", "host", "complexAttr", []string{"literal"}}, false, true, `5`},
		{"TestCapabilityAttrComplexTypeLitDef", capAttrArgs{"VANode1", "0", "host", "complexAttr", []string{"literalDefault"}}, false, true, `CapNode1`},
		{"TestCapabilityAttrComplexTypeAll", capAttrArgs{"VANode1", "0", "host", "complexAttr", nil}, false, true, `{"literal":"5","literalDefault":"CapNode1"}`},
		{"TestCapabilityAttrComplexDefaultAll", capAttrArgs{"VANode1", "0", "host", "complexDefAttr", nil}, false, true, `{"literal":"1","literalDefault":"ComplexDataTypeDefault"}`},
		{"TestCapabilityAttrComplexDefaultFromDT", capAttrArgs{"VANode1", "0", "host", "complexDefAttr", []string{"literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		{"TestCapabilityAttrComplexDTNestedListOfString", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "listofstring"}}, false, true, `["CapNode1L1","CapNode1L2"]`},
		{"TestCapabilityAttrComplexDTNestedSubComplexLiteral", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "subcomplex", "literal"}}, false, true, `2`},
		{"TestCapabilityAttrComplexDTNestedSubComplexLiteralDefault", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "subcomplex", "literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		{"TestCapabilityAttrComplexDTNestedSubComplexAll", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "subcomplex"}}, false, true, `{"literal":"2","literalDefault":"ComplexDataTypeDefault"}`},
		{"TestCapabilityAttrComplexDTNestedListOfComplex0Literal", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "listofcomplex", "0", "literal"}}, false, true, `2`},
		{"TestCapabilityAttrComplexDTNestedListOfComplex0MyMap", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "listofcomplex", "0", "mymap"}}, false, true, `{"CapNode1":"1"}`},
		{"TestCapabilityAttrComplexDTNestedListOfComplex0All", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "listofcomplex", "0"}}, false, true, `{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"1"}}`},
		{"TestCapabilityAttrComplexDTNestedListOfComplex1Literal", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "listofcomplex", "1", "literal"}}, false, true, `3`},
		{"TestCapabilityAttrComplexDTNestedListOfComplex1MyMap", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "listofcomplex", "1", "mymap"}}, false, true, `{"CapNode1":"2"}`},
		{"TestCapabilityAttrComplexDTNestedListOfComplex1All", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "listofcomplex", "1"}}, false, true, `{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"2"}}`},
		{"TestCapabilityAttrComplexDTNestedListOfComplexAll", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "listofcomplex"}}, false, true, `[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"2"}}]`},
		{"TestCapabilityAttrComplexDTNestedMapOfComplex1Literal", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "mapofcomplex", "m1", "literal"}}, false, true, `4`},
		{"TestCapabilityAttrComplexDTNestedMapOfComplex1MyMap", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "mapofcomplex", "m1", "mymap"}}, false, true, `{"CapNode1":"3"}`},
		{"TestCapabilityAttrComplexDTNestedMapOfComplex1All", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "mapofcomplex", "m1"}}, false, true, `{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"3"}}`},
		{"TestCapabilityAttrComplexDTNestedMapOfComplexAll", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType", "mapofcomplex"}}, false, true, `{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"3"}}}`},
		{"TestCapabilityAttrComplexDTNestedAll", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", []string{"nestedType"}}, false, true, `{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"2"}}],"listofstring":["CapNode1L1","CapNode1L2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}`},
		{"TestCapabilityAttrComplexDTAll", capAttrArgs{"VANode1", "0", "host", "baseComplexAttr", nil}, false, true, `{"nestedType":{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"2"}}],"listofstring":["CapNode1L1","CapNode1L2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}}`},

		// we reflect Node 1 attributes on Node 2 due to the hostedOn relationship
		{"TestCapabilityNode2AttrliteralAttr", capAttrArgs{"VANode2", "0", "host", "literalAttr", nil}, false, true, `user cap literal attr`},
		{"TestCapabilityNode2AttrMapAll", capAttrArgs{"VANode2", "0", "host", "mapAttr", nil}, false, true, `{"U1":"V1","U2":"V2"}`},
		{"TestCapabilityNode2AttrMapKey1", capAttrArgs{"VANode2", "0", "host", "mapAttr", []string{"U1"}}, false, true, `V1`},
		{"TestCapabilityNode2AttrMapKey2", capAttrArgs{"VANode2", "0", "host", "mapAttr", []string{"U2"}}, false, true, `V2`},
		{"TestCapabilityNode2AttrListAll", capAttrArgs{"VANode2", "0", "host", "listAttr", nil}, false, true, `["UV1","UV2","UV3"]`},
		{"TestCapabilityNode2AttrListIndex0", capAttrArgs{"VANode2", "0", "host", "listAttr", []string{"0"}}, false, true, `UV1`},
		{"TestCapabilityNode2AttrListIndex1", capAttrArgs{"VANode2", "0", "host", "listAttr", []string{"1"}}, false, true, `UV2`},
		{"TestCapabilityNode2AttrListIndex2", capAttrArgs{"VANode2", "0", "host", "listAttr", []string{"2"}}, false, true, `UV3`},
		{"TestCapabilityAttrComplexDTAll", capAttrArgs{"VANode2", "0", "host", "baseComplexAttr", nil}, false, true, `{"nestedType":{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"2"}}],"listofstring":["CapNode1L1","CapNode1L2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}}`},
		{"TestCapabilityAttrComplexTypeAll", capAttrArgs{"VANode2", "0", "host", "complexAttr", nil}, false, true, `{"literal":"5","literalDefault":"CapNode1"}`},
		{"TestCapabilityAttrComplexDefaultAll", capAttrArgs{"VANode2", "0", "host", "complexDefAttr", nil}, false, true, `{"literal":"1","literalDefault":"ComplexDataTypeDefault"}`},

		// Now check that we reflect properties as attributes
		{"TestCapabilityPropAsAttrLitDefaultNode1", capAttrArgs{"VANode1", "0", "host", "literalDefault", nil}, false, true, `capDefault`},
		{"TestCapabilityPropAsAttrMapDefaultAllNode1", capAttrArgs{"VANode1", "0", "host", "mapPropDefault", nil}, false, true, `{"capProp1":"capPropVal1","capProp2":"capPropVal2"}`},
		{"TestCapabilityPropAsAttrListDefaultAllNode1", capAttrArgs{"VANode1", "0", "host", "listPropDefault", nil}, false, true, `["capPropI1","capPropI2","capPropI3"]`},
		{"TestCapabilityPropAsAttrLiteral", capAttrArgs{"VANode1", "0", "host", "literal", nil}, false, true, `user cap literal`},
		{"TestCapabilityPropAsAttrMapAll", capAttrArgs{"VANode1", "0", "host", "mapProp", nil}, false, true, `{"U1":"V1","U2":"V2"}`},
		{"TestCapabilityPropAsAttrListAll", capAttrArgs{"VANode1", "0", "host", "listProp", nil}, false, true, `["UV1","UV2","UV3"]`},
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
		{"NestedTypeSubComplexMapOnBaseType", args{"yorc.tests.datatypes.BaseType", []string{"nestedType", "mapofcomplex", "something", "literal"}}, map[string]interface{}{
			"nestedType": map[string]interface{}{
				"mapofcomplex": map[string]interface{}{
					"something": map[string]interface{}{
						"literal": map[string]interface{}{},
					},
				},
			},
		}, false},
		{"NestedTypeSubComplexMapOnBaseType", args{"yorc.tests.datatypes.BaseType", []string{"nestedType", "listofcomplex", "5", "mymap"}}, map[string]interface{}{
			"nestedType": map[string]interface{}{
				"listofcomplex": []interface{}{nil, nil, nil, nil, nil, map[string]interface{}{"mymap": map[string]interface{}{}}},
			},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildValue(ctx, deploymentID, tt.args.baseType, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
