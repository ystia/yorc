package tosca

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestGroupedTypesParallel(t *testing.T)  {
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
}
