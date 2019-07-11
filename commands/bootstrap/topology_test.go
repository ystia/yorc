// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package bootstrap

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/commands"
)

// TestCreateTopology creates a topology from templates

func TestCreateTopology(t *testing.T) {

	tempdir, err := ioutil.TempDir("", path.Base(t.Name()))
	require.NoError(t, err, "Failed to create temporary directory")
	defer os.RemoveAll(tempdir)

	workingDirectoryPath = tempdir
	resourcesDir := filepath.Join(workingDirectoryPath, "bootstrapResources")

	resourcesZip := filepath.Join("resources", "topology", "tosca_types.zip")
	// copying resources zip file to temporary directory
	input, err := ioutil.ReadFile(resourcesZip)
	require.NoError(t, err, "Failed to ready resource zip file %s", resourcesZip)

	err = ioutil.WriteFile(filepath.Join(tempdir, "tosca_types.zip"), input, 0744)
	require.NoError(t, err, "Failed to copy resources zip file %s to temp dir", resourcesZip)

	err = extractResources(resourcesZip,
		filepath.Join(resourcesDir, "topology"))
	require.NoError(t, err, "Failed to extract resources")

	configuration := commands.GetConfig()
	err = initializeInputs("testdata/inputs_test.yaml", resourcesDir, configuration)
	require.NoError(t, err, "Failed to initialize inputs")

	err = createTopology(tempdir)
	require.NoError(t, err, "Failed to create topology")

}
