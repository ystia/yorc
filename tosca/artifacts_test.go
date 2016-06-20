package tosca

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestArtifactDefinitionConcrete_UnmarshalYAML_SimpleGrammar(t *testing.T) {
	var data = `artifact: ./artifact.txt`

	artMap := ArtifactDefinitionMap{}

	err := yaml.Unmarshal([]byte(data), &artMap)
	assert.Nil(t, err, "Expecting no error when unmarshaling artifact with simple grammar")
	assert.Len(t, artMap, 1, "Expecting 1 artifact")
	assert.Contains(t, artMap, "artifact", "Artifact 'artifact' not found")
	art := artMap["artifact"]
	assert.Equal(t, "./artifact.txt", art.File)
}

func TestArtifactDefinitionConcrete_UnmarshalYAML_ComplexGrammar(t *testing.T) {
	var data = `
artifact:
  file: ./artifact.txt
  description: artifact_description
  type: artifact_type_name
  repository: artifact_repository_name
  deploy_path: file_deployment_path`

	artMap := ArtifactDefinitionMap{}

	err := yaml.Unmarshal([]byte(data), &artMap)
	assert.Nil(t, err, "Expecting no error when unmarshaling artifact with simple grammar")
	assert.Len(t, artMap, 1, "Expecting 1 artifact")
	assert.Contains(t, artMap, "artifact", "Artifact 'artifact' not found")
	art := artMap["artifact"]
	assert.Equal(t, "./artifact.txt", art.File)
	assert.Equal(t, "artifact_description", art.Description)
	assert.Equal(t, "artifact_type_name", art.Type)
	assert.Equal(t, "artifact_repository_name", art.Repository)
	assert.Equal(t, "file_deployment_path", art.DeployPath)
}

func TestArtifactDefinitionConcrete_UnmarshalYAML_Failure(t *testing.T) {
	var data = `
artifact:
  file: [ ./artifact.txt, art2 ]
  description: artifact_description
  type: artifact_type_name
  repository: artifact_repository_name
  deploy_path: file_deployment_path`

	artMap := ArtifactDefinitionMap{}

	err := yaml.Unmarshal([]byte(data), &artMap)
	assert.NotNil(t, err, "Expecting error when unmarshaling artifact with an array as file")
}
