package tosca

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestImportDefinitionConcrete_UnmarshalYAML_SimpleGrammar(t *testing.T) {

	var data = `
imports:
  - some_definition_file: path1/path2/some_defs.yaml
  - another_definition_file:
      file: path1/path2/file2.yaml
      repository: my_service_catalog
      namespace_uri: http://mycompany.com/tosca/1.0/platform
      namespace_prefix: mycompany
`
	importMap := ImportMapInteface{}
	err := yaml.Unmarshal([]byte(data), &importMap)
	if err == nil {
		assert.Len(t, importMap, 1)
		assert.Contains(t, importMap, "imports")
		importDef := importMap["imports"]
		assert.Len(t, importDef, 2)
		importDefMap := importDef[0]
		assert.Contains(t, importDefMap, "some_definition_file")
		importDefMap = importDef[1]
		assert.Contains(t, importDefMap, "another_definition_file")
	}
}
