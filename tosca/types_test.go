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
	"gopkg.in/yaml.v2"
)

func TestGroupedTypesParallel(t *testing.T) {
	t.Run("groupTypes", func(t *testing.T) {
		t.Run("TestNodeTypeParsing", nodeTypeParsing)
	})
}

func nodeTypeParsing(t *testing.T) {

	var data = `
  starlings.samples.nodes.Welcome:
    derived_from: tosca.nodes.SoftwareComponent
    description: Installation of the Welcome Very Simple HTTP Server, a Starlings Sample.
    tags:
      icon: /images/welcome-icon.png
    attributes:
      url: { concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port] ] }
    properties:
      component_version:
        type: version
        default: 2.1-SNAPSHOT
        constraints:
          - equal: 2.1-SNAPSHOT
      port:
        type: integer
        description: |
          Port number of the Welcome HTTP server.
        required: true
        default: 8111
    interfaces:
      Standard:
        start:
          inputs:
            PORT: { get_property: [SELF, port] }
          implementation: scripts/welcome_start.sh
        stop: scripts/welcome_stop.sh
    artifacts:
      scripts:
        file: scripts
        type: tosca.artifacts.File
      utils_scripts:
        file: utils_scripts
        type: tosca.artifacts.File`

	type nodeTypeMapTest map[string]NodeType
	var nodeTypeMap nodeTypeMapTest

	err := yaml.Unmarshal([]byte(data), &nodeTypeMap)
	assert.Nil(t, err, "Expecting no error when unmarshaling a node type")
	assert.Len(t, nodeTypeMap, 1)
	assert.Contains(t, nodeTypeMap, "starlings.samples.nodes.Welcome")
	nodeType := nodeTypeMap["starlings.samples.nodes.Welcome"]
	assert.Equal(t, "tosca.nodes.SoftwareComponent", nodeType.DerivedFrom)
	assert.Equal(t, "tosca.nodes.SoftwareComponent", nodeType.DerivedFrom)
	assert.Equal(t, ValueAssignmentFunction.String(), nodeType.Attributes["url"].Type)
	assert.Equal(t, ValueAssignmentFunction.String(), nodeType.Attributes["url"].Default.Type.String())
	assert.Equal(t, `concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port]]`, nodeType.Attributes["url"].Default.String())
}
