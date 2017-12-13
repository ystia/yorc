package deployments

import (
	"context"
	"path"
	"testing"

	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/prov"

	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
)

func testDefinitionStore(t *testing.T, kv *api.KV) {
	t.Run("groupDeploymentsDefinitionStore", func(t *testing.T) {
		t.Run("TestImplementationArtifacts", func(t *testing.T) {
			testImplementationArtifacts(t, kv)
		})
		t.Run("TestImplementationArtifactsDuplicates", func(t *testing.T) {
			testImplementationArtifactsDuplicates(t, kv)
		})
		t.Run("TestValueAssignments", func(t *testing.T) {
			testValueAssignments(t, kv)
		})
	})
}

func testImplementationArtifacts(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/get_op_output.yaml")
	require.Nil(t, err, "Failed to parse testdata/get_op_output.yaml definition")

	impl, err := GetImplementationArtifactForExtension(kv, deploymentID, "sh")
	require.Nil(t, err)
	require.Equal(t, "tosca.artifacts.Implementation.Bash", impl)

	impl, err = GetImplementationArtifactForExtension(kv, deploymentID, "SH")
	require.Nil(t, err)
	require.Equal(t, "tosca.artifacts.Implementation.Bash", impl)

	impl, err = GetImplementationArtifactForExtension(kv, deploymentID, "py")
	require.Nil(t, err)
	require.Equal(t, "tosca.artifacts.Implementation.Python", impl)

	impl, err = GetImplementationArtifactForExtension(kv, deploymentID, "Py")
	require.Nil(t, err)
	require.Equal(t, "tosca.artifacts.Implementation.Python", impl)

	impl, err = GetImplementationArtifactForExtension(kv, deploymentID, "yaml")
	require.Nil(t, err)
	require.Equal(t, "tosca.artifacts.Implementation.Ansible", impl)

	impl, err = GetImplementationArtifactForExtension(kv, deploymentID, "yml")
	require.Nil(t, err)
	require.Equal(t, "tosca.artifacts.Implementation.Ansible", impl)

}

func testImplementationArtifactsDuplicates(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/artifacts_ext_duplicate.yaml")
	require.Error(t, err, "Expecting for a duplicate extension for artifact implementation")

}

