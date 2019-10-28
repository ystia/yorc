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

package openstack

import (
	"context"
	"path"
	"strconv"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
)

func testGenerateOSBSVolumeSizeConvert(t *testing.T, srv1 *testutil.TestServer) {
	t.Parallel()
	log.SetDebug(true)

	depID := path.Base(t.Name())
	yamlName := "testdata/OSBaseImports.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), depID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")

	locationProps := config.DynamicMap{"region": "Region_" + depID}
	var cfg config.Configuration
	g := osGenerator{}

	var testData = []struct {
		nodeName     string
		inputSize    string
		expectedSize int
	}{
		{"volume1", "1", 1},
		{"volume10000000", "100", 1},
		{"volume10000000", "1500 M", 2},
		{"volume1GB", "1GB", 1},
		{"volume1GBS", "1      GB", 1},
		{"volume1GiB", "1 GiB", 2},
		{"volume2GIB", "2 GIB", 3},
		{"volume1TB", "1 tb", 1000},
		{"volume1TiB", "1 TiB", 1100},
	}
	nodesPrefix := path.Join(consulutil.DeploymentKVPrefix, depID, "topology/nodes")
	for i, tt := range testData {
		t.Log("Registering Key")
		// Create a test key/value pair
		data := make(map[string][]byte)
		data[path.Join(nodesPrefix, tt.nodeName, "type")] = []byte("yorc.nodes.openstack.BlockStorage")
		data[path.Join(nodesPrefix, tt.nodeName, "properties/size")] = []byte(tt.inputSize)

		srv1.PopulateKV(t, data)
		bsv, err := g.generateOSBSVolume(cfg, locationProps, depID, tt.nodeName, strconv.Itoa(i))
		assert.Nil(t, err)
		assert.Equal(t, tt.expectedSize, bsv.Size)
		// Default region
		assert.Equal(t, "Region_"+depID, bsv.Region)
	}
}

func testGenerateOSBSVolumeSizeConvertError(t *testing.T, srv1 *testutil.TestServer) {
	t.Parallel()
	log.SetDebug(true)

	depID := path.Base(t.Name())
	yamlName := "testdata/OSBaseImports.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), depID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")

	locationProps := config.DynamicMap{"region": "Region_" + depID}
	var cfg config.Configuration
	g := osGenerator{}

	var testData = []struct {
		nodeName  string
		inputSize string
	}{
		{"volume1", "1 bar"},
		{"volume2", "100 BAZ"},
		{"volume3", "M 1500"},
		{"volume4", "GB"},
	}
	nodesPrefix := path.Join(consulutil.DeploymentKVPrefix, depID, "topology/nodes")
	for i, tt := range testData {
		t.Log("Registering Key")
		// Create a test key/value pair
		data := make(map[string][]byte)
		data[path.Join(nodesPrefix, tt.nodeName, "type")] = []byte("yorc.nodes.openstack.BlockStorage")
		data[path.Join(nodesPrefix, tt.nodeName, "properties/size")] = []byte(tt.inputSize)

		srv1.PopulateKV(t, data)
		_, err := g.generateOSBSVolume(cfg, locationProps, depID, tt.nodeName, strconv.Itoa(i))
		assert.NotNil(t, err)
	}
}

func testGenerateOSBSVolumeMissingSize(t *testing.T, srv1 *testutil.TestServer) {
	t.Parallel()
	log.SetDebug(true)

	depID := path.Base(t.Name())
	yamlName := "testdata/OSBaseImports.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), depID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")

	locationProps := config.DynamicMap{"region": "Region_" + depID}
	var cfg config.Configuration
	g := osGenerator{}

	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	nodeName := "volumeMissingSize"

	data[path.Join(consulutil.DeploymentKVPrefix, depID, "topology/nodes", nodeName, "type")] = []byte("yorc.nodes.openstack.BlockStorage")

	srv1.PopulateKV(t, data)
	_, err = g.generateOSBSVolume(cfg, locationProps, depID, nodeName, "0")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Missing mandatory property 'size'")
}

func testGenerateOSBSVolumeWrongType(t *testing.T, srv1 *testutil.TestServer) {
	t.Parallel()
	log.SetDebug(true)

	depID := path.Base(t.Name())
	yamlName := "testdata/OSBaseImports.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), depID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")

	locationProps := config.DynamicMap{"region": "Region_" + depID}
	var cfg config.Configuration
	g := osGenerator{}

	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	nodeName := "volumeWrongType"

	data[path.Join(consulutil.DeploymentKVPrefix, depID, "topology/nodes", nodeName, "type")] = []byte("someorchestrator.nodes.openstack.BlockStorage")

	srv1.PopulateKV(t, data)
	_, err = g.generateOSBSVolume(cfg, locationProps, depID, nodeName, "0")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unsupported node type for")
}

func testGenerateOSBSVolumeCheckOptionalValues(t *testing.T, srv1 *testutil.TestServer) {
	t.Parallel()
	log.SetDebug(true)

	depID := path.Base(t.Name())
	yamlName := "testdata/OSBaseImports.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), depID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")

	locationProps := config.DynamicMap{"region": "Region_" + depID}
	var cfg config.Configuration
	g := osGenerator{}

	t.Log("Registering Key")
	// Create a test key/value pair
	nodeName := "volumeOpts"

	nodePrefix := path.Join(consulutil.DeploymentKVPrefix, depID, "topology/nodes", nodeName)

	data := make(map[string][]byte)
	data[path.Join(nodePrefix, "type")] = []byte("yorc.nodes.openstack.BlockStorage")
	data[path.Join(nodePrefix, "properties/size")] = []byte("1 GB")
	data[path.Join(nodePrefix, "properties/availability_zone")] = []byte("az1")
	data[path.Join(nodePrefix, "properties/region")] = []byte("Region2")

	srv1.PopulateKV(t, data)
	bsv, err := g.generateOSBSVolume(cfg, locationProps, depID, nodeName, "0")
	assert.Nil(t, err)
	assert.Equal(t, "az1", bsv.AvailabilityZone)
	assert.Equal(t, "Region2", bsv.Region)
}

func testComputeBootVolumeWrongSize(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()
	log.SetDebug(true)

	depID := path.Base(t.Name())
	yamlName := "testdata/BootVolumeWrongSize.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), depID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")

	_, err = computeBootVolume(depID, "Compute")
	require.Error(t, err, "Expected a failure to parse %s boot volume definition", yamlName)
}

func testComputeBootVolumeWrongType(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()
	log.SetDebug(true)

	depID := path.Base(t.Name())
	yamlName := "testdata/BootVolumeWrongType.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), depID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")

	_, err = computeBootVolume(depID, "Compute")
	require.Error(t, err, "Expected a failure to parse %s boot volume definition", yamlName)
}
