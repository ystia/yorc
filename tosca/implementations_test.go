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

func TestGroupedImplementationsParallel(t *testing.T) {
	t.Run("groupImplementations", func(t *testing.T) {
		t.Run("TestImplementationSimpleGrammar", implementationSimpleGrammar)
		t.Run("TestImplementationComplexGrammar", implementationComplexGrammar)
		t.Run("TestImplementationArtifact", implementationArtifact)
		t.Run("TestImplementationComplexGrammarWithDependencies", implementationComplexGrammarWithDependencies)
		t.Run("TestImplementationFailing", implementationFailing)
	})
}

type implementationTestType struct {
	Implementation Implementation
}

func implementationSimpleGrammar(t *testing.T) {
	t.Parallel()
	var inputYaml = `implementation: scripts/start_server.sh`
	implem := implementationTestType{}

	err := yaml.Unmarshal([]byte(inputYaml), &implem)
	assert.Nil(t, err, "Expecting no error when unmarshaling Implementation with simple grammar")
	assert.Equal(t, "scripts/start_server.sh", implem.Implementation.Primary)
	assert.Len(t, implem.Implementation.Dependencies, 0, "Expecting no dependencies but found %d", len(implem.Implementation.Dependencies))
}

func implementationComplexGrammar(t *testing.T) {
	t.Parallel()
	var inputYaml = `
implementation:
  primary: scripts/start_server.sh
  operation_host: HOST`
	implem := implementationTestType{}

	err := yaml.Unmarshal([]byte(inputYaml), &implem)
	assert.Nil(t, err, "Expecting no error when unmarshaling Implementation with simple grammar")
	assert.Equal(t, "scripts/start_server.sh", implem.Implementation.Primary)
	assert.Len(t, implem.Implementation.Dependencies, 0, "Expecting no dependencies but found %d", len(implem.Implementation.Dependencies))
	assert.Equal(t, "HOST", implem.Implementation.OperationHost)
}

func implementationArtifact(t *testing.T) {
	t.Parallel()
	var inputYaml = `
implementation:
    file: nginx
    repository: docker
    type: tosca.artifacts.Deployment.Image.Container.Docker.Kubernetes`
	implem := implementationTestType{}

	err := yaml.Unmarshal([]byte(inputYaml), &implem)
	assert.Nil(t, err, "Expecting no error when unmarshaling Implementation with simple grammar")
	assert.Equal(t, "nginx", implem.Implementation.Artifact.File)
	assert.Len(t, implem.Implementation.Dependencies, 0, "Expecting no dependencies but found %d", len(implem.Implementation.Dependencies))
}

func implementationComplexGrammarWithDependencies(t *testing.T) {
	t.Parallel()
	var inputYaml = `
implementation:
  primary: scripts/start_server.sh
  dependencies:
    - utils/utils.sh
    - utils/log.sh
  operation_host: SELF`
	implem := implementationTestType{}

	err := yaml.Unmarshal([]byte(inputYaml), &implem)
	assert.Nil(t, err, "Expecting no error when unmarshaling Implementation with simple grammar")
	assert.Equal(t, "scripts/start_server.sh", implem.Implementation.Primary)
	assert.Len(t, implem.Implementation.Dependencies, 2, "Expecting 2 dependencies but found %d", len(implem.Implementation.Dependencies))
	assert.Contains(t, implem.Implementation.Dependencies, "utils/utils.sh")
	assert.Contains(t, implem.Implementation.Dependencies, "utils/log.sh")
	assert.Equal(t, "SELF", implem.Implementation.OperationHost)
}

func implementationFailing(t *testing.T) {
	t.Parallel()
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
