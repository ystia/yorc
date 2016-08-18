package tosca

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

type ReqTestNode struct {
	Requirements []RequirementAssignmentMap `yaml:"requirements,omitempty"`
}

func TestRequirementAssignment_Complex(t *testing.T) {
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

func TestRequirementAssignment_Simple(t *testing.T) {
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

func TestRequirementAssignment_SimpleRelationship(t *testing.T) {
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
