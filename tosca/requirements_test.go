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

package tosca

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v3/log"
)

type ReqTestNode struct {
	Requirements []RequirementAssignmentMap `yaml:"requirements,omitempty"`
}

type ReqDefTestNode struct {
	Requirements []RequirementDefinitionMap `yaml:"requirements,omitempty"`
}

func TestRequirementAssignmentComplex(t *testing.T) {
	t.Parallel()
	data := `Compute:
  requirements:
    - local_storage:
        node: my_block_storage
        capability: tosca.capabilities.Attachment
        type_requirement: host
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
	assert.Contains(t, lc.TypeRequirement, "host")
}

func TestRequirementAssignmentSimple(t *testing.T) {
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

func TestRequirementAssignmentSimpleRelationship(t *testing.T) {
	t.Parallel()
	data := `
Compute:
  requirements:
    - req:
        node: nodeName
        capability: tosca.capabilities.cap
        relationship: tosca.relationships.rs
        properties:
          literal: "value"
`
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
	assert.NotNil(t, req.RelationshipProps["literal"])
	assert.Equal(t, "value", req.RelationshipProps["literal"].GetLiteral())
}

func TestRequirementDefinitionStandard(t *testing.T) {
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

func TestRequirementDefinitionAlien(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	data := `NodeType:
    requirements:
      - server_endpoint: starlings.capabilities.ConsulServer
        relationship_type: starlings.relationships.ConnectsConsulAgentToServer
        lower_bound: 0
        upper_bound: 1
        capability_name: server
      - wan_endpoint: starlings.capabilities.ConsulServerWAN
        relationship_type: starlings.relationships.ConnectsConsulServerWAN
        lower_bound: 0
        upper_bound: UNBOUNDED
        capability_name: server
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
	require.Equal(t, "server", req.CapabilityName)

}