func testValueAssignments(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/value_assignments.yaml")
	require.Nil(t, err)
	// First test operation outputs detection
	vaTypePrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types/janus.tests.nodes.ValueAssignmentNode")
	kvp, _, err := kv.Get(path.Join(vaTypePrefix, "interfaces/standard/create/outputs/SELF/CREATE_OUTPUT/expression"), nil)
	require.Nil(t, err)
	require.NotNil(t, kvp)
	require.Equal(t, "get_operation_output: [SELF, Standard, create, CREATE_OUTPUT]", string(kvp.Value))
	kvp, _, err = kv.Get(path.Join(vaTypePrefix, "interfaces/standard/configure/outputs/SELF/PARTITION_NAME/expression"), nil)
	require.Nil(t, err)
	require.NotNil(t, kvp)
	require.Equal(t, "get_operation_output: [SELF, Standard, configure, PARTITION_NAME]", string(kvp.Value))

	// Then test node properties
	type nodePropArgs struct {
		nodeName     string
		propertyName string
		nestedKeys   []string
	}
	nodePropTests := []struct {
		name      string
		args      nodePropArgs
		wantErr   bool
		wantFound bool
		want      string
	}{
		{"TestEmptyProp", nodePropArgs{"VANode1", "empty", nil}, false, true, ``},
		{"TestList0", nodePropArgs{"VANode1", "list", []string{"0"}}, false, true, `http://`},
		{"TestList1", nodePropArgs{"VANode1", "list", []string{"1"}}, false, true, `janus`},
		{"TestList2", nodePropArgs{"VANode1", "list", []string{"2"}}, false, true, `.io`},
		{"TestListComplex", nodePropArgs{"VANode1", "list", nil}, false, true, `["http://","janus",".io"]`},
		{"TestListExt0", nodePropArgs{"VANode2", "list", []string{"0"}}, false, true, `http://`},
		{"TestListExt1", nodePropArgs{"VANode2", "list", []string{"1"}}, false, true, `janus`},
		{"TestListExt2", nodePropArgs{"VANode2", "list", []string{"2"}}, false, true, `.io`},
		{"TestListExtComplex", nodePropArgs{"VANode2", "list", nil}, false, true, `["http://","janus",".io"]`},
		{"TestMap0", nodePropArgs{"VANode1", "map", []string{"one"}}, false, true, `1`},
		{"TestMap1", nodePropArgs{"VANode1", "map", []string{"two"}}, false, true, `2`},
		{"TestMapComplex", nodePropArgs{"VANode1", "map", nil}, false, true, `{"one":"1","two":"2"}`},
		{"TestMapExt0", nodePropArgs{"VANode2", "map", []string{"one"}}, false, true, `1`},
		{"TestMapExt1", nodePropArgs{"VANode2", "map", []string{"two"}}, false, true, `2`},
		{"TestMapExtComplex", nodePropArgs{"VANode2", "map", nil}, false, true, `{"one":"1","two":"2"}`},
		{"TestLiteralN1", nodePropArgs{"VANode1", "literal", nil}, false, true, `1`},
		{"TestLiteralN2", nodePropArgs{"VANode2", "literal", []string{}}, false, true, `1`},
		{"TestPropNotFound", nodePropArgs{"VANode2", "do_not_exits", nil}, false, false, ``},
		{"TestNestedKeyNotFound", nodePropArgs{"VANode2", "map", []string{"do_not_exits"}}, false, false, ``},
		{"TestIndexNotFound", nodePropArgs{"VANode2", "list", []string{"42"}}, false, false, ``},
		{"TestDefaultMapAll", nodePropArgs{"VANode1", "mapdef", nil}, false, true, `{"def1":"1","def2":"2"}`},
		{"TestDefaultMap", nodePropArgs{"VANode1", "mapdef", []string{"def1"}}, false, true, `1`},
		{"TestDefaultListAll", nodePropArgs{"VANode1", "listdef", nil}, false, true, `["l1","l2"]`},
		{"TestDefaultlist", nodePropArgs{"VANode1", "listdef", []string{"1"}}, false, true, `l2`},
		{"TestComplexTypeLit", nodePropArgs{"VANode1", "complex", []string{"literal"}}, false, true, `11`},
		{"TestComplexTypeLitDef", nodePropArgs{"VANode1", "complex", []string{"literalDefault"}}, false, true, `VANode1LitDef`},
		{"TestComplexTypeAll", nodePropArgs{"VANode1", "complex", nil}, false, true, `{"literal":"11","literalDefault":"VANode1LitDef"}`},
		{"TestComplexDefaultAll", nodePropArgs{"VANode2", "complexDef", nil}, false, true, `{"literal":"1","literalDefault":"ComplexDataTypeDefault"}`},
		{"TestComplexDefaultFromDT", nodePropArgs{"VANode2", "complexDef", []string{"literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		{"TestComplexDTNestedListOfString", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType", "listofstring"}}, false, true, `["VANode2L1","VANode2L2"]`},
		{"TestComplexDTNestedSubComplexLiteral", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType", "subcomplex", "literal"}}, false, true, `2`},
		{"TestComplexDTNestedSubComplexLiteralDefault", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType", "subcomplex", "literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		{"TestComplexDTNestedSubComplexAll", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType", "subcomplex"}}, false, true, `{"literal":"2","literalDefault":"ComplexDataTypeDefault"}`},
		{"TestComplexDTNestedListOfComplex0Literal", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType", "listofcomplex", "0", "literal"}}, false, true, `2`},
		{"TestComplexDTNestedListOfComplex0MyMap", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType", "listofcomplex", "0", "mymap"}}, false, true, `{"VANode2":"1"}`},
		{"TestComplexDTNestedListOfComplex0All", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType", "listofcomplex", "0"}}, false, true, `{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"1"}}`},
		{"TestComplexDTNestedListOfComplex1Literal", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType", "listofcomplex", "1", "literal"}}, false, true, `3`},
		{"TestComplexDTNestedListOfComplex1MyMap", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType", "listofcomplex", "1", "mymap"}}, false, true, `{"VANode2":"2"}`},
		{"TestComplexDTNestedListOfComplex1All", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType", "listofcomplex", "1"}}, false, true, `{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"2"}}`},
		{"TestComplexDTNestedListOfComplexAll", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType", "listofcomplex"}}, false, true, `[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"2"}}]`},
		{"TestComplexDTNestedMapOfComplex1Literal", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType", "mapofcomplex", "m1", "literal"}}, false, true, `4`},
		{"TestComplexDTNestedMapOfComplex1MyMap", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType", "mapofcomplex", "m1", "mymap"}}, false, true, `{"VANode2":"3"}`},
		{"TestComplexDTNestedMapOfComplex1All", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType", "mapofcomplex", "m1"}}, false, true, `{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"3"}}`},
		{"TestComplexDTNestedMapOfComplexAll", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType", "mapofcomplex"}}, false, true, `{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"3"}}}`},
		{"TestComplexDTNestedAll", nodePropArgs{"VANode2", "baseComplex", []string{"nestedType"}}, false, true, `{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"2"}}],"listofstring":["VANode2L1","VANode2L2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}`},
		{"TestComplexDTAll", nodePropArgs{"VANode2", "baseComplex", nil}, false, true, `{"nestedType":{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"2"}}],"listofstring":["VANode2L1","VANode2L2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}}`},
	}
	for _, tt := range nodePropTests {
		t.Run(tt.name, func(t *testing.T) {
			found, got, err := GetNodeProperty(kv, deploymentID, tt.args.nodeName, tt.args.propertyName, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNodeProperty() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && found != tt.wantFound {
				t.Errorf("GetNodeProperty() found result = %v, want %v", found, tt.wantFound)
			}
			if err == nil && got != tt.want {
				t.Errorf("GetNodeProperty() = %q, want %q", got, tt.want)
			}
		})
	}

	// Then test node attributes
	err = SetInstanceAttribute(deploymentID, "VANode1", "0", "lit", "myLiteral")
	require.NoError(t, err)
	err = SetInstanceAttributeComplex(deploymentID, "VANode1", "0", "listAttr", []int{42, 43, 44})
	require.NoError(t, err)
	err = SetInstanceAttributeComplex(deploymentID, "VANode1", "0", "mapAttr", map[string]interface{}{"map1": "v1", "map2": "v2", "map3": "v3"})
	require.NoError(t, err)
	err = SetInstanceAttributeComplex(deploymentID, "VANode1", "0", "complexAttr", map[string]interface{}{"literal": "11", "literalDefault": "VANode1LitDef"})
	require.NoError(t, err)
	err = SetInstanceAttributeComplex(deploymentID, "VANode2", "0", "baseComplexAttr", map[string]interface{}{
		"nestedType": map[string]interface{}{
			"listofstring":  []string{"VANode2L1", "VANode2L2"},
			"subcomplex":    map[string]interface{}{"literal": 2},
			"listofcomplex": []interface{}{map[string]interface{}{"literal": 2, "mymap": map[string]interface{}{"VANode2": 1}}, map[string]interface{}{"literal": 3, "mymap": map[string]interface{}{"VANode2": 2}}},
			"mapofcomplex":  map[string]interface{}{"m1": map[string]interface{}{"literal": 4, "mymap": map[string]interface{}{"VANode2": 3}}},
		},
	})
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
		{"TestNodeAttrListAll", nodeAttrArgs{"VANode1", "0", "listAttr", nil}, false, true, `["42","43","44"]`},
		{"TestNodeAttrListIndex0", nodeAttrArgs{"VANode1", "0", "listAttr", []string{"0"}}, false, true, `42`},
		{"TestNodeAttrListIndex1", nodeAttrArgs{"VANode1", "0", "listAttr", []string{"1"}}, false, true, `43`},
		{"TestNodeAttrListIndex2", nodeAttrArgs{"VANode1", "0", "listAttr", []string{"2"}}, false, true, `44`},
		{"TestNodeAttrMapAll", nodeAttrArgs{"VANode1", "0", "mapAttr", nil}, false, true, `{"map1":"v1","map2":"v2","map3":"v3"}`},
		{"TestNodeAttrMapKey1", nodeAttrArgs{"VANode1", "0", "mapAttr", []string{"map1"}}, false, true, `v1`},
		{"TestNodeAttrMapKey2", nodeAttrArgs{"VANode1", "0", "mapAttr", []string{"map2"}}, false, true, `v2`},
		{"TestNodeAttrMapKey3", nodeAttrArgs{"VANode1", "0", "mapAttr", []string{"map3"}}, false, true, `v3`},
		{"TestAttrComplexTypeLit", nodeAttrArgs{"VANode1", "0", "complexAttr", []string{"literal"}}, false, true, `11`},
		{"TestAttrComplexTypeLitDef", nodeAttrArgs{"VANode1", "0", "complexAttr", []string{"literalDefault"}}, false, true, `VANode1LitDef`},
		{"TestAttrComplexTypeAll", nodeAttrArgs{"VANode1", "0", "complexAttr", nil}, false, true, `{"literal":"11","literalDefault":"VANode1LitDef"}`},
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
			found, got, err := GetInstanceAttribute(kv, deploymentID, tt.args.nodeName, tt.args.instanceName, tt.args.attributeName, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInstanceAttribute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && found != tt.wantFound {
				t.Errorf("GetInstanceAttribute() found result = %v, want %v", found, tt.wantFound)
			}
			if err == nil && got != tt.want {
				t.Errorf("GetInstanceAttribute() = %q, want %q", got, tt.want)
			}
		})
	}

	// Then test relationship properties
	type relPropArgs struct {
		nodeName     string
		reqIndex     string
		propertyName string
		nestedKeys   []string
	}
	relPropTests := []struct {
		name      string
		args      relPropArgs
		wantErr   bool
		wantFound bool
		want      string
	}{
		{"TestRelationshipPropLitDefaultNode2", relPropArgs{"VANode2", "0", "literalDefault", nil}, false, true, `relDefault`},
		{"TestRelationshipPropMapDefaultAllNode2", relPropArgs{"VANode2", "0", "mapPropDefault", nil}, false, true, `{"relProp1":"relPropVal1","relProp2":"relPropVal2"}`},
		{"TestRelationshipPropMapDefaultKey1Node2", relPropArgs{"VANode2", "0", "mapPropDefault", []string{"relProp1"}}, false, true, `relPropVal1`},
		{"TestRelationshipPropMapDefaultKey2Node2", relPropArgs{"VANode2", "0", "mapPropDefault", []string{"relProp2"}}, false, true, `relPropVal2`},
		{"TestRelationshipPropListDefaultAllNode2", relPropArgs{"VANode2", "0", "listPropDefault", nil}, false, true, `["relPropI1","relPropI2","relPropI3"]`},
		{"TestRelationshipPropListDefaultKey1Node2", relPropArgs{"VANode2", "0", "listPropDefault", []string{"0"}}, false, true, `relPropI1`},
		{"TestRelationshipPropListDefaultKey2Node2", relPropArgs{"VANode2", "0", "listPropDefault", []string{"1"}}, false, true, `relPropI2`},
		{"TestRelationshipPropListDefaultKey3Node2", relPropArgs{"VANode2", "0", "listPropDefault", []string{"2"}}, false, true, `relPropI3`},
		{"TestRelationshipPropPropNotFound", relPropArgs{"VANode2", "0", "doesnotexist", nil}, false, false, ``},
		{"TestRelationshipPropListIndexNotFound", relPropArgs{"VANode2", "0", "listPropDefault", []string{"42"}}, false, false, ``},
		{"TestRelationshipPropMapKeyNotFound", relPropArgs{"VANode2", "0", "mapPropDefault", []string{"42"}}, false, false, ``},
		{"TestRelationshipPropLiteral", relPropArgs{"VANode2", "0", "literal", nil}, false, true, `user rel literal`},
		{"TestRelationshipPropMapAll", relPropArgs{"VANode2", "0", "mapProp", nil}, false, true, `{"U1":"V1","U2":"V2"}`},
		{"TestRelationshipPropMapKey1", relPropArgs{"VANode2", "0", "mapProp", []string{"U1"}}, false, true, `V1`},
		{"TestRelationshipPropMapKey2", relPropArgs{"VANode2", "0", "mapProp", []string{"U2"}}, false, true, `V2`},
		{"TestRelationshipPropListAll", relPropArgs{"VANode2", "0", "listProp", nil}, false, true, `["UV1","UV2","UV3"]`},
		{"TestRelationshipPropListIndex0", relPropArgs{"VANode2", "0", "listProp", []string{"0"}}, false, true, `UV1`},
		{"TestRelationshipPropListIndex1", relPropArgs{"VANode2", "0", "listProp", []string{"1"}}, false, true, `UV2`},
		{"TestRelationshipPropListIndex2", relPropArgs{"VANode2", "0", "listProp", []string{"2"}}, false, true, `UV3`},
		{"TestRelationshipPropComplexTypeLit", relPropArgs{"VANode2", "0", "complex", []string{"literal"}}, false, true, `5`},
		{"TestRelationshipPropComplexTypeLitDef", relPropArgs{"VANode2", "0", "complex", []string{"literalDefault"}}, false, true, `VANode2ToVANode1`},
		{"TestRelationshipPropComplexTypeAll", relPropArgs{"VANode2", "0", "complex", nil}, false, true, `{"literal":"5","literalDefault":"VANode2ToVANode1"}`},
		{"TestRelationshipPropComplexDefaultAll", relPropArgs{"VANode2", "0", "complexDef", nil}, false, true, `{"literal":"1","literalDefault":"ComplexDataTypeDefault"}`},
		{"TestRelationshipPropComplexDefaultFromDT", relPropArgs{"VANode2", "0", "complexDef", []string{"literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		{"TestRelationshipPropComplexDTNestedListOfString", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType", "listofstring"}}, false, true, `["VANode2ToVANode1L1","VANode2ToVANode1L2"]`},
		{"TestRelationshipPropComplexDTNestedSubComplexLiteral", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType", "subcomplex", "literal"}}, false, true, `2`},
		{"TestRelationshipPropComplexDTNestedSubComplexLiteralDefault", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType", "subcomplex", "literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		{"TestRelationshipPropComplexDTNestedSubComplexAll", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType", "subcomplex"}}, false, true, `{"literal":"2","literalDefault":"ComplexDataTypeDefault"}`},
		{"TestRelationshipPropComplexDTNestedListOfComplex0Literal", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType", "listofcomplex", "0", "literal"}}, false, true, `2`},
		{"TestRelationshipPropComplexDTNestedListOfComplex0MyMap", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType", "listofcomplex", "0", "mymap"}}, false, true, `{"VANode2ToVANode1":"1"}`},
		{"TestRelationshipPropComplexDTNestedListOfComplex0All", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType", "listofcomplex", "0"}}, false, true, `{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"1"}}`},
		{"TestRelationshipPropComplexDTNestedListOfComplex1Literal", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType", "listofcomplex", "1", "literal"}}, false, true, `3`},
		{"TestRelationshipPropComplexDTNestedListOfComplex1MyMap", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType", "listofcomplex", "1", "mymap"}}, false, true, `{"VANode2ToVANode1":"2"}`},
		{"TestRelationshipPropComplexDTNestedListOfComplex1All", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType", "listofcomplex", "1"}}, false, true, `{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"2"}}`},
		{"TestRelationshipPropComplexDTNestedListOfComplexAll", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType", "listofcomplex"}}, false, true, `[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"2"}}]`},
		{"TestRelationshipPropComplexDTNestedMapOfComplex1Literal", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType", "mapofcomplex", "m1", "literal"}}, false, true, `4`},
		{"TestRelationshipPropComplexDTNestedMapOfComplex1MyMap", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType", "mapofcomplex", "m1", "mymap"}}, false, true, `{"VANode2ToVANode1":"3"}`},
		{"TestRelationshipPropComplexDTNestedMapOfComplex1All", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType", "mapofcomplex", "m1"}}, false, true, `{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"3"}}`},
		{"TestRelationshipPropComplexDTNestedMapOfComplexAll", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType", "mapofcomplex"}}, false, true, `{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"3"}}}`},
		{"TestRelationshipPropComplexDTNestedAll", relPropArgs{"VANode2", "0", "baseComplex", []string{"nestedType"}}, false, true, `{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"2"}}],"listofstring":["VANode2ToVANode1L1","VANode2ToVANode1L2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}`},
		{"TestRelationshipPropComplexDTAll", relPropArgs{"VANode2", "0", "baseComplex", nil}, false, true, `{"nestedType":{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"2"}}],"listofstring":["VANode2ToVANode1L1","VANode2ToVANode1L2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}}`},
	}
	for _, tt := range relPropTests {
		t.Run(tt.name, func(t *testing.T) {
			found, got, err := GetRelationshipPropertyFromRequirement(kv, deploymentID, tt.args.nodeName, tt.args.reqIndex, tt.args.propertyName, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRelationshipPropertyFromRequirement() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && found != tt.wantFound {
				t.Errorf("GetRelationshipPropertyFromRequirement() found result = %v, want %v", found, tt.wantFound)
			}
			if err == nil && got != tt.want {
				t.Errorf("GetRelationshipPropertyFromRequirement() = %q, want %q", got, tt.want)
			}
		})
	}

	// Then test relationship attributes value assignment
	err = SetInstanceRelationshipAttribute(deploymentID, "VANode2", "0", "0", "literalAttr", "user rel literal attr")
	require.NoError(t, err)
	err = SetInstanceRelationshipAttributeComplex(deploymentID, "VANode2", "0", "0", "mapAttr", map[interface{}]string{"U1": "V1", "U2": "V2"})
	require.NoError(t, err)
	err = SetInstanceRelationshipAttributeComplex(deploymentID, "VANode2", "0", "0", "listAttr", []interface{}{"UV1", "UV2", "UV3"})
	require.NoError(t, err)
	err = SetInstanceRelationshipAttributeComplex(deploymentID, "VANode2", "0", "0", "complexAttr", map[string]interface{}{"literal": 5, "literalDefault": "VANode2ToVANode1"})
	require.NoError(t, err)
	err = SetInstanceRelationshipAttributeComplex(deploymentID, "VANode2", "0", "0", "baseComplexAttr", map[string]interface{}{
		"nestedType": map[string]interface{}{
			"listofstring":  []string{"VANode2ToVANode1L1", "VANode2ToVANode1L2"},
			"subcomplex":    map[string]interface{}{"literal": 2},
			"listofcomplex": []interface{}{map[string]interface{}{"literal": 2, "mymap": map[string]interface{}{"VANode2ToVANode1": 1}}, map[string]interface{}{"literal": 3, "mymap": map[string]interface{}{"VANode2ToVANode1": 2}}},
			"mapofcomplex":  map[string]interface{}{"m1": map[string]interface{}{"literal": 4, "mymap": map[string]interface{}{"VANode2ToVANode1": 3}}},
		},
	})
	require.NoError(t, err)
	type relAttrArgs struct {
		nodeName      string
		instanceName  string
		reqIndex      string
		attributeName string
		nestedKeys    []string
	}
	relAttrTests := []struct {
		name      string
		args      relAttrArgs
		wantErr   bool
		wantFound bool
		want      string
	}{
		{"TestRelationshipAttrLitDefaultNode2", relAttrArgs{"VANode2", "0", "0", "literalDefault", nil}, false, true, `relDefault`},
		{"TestRelationshipAttrMapDefaultAllNode2", relAttrArgs{"VANode2", "0", "0", "mapAttrDefault", nil}, false, true, `{"relAttr1":"relAttrVal1","relAttr2":"relAttrVal2"}`},
		{"TestRelationshipAttrMapDefaultKey1Node2", relAttrArgs{"VANode2", "0", "0", "mapAttrDefault", []string{"relAttr1"}}, false, true, `relAttrVal1`},
		{"TestRelationshipAttrMapDefaultKey2Node2", relAttrArgs{"VANode2", "0", "0", "mapAttrDefault", []string{"relAttr2"}}, false, true, `relAttrVal2`},
		{"TestRelationshipAttrListDefaultAllNode2", relAttrArgs{"VANode2", "0", "0", "listAttrDefault", nil}, false, true, `["relAttrI1","relAttrI2","relAttrI3"]`},
		{"TestRelationshipAttrListDefaultKey1Node2", relAttrArgs{"VANode2", "0", "0", "listAttrDefault", []string{"0"}}, false, true, `relAttrI1`},
		{"TestRelationshipAttrListDefaultKey2Node2", relAttrArgs{"VANode2", "0", "0", "listAttrDefault", []string{"1"}}, false, true, `relAttrI2`},
		{"TestRelationshipAttrListDefaultKey3Node2", relAttrArgs{"VANode2", "0", "0", "listAttrDefault", []string{"2"}}, false, true, `relAttrI3`},
		{"TestRelationshipAttrAttrNotFound", relAttrArgs{"VANode2", "0", "0", "doesnotexist", nil}, false, false, ``},
		{"TestRelationshipAttrListIndexNotFound", relAttrArgs{"VANode2", "0", "0", "listAttrDefault", []string{"42"}}, false, false, ``},
		{"TestRelationshipAttrMapKeyNotFound", relAttrArgs{"VANode2", "0", "0", "mapAttrDefault", []string{"42"}}, false, false, ``},
		{"TestRelationshipAttrliteralAttr", relAttrArgs{"VANode2", "0", "0", "literalAttr", nil}, false, true, `user rel literal attr`},
		{"TestRelationshipAttrMapAll", relAttrArgs{"VANode2", "0", "0", "mapAttr", nil}, false, true, `{"U1":"V1","U2":"V2"}`},
		{"TestRelationshipAttrMapKey1", relAttrArgs{"VANode2", "0", "0", "mapAttr", []string{"U1"}}, false, true, `V1`},
		{"TestRelationshipAttrMapKey2", relAttrArgs{"VANode2", "0", "0", "mapAttr", []string{"U2"}}, false, true, `V2`},
		{"TestRelationshipAttrListAll", relAttrArgs{"VANode2", "0", "0", "listAttr", nil}, false, true, `["UV1","UV2","UV3"]`},
		{"TestRelationshipAttrListIndex0", relAttrArgs{"VANode2", "0", "0", "listAttr", []string{"0"}}, false, true, `UV1`},
		{"TestRelationshipAttrListIndex1", relAttrArgs{"VANode2", "0", "0", "listAttr", []string{"1"}}, false, true, `UV2`},
		{"TestRelationshipAttrListIndex2", relAttrArgs{"VANode2", "0", "0", "listAttr", []string{"2"}}, false, true, `UV3`},
		{"TestRelationshipAttrComplexTypeLit", relAttrArgs{"VANode2", "0", "0", "complexAttr", []string{"literal"}}, false, true, `5`},
		{"TestRelationshipAttrComplexTypeLitDef", relAttrArgs{"VANode2", "0", "0", "complexAttr", []string{"literalDefault"}}, false, true, `VANode2ToVANode1`},
		{"TestRelationshipAttrComplexTypeAll", relAttrArgs{"VANode2", "0", "0", "complexAttr", nil}, false, true, `{"literal":"5","literalDefault":"VANode2ToVANode1"}`},
		{"TestRelationshipAttrComplexDefaultAll", relAttrArgs{"VANode2", "0", "0", "complexDefAttr", nil}, false, true, `{"literal":"1","literalDefault":"ComplexDataTypeDefault"}`},
		{"TestRelationshipAttrComplexDefaultFromDT", relAttrArgs{"VANode2", "0", "0", "complexDefAttr", []string{"literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		{"TestRelationshipAttrComplexDTNestedListOfString", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType", "listofstring"}}, false, true, `["VANode2ToVANode1L1","VANode2ToVANode1L2"]`},
		{"TestRelationshipAttrComplexDTNestedSubComplexLiteral", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType", "subcomplex", "literal"}}, false, true, `2`},
		{"TestRelationshipAttrComplexDTNestedSubComplexLiteralDefault", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType", "subcomplex", "literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		{"TestRelationshipAttrComplexDTNestedSubComplexAll", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType", "subcomplex"}}, false, true, `{"literal":"2","literalDefault":"ComplexDataTypeDefault"}`},
		{"TestRelationshipAttrComplexDTNestedListOfComplex0Literal", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex", "0", "literal"}}, false, true, `2`},
		{"TestRelationshipAttrComplexDTNestedListOfComplex0MyMap", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex", "0", "mymap"}}, false, true, `{"VANode2ToVANode1":"1"}`},
		{"TestRelationshipAttrComplexDTNestedListOfComplex0All", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex", "0"}}, false, true, `{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"1"}}`},
		{"TestRelationshipAttrComplexDTNestedListOfComplex1Literal", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex", "1", "literal"}}, false, true, `3`},
		{"TestRelationshipAttrComplexDTNestedListOfComplex1MyMap", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex", "1", "mymap"}}, false, true, `{"VANode2ToVANode1":"2"}`},
		{"TestRelationshipAttrComplexDTNestedListOfComplex1All", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex", "1"}}, false, true, `{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"2"}}`},
		{"TestRelationshipAttrComplexDTNestedListOfComplexAll", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType", "listofcomplex"}}, false, true, `[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"2"}}]`},
		{"TestRelationshipAttrComplexDTNestedMapOfComplex1Literal", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType", "mapofcomplex", "m1", "literal"}}, false, true, `4`},
		{"TestRelationshipAttrComplexDTNestedMapOfComplex1MyMap", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType", "mapofcomplex", "m1", "mymap"}}, false, true, `{"VANode2ToVANode1":"3"}`},
		{"TestRelationshipAttrComplexDTNestedMapOfComplex1All", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType", "mapofcomplex", "m1"}}, false, true, `{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"3"}}`},
		{"TestRelationshipAttrComplexDTNestedMapOfComplexAll", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType", "mapofcomplex"}}, false, true, `{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"3"}}}`},
		{"TestRelationshipAttrComplexDTNestedAll", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", []string{"nestedType"}}, false, true, `{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"2"}}],"listofstring":["VANode2ToVANode1L1","VANode2ToVANode1L2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}`},
		{"TestRelationshipAttrComplexDTAll", relAttrArgs{"VANode2", "0", "0", "baseComplexAttr", nil}, false, true, `{"nestedType":{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"2"}}],"listofstring":["VANode2ToVANode1L1","VANode2ToVANode1L2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2ToVANode1":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}}`},
		// Now check that we reflect properties as attributes
		{"TestRelationshipPropAsAttrLitDefaultNode2", relAttrArgs{"VANode2", "0", "0", "literalDefault", nil}, false, true, `relDefault`},
		{"TestRelationshipPropAsAttrMapDefaultAllNode2", relAttrArgs{"VANode2", "0", "0", "mapPropDefault", nil}, false, true, `{"relProp1":"relPropVal1","relProp2":"relPropVal2"}`},
		{"TestRelationshipPropAsAttrListDefaultAllNode2", relAttrArgs{"VANode2", "0", "0", "listPropDefault", nil}, false, true, `["relPropI1","relPropI2","relPropI3"]`},
		{"TestRelationshipPropAsAttrLiteral", relAttrArgs{"VANode2", "0", "0", "literal", nil}, false, true, `user rel literal`},
		{"TestRelationshipPropAsAttrMapAll", relAttrArgs{"VANode2", "0", "0", "mapProp", nil}, false, true, `{"U1":"V1","U2":"V2"}`},
		{"TestRelationshipPropAsAttrListAll", relAttrArgs{"VANode2", "0", "0", "listProp", nil}, false, true, `["UV1","UV2","UV3"]`},
	}
	for _, tt := range relAttrTests {
		t.Run(tt.name, func(t *testing.T) {
			found, got, err := GetRelationshipAttributeFromRequirement(kv, deploymentID, tt.args.nodeName, tt.args.instanceName, tt.args.reqIndex, tt.args.attributeName, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRelationshipAttributeFromRequirement() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && found != tt.wantFound {
				t.Errorf("GetRelationshipAttributeFromRequirement() found result = %v, want %v", found, tt.wantFound)
			}
			if err == nil && got != tt.want {
				t.Errorf("GetRelationshipAttributeFromRequirement() = %q, want %q", got, tt.want)
			}
		})
	}

	// Then test capabilities properties value assignment
	type capPropArgs struct {
		nodeName     string
		capability   string
		propertyName string
		nestedKeys   []string
	}
	capPropTests := []struct {
		name      string
		args      capPropArgs
		wantErr   bool
		wantFound bool
		want      string
	}{
		{"TestCapabilityPropLitDefaultNode1", capPropArgs{"VANode1", "host", "literalDefault", nil}, false, true, `capDefault`},
		{"TestCapabilityPropLitDefaultNode2", capPropArgs{"VANode2", "host", "literalDefault", nil}, false, true, `capDefault`},
		{"TestCapabilityPropMapDefaultAllNode1", capPropArgs{"VANode1", "host", "mapPropDefault", nil}, false, true, `{"capProp1":"capPropVal1","capProp2":"capPropVal2"}`},
		{"TestCapabilityPropMapDefaultAllNode2", capPropArgs{"VANode2", "host", "mapPropDefault", nil}, false, true, `{"capProp1":"capPropVal1","capProp2":"capPropVal2"}`},
		{"TestCapabilityPropMapDefaultKey1Node1", capPropArgs{"VANode1", "host", "mapPropDefault", []string{"capProp1"}}, false, true, `capPropVal1`},
		{"TestCapabilityPropMapDefaultKey2Node1", capPropArgs{"VANode1", "host", "mapPropDefault", []string{"capProp2"}}, false, true, `capPropVal2`},
		{"TestCapabilityPropListDefaultAllNode1", capPropArgs{"VANode1", "host", "listPropDefault", nil}, false, true, `["capPropI1","capPropI2","capPropI3"]`},
		{"TestCapabilityPropListDefaultAllNode2", capPropArgs{"VANode2", "host", "listPropDefault", nil}, false, true, `["capPropI1","capPropI2","capPropI3"]`},
		{"TestCapabilityPropListDefaultKey1Node1", capPropArgs{"VANode1", "host", "listPropDefault", []string{"0"}}, false, true, `capPropI1`},
		{"TestCapabilityPropListDefaultKey2Node1", capPropArgs{"VANode1", "host", "listPropDefault", []string{"1"}}, false, true, `capPropI2`},
		{"TestCapabilityPropListDefaultKey3Node1", capPropArgs{"VANode1", "host", "listPropDefault", []string{"2"}}, false, true, `capPropI3`},
		{"TestCapabilityPropPropNotFound", capPropArgs{"VANode1", "host", "doesnotexist", nil}, false, false, ``},
		{"TestCapabilityPropListIndexNotFound", capPropArgs{"VANode1", "host", "listPropDefault", []string{"42"}}, false, false, ``},
		{"TestCapabilityPropMapKeyNotFound", capPropArgs{"VANode1", "host", "mapPropDefault", []string{"42"}}, false, false, ``},
		{"TestCapabilityPropLiteral", capPropArgs{"VANode1", "host", "literal", nil}, false, true, `user cap literal`},
		{"TestCapabilityPropMapAll", capPropArgs{"VANode1", "host", "mapProp", nil}, false, true, `{"U1":"V1","U2":"V2"}`},
		{"TestCapabilityPropMapKey1", capPropArgs{"VANode1", "host", "mapProp", []string{"U1"}}, false, true, `V1`},
		{"TestCapabilityPropMapKey2", capPropArgs{"VANode1", "host", "mapProp", []string{"U2"}}, false, true, `V2`},
		{"TestCapabilityPropListAll", capPropArgs{"VANode1", "host", "listProp", nil}, false, true, `["UV1","UV2","UV3"]`},
		{"TestCapabilityPropListIndex0", capPropArgs{"VANode1", "host", "listProp", []string{"0"}}, false, true, `UV1`},
		{"TestCapabilityPropListIndex1", capPropArgs{"VANode1", "host", "listProp", []string{"1"}}, false, true, `UV2`},
		{"TestCapabilityPropListIndex2", capPropArgs{"VANode1", "host", "listProp", []string{"2"}}, false, true, `UV3`},
		{"TestCapabilityPropComplexTypeLit", capPropArgs{"VANode1", "host", "complex", []string{"literal"}}, false, true, `5`},
		{"TestCapabilityPropComplexTypeLitDef", capPropArgs{"VANode1", "host", "complex", []string{"literalDefault"}}, false, true, `CapNode1`},
		{"TestCapabilityPropComplexTypeAll", capPropArgs{"VANode1", "host", "complex", nil}, false, true, `{"literal":"5","literalDefault":"CapNode1"}`},
		{"TestCapabilityPropComplexDefaultAll", capPropArgs{"VANode1", "host", "complexDef", nil}, false, true, `{"literal":"1","literalDefault":"ComplexDataTypeDefault"}`},
		{"TestCapabilityPropComplexDefaultFromDT", capPropArgs{"VANode1", "host", "complexDef", []string{"literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		{"TestCapabilityPropComplexDTNestedListOfString", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType", "listofstring"}}, false, true, `["CapNode1L1","CapNode1L2"]`},
		{"TestCapabilityPropComplexDTNestedSubComplexLiteral", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType", "subcomplex", "literal"}}, false, true, `2`},
		{"TestCapabilityPropComplexDTNestedSubComplexLiteralDefault", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType", "subcomplex", "literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		{"TestCapabilityPropComplexDTNestedSubComplexAll", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType", "subcomplex"}}, false, true, `{"literal":"2","literalDefault":"ComplexDataTypeDefault"}`},
		{"TestCapabilityPropComplexDTNestedListOfComplex0Literal", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType", "listofcomplex", "0", "literal"}}, false, true, `2`},
		{"TestCapabilityPropComplexDTNestedListOfComplex0MyMap", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType", "listofcomplex", "0", "mymap"}}, false, true, `{"CapNode1":"1"}`},
		{"TestCapabilityPropComplexDTNestedListOfComplex0All", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType", "listofcomplex", "0"}}, false, true, `{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"1"}}`},
		{"TestCapabilityPropComplexDTNestedListOfComplex1Literal", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType", "listofcomplex", "1", "literal"}}, false, true, `3`},
		{"TestCapabilityPropComplexDTNestedListOfComplex1MyMap", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType", "listofcomplex", "1", "mymap"}}, false, true, `{"CapNode1":"2"}`},
		{"TestCapabilityPropComplexDTNestedListOfComplex1All", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType", "listofcomplex", "1"}}, false, true, `{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"2"}}`},
		{"TestCapabilityPropComplexDTNestedListOfComplexAll", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType", "listofcomplex"}}, false, true, `[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"2"}}]`},
		{"TestCapabilityPropComplexDTNestedMapOfComplex1Literal", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType", "mapofcomplex", "m1", "literal"}}, false, true, `4`},
		{"TestCapabilityPropComplexDTNestedMapOfComplex1MyMap", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType", "mapofcomplex", "m1", "mymap"}}, false, true, `{"CapNode1":"3"}`},
		{"TestCapabilityPropComplexDTNestedMapOfComplex1All", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType", "mapofcomplex", "m1"}}, false, true, `{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"3"}}`},
		{"TestCapabilityPropComplexDTNestedMapOfComplexAll", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType", "mapofcomplex"}}, false, true, `{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"3"}}}`},
		{"TestCapabilityPropComplexDTNestedAll", capPropArgs{"VANode1", "host", "baseComplex", []string{"nestedType"}}, false, true, `{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"2"}}],"listofstring":["CapNode1L1","CapNode1L2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}`},
		{"TestCapabilityPropComplexDTAll", capPropArgs{"VANode1", "host", "baseComplex", nil}, false, true, `{"nestedType":{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"2"}}],"listofstring":["CapNode1L1","CapNode1L2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"CapNode1":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}}`},

		// we reflect Node 1 properties on Node 2 due to the hostedOn relationship
		{"TestCapabilityNode2PropLiteral", capPropArgs{"VANode2", "host", "literal", nil}, false, true, `user cap literal`},
		{"TestCapabilityNode2PropMapAll", capPropArgs{"VANode2", "host", "mapProp", nil}, false, true, `{"U1":"V1","U2":"V2"}`},
		{"TestCapabilityNode2PropMapKey1", capPropArgs{"VANode2", "host", "mapProp", []string{"U1"}}, false, true, `V1`},
		{"TestCapabilityNode2PropMapKey2", capPropArgs{"VANode2", "host", "mapProp", []string{"U2"}}, false, true, `V2`},
		{"TestCapabilityNode2PropListAll", capPropArgs{"VANode2", "host", "listProp", nil}, false, true, `["UV1","UV2","UV3"]`},
		{"TestCapabilityNode2PropListIndex0", capPropArgs{"VANode2", "host", "listProp", []string{"0"}}, false, true, `UV1`},
		{"TestCapabilityNode2PropListIndex1", capPropArgs{"VANode2", "host", "listProp", []string{"1"}}, false, true, `UV2`},
		{"TestCapabilityNode2PropListIndex2", capPropArgs{"VANode2", "host", "listProp", []string{"2"}}, false, true, `UV3`},
	}
	for _, tt := range capPropTests {
		t.Run(tt.name, func(t *testing.T) {
			found, got, err := GetCapabilityProperty(kv, deploymentID, tt.args.nodeName, tt.args.capability, tt.args.propertyName, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCapabilityProperty() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && found != tt.wantFound {
				t.Errorf("GetCapabilityProperty() found result = %v, want %v", found, tt.wantFound)
			}
			if err == nil && got != tt.want {
				t.Errorf("GetCapabilityProperty() = %q, want %q", got, tt.want)
			}
		})
	}

	// Then test capabilities attributes value assignment
	err = SetInstanceCapabilityAttribute(deploymentID, "VANode1", "0", "host", "literalAttr", "user cap literal attr")
	require.NoError(t, err)
	err = SetInstanceCapabilityAttributeComplex(deploymentID, "VANode1", "0", "host", "mapAttr", map[interface{}]string{"U1": "V1", "U2": "V2"})
	require.NoError(t, err)
	err = SetInstanceCapabilityAttributeComplex(deploymentID, "VANode1", "0", "host", "listAttr", []interface{}{"UV1", "UV2", "UV3"})
	require.NoError(t, err)
	err = SetInstanceCapabilityAttributeComplex(deploymentID, "VANode1", "0", "host", "complexAttr", map[string]interface{}{"literal": 5, "literalDefault": "CapNode1"})
	require.NoError(t, err)
	err = SetInstanceCapabilityAttributeComplex(deploymentID, "VANode1", "0", "host", "baseComplexAttr", map[string]interface{}{
		"nestedType": map[string]interface{}{
			"listofstring":  []string{"CapNode1L1", "CapNode1L2"},
			"subcomplex":    map[string]interface{}{"literal": 2},
			"listofcomplex": []interface{}{map[string]interface{}{"literal": 2, "mymap": map[string]interface{}{"CapNode1": 1}}, map[string]interface{}{"literal": 3, "mymap": map[string]interface{}{"CapNode1": 2}}},
			"mapofcomplex":  map[string]interface{}{"m1": map[string]interface{}{"literal": 4, "mymap": map[string]interface{}{"CapNode1": 3}}},
		},
	})
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
			found, got, err := GetInstanceCapabilityAttribute(kv, deploymentID, tt.args.nodeName, tt.args.instanceName, tt.args.capability, tt.args.attributeName, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInstanceCapabilityAttribute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && found != tt.wantFound {
				t.Errorf("GetInstanceCapabilityAttribute() found result = %v, want %v", found, tt.wantFound)
			}
			if err == nil && got != tt.want {
				t.Errorf("GetInstanceCapabilityAttribute() = %q, want %q", got, tt.want)
			}
		})
	}
	type topoInputArgs struct {
		inputName  string
		nestedKeys []string
	}
	topoInputTests := []struct {
		name    string
		args    topoInputArgs
		wantErr bool
		want    string
	}{
		{"InputValueLiteral", topoInputArgs{"literal", nil}, false, `literalInput`},
		{"InputValueLiteralDefault", topoInputArgs{"literalDefault", nil}, false, `1`},
		{"InputValueComplexDTLiteral", topoInputArgs{"complex", []string{"literal"}}, false, `11`},
		{"InputValueComplexDTLiteral", topoInputArgs{"complex", []string{"literalDefault"}}, false, `InputLitDef`},
		{"InputValueComplexDTLiteralDefault", topoInputArgs{"complex", []string{"literalDefault"}}, false, `InputLitDef`},
		{"InputValueComplexDTAll", topoInputArgs{"complex", []string{}}, false, `{"literal":"11","literalDefault":"InputLitDef"}`},
		{"InputValueComplexDefaultDTAll", topoInputArgs{"complexDef", nil}, false, `{"literal":"1","literalDefault":"ComplexDataTypeDefault"}`},
		{"InputValueComplexDefaultFromDT", topoInputArgs{"complexDef", []string{"literalDefault"}}, false, `ComplexDataTypeDefault`},
		{"InputValueBaseComplexDTNestedListOfString", topoInputArgs{"baseComplex", []string{"nestedType", "listofstring"}}, false, `["InputL1","InputL2"]`},
		{"InputValueBaseComplexDTNestedSubComplexLiteral", topoInputArgs{"baseComplex", []string{"nestedType", "subcomplex", "literal"}}, false, `2`},
		{"InputValueBaseComplexDTNestedSubComplexLiteralDefault", topoInputArgs{"baseComplex", []string{"nestedType", "subcomplex", "literalDefault"}}, false, `ComplexDataTypeDefault`},
		{"InputValueBaseComplexDTNestedSubComplexAll", topoInputArgs{"baseComplex", []string{"nestedType", "subcomplex"}}, false, `{"literal":"2","literalDefault":"ComplexDataTypeDefault"}`},
		{"InputValueBaseComplexDTNestedListOfComplex0Literal", topoInputArgs{"baseComplex", []string{"nestedType", "listofcomplex", "0", "literal"}}, false, `2`},
		{"InputValueBaseComplexDTNestedListOfComplex0MyMap", topoInputArgs{"baseComplex", []string{"nestedType", "listofcomplex", "0", "mymap"}}, false, `{"Input":"1"}`},
		{"InputValueBaseComplexDTNestedListOfComplex0All", topoInputArgs{"baseComplex", []string{"nestedType", "listofcomplex", "0"}}, false, `{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"Input":"1"}}`},
		{"InputValueBaseComplexDTNestedListOfComplex1Literal", topoInputArgs{"baseComplex", []string{"nestedType", "listofcomplex", "1", "literal"}}, false, `3`},
		{"InputValueBaseComplexDTNestedListOfComplex1MyMap", topoInputArgs{"baseComplex", []string{"nestedType", "listofcomplex", "1", "mymap"}}, false, `{"Input":"2"}`},
		{"InputValueBaseComplexDTNestedListOfComplex1All", topoInputArgs{"baseComplex", []string{"nestedType", "listofcomplex", "1"}}, false, `{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"Input":"2"}}`},
		{"InputValueBaseComplexDTNestedListOfComplexAll", topoInputArgs{"baseComplex", []string{"nestedType", "listofcomplex"}}, false, `[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"Input":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"Input":"2"}}]`},
		{"InputValueBaseComplexDTNestedMapOfComplex1Literal", topoInputArgs{"baseComplex", []string{"nestedType", "mapofcomplex", "m1", "literal"}}, false, `4`},
		{"InputValueBaseComplexDTNestedMapOfComplex1MyMap", topoInputArgs{"baseComplex", []string{"nestedType", "mapofcomplex", "m1", "mymap"}}, false, `{"Input":"3"}`},
		{"InputValueBaseComplexDTNestedMapOfComplex1All", topoInputArgs{"baseComplex", []string{"nestedType", "mapofcomplex", "m1"}}, false, `{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"Input":"3"}}`},
		{"InputValueBaseComplexDTNestedMapOfComplexAll", topoInputArgs{"baseComplex", []string{"nestedType", "mapofcomplex"}}, false, `{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"Input":"3"}}}`},
		{"InputValueBaseComplexDTNestedAll", topoInputArgs{"baseComplex", []string{"nestedType"}}, false, `{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"Input":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"Input":"2"}}],"listofstring":["InputL1","InputL2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"Input":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}`},
		{"InputValueBaseComplexDTAll", topoInputArgs{"baseComplex", nil}, false, `{"nestedType":{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"Input":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"Input":"2"}}],"listofstring":["InputL1","InputL2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"Input":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}}`},
	}
	for _, tt := range topoInputTests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetInputValue(kv, deploymentID, tt.args.inputName, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInputValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && got != tt.want {
				t.Errorf("GetInputValue() = %q, want %q", got, tt.want)
			}
		})
	}

	type topoOutputArgs struct {
		OutputName string
		nestedKeys []string
	}
	topoOutputTests := []struct {
		name      string
		args      topoOutputArgs
		wantErr   bool
		wantFound bool
		want      string
	}{
		// {"OutputValueLiteral", topoOutputArgs{"literal", nil}, false, true, `literalOutput`},
		// {"OutputValueLiteralDefault", topoOutputArgs{"literalDefault", nil}, false, true, `1`},
		// {"OutputValueComplexDTLiteral", topoOutputArgs{"complex", []string{"literal"}}, false, true, `11`},
		// {"OutputValueComplexDTLiteral", topoOutputArgs{"complex", []string{"literalDefault"}}, false, true, `OutputLitDef`},
		// {"OutputValueComplexDTLiteralDefault", topoOutputArgs{"complex", []string{"literalDefault"}}, false, true, `OutputLitDef`},
		// {"OutputValueComplexDTAll", topoOutputArgs{"complex", []string{}}, false, true, `{"literal":"11","literalDefault":"OutputLitDef"}`},
		// {"OutputValueComplexDefaultDTAll", topoOutputArgs{"complexDef", nil}, false, true, `{"literal":"1","literalDefault":"ComplexDataTypeDefault"}`},
		// {"OutputValueComplexDefaultFromDT", topoOutputArgs{"complexDef", []string{"literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		// {"OutputValueBaseComplexDTNestedListOfString", topoOutputArgs{"baseComplex", []string{"nestedType", "listofstring"}}, false, true, `["OutputL1","OutputL2"]`},
		// {"OutputValueBaseComplexDTNestedSubComplexLiteral", topoOutputArgs{"baseComplex", []string{"nestedType", "subcomplex", "literal"}}, false, true, `2`},
		// {"OutputValueBaseComplexDTNestedSubComplexLiteralDefault", topoOutputArgs{"baseComplex", []string{"nestedType", "subcomplex", "literalDefault"}}, false, true, `ComplexDataTypeDefault`},
		// {"OutputValueBaseComplexDTNestedSubComplexAll", topoOutputArgs{"baseComplex", []string{"nestedType", "subcomplex"}}, false, true, `{"literal":"2","literalDefault":"ComplexDataTypeDefault"}`},
		// {"OutputValueBaseComplexDTNestedListOfComplex0Literal", topoOutputArgs{"baseComplex", []string{"nestedType", "listofcomplex", "0", "literal"}}, false, true, `2`},
		// {"OutputValueBaseComplexDTNestedListOfComplex0MyMap", topoOutputArgs{"baseComplex", []string{"nestedType", "listofcomplex", "0", "mymap"}}, false, true, `{"Output":"1"}`},
		// {"OutputValueBaseComplexDTNestedListOfComplex0All", topoOutputArgs{"baseComplex", []string{"nestedType", "listofcomplex", "0"}}, false, true, `{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"Output":"1"}}`},
		// {"OutputValueBaseComplexDTNestedListOfComplex1Literal", topoOutputArgs{"baseComplex", []string{"nestedType", "listofcomplex", "1", "literal"}}, false, true, `3`},
		// {"OutputValueBaseComplexDTNestedListOfComplex1MyMap", topoOutputArgs{"baseComplex", []string{"nestedType", "listofcomplex", "1", "mymap"}}, false, true, `{"Output":"2"}`},
		// {"OutputValueBaseComplexDTNestedListOfComplex1All", topoOutputArgs{"baseComplex", []string{"nestedType", "listofcomplex", "1"}}, false, true, `{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"Output":"2"}}`},
		// {"OutputValueBaseComplexDTNestedListOfComplexAll", topoOutputArgs{"baseComplex", []string{"nestedType", "listofcomplex"}}, false, true, `[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"Output":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"Output":"2"}}]`},
		// {"OutputValueBaseComplexDTNestedMapOfComplex1Literal", topoOutputArgs{"baseComplex", []string{"nestedType", "mapofcomplex", "m1", "literal"}}, false, true, `4`},
		// {"OutputValueBaseComplexDTNestedMapOfComplex1MyMap", topoOutputArgs{"baseComplex", []string{"nestedType", "mapofcomplex", "m1", "mymap"}}, false, true, `{"Output":"3"}`},
		// {"OutputValueBaseComplexDTNestedMapOfComplex1All", topoOutputArgs{"baseComplex", []string{"nestedType", "mapofcomplex", "m1"}}, false, true, `{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"Output":"3"}}`},
		// {"OutputValueBaseComplexDTNestedMapOfComplexAll", topoOutputArgs{"baseComplex", []string{"nestedType", "mapofcomplex"}}, false, true, `{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"Output":"3"}}}`},
		// {"OutputValueBaseComplexDTNestedAll", topoOutputArgs{"baseComplex", []string{"nestedType"}}, false, true, `{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"Output":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"Output":"2"}}],"listofstring":["OutputL1","OutputL2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"Output":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}`},
		// {"OutputValueBaseComplexDTAll", topoOutputArgs{"baseComplex", nil}, false, true, `{"nestedType":{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"Output":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"Output":"2"}}],"listofstring":["OutputL1","OutputL2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"Output":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}}`},

		// Test functions
		// {"OutputFunctionLiteral", topoOutputArgs{"node1Lit", nil}, false, true, `myLiteral`},
		{"OutputFunctionNode2BaseComplexPropAll", topoOutputArgs{"node2BaseComplexPropAll", nil}, false, true, `{"nestedType":{"listofcomplex":[{"literal":"2","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"1"}},{"literal":"3","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"2"}}],"listofstring":["VANode2L1","VANode2L2"],"mapofcomplex":{"m1":{"literal":"4","literalDefault":"ComplexDataTypeDefault","mymap":{"VANode2":"3"}}},"subcomplex":{"literal":"2","literalDefault":"ComplexDataTypeDefault"}}}`},
		// {"OutputFunctionNode2BaseComplexPropNestedSubComplexLiteral", topoOutputArgs{"node2BaseComplexPropNestedSubComplexLiteral", nil}, false, true, `2`},
	}
	for _, tt := range topoOutputTests {
		t.Run(tt.name, func(t *testing.T) {
			found, got, err := GetTopologyOutput(kv, deploymentID, tt.args.OutputName, tt.args.nestedKeys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTopologyOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && found != tt.wantFound {
				t.Errorf("GetTopologyOutput() found result = %v, want %v", found, tt.wantFound)
			}
			if err == nil && got != tt.want {
				t.Errorf("GetTopologyOutput() = %q, want %q", got, tt.want)
			}
		})
	}

}

func testIssueGetEmptyPropRel(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/issue_get_empty_prop_rel.yaml")
	require.Nil(t, err)
	// First test operation outputs detection

	results, err := GetOperationInput(kv, deploymentID, "ValueAssignmentNode2", prov.Operation{
		Name:                   "tosca.interfaces.relationship.configure.pre_configure_target",
		ImplementedInType:      "janus.tests.relationships.ValueAssignmentConnectsTo",
		ImplementationArtifact: "",
		RelOp: prov.RelationshipOperation{
			IsRelationshipOperation: true,
			RequirementIndex:        "1",
			TargetNodeName:          "ValueAssignmentNode1",
		}}, "input_empty")
	require.Nil(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "", results[0].Value)
}

func testRelationshipWorkflow(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/relationship_workflow.yaml")
	require.Nil(t, err)

	workflows, err := GetWorkflows(kv, deploymentID)
	require.Nil(t, err)
	require.Equal(t, len(workflows), 4)
	require.Equal(t, workflows[0], "install")

	wfInstall, err := ReadWorkflow(kv, deploymentID, "install")
	require.Nil(t, err)
	require.Equal(t, len(wfInstall.Steps), 14)

	step := wfInstall.Steps["OracleJDK_hostedOnComputeHost_pre_configure_source"]
	require.Equal(t, step.Target, "OracleJDK")
	require.Equal(t, step.OperationHost, "SOURCE")
	require.Equal(t, step.TargetRelationShip, "hostedOnComputeHost")

}
