package deployments

import (
	"context"
	"path"
	"testing"

	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"

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
		// we reflect Node 1 attributes on Node 2 due to the hostedOn relationship
		{"TestCapabilityNode2AttrliteralAttr", capAttrArgs{"VANode2", "0", "host", "literalAttr", nil}, false, true, `user cap literal attr`},
		{"TestCapabilityNode2AttrMapAll", capAttrArgs{"VANode2", "0", "host", "mapAttr", nil}, false, true, `{"U1":"V1","U2":"V2"}`},
		{"TestCapabilityNode2AttrMapKey1", capAttrArgs{"VANode2", "0", "host", "mapAttr", []string{"U1"}}, false, true, `V1`},
		{"TestCapabilityNode2AttrMapKey2", capAttrArgs{"VANode2", "0", "host", "mapAttr", []string{"U2"}}, false, true, `V2`},
		{"TestCapabilityNode2AttrListAll", capAttrArgs{"VANode2", "0", "host", "listAttr", nil}, false, true, `["UV1","UV2","UV3"]`},
		{"TestCapabilityNode2AttrListIndex0", capAttrArgs{"VANode2", "0", "host", "listAttr", []string{"0"}}, false, true, `UV1`},
		{"TestCapabilityNode2AttrListIndex1", capAttrArgs{"VANode2", "0", "host", "listAttr", []string{"1"}}, false, true, `UV2`},
		{"TestCapabilityNode2AttrListIndex2", capAttrArgs{"VANode2", "0", "host", "listAttr", []string{"2"}}, false, true, `UV3`},
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

}
