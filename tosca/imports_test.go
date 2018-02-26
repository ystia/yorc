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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"testing"
)

type ImportsMap map[string][]ImportDefinition

func TestGroupedImportsParallel(t *testing.T) {
	t.Run("groupImports", func(t *testing.T) {
		t.Run("TestimportDefinitionConcreteUnmarshalYAMLGrammar", importDefinitionConcreteUnmarshalYAMLGrammar)
		t.Run("TestimportWrongDefinitionUnmarshalYAMLGrammar", importWrongDefinitionUnmarshalYAMLGrammar)
	})
}

func importDefinitionConcreteUnmarshalYAMLGrammar(t *testing.T) {
	t.Parallel()
	var data = `
imports:
  - some_definition_file: path0/some_defs.yaml
  - another_definition_file:
      file: path1/file2.yaml
      repository: my_service_catalog
      namespace_uri: http://mycompany.com/tosca/1.0/platform
      namespace_prefix: mycompany
  - file: path2/some_defs.yaml
  - file: path3/some_defs.yaml
    repository: mySecondRepository
    namespace_prefix: mySecondNamespace
  - path4/some_defs.yml
  - {file: path5/some_defs.yml}
  - path6/some_defs.yml
`
	importMap := ImportsMap{}
	err := yaml.Unmarshal([]byte(data), &importMap)
	require.NoError(t, err, "Failure umarshalling %s", data)
	assert.Len(t, importMap, 1)
	assert.Contains(t, importMap, "imports")
	imports := importMap["imports"]
	assert.Len(t, imports, 7)

	assert.Contains(t, imports[0].File, "path0")

	assert.Contains(t, imports[1].File, "path1")
	assert.Equal(t, imports[1].Repository, "my_service_catalog")
	assert.Equal(t, imports[1].NamespacePrefix, "mycompany")

	assert.Contains(t, imports[2].File, "path2")

	assert.Contains(t, imports[3].File, "path3")
	assert.Equal(t, imports[3].Repository, "mySecondRepository")
	assert.Equal(t, imports[3].NamespacePrefix, "mySecondNamespace")

	assert.Contains(t, imports[4].File, "path4")

	assert.Contains(t, imports[5].File, "path5")

	assert.Contains(t, imports[6].File, "path6")
}

func importWrongDefinitionUnmarshalYAMLGrammar(t *testing.T) {
	t.Parallel()
	var typoInFileKeyData = `
imports:
  - fail: path0/some_defs.yaml
    repository: mySecondRepository
    namespace_prefix: mySecondNamespace
`
	importMap := ImportsMap{}
	err := yaml.Unmarshal([]byte(typoInFileKeyData), &importMap)
	require.Error(t, err, "Failed to detect missing key 'file' in imports")
	assert.Contains(t, err.Error(), "Missing required key")
}
