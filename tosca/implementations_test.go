package tosca

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

type implementationTestType struct {
	Implementation Implementation
}

func TestImplementationSimpleGrammar(t *testing.T) {
	var inputYaml = `implementation: scripts/start_server.sh`
	implem := implementationTestType{}

	err := yaml.Unmarshal([]byte(inputYaml), &implem)
	assert.Nil(t, err, "Expecting no error when unmarshaling Implementation with simple grammar")
	assert.Equal(t, "scripts/start_server.sh", implem.Implementation.Primary)
	assert.Len(t, implem.Implementation.Dependencies, 0, "Expecting no dependencies but found %d", len(implem.Implementation.Dependencies))
}

func TestImplementationComplexGrammar(t *testing.T) {
	var inputYaml = `
implementation:
  primary: scripts/start_server.sh`
	implem := implementationTestType{}

	err := yaml.Unmarshal([]byte(inputYaml), &implem)
	assert.Nil(t, err, "Expecting no error when unmarshaling Implementation with simple grammar")
	assert.Equal(t, "scripts/start_server.sh", implem.Implementation.Primary)
	assert.Len(t, implem.Implementation.Dependencies, 0, "Expecting no dependencies but found %d", len(implem.Implementation.Dependencies))
}

func TestImplementationComplexGrammarWithDependencies(t *testing.T) {
	var inputYaml = `
implementation:
  primary: scripts/start_server.sh
  dependencies:
    - utils/utils.sh
    - utils/log.sh`
	implem := implementationTestType{}

	err := yaml.Unmarshal([]byte(inputYaml), &implem)
	assert.Nil(t, err, "Expecting no error when unmarshaling Implementation with simple grammar")
	assert.Equal(t, "scripts/start_server.sh", implem.Implementation.Primary)
	assert.Len(t, implem.Implementation.Dependencies, 2, "Expecting 2 dependencies but found %d", len(implem.Implementation.Dependencies))
	assert.Contains(t, implem.Implementation.Dependencies, "utils/utils.sh")
	assert.Contains(t, implem.Implementation.Dependencies, "utils/log.sh")
}

func TestImplementationFailing(t *testing.T) {
	var inputYaml = `
implementation:
  primary: [scripts/start_server.sh, scripts/start_server.sh]
  dependencies:
    - utils/utils.sh
    - utils/log.sh`
	implem := implementationTestType{}

	err := yaml.Unmarshal([]byte(inputYaml), &implem)
	assert.NotNil(t, err, "Expecting an error when unmarshaling Implementation with an array as primary")

}
