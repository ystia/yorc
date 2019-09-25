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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

	topologyDir := filepath.Join(resourcesDir, "topology")
	srcResourcesDir := filepath.Join("resources", "topology")
	err = copyFiles(srcResourcesDir, topologyDir)
	require.NoError(t, err, "Failed to copy resources from %s to %s", srcResourcesDir, topologyDir)
	resourcesZip, _ := getResourcesZipPath()
	err = extractResources(resourcesZip, topologyDir)
	require.NoError(t, err, "Failed to extract resources")

	configuration := commands.GetConfig()
	err = initializeInputs("testdata/inputs_test.yaml", resourcesDir, configuration)
	require.NoError(t, err, "Failed to initialize inputs")

	// For the moment, viper when reading the configuration is lowercasing Map keys.
	// So a location name will be lowercased by viper
	assert.Equal(t, strings.ToLower("myLocation"), inputValues.Location.Name, "Unexpected location name in input values %+v", inputValues)

	err = createTopology(topologyDir)
	require.NoError(t, err, "Failed to create topology")

	// Check a topology file was created
	topologyFile := filepath.Join(topologyDir, "topology.yaml")
	_, err = os.Stat(topologyFile)
	require.NoError(t, err, "No topology.yaml file was generated")

}

func copyFiles(srcDir, dstDir string) error {
	err := os.MkdirAll(dstDir, 0700)
	if err != nil {
		return err
	}

	err = filepath.Walk(srcDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			input, newErr := ioutil.ReadFile(path)
			if newErr != nil {
				return newErr
			}

			newErr = ioutil.WriteFile(filepath.Join(dstDir, filepath.Base(path)), input, 0744)
			return newErr
		})

	return err

}
