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
	"fmt"
	"testing"

	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"

	ctu "github.com/hashicorp/consul/sdk/testutil"

	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
)

func testDeploymentNodes(t *testing.T, srv1 *ctu.TestServer) {
	log.SetDebug(true)
	ctx := context.Background()
	// Test testIsNodeTypeDerivedFrom
	type1 := tosca.RelationshipType{
		Type: tosca.Type{
			Base:        tosca.TypeBaseRELATIONSHIP,
			DerivedFrom: "yorc.type.2",
		},
	}
	type2 := tosca.RelationshipType{
		Type: tosca.Type{
			Base:        tosca.TypeBaseRELATIONSHIP,
			DerivedFrom: "yorc.type.3",
		},
	}
	type3 := tosca.RelationshipType{
		Type: tosca.Type{
			Base:        tosca.TypeBaseRELATIONSHIP,
			DerivedFrom: "tosca.relationships.HostedOn",
		},
	}

	typeRelationshipHostedOn := tosca.RelationshipType{
		Type: tosca.Type{
			Base:        tosca.TypeBaseRELATIONSHIP,
			DerivedFrom: "tosca.relationships.Root",
		},
	}

	typeRelationshipRoot := tosca.RelationshipType{
		Type: tosca.Type{
			Base: tosca.TypeBaseRELATIONSHIP,
		},
	}

	err := storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testIsNodeTypeDerivedFrom/topology/types/yorc.type.1", type1)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testIsNodeTypeDerivedFrom/topology/types/yorc.type.2", type2)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testIsNodeTypeDerivedFrom/topology/types/yorc.type.3", type3)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testIsNodeTypeDerivedFrom/topology/types/tosca.relationships.HostedOn", typeRelationshipHostedOn)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testIsNodeTypeDerivedFrom/topology/types/tosca.relationships.Root", typeRelationshipRoot)
	require.Nil(t, err)

	// Test testGetNbInstancesForNode
	// Default case type "tosca.nodes.Compute" default_instance specified
	nodeCompute1 := tosca.NodeTemplate{
		Type: "tosca.nodes.Compute",
		Attributes: map[string]*tosca.ValueAssignment{"id": &tosca.ValueAssignment{
			Type:  0,
			Value: "Not Used as it exists in instances",
		}},
		Capabilities: map[string]tosca.CapabilityAssignment{"scalable": tosca.CapabilityAssignment{
			Properties: map[string]*tosca.ValueAssignment{"default_instances": &tosca.ValueAssignment{
				Type:  0,
				Value: "10",
			}, "max_instances": &tosca.ValueAssignment{
				Type:  0,
				Value: "20",
			}, "min_instances": &tosca.ValueAssignment{
				Type:  0,
				Value: "2",
			}}},
		},
	}
	// Case type "tosca.nodes.Compute" default_instance not specified (1 assumed)
	nodeCompute2 := tosca.NodeTemplate{
		Type: "tosca.nodes.Compute",
	}
	// Error case default_instance specified but not an uint
	nodeCompute3 := tosca.NodeTemplate{
		Type: "tosca.nodes.Compute",
		Attributes: map[string]*tosca.ValueAssignment{"id": &tosca.ValueAssignment{
			Type:  0,
			Value: "Not Used as it exists in instances",
		}},
		Capabilities: map[string]tosca.CapabilityAssignment{"scalable": tosca.CapabilityAssignment{
			Properties: map[string]*tosca.ValueAssignment{"default_instances": &tosca.ValueAssignment{
				Type:  0,
				Value: "-10",
			}, "max_instances": &tosca.ValueAssignment{
				Type:  0,
				Value: "-15",
			}, "min_instances": &tosca.ValueAssignment{
				Type:  0,
				Value: "-15",
			}}},
		},
	}

	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/nodes/Compute1", nodeCompute1)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/nodes/Compute2", nodeCompute2)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/nodes/Compute3", nodeCompute3)
	require.Nil(t, err)
	// Case Node Hosted on another node
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/types/yorc.type.1", type1)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/types/yorc.type.2", type2)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/types/yorc.type.3", type3)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/types/tosca.relationships.HostedOn", typeRelationshipHostedOn)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/types/tosca.relationships.Root", typeRelationshipRoot)
	require.Nil(t, err)

	typeCompute := tosca.NodeType{
		Type: tosca.Type{
			Base:        tosca.TypeBaseNODE,
			DerivedFrom: "tosca.nodes.Root",
		},
		Attributes: map[string]tosca.AttributeDefinition{"id": tosca.AttributeDefinition{Default: &tosca.ValueAssignment{
			Type:  0,
			Value: "DefaultComputeTypeid",
		}}, "ip": tosca.AttributeDefinition{Default: &tosca.ValueAssignment{
			Type:  0,
			Value: "",
		}}},
		Capabilities: map[string]tosca.CapabilityDefinition{"scalable": tosca.CapabilityDefinition{Type: "tosca.capabilities.Scalable"}},
	}

	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/types/tosca.nodes.Compute", typeCompute)
	require.Nil(t, err)

	scalableCap := tosca.NodeType{
		Type: tosca.Type{Base: tosca.TypeBaseCAPABILITY},
	}

	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/types/tosca.capabilities.Scalable", scalableCap)
	require.Nil(t, err)

	derivedSC1 := tosca.NodeType{
		Type: tosca.Type{
			Base:        tosca.TypeBaseNODE,
			DerivedFrom: "tosca.nodes.SoftwareComponent",
		},
	}
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC1", derivedSC1)
	require.Nil(t, err)

	derivedSC2 := tosca.NodeType{
		Type: tosca.Type{
			Base:        tosca.TypeBaseNODE,
			DerivedFrom: "tosca.nodes.SoftwareComponent",
		},
		Attributes: map[string]tosca.AttributeDefinition{"dsc2": tosca.AttributeDefinition{
			Default: &tosca.ValueAssignment{Value: "yorc.type.DerivedSC2"},
		}},
	}
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC2", derivedSC2)
	require.Nil(t, err)

	derivedSC3 := tosca.NodeType{
		Type: tosca.Type{
			Base:        tosca.TypeBaseNODE,
			DerivedFrom: "yorc.type.DerivedSC2",
		},
	}
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC3", derivedSC3)
	require.Nil(t, err)

	derivedSC4 := tosca.NodeType{
		Type: tosca.Type{
			Base:        tosca.TypeBaseNODE,
			DerivedFrom: "yorc.type.DerivedSC3",
		},
		Properties: nil,
		Attributes: map[string]tosca.AttributeDefinition{"dsc4": tosca.AttributeDefinition{
			Default: &tosca.ValueAssignment{Value: "yorc.type.DerivedSC4"},
		}},
	}
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/types/yorc.type.DerivedSC4", derivedSC4)
	require.Nil(t, err)

	root := tosca.NodeType{
		Type: tosca.Type{
			Base: tosca.TypeBaseNODE,
		},
	}
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/types/tosca.nodes.Root", root)
	require.Nil(t, err)

	softwareComponent := tosca.NodeType{
		Type: tosca.Type{
			Base:        tosca.TypeBaseNODE,
			DerivedFrom: "tosca.nodes.Root",
		},
		Properties: map[string]tosca.PropertyDefinition{"typeprop": tosca.PropertyDefinition{
			Default: &tosca.ValueAssignment{Value: "SoftwareComponentTypeProp"},
		},
			"parenttypeprop": tosca.PropertyDefinition{
				Default: &tosca.ValueAssignment{Value: "RootComponentTypeProp"},
			}},
		Attributes: map[string]tosca.AttributeDefinition{"id": tosca.AttributeDefinition{
			Default: &tosca.ValueAssignment{Value: "DefaultSoftwareComponentTypeid"},
		}, "type": tosca.AttributeDefinition{
			Default: &tosca.ValueAssignment{Value: "DefaultSoftwareComponentTypeid"},
		}},
	}
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/types/tosca.nodes.SoftwareComponent", softwareComponent)
	require.Nil(t, err)

	node1 := tosca.NodeTemplate{
		Type: "tosca.nodes.SoftwareComponent",
		Properties: map[string]*tosca.ValueAssignment{"simple": &tosca.ValueAssignment{
			Type:  0,
			Value: "simple",
		}},
		Attributes: map[string]*tosca.ValueAssignment{"id": &tosca.ValueAssignment{
			Type:  0,
			Value: "Node1-id",
		}},
		Requirements: []tosca.RequirementAssignmentMap{
			{"req1": tosca.RequirementAssignment{Relationship: "tosca.relationships.Root"}},
			{"req2": tosca.RequirementAssignment{Relationship: "tosca.relationships.Root"}},
			{"req3": tosca.RequirementAssignment{Relationship: "yorc.type.1", Node: "Node2"}},
			{"req4": tosca.RequirementAssignment{Relationship: "tosca.relationships.Root"}},
		},
	}
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/nodes/Node1", node1)
	require.Nil(t, err)

	node2 := tosca.NodeTemplate{
		Type: "tosca.nodes.SoftwareComponent",
		Properties: map[string]*tosca.ValueAssignment{"recurse": &tosca.ValueAssignment{
			Type:  0,
			Value: "Node2",
		}},
		Requirements: []tosca.RequirementAssignmentMap{
			{"req1": tosca.RequirementAssignment{Relationship: "tosca.relationships.Root"}},
			{"req2": tosca.RequirementAssignment{Relationship: "tosca.relationships.HostedOn", Node: "Compute1"}},
		},
	}
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/nodes/Node2", node2)
	require.Nil(t, err)

	node3 := tosca.NodeTemplate{
		Type: "tosca.nodes.SoftwareComponent",
		Properties: map[string]*tosca.ValueAssignment{"simple": &tosca.ValueAssignment{
			Type:  0,
			Value: "simple",
		}},
		Attributes: map[string]*tosca.ValueAssignment{"simple": &tosca.ValueAssignment{
			Type:  0,
			Value: "simple",
		}},
		Requirements: []tosca.RequirementAssignmentMap{
			{"req1": tosca.RequirementAssignment{Relationship: "yorc.type.3", Node: "Compute2"}},
		},
	}
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/nodes/Node3", node3)
	require.Nil(t, err)
	node4 := tosca.NodeTemplate{
		Type: "tosca.nodes.SoftwareComponent",
		Requirements: []tosca.RequirementAssignmentMap{
			{"host": tosca.RequirementAssignment{Relationship: "tosca.relationships.HostedOn", Node: "Node2"}},
		},
	}
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testGetNbInstancesForNode/topology/nodes/Node4", node4)
	require.Nil(t, err)

	// Test testDeleteNode
	nodeCompute1ToDel := tosca.NodeTemplate{
		Type: "tosca.nodes.Compute",
		Capabilities: map[string]tosca.CapabilityAssignment{"scalable": tosca.CapabilityAssignment{
			Properties: map[string]*tosca.ValueAssignment{"default_instances": &tosca.ValueAssignment{
				Type:  0,
				Value: "10",
			}, "max_instances": &tosca.ValueAssignment{
				Type:  0,
				Value: "20",
			}, "min_instances": &tosca.ValueAssignment{
				Type:  0,
				Value: "2",
			}}},
		},
	}
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, consulutil.DeploymentKVPrefix+"/testDeleteNode/topology/nodes/Compute1", nodeCompute1ToDel)
	require.Nil(t, err)

	// Instancesz
	srv1.PopulateKV(t, map[string][]byte{
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/0/attributes/id": []byte("Compute1-0"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/1/attributes/id": []byte("Compute1-1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/2/attributes/id": []byte("Compute1-2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/3/attributes/id": []byte("Compute1-3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/4/attributes/id": []byte("Compute1-4"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/5/attributes/id": []byte("Compute1-5"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/6/attributes/id": []byte("Compute1-6"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/7/attributes/id": []byte("Compute1-7"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/8/attributes/id": []byte("Compute1-8"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/9/attributes/id": []byte("Compute1-9"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/0/attributes/recurse": []byte("Recurse-Compute1-0"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/1/attributes/recurse": []byte("Recurse-Compute1-1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/2/attributes/recurse": []byte("Recurse-Compute1-2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/3/attributes/recurse": []byte("Recurse-Compute1-3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/4/attributes/recurse": []byte("Recurse-Compute1-4"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/5/attributes/recurse": []byte("Recurse-Compute1-5"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/6/attributes/recurse": []byte("Recurse-Compute1-6"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/7/attributes/recurse": []byte("Recurse-Compute1-7"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/8/attributes/recurse": []byte("Recurse-Compute1-8"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Compute1/9/attributes/recurse": []byte("Recurse-Compute1-9"),

		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/0/attributes/id": []byte("Node2-0"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/1/attributes/id": []byte("Node2-1"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/2/attributes/id": []byte("Node2-2"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/3/attributes/id": []byte("Node2-3"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/4/attributes/id": []byte("Node2-4"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/5/attributes/id": []byte("Node2-5"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/6/attributes/id": []byte("Node2-6"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/7/attributes/id": []byte("Node2-7"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/8/attributes/id": []byte("Node2-8"),
		consulutil.DeploymentKVPrefix + "/testGetNbInstancesForNode/topology/instances/Node2/9/attributes/id": []byte("Node2-9"),

		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/0/attributes/id":  []byte("Node1-0"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/1/attributes/id":  []byte("Node1-1"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/10/attributes/id": []byte("Node1-10"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/11/attributes/id": []byte("Node1-11"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/20/attributes/id": []byte("Node1-20"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/2/attributes/id":  []byte("Node1-2"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/3/attributes/id":  []byte("Node1-3"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/4/attributes/id":  []byte("Node1-4"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/5/attributes/id":  []byte("Node1-5"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/6/attributes/id":  []byte("Node1-6"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/7/attributes/id":  []byte("Node1-7"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/8/attributes/id":  []byte("Node1-8"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node1/9/attributes/id":  []byte("Node1-9"),

		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node2/ab0/attributes/id":   []byte("Node1-0"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node2/ab1/attributes/id":   []byte("Node1-1"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node2/ab10/attributes/id":  []byte("Node1-10"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node2/za11/attributes/id":  []byte("Node1-11"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node2/ab20a/attributes/id": []byte("Node1-20"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node2/ab2/attributes/id":   []byte("Node1-2"),
		consulutil.DeploymentKVPrefix + "/testGetNodeInstancesIds/topology/instances/Node2/za3/attributes/id":   []byte("Node1-3"),
	})

	t.Run("groupDeploymentsNodes", func(t *testing.T) {
		t.Run("TestIsNodeTypeDerivedFrom", func(t *testing.T) {
			testIsNodeTypeDerivedFrom(t)
		})
		t.Run("TestGetDefaultNbInstancesForNode", func(t *testing.T) {
			testGetDefaultNbInstancesForNode(t)
		})
		t.Run("TestGetMaxNbInstancesForNode", func(t *testing.T) {
			testGetMaxNbInstancesForNode(t)
		})
		t.Run("TesttestGetMinNbInstancesForNode", func(t *testing.T) {
			testGetMinNbInstancesForNode(t)
		})
		t.Run("TestGetNodeProperty", func(t *testing.T) {
			testGetNodeProperty(t)
		})
		// t.Run("TestGetNodeAttributes", func(t *testing.T) {
		// 	testGetNodeAttributes(t, kv)
		// })
		t.Run("TestGetNodeAttributesNames", func(t *testing.T) {
			testGetNodeAttributesNames(t)
		})
		t.Run("TestGetNodeInstancesIds", func(t *testing.T) {
			testGetNodeInstancesIds(t)
		})
		t.Run("TestGetTypeAttributesNames", func(t *testing.T) {
			testGetTypeAttributesNames(t)
		})
		t.Run("TestDeleteNode", func(t *testing.T) {
			testDeleteNode(t)
		})
	})
}

func testIsNodeTypeDerivedFrom(t *testing.T) {
	// t.Parallel()
	ctx := context.Background()
	ok, err := IsTypeDerivedFrom(ctx, "testIsNodeTypeDerivedFrom", "yorc.type.1", "tosca.relationships.HostedOn")
	require.Nil(t, err)
	require.True(t, ok)

	ok, err = IsTypeDerivedFrom(ctx, "testIsNodeTypeDerivedFrom", "yorc.type.1", "tosca.relationships.ConnectsTo")
	require.Nil(t, err)
	require.False(t, ok)

	ok, err = IsTypeDerivedFrom(ctx, "testIsNodeTypeDerivedFrom", "yorc.type.1", "yorc.type.1")
	require.Nil(t, err)
	require.True(t, ok)
}

func testGetDefaultNbInstancesForNode(t *testing.T) {
	// t.Parallel()
	ctx := context.Background()
	nb, err := GetDefaultNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Compute1")
	require.Nil(t, err)
	require.Equal(t, uint32(10), nb)

	nb, err = GetDefaultNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Compute2")
	require.Nil(t, err)
	require.Equal(t, uint32(1), nb)

	_, err = GetDefaultNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Compute3")
	require.NotNil(t, err)

	nb, err = GetDefaultNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Node1")
	require.Nil(t, err)
	require.Equal(t, uint32(10), nb)

	nb, err = GetDefaultNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Node2")
	require.Nil(t, err)
	require.Equal(t, uint32(10), nb)

	nb, err = GetDefaultNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Node3")
	require.Nil(t, err)
	require.Equal(t, uint32(1), nb)
}

func testGetMaxNbInstancesForNode(t *testing.T) {
	// t.Parallel()
	ctx := context.Background()
	nb, err := GetMaxNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Compute1")
	require.Nil(t, err)
	require.Equal(t, uint32(20), nb)

	nb, err = GetMaxNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Compute2")
	require.Nil(t, err)
	require.Equal(t, uint32(1), nb)

	_, err = GetMaxNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Compute3")
	fmt.Println(err)
	require.NotNil(t, err)

	nb, err = GetMaxNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Node1")
	require.Nil(t, err)
	require.Equal(t, uint32(20), nb)

	nb, err = GetMaxNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Node2")
	require.Nil(t, err)
	require.Equal(t, uint32(20), nb)

	nb, err = GetMaxNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Node3")
	require.Nil(t, err)
	require.Equal(t, uint32(1), nb)
}

func testGetMinNbInstancesForNode(t *testing.T) {
	// t.Parallel()
	ctx := context.Background()
	nb, err := GetMinNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Compute1")
	require.Nil(t, err)
	require.Equal(t, uint32(2), nb)

	nb, err = GetMinNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Compute2")
	require.Nil(t, err)
	require.Equal(t, uint32(1), nb)

	_, err = GetMinNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Compute3")
	fmt.Println(err)
	require.NotNil(t, err)

	nb, err = GetMinNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Node1")
	require.Nil(t, err)
	require.Equal(t, uint32(2), nb)

	nb, err = GetMinNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Node2")
	require.Nil(t, err)
	require.Equal(t, uint32(2), nb)

	nb, err = GetMinNbInstancesForNode(ctx, "testGetNbInstancesForNode", "Node3")
	require.Nil(t, err)
	require.Equal(t, uint32(1), nb)
}

func testGetNodeProperty(t *testing.T) {
	// t.Parallel()
	ctx := context.Background()
	// Property is directly in node
	value, err := GetNodePropertyValue(ctx, "testGetNbInstancesForNode", "Node1", "simple")
	require.Nil(t, err)
	require.NotNil(t, value)
	require.Equal(t, "simple", value.RawString())

	// Property is in a parent node we found it with recurse
	value, err = GetNodePropertyValue(ctx, "testGetNbInstancesForNode", "Node4", "recurse")
	require.Nil(t, err)
	require.NotNil(t, value)
	require.Equal(t, "Node2", value.RawString())

	// Property has a default in node type
	value, err = GetNodePropertyValue(ctx, "testGetNbInstancesForNode", "Node4", "typeprop")
	require.Nil(t, err)
	require.NotNil(t, value)
	require.Equal(t, "SoftwareComponentTypeProp", value.RawString())

	value, err = GetNodePropertyValue(ctx, "testGetNbInstancesForNode", "Node4", "typeprop")
	require.Nil(t, err)
	require.NotNil(t, value)
	require.Equal(t, "SoftwareComponentTypeProp", value.RawString())

	// Property has a default in a parent of the node type
	value, err = GetNodePropertyValue(ctx, "testGetNbInstancesForNode", "Node4", "parenttypeprop")
	require.Nil(t, err)
	require.NotNil(t, value)
	require.Equal(t, "RootComponentTypeProp", value.RawString())

	value, err = GetNodePropertyValue(ctx, "testGetNbInstancesForNode", "Node4", "parenttypeprop")
	require.Nil(t, err)
	require.NotNil(t, value)
	require.Equal(t, "RootComponentTypeProp", value.RawString())

}

func testGetNodeAttributes(t *testing.T) {
	ctx := context.Background()
	// t.Parallel()
	// Attribute is directly in node
	// res, instancesValues, err := GetNodeAttributes(kv, "testGetNbInstancesForNode", "Node3", "simple")
	// require.Nil(t, err)
	// require.True(t, res)
	// require.Len(t, instancesValues, 1)
	// require.Equal(t, "simple", instancesValues[""])

	// Attribute is directly in instances
	instancesValues, err := GetNodeAttributesValues(ctx, "testGetNbInstancesForNode", "Compute1", "id")
	require.Nil(t, err)
	require.Len(t, instancesValues, 10)
	require.NotNil(t, instancesValues["0"])
	require.Equal(t, "Compute1-0", instancesValues["0"].RawString())
	require.NotNil(t, instancesValues["1"])
	require.Equal(t, "Compute1-1", instancesValues["1"].RawString())
	require.NotNil(t, instancesValues["2"])
	require.Equal(t, "Compute1-2", instancesValues["2"].RawString())
	require.NotNil(t, instancesValues["3"])
	require.Equal(t, "Compute1-3", instancesValues["3"].RawString())

	// Look at generic node attribute before parents
	// res, instancesValues, err = GetNodeAttributes(kv, "testGetNbInstancesForNode", "Node1", "id")
	// require.Nil(t, err)
	// require.True(t, res)
	// require.Len(t, instancesValues, 1)
	// require.Equal(t, "Node1-id", instancesValues[""])

	// Look at generic node type attribute before parents
	// res, instancesValues, err = GetNodeAttributes(kv, "testGetNbInstancesForNode", "Node3", "id")
	// require.Nil(t, err)
	// require.True(t, res)
	// require.Len(t, instancesValues, 1)
	// require.Equal(t, "DefaultSoftwareComponentTypeid", instancesValues[""])

	// Look at generic node type attribute before parents
	instancesValues, err = GetNodeAttributesValues(ctx, "testGetNbInstancesForNode", "Node2", "type")
	require.Nil(t, err)
	require.Len(t, instancesValues, 10)
	require.NotNil(t, instancesValues["0"])
	require.Equal(t, "DefaultSoftwareComponentTypeid", instancesValues["0"].RawString())
	require.NotNil(t, instancesValues["3"])
	require.Equal(t, "DefaultSoftwareComponentTypeid", instancesValues["3"].RawString())
	require.NotNil(t, instancesValues["6"])
	require.Equal(t, "DefaultSoftwareComponentTypeid", instancesValues["6"].RawString())

	//
	instancesValues, err = GetNodeAttributesValues(ctx, "testGetNbInstancesForNode", "Node2", "recurse")
	require.Nil(t, err)
	require.Len(t, instancesValues, 10)
	require.NotNil(t, instancesValues["0"])
	require.Equal(t, "Recurse-Compute1-0", instancesValues["0"].RawString())
	require.NotNil(t, instancesValues["3"])
	require.Equal(t, "Recurse-Compute1-3", instancesValues["3"].RawString())
	require.NotNil(t, instancesValues["6"])
	require.Equal(t, "Recurse-Compute1-6", instancesValues["6"].RawString())

	//
	instancesValues, err = GetNodeAttributesValues(ctx, "testGetNbInstancesForNode", "Node1", "recurse")
	require.Nil(t, err)
	require.Len(t, instancesValues, 10)
	require.NotNil(t, instancesValues["0"])
	require.Equal(t, "Recurse-Compute1-0", instancesValues["0"].RawString())
	require.NotNil(t, instancesValues["3"])
	require.Equal(t, "Recurse-Compute1-3", instancesValues["3"].RawString())
	require.NotNil(t, instancesValues["6"])
	require.Equal(t, "Recurse-Compute1-6", instancesValues["6"].RawString())
}

func testGetNodeAttributesNames(t *testing.T) {
	// t.Parallel()
	ctx := context.Background()
	attrNames, err := GetNodeAttributesNames(ctx, "testGetNbInstancesForNode", "Compute1")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 3)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "ip")
	require.Contains(t, attrNames, "recurse")

	attrNames, err = GetNodeAttributesNames(ctx, "testGetNbInstancesForNode", "Node1")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 2)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")

	attrNames, err = GetNodeAttributesNames(ctx, "testGetNbInstancesForNode", "Node2")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 2)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")

	attrNames, err = GetNodeAttributesNames(ctx, "testGetNbInstancesForNode", "Node3")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 3)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")
	require.Contains(t, attrNames, "simple")

	attrNames, err = GetNodeAttributesNames(ctx, "testGetNbInstancesForNode", "Node4")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 2)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")
}

func testGetTypeAttributesNames(t *testing.T) {
	// t.Parallel()
	ctx := context.Background()
	attrNames, err := GetTypeAttributesNames(ctx, "testGetNbInstancesForNode", "tosca.nodes.SoftwareComponent")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 2)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")

	attrNames, err = GetTypeAttributesNames(ctx, "testGetNbInstancesForNode", "yorc.type.DerivedSC1")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 2)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")

	attrNames, err = GetTypeAttributesNames(ctx, "testGetNbInstancesForNode", "yorc.type.DerivedSC2")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 3)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")
	require.Contains(t, attrNames, "dsc2")

	attrNames, err = GetTypeAttributesNames(ctx, "testGetNbInstancesForNode", "yorc.type.DerivedSC3")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 3)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")
	require.Contains(t, attrNames, "dsc2")

	attrNames, err = GetTypeAttributesNames(ctx, "testGetNbInstancesForNode", "yorc.type.DerivedSC4")
	require.Nil(t, err)
	require.NotNil(t, attrNames)
	require.Len(t, attrNames, 4)

	require.Contains(t, attrNames, "id")
	require.Contains(t, attrNames, "type")
	require.Contains(t, attrNames, "dsc2")
	require.Contains(t, attrNames, "dsc4")
}

func testGetNodeInstancesIds(t *testing.T) {
	// t.Parallel()

	ctx := context.Background()
	node1ExpectedResult := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "20"}
	instancesIDs, err := GetNodeInstancesIds(ctx, "testGetNodeInstancesIds", "Node1")
	require.NoError(t, err)
	require.Equal(t, node1ExpectedResult, instancesIDs)

	node2ExpectedResult := []string{"ab0", "ab1", "ab2", "ab10", "ab20a", "za3", "za11"}
	instancesIDs, err = GetNodeInstancesIds(ctx, "testGetNodeInstancesIds", "Node2")
	require.NoError(t, err)
	require.Equal(t, node2ExpectedResult, instancesIDs)
}

func testDeleteNode(t *testing.T) {
	ctx := context.Background()
	deploymentID := "testDeleteNode"
	nodeName := "Compute1"
	err := DeleteNode(ctx, deploymentID, nodeName)
	require.NoError(t, err, "Unexpected error deleting node %s", nodeName)

	instancesValues, err := GetNodeAttributesValues(ctx, deploymentID, nodeName, "type")
	require.NoError(t, err, "Unexpected error calling GetNodeAttributesValues")
	require.Len(t, instancesValues, 0, "Expected an empty map, got %+v", instancesValues)
}

func testNodeHasAttribute(t *testing.T, deploymentID string) {
	type args struct {
		nodeName       string
		attributeName  string
		exploreParents bool
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{"NodeHasAttribute", args{"TestCompute", "public_address", true}, true, false},
		{"NodeDoesntHaveAttribute", args{"TestCompute", "missing_attribute", true}, false, false},
		{"NodeHasAttribute", args{"TestCompute", "public_address", false}, false, false},
		{"NodeDoesntExist", args{"DoNotExist", "missing_attribute", true}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NodeHasAttribute(context.Background(), deploymentID, tt.args.nodeName, tt.args.attributeName, tt.args.exploreParents)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeHasAttribute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NodeHasAttribute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testNodeHasProperty(t *testing.T, deploymentID string) {
	ctx := context.Background()
	type args struct {
		nodeName       string
		propertyName   string
		exploreParents bool
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{"NodeHasProperty", args{"TestModule", "component_version", true}, true, false},
		{"NodeDoesntHaveProperty", args{"TestModule", "missing_property", true}, false, false},
		{"NodeHasPropertyOnlyOnParentType", args{"TestModule", "admin_credential", false}, false, false},
		{"NodeDoesntExist", args{"DoNotExist", "missing_property", true}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NodeHasProperty(ctx, deploymentID, tt.args.nodeName, tt.args.propertyName, tt.args.exploreParents)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeHasProperty() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NodeHasProperty() = %v, want %v", got, tt.want)
			}
		})
	}
}
