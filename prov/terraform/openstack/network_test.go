package openstack

import (
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/config"
)

func TestGroupedNetworkParallel(t *testing.T) {
	t.Run("groupVolume", func(t *testing.T) {
		t.Run("generateNetwork", generateNetwork)
		t.Run("generateSubNetwork", generateSubNetwork)
	})
}

func generateNetwork(t *testing.T) {
	t.Parallel()
	srv1, err := testutil.NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create consul server: %v", err)
	}
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	assert.Nil(t, err)

	kv := client.KV()
	g := osGenerator{}

	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	ipURL := "node/NetworkFIP"
	data[ipURL+"/type"] = []byte("janus.nodes.openstack.Network")
	data[ipURL+"/properties/network_name"] = []byte("Public_Network")

	srv1.PopulateKV(t, data)
	net, err := g.generateNetwork(kv, config.Configuration{ ResourcesPrefix: "test-"}, ipURL, "0")
	assert.Nil(t, err)
	assert.Equal(t, "test-Public_Network", net.Name)
}

func generateSubNetwork(t *testing.T) {
	t.Parallel()
	srv1, err := testutil.NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create consul server: %v", err)
	}
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	assert.Nil(t, err)

	kv := client.KV()
	g := osGenerator{}

	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	ipURL := "node/NetworkFIP"
	data[ipURL+"/type"] = []byte("janus.nodes.openstack.Network")
	data[ipURL+"/properties/network_name"] = []byte("Public_Network")
	data[ipURL+"/properties/cidr"] = []byte("/24")
	data[ipURL+"/properties/gateway_ip"] = []byte("10.0.0.0")
	data[ipURL+"/properties/start_ip"] = []byte("10.0.0.1")
	data[ipURL+"/properties/end_ip"] = []byte("10.0.0.253")

	srv1.PopulateKV(t, data)
	net, err := g.generateSubnet(kv, config.Configuration{ OSRegion: "test", ResourcesPrefix: "test-"}, ipURL, "0", "nodeA")
	assert.Nil(t, err)
	assert.Equal(t, "/24", net.CIDR)
	assert.Equal(t, "10.0.0.0", net.GatewayIP)
	assert.Equal(t, 4, net.IPVersion)

}
