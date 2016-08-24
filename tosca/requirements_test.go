package tosca

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestGroupedRequirementParallel(t *testing.T) {
	t.Run("groupRequirement", func(t *testing.T) {
		t.Run("TestRequirementAssignment_Complex", requirementAssignment_Complex)
		t.Run("TestRequirementAssignment_Simple", requirementAssignment_Simple)
		t.Run("TestRequirementAssignment_SimpleRelationship", requirementAssignment_SimpleRelationship)
	})
}

type ReqTestNode struct {
	Requirements []RequirementAssignmentMap `yaml:"requirements,omitempty"`
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
