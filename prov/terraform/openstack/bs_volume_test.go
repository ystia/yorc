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
	"path"
	"strconv"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/log"
)

func testGenerateOSBSVolumeSizeConvert(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()
	log.SetDebug(true)

	indexSuffix := path.Base(t.Name())
	cfg := config.Configuration{
		Infrastructures: map[string]config.DynamicMap{
			infrastructureName: config.DynamicMap{
				"region": "Region_" + indexSuffix,
			}}}
	g := osGenerator{}

	var testData = []struct {
		volURL       string
		inputSize    string
		expectedSize int
	}{
		{"node_" + indexSuffix + "/volume1", "1", 1},
		{"node_" + indexSuffix + "/volume10000000", "100", 1},
		{"node_" + indexSuffix + "/volume10000000", "1500 M", 2},
		{"node_" + indexSuffix + "/volume1GB", "1GB", 1},
		{"node_" + indexSuffix + "/volume1GBS", "1      GB", 1},
		{"node_" + indexSuffix + "/volume1GiB", "1 GiB", 2},
		{"node_" + indexSuffix + "/volume2GIB", "2 GIB", 3},
		{"node_" + indexSuffix + "/volume1TB", "1 tb", 1000},
		{"node_" + indexSuffix + "/volume1TiB", "1 TiB", 1100},
	}
	for i, tt := range testData {
		t.Log("Registering Key")
		// Create a test key/value pair
		data := make(map[string][]byte)
		data[tt.volURL+"/type"] = []byte("yorc.nodes.openstack.BlockStorage")
		data[tt.volURL+"/properties/size"] = []byte(tt.inputSize)

		srv1.PopulateKV(t, data)
		bsv, err := g.generateOSBSVolume(kv, cfg, tt.volURL, strconv.Itoa(i))
		assert.Nil(t, err)
		assert.Equal(t, tt.expectedSize, bsv.Size)
		// Default region
		assert.Equal(t, "Region_"+indexSuffix, bsv.Region)
	}
}

func testGenerateOSBSVolumeSizeConvertError(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()
	log.SetDebug(true)

	indexSuffix := path.Base(t.Name())
	cfg := config.Configuration{
		Infrastructures: map[string]config.DynamicMap{
			infrastructureName: config.DynamicMap{
				"region": "Region_" + indexSuffix,
			}}}
	g := osGenerator{}

	var testData = []struct {
		volURL    string
		inputSize string
	}{
		{"node_" + indexSuffix + "/volume1", "1 bar"},
		{"node_" + indexSuffix + "/volume2", "100 BAZ"},
		{"node_" + indexSuffix + "/volume3", "M 1500"},
		{"node_" + indexSuffix + "/volume4", "GB"},
	}
	for i, tt := range testData {
		t.Log("Registering Key")
		// Create a test key/value pair
		data := make(map[string][]byte)
		data[tt.volURL+"/type"] = []byte("yorc.nodes.openstack.BlockStorage")
		data[tt.volURL+"/properties/size"] = []byte(tt.inputSize)

		srv1.PopulateKV(t, data)
		_, err := g.generateOSBSVolume(kv, cfg, tt.volURL, strconv.Itoa(i))
		assert.NotNil(t, err)
	}
}

func testGenerateOSBSVolumeMissingSize(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()
	log.SetDebug(true)

	indexSuffix := path.Base(t.Name())
	cfg := config.Configuration{
		Infrastructures: map[string]config.DynamicMap{
			infrastructureName: config.DynamicMap{
				"region": "Region_" + indexSuffix,
			}}}
	g := osGenerator{}

	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	data["vol_"+indexSuffix+"/type"] = []byte("yorc.nodes.openstack.BlockStorage")

	srv1.PopulateKV(t, data)
	_, err := g.generateOSBSVolume(kv, cfg, "vol_"+indexSuffix, "0")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Missing mandatory property 'size'")
}

func testGenerateOSBSVolumeWrongType(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()
	log.SetDebug(true)

	indexSuffix := path.Base(t.Name())
	cfg := config.Configuration{
		Infrastructures: map[string]config.DynamicMap{
			infrastructureName: config.DynamicMap{
				"region": "Region_" + indexSuffix,
			}}}
	g := osGenerator{}

	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	data["vol_"+indexSuffix+"/type"] = []byte("someorchestrator.nodes.openstack.BlockStorage")

	srv1.PopulateKV(t, data)
	_, err := g.generateOSBSVolume(kv, cfg, "vol_"+indexSuffix, "0")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unsupported node type for")
}

func testGenerateOSBSVolumeCheckOptionalValues(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()
	log.SetDebug(true)

	indexSuffix := path.Base(t.Name())
	cfg := config.Configuration{
		Infrastructures: map[string]config.DynamicMap{
			infrastructureName: config.DynamicMap{
				"region": "Region_" + indexSuffix,
			}}}
	g := osGenerator{}

	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	data["vol_"+indexSuffix+"/type"] = []byte("yorc.nodes.openstack.BlockStorage")
	data["vol_"+indexSuffix+"/properties/size"] = []byte("1 GB")
	data["vol_"+indexSuffix+"/properties/availability_zone"] = []byte("az1")
	data["vol_"+indexSuffix+"/properties/region"] = []byte("Region2")

	srv1.PopulateKV(t, data)
	bsv, err := g.generateOSBSVolume(kv, cfg, "vol_"+indexSuffix, "0")
	assert.Nil(t, err)
	assert.Equal(t, "az1", bsv.AvailabilityZone)
	assert.Equal(t, "Region2", bsv.Region)
}
