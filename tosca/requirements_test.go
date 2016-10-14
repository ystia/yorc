package tosca

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"novaforge.bull.com/starlings-janus/janus/log"
	"testing"
)

func TestGroupedRequirementParallel(t *testing.T) {
	t.Run("groupRequirement", func(t *testing.T) {
		t.Run("TestRequirementAssignment_Complex", requirementAssignment_Complex)
		t.Run("TestRequirementAssignment_Simple", requirementAssignment_Simple)
		t.Run("TestRequirementAssignment_SimpleRelationship", requirementAssignment_SimpleRelationship)
		t.Run("TestRequirementDefinition_Standard", requirementDefinition_Standard)
		t.Run("TestRequirementDefinition_Alien", requirementDefinition_Alien)
	})
}

type ReqTestNode struct {
	Requirements []RequirementAssignmentMap `yaml:"requirements,omitempty"`
}

type ReqDefTestNode struct {
	Requirements []RequirementDefinitionMap `yaml:"requirements,omitempty"`
}

func requirementAssignment_Complex(t *testing.T) {
	t.Parallel()
	data := `Compute:
  requirements:
    - local_storage:
        node: my_block_storage
        capability: tosca.capabilities.Attachment
        relationship:
          type: tosca.relationships.AttachesTo
          properties:
            location: /dev/vde`
	nodes := make(map[string]ReqTestNode)
	err := yaml.Unmarshal([]byte(data), &nodes)
	assert.Nil(t, err)
	assert.Contains(t, nodes, "Compute")
	compute := nodes["Compute"]

	assert.Contains(t, compute.Requirements[0], "local_storage")
	lc := compute.Requirements[0]["local_storage"]

	assert.Equal(t, "tosca.relationships.AttachesTo", lc.Relationship)
	assert.Contains(t, lc.RelationshipProps, "location")
	assert.Equal(t, "/dev/vde", lc.RelationshipProps["location"].String())
}

func requirementAssignment_Simple(t *testing.T) {
	t.Parallel()
	data := `Compute:
  requirements:
    - req: nodeName`
	nodes := make(map[string]ReqTestNode)
	err := yaml.Unmarshal([]byte(data), &nodes)
	assert.Nil(t, err)
	assert.Contains(t, nodes, "Compute")
	compute := nodes["Compute"]

	assert.Contains(t, compute.Requirements[0], "req")
	req := compute.Requirements[0]["req"]

	assert.Equal(t, "nodeName", req.Node)
}

func requirementAssignment_SimpleRelationship(t *testing.T) {
	t.Parallel()
	data := `Compute:
  requirements:
    - req:
        node: nodeName
        capability: tosca.capabilities.cap
        relationship: tosca.relationships.rs`
	nodes := make(map[string]ReqTestNode)
	err := yaml.Unmarshal([]byte(data), &nodes)
	assert.Nil(t, err)
	assert.Contains(t, nodes, "Compute")
	compute := nodes["Compute"]

	assert.Contains(t, compute.Requirements[0], "req")
	req := compute.Requirements[0]["req"]

	assert.Equal(t, "nodeName", req.Node)
	assert.Equal(t, "tosca.capabilities.cap", req.Capability)
	assert.Equal(t, "tosca.relationships.rs", req.Relationship)
}

func requirementDefinition_Standard(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	data := `NodeType:
    requirements:
      - server_endpoint:
          capability: starlings.capabilities.ConsulServer
          relationship_type: starlings.relationships.ConnectsConsulAgentToServer
          occurrences: [ 0, 1 ]
      - wan_endpoint:
          capability: starlings.capabilities.ConsulServerWAN
          relationship_type: starlings.relationships.ConnectsConsulServerWAN
          occurrences: [ 0, UNBOUNDED ]
      - simple_endpoint: starlings.capabilities.Simple
        `

	nodes := make(map[string]ReqDefTestNode)
	err := yaml.Unmarshal([]byte(data), &nodes)

	log.Printf("%+v", nodes)

	require.Nil(t, err)
	require.Contains(t, nodes, "NodeType")
	node := nodes["NodeType"]

	require.Len(t, node.Requirements, 3)

	require.Contains(t, node.Requirements[0], "server_endpoint")
	req := node.Requirements[0]["server_endpoint"]

	require.Equal(t, "starlings.capabilities.ConsulServer", req.Capability)
	require.Equal(t, "starlings.relationships.ConnectsConsulAgentToServer", req.Relationship)
	require.Equal(t, uint64(0), req.Occurrences.LowerBound)
	require.Equal(t, uint64(1), req.Occurrences.UpperBound)

	require.Contains(t, node.Requirements[1], "wan_endpoint")
	req = node.Requirements[1]["wan_endpoint"]

	require.Equal(t, "starlings.capabilities.ConsulServerWAN", req.Capability)
	require.Equal(t, "starlings.relationships.ConnectsConsulServerWAN", req.Relationship)
	require.Equal(t, uint64(0), req.Occurrences.LowerBound)
	require.Equal(t, uint64(UNBOUNDED), req.Occurrences.UpperBound)

	require.Contains(t, node.Requirements[2], "simple_endpoint")
	req = node.Requirements[2]["simple_endpoint"]

	require.Equal(t, "starlings.capabilities.Simple", req.Capability)
	require.Equal(t, "", req.Relationship)
	require.Equal(t, uint64(0), req.Occurrences.LowerBound)
	require.Equal(t, uint64(0), req.Occurrences.UpperBound)

}

func requirementDefinition_Alien(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	data := `NodeType:
    requirements:
      - server_endpoint: starlings.capabilities.ConsulServer
        relationship_type: starlings.relationships.ConnectsConsulAgentToServer
        lower_bound: 0
        upper_bound: 1
      - wan_endpoint: starlings.capabilities.ConsulServerWAN
        relationship_type: starlings.relationships.ConnectsConsulServerWAN
        lower_bound: 0
        upper_bound: UNBOUNDED
        `

	nodes := make(map[string]ReqDefTestNode)
	err := yaml.Unmarshal([]byte(data), &nodes)

	log.Printf("%+v", nodes)

	require.Nil(t, err)
	require.Contains(t, nodes, "NodeType")
	node := nodes["NodeType"]

	require.Len(t, node.Requirements, 2)

	require.Contains(t, node.Requirements[0], "server_endpoint")
	req := node.Requirements[0]["server_endpoint"]

	require.Equal(t, "starlings.capabilities.ConsulServer", req.Capability)
	require.Equal(t, "starlings.relationships.ConnectsConsulAgentToServer", req.Relationship)
	require.Equal(t, uint64(0), req.Occurrences.LowerBound)
	require.Equal(t, uint64(1), req.Occurrences.UpperBound)

	require.Contains(t, node.Requirements[1], "wan_endpoint")
	req = node.Requirements[1]["wan_endpoint"]

	require.Equal(t, "starlings.capabilities.ConsulServerWAN", req.Capability)
	require.Equal(t, "starlings.relationships.ConnectsConsulServerWAN", req.Relationship)
	require.Equal(t, uint64(0), req.Occurrences.LowerBound)
	require.Equal(t, uint64(UNBOUNDED), req.Occurrences.UpperBound)

}
