package tosca

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestGroupedArtifactParallel(t *testing.T)  {
	t.Run("groupArtifact", func(t *testing.T) {
		t.Run("TestArtifactDefinitionConcrete_UnmarshalYAML_SimpleGrammar", artifactDefinitionConcrete_UnmarshalYAML_SimpleGrammar)
		t.Run("TestArtifactDefinitionConcrete_UnmarshalYAML_ComplexGrammar", artifactDefinitionConcrete_UnmarshalYAML_ComplexGrammar)
		t.Run("TestArtifactDefinitionConcrete_UnmarshalYAML_Failure", artifactDefinitionConcrete_UnmarshalYAML_Failure)
		t.Run("TestArtifactsInNodeType", artifactsInNodeType)
		t.Run("TestArtifactsAlien", artifactsAlien)
		t.Run("TestArtifactsAlien2", artifactsAlien2)
		t.Run("TestArtifactsInNodeTypeAlien", artifactsInNodeTypeAlien)
		t.Run("TestArtifactsInNodeTemplateAlien", artifactsInNodeTemplateAlien)
	})
}

func artifactDefinitionConcrete_UnmarshalYAML_SimpleGrammar(t *testing.T) {
	t.Parallel()
	var data = `artifact: ./artifact.txt`

	var artMap map[string]ArtifactDefinition

	err := yaml.Unmarshal([]byte(data), &artMap)
	assert.Nil(t, err, "Expecting no error when unmarshaling artifact with simple grammar")
	assert.Len(t, artMap, 1, "Expecting 1 artifact")
	assert.Contains(t, artMap, "artifact", "Artifact 'artifact' not found")
	art := artMap["artifact"]
	assert.Equal(t, "./artifact.txt", art.File)
}

func artifactDefinitionConcrete_UnmarshalYAML_ComplexGrammar(t *testing.T) {
	t.Parallel()
	var data = `
artifact:
  file: ./artifact.txt
  description: artifact_description
  type: artifact_type_name
  repository: artifact_repository_name
  deploy_path: file_deployment_path`

	var artMap map[string]ArtifactDefinition

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

func artifactDefinitionConcrete_UnmarshalYAML_Failure(t *testing.T) {
	t.Parallel()
	var data = `
artifact:
  file: [ ./artifact.txt, art2 ]
  description: artifact_description
  type: artifact_type_name
  repository: artifact_repository_name
  deploy_path: file_deployment_path`

	var artMap map[string]ArtifactDefinition

	err := yaml.Unmarshal([]byte(data), &artMap)
	assert.NotNil(t, err, "Expecting error when unmarshaling artifact with an array as file")
}

func artifactsInNodeType(t *testing.T) {
	t.Parallel()
	var data = `
nodes.ANode:
  artifacts:
    scripts:
      file: scripts
      type: tosca.artifacts.File
    utils_scripts: utils_scripts
`

	var nodeType map[string]NodeType
	err := yaml.Unmarshal([]byte(data), &nodeType)
	assert.Nil(t, err)
	//t.Log(nodeType["nodes.ANode"].Artifacts)
	assert.Contains(t, nodeType["nodes.ANode"].Artifacts, "scripts")
	assert.Contains(t, nodeType["nodes.ANode"].Artifacts, "utils_scripts")
	artScripts := nodeType["nodes.ANode"].Artifacts["scripts"]
	assert.Equal(t, "scripts", artScripts.File)
	assert.Equal(t, "tosca.artifacts.File", artScripts.Type)
	artUtilsScripts := nodeType["nodes.ANode"].Artifacts["utils_scripts"]
	assert.Equal(t, "utils_scripts", artUtilsScripts.File)
}

func artifactsAlien(t *testing.T) {
	t.Parallel()
	var data = `
- scripts: scripts
  type: tosca.artifacts.File
- utils_scripts: utils_scripts
- test_scripts: test_scripts
  description: blahblah
  deploy_path: /my/path
`

	var arts ArtifactDefMap
	err := yaml.Unmarshal([]byte(data), &arts)
	//t.Log("Alien1", arts)
	assert.Nil(t, err)
	assert.Len(t, arts, 3)
	assert.Equal(t, "scripts", arts["scripts"].File)
	assert.Equal(t, "tosca.artifacts.File", arts["scripts"].Type)
	assert.Equal(t, "utils_scripts", arts["utils_scripts"].File)
	assert.Equal(t, "test_scripts", arts["test_scripts"].File)
	assert.Equal(t, "blahblah", arts["test_scripts"].Description)
	assert.Equal(t, "/my/path", arts["test_scripts"].DeployPath)

}

func artifactsAlien2(t *testing.T) {
	t.Parallel()
	var data = `
- scripts: scripts
- utils_scripts: utils_scripts
`

	var arts ArtifactDefMap
	err := yaml.Unmarshal([]byte(data), &arts)
	//t.Log("Alien2", arts)
	assert.Nil(t, err)
	assert.Len(t, arts, 2)
	assert.Equal(t, "scripts", arts["scripts"].File)
	//	assert.Equal(t, "tosca.artifacts.File", arts[0].Type)
	assert.Equal(t, "utils_scripts", arts["utils_scripts"].File)
}

func artifactsInNodeTypeAlien(t *testing.T) {
	t.Parallel()
	var data = `
nodes.ANode:
  artifacts:
    - scripts: scripts
      type: tosca.artifacts.File
    - utils_scripts: utils_scripts
`

	var nodeType map[string]NodeType
	err := yaml.Unmarshal([]byte(data), &nodeType)
	assert.Nil(t, err)
	//t.Log(nodeType["nodes.ANode"].Artifacts)
	assert.Contains(t, nodeType["nodes.ANode"].Artifacts, "scripts")
	assert.Contains(t, nodeType["nodes.ANode"].Artifacts, "utils_scripts")
	artScripts := nodeType["nodes.ANode"].Artifacts["scripts"]
	assert.Equal(t, "scripts", artScripts.File)
	assert.Equal(t, "tosca.artifacts.File", artScripts.Type)
	artUtilsScripts := nodeType["nodes.ANode"].Artifacts["utils_scripts"]
	assert.Equal(t, "utils_scripts", artUtilsScripts.File)
}

func artifactsInNodeTemplateAlien(t *testing.T) {
	t.Parallel()
	var data = `
nodes.ANode:
  artifacts:
    - scripts: scripts
      type: tosca.artifacts.File
    - utils_scripts: utils_scripts
`

	var nodeTemplate map[string]NodeTemplate
	err := yaml.Unmarshal([]byte(data), &nodeTemplate)
	assert.Nil(t, err)
	//t.Log(nodeTemplate["nodes.ANode"].Artifacts)
	assert.Contains(t, nodeTemplate["nodes.ANode"].Artifacts, "scripts")
	assert.Contains(t, nodeTemplate["nodes.ANode"].Artifacts, "utils_scripts")
	artScripts := nodeTemplate["nodes.ANode"].Artifacts["scripts"]
	assert.Equal(t, "scripts", artScripts.File)
	assert.Equal(t, "tosca.artifacts.File", artScripts.Type)
	artUtilsScripts := nodeTemplate["nodes.ANode"].Artifacts["utils_scripts"]
	assert.Equal(t, "utils_scripts", artUtilsScripts.File)
}
