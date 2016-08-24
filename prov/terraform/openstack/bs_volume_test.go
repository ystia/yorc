package openstack

import (
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/log"
	"testing"
)

func TestGroupedVolumeParallel(t *testing.T) {
	t.Run("groupVolume", func(t *testing.T) {
		t.Run("generateOSBSVolumeSizeConvert", generateOSBSVolumeSizeConvert)
		t.Run("Test_generateOSBSVolumeSizeConvertError", generateOSBSVolumeSizeConvertError)
		t.Run("Test_generateOSBSVolumeMissingSize", generateOSBSVolumeMissingSize)
		t.Run("generateOSBSVolumeWrongType", generateOSBSVolumeWrongType)
		t.Run("Test_generateOSBSVolumeCheckOptionalValues", generateOSBSVolumeCheckOptionalValues)
	})
}

func generateOSBSVolumeSizeConvert(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	assert.Nil(t, err)

	kv := client.KV()
	cfg := config.Configuration{}
	g := NewGenerator(kv, cfg)

	var testData = []struct {
		volUrl       string
		inputSize    string
		expectedSize int
	}{
		{"node/volume1", "1", 1},
		{"node/volume10000000", "100", 1},
		{"node/volume10000000", "1500 M", 2},
		{"node/volume1GB", "1GB", 1},
		{"node/volume1GBS", "1      GB", 1},
		{"node/volume1GiB", "1 GiB", 2},
		{"node/volume2GIB", "2 GIB", 3},
		{"node/volume1TB", "1 tb", 1000},
		{"node/volume1TiB", "1 TiB", 1100},
	}
	for _, tt := range testData {
		t.Log("Registering Key")
		// Create a test key/value pair
		data := make(map[string][]byte)
		data[tt.volUrl+"/type"] = []byte("janus.nodes.openstack.BlockStorage")
		data[tt.volUrl+"/properties/size"] = []byte(tt.inputSize)

		srv1.PopulateKV(data)
		bsv, err := g.generateOSBSVolume(tt.volUrl)
		assert.Nil(t, err)
		assert.Equal(t, tt.expectedSize, bsv.Size)
		// Default region
		assert.Equal(t, "RegionOne", bsv.Region)
	}
}
func generateOSBSVolumeSizeConvertError(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	assert.Nil(t, err)

	kv := client.KV()
	cfg := config.Configuration{}
	g := NewGenerator(kv, cfg)

	var testData = []struct {
		volUrl    string
		inputSize string
	}{
		{"node/volume1", "1 bar"},
		{"node/volume2", "100 BAZ"},
		{"node/volume3", "M 1500"},
		{"node/volume4", "GB"},
	}
	for _, tt := range testData {
		t.Log("Registering Key")
		// Create a test key/value pair
		data := make(map[string][]byte)
		data[tt.volUrl+"/type"] = []byte("janus.nodes.openstack.BlockStorage")
		data[tt.volUrl+"/properties/size"] = []byte(tt.inputSize)

		srv1.PopulateKV(data)
		_, err := g.generateOSBSVolume(tt.volUrl)
		assert.NotNil(t, err)
	}
}

func generateOSBSVolumeMissingSize(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	assert.Nil(t, err)

	kv := client.KV()
	cfg := config.Configuration{}
	g := NewGenerator(kv, cfg)

	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	data["vol/type"] = []byte("janus.nodes.openstack.BlockStorage")

	srv1.PopulateKV(data)
	_, err = g.generateOSBSVolume("vol")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Missing mandatory property 'size'")
}
func generateOSBSVolumeWrongType(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	assert.Nil(t, err)

	kv := client.KV()
	cfg := config.Configuration{}
	g := NewGenerator(kv, cfg)

	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	data["vol/type"] = []byte("someorchestrator.nodes.openstack.BlockStorage")

	srv1.PopulateKV(data)
	_, err = g.generateOSBSVolume("vol")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unsupported node type for")
}

func generateOSBSVolumeCheckOptionalValues(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	assert.Nil(t, err)

	kv := client.KV()
	cfg := config.Configuration{}
	g := NewGenerator(kv, cfg)

	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	data["vol/type"] = []byte("janus.nodes.openstack.BlockStorage")
	data["vol/properties/size"] = []byte("1 GB")
	data["vol/properties/availability_zone"] = []byte("az1")
	data["vol/properties/region"] = []byte("Region2")

	srv1.PopulateKV(data)
	bsv, err := g.generateOSBSVolume("vol")
	assert.Nil(t, err)
	assert.Equal(t, "az1", bsv.AvailabilityZone)
	assert.Equal(t, "Region2", bsv.Region)
}
