package openstack

import (
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"novaforge.bull.com/starlings-janus/janus/log"
	"testing"
)

func Test_generateOSBSVolumeSizeConvert(t *testing.T) {
	log.SetDebug(true)
	srv1 := testutil.NewTestServerConfig(t, nil)
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

	kv := client.KV()
	g := NewGenerator(kv)

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
func Test_generateOSBSVolumeSizeConvertError(t *testing.T) {
	log.SetDebug(true)
	srv1 := testutil.NewTestServerConfig(t, nil)
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

	kv := client.KV()
	g := NewGenerator(kv)

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

func Test_generateOSBSVolumeMissingSize(t *testing.T) {
	log.SetDebug(true)
	srv1 := testutil.NewTestServerConfig(t, nil)
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

	kv := client.KV()
	g := NewGenerator(kv)

	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	data["vol/type"] = []byte("janus.nodes.openstack.BlockStorage")

	srv1.PopulateKV(data)
	_, err = g.generateOSBSVolume("vol")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Missing mandatory property 'size'")
}
func Test_generateOSBSVolumeWrongType(t *testing.T) {
	log.SetDebug(true)
	srv1 := testutil.NewTestServerConfig(t, nil)
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

	kv := client.KV()
	g := NewGenerator(kv)

	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	data["vol/type"] = []byte("someorchestrator.nodes.openstack.BlockStorage")

	srv1.PopulateKV(data)
	_, err = g.generateOSBSVolume("vol")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unsupported node type for")
}

func Test_generateOSBSVolumeCheckOptionalValues(t *testing.T) {
	log.SetDebug(true)
	srv1 := testutil.NewTestServerConfig(t, nil)
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

	kv := client.KV()
	g := NewGenerator(kv)

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
