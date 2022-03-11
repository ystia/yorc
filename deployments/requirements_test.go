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
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
	"reflect"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
)

func testRequirements(t *testing.T, srv1 *testutil.TestServer) {
	log.SetDebug(true)
	ctx := context.Background()
	node := tosca.NodeTemplate{Type: "tosca.nodes.Compute"}
	node.Requirements = []tosca.RequirementAssignmentMap{
		{"network": tosca.RequirementAssignment{Node: "TNode1"}},
		{"host": tosca.RequirementAssignment{Node: "TNode1"}},
		{"network_1": tosca.RequirementAssignment{Node: "TNode2", TypeRequirement: "network"}},
		{"host_1": tosca.RequirementAssignment{Node: "TNode2", TypeRequirement: "host"}},
		{"network_2": tosca.RequirementAssignment{Node: "TNode3", TypeRequirement: "network"}},
		{"storage": tosca.RequirementAssignment{Node: "TNode4"}},
		{"storage_other": tosca.RequirementAssignment{Node: "TNode5", TypeRequirement: "storage", Relationship: "my_relationship", Capability: "yorc.capabilities.Assignable"}},
		{"req_name": tosca.RequirementAssignment{Node: "myNode", TypeRequirement: "req_type"}},
	}

	err := storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1", node)
	require.Nil(t, err)

	node2 := tosca.NodeTemplate{Type: "yorc.nodes.google.Compute"}
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/TNode5", node2)
	require.Nil(t, err)

	t.Run("groupDeploymentsRequirements", func(t *testing.T) {
		t.Run("TestGetNbRequirementsForNode", func(t *testing.T) {
			testGetNbRequirementsForNode(t)
		})
		t.Run("testGetRequirementsByTypeForNode", func(t *testing.T) {
			testGetRequirementsByTypeForNode(t)
		})
		t.Run("testGetRequirementIndexByNameForNode", func(t *testing.T) {
			testGetRequirementIndexByNameForNode(t)
		})
		t.Run("testGetRequirementNameByIndexForNode", func(t *testing.T) {
			testGetRequirementNameByIndexForNode(t)
		})
		t.Run("testGetRequirementsIndexes", func(t *testing.T) {
			testGetRequirementsIndexes(t)
		})
		t.Run("testGetRelationshipForRequirement", func(t *testing.T) {
			testGetRelationshipForRequirement(t)
		})
		t.Run("testGetCapabilityForRequirement", func(t *testing.T) {
			testGetCapabilityForRequirement(t)
		})
		t.Run("testGetTargetNodeForRequirementByName", func(t *testing.T) {
			testGetTargetNodeForRequirementByName(t)
		})
		t.Run("testHasAnyRequirementCapability", func(t *testing.T) {
			testHasAnyRequirementCapability(t)
		})
		t.Run("testHasAnyRequirementFromNodeType", func(t *testing.T) {
			testHasAnyRequirementFromNodeType(t)
		})
	})
}

func testGetNbRequirementsForNode(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	reqNb, err := GetNbRequirementsForNode(ctx, "t1", "Compute1")
	require.Nil(t, err)
	require.Equal(t, 8, reqNb)

	reqNb, err = GetNbRequirementsForNode(ctx, "t1", "do_not_exits")
	require.Nil(t, err)
	require.Equal(t, 0, reqNb)

}

func testGetRequirementsByTypeForNode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name     string
		typ      string
		expected []Requirement
	}{
		{
			name: "getHostRequirements",
			typ:  "host",
			expected: []Requirement{{Name: "host", Index: "1", RequirementAssignment: tosca.RequirementAssignment{Node: "TNode1"}},
				{Name: "host_1", Index: "3", RequirementAssignment: tosca.RequirementAssignment{Node: "TNode2", TypeRequirement: "host"}}},
		},
		{
			name: "getNetworkRequirements",
			typ:  "network",
			expected: []Requirement{{Name: "network", Index: "0", RequirementAssignment: tosca.RequirementAssignment{Node: "TNode1"}},
				{Name: "network_1", Index: "2", RequirementAssignment: tosca.RequirementAssignment{Node: "TNode2", TypeRequirement: "network"}},
				{Name: "network_2", Index: "4", RequirementAssignment: tosca.RequirementAssignment{Node: "TNode3", TypeRequirement: "network"}}},
		},
		{
			name: "getStorageRequirements",
			typ:  "storage",
			expected: []Requirement{{Name: "storage", Index: "5", RequirementAssignment: tosca.RequirementAssignment{Node: "TNode4"}},
				{Name: "storage_other", Index: "6", RequirementAssignment: tosca.RequirementAssignment{Node: "TNode5", TypeRequirement: "storage", Relationship: "my_relationship", Capability: "yorc.capabilities.Assignable"}}},
		},
		{
			name:     "getNotExistingRequirements",
			typ:      "not_exists",
			expected: []Requirement{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqs, err := GetRequirementsByTypeForNode(ctx, "t1", "Compute1", tt.typ)
			require.Nil(t, err)
			require.Len(t, reqs, len(tt.expected), "unexpected nb of requirements of type %q", tt.typ)
			if !reflect.DeepEqual(reqs, tt.expected) {
				t.Errorf("GetRequirementsByTypeForNode = %v, want %v", reqs, tt.expected)
			}
		})
	}
}

