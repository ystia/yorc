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

type ImportMapInterface map[string][]map[string]ImportDefinition

func TestGroupedImportsParallel(t *testing.T) {
	t.Run("groupImports", func(t *testing.T) {
		t.Run("TestimportDefinitionConcreteUnmarshalYAMLSimpleGrammar", importDefinitionConcreteUnmarshalYAMLSimpleGrammar)
	})
}

func importDefinitionConcreteUnmarshalYAMLSimpleGrammar(t *testing.T) {
	t.Parallel()
	var data = `
imports:
  - some_definition_file: path1/path2/some_defs.yaml
  - another_definition_file:
      file: path1/path2/file2.yaml
      repository: my_service_catalog
      namespace_uri: http://mycompany.com/tosca/1.0/platform
      namespace_prefix: mycompany
`
	importMap := ImportMapInterface{}
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