func testGetRequirementIndexByNameForNode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		nameTest string
		name     string
		expected string
	}{
		{
			nameTest: "IndexForReqHost",
			name:     "host",
			expected: "1",
		},
		{
			nameTest: "IndexForReqNetwork",
			name:     "network_2",
			expected: "4",
		},
		{
			nameTest: "IndexForUnknownName",
			name:     "unknown",
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.nameTest, func(t *testing.T) {
			reqs, err := GetRequirementIndexByNameForNode(ctx, "t1", "Compute1", tt.name)
			require.Nil(t, err)
			if !reflect.DeepEqual(reqs, tt.expected) {
				t.Errorf("GetRequirementIndexByNameForNode = %v, want %v", reqs, tt.expected)
			}
		})
	}
}

func testGetRequirementNameByIndexForNode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		nameTest string
		index    string
		expected string
	}{
		{
			nameTest: "NameForIndex2",
			index:    "2",
			expected: "network_1",
		},
		{
			nameTest: "NameForIndex6",
			index:    "6",
			expected: "storage_other",
		},
		{
			nameTest: "NameForIndex12",
			index:    "12",
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.nameTest, func(t *testing.T) {
			reqs, err := GetRequirementNameByIndexForNode(ctx, "t1", "Compute1", tt.index)
			require.Nil(t, err)
			if !reflect.DeepEqual(reqs, tt.expected) {
				t.Errorf("GetRequirementIndexByNameForNode = %v, want %v", reqs, tt.expected)
			}
		})
	}
}

func testGetRequirementsIndexes(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	indexes, err := GetRequirementsIndexes(ctx, "t1", "Compute1")
	expected := []string{"0", "1", "2", "3", "4", "5", "6", "7"}
	require.Nil(t, err)
	require.Equal(t, 8, len(indexes))
	if !reflect.DeepEqual(indexes, expected) {
		t.Errorf("GetRequirementsIndexes = %v, want %v", indexes, expected)
	}
}

func testGetRelationshipForRequirement(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	relationship, err := GetRelationshipForRequirement(ctx, "t1", "Compute1", "6")
	expected := "my_relationship"
	require.Nil(t, err)
	require.Equal(t, expected, relationship)
}

func testGetCapabilityForRequirement(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	capability, err := GetCapabilityForRequirement(ctx, "t1", "Compute1", "6")
	expected := "yorc.capabilities.Assignable"
	require.Nil(t, err)
	require.Equal(t, expected, capability)
}

func testGetTargetNodeForRequirementByName(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	node, err := GetTargetNodeForRequirementByName(ctx, "t1", "Compute1", "storage_other")
	expected := "TNode5"
	require.Nil(t, err)
	require.Equal(t, expected, node)

	node, err = GetTargetNodeForRequirementByName(ctx, "t1", "Compute1", "req_type")
	expected = "myNode"
	require.Nil(t, err)
	require.Equal(t, expected, node)
}

func testHasAnyRequirementCapability(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	has, targetNode, err := HasAnyRequirementCapability(ctx, "t1", "Compute1", "storage_other", "yorc.capabilities.Assignable")
	require.Nil(t, err)
	require.Equal(t, true, has)
	require.Equal(t, "TNode5", targetNode)

	has, targetNode, err = HasAnyRequirementCapability(ctx, "t1", "Compute1", "storage_other", "tosca.capabilities.Root")
	require.Nil(t, err)
	require.Equal(t, true, has)
	require.Equal(t, "TNode5", targetNode)

}

func testHasAnyRequirementFromNodeType(t *testing.T) {
	ctx := context.Background()
	t.Parallel()
	has, targetNode, err := HasAnyRequirementFromNodeType(ctx, "t1", "Compute1", "storage_other", "yorc.nodes.Compute")
	require.Nil(t, err)
	require.Equal(t, true, has)
	require.Equal(t, "TNode5", targetNode)

	has, targetNode, err = HasAnyRequirementFromNodeType(ctx, "t1", "Compute1", "storage_other", "yorc.nodes.google.Subnetwork")
	require.Nil(t, err)
	require.Equal(t, false, has)
	require.Equal(t, "", targetNode)
}
