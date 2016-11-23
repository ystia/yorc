package openstack

import (
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/log"
	"strings"
	"testing"
)

func TestGroupedIpParallel(t *testing.T) {
	t.Run("groupVolume", func(t *testing.T) {
		t.Run("generatePoolIp", generatePoolIp)
		t.Run("generateSingleIp", generateSingleIp)
		t.Run("generateMultipleIp", generateMultipleIp)
	})
}

func generatePoolIp(t *testing.T) {
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
	ipUrl := "node/NetworkFIP"
	data[ipUrl+"/type"] = []byte("janus.nodes.openstack.FloatingIP")
	data[ipUrl+"/properties/floating_network_name"] = []byte("Public_Network")

	srv1.PopulateKV(data)
	gia, err := g.generateFloatingIP(ipUrl, "0")
	assert.Nil(t, err)
	assert.Equal(t, "Public_Network", gia.Pool)
	assert.False(t, gia.IsIp)
}

func generateSingleIp(t *testing.T) {
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
	ipUrl := "node/NetworkFIP"
	data[ipUrl+"/type"] = []byte("janus.nodes.openstack.FloatingIP")
	data[ipUrl+"/properties/ip"] = []byte("10.0.0.2")

	srv1.PopulateKV(data)
	gia, err := g.generateFloatingIP(ipUrl, "0")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.2", gia.Pool)
	assert.True(t, gia.IsIp)
}

func generateMultipleIp(t *testing.T) {
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
	ipUrl := "node/NetworkFIP"
	data[ipUrl+"/type"] = []byte("janus.nodes.openstack.FloatingIP")
	data[ipUrl+"/properties/ip"] = []byte("10.0.0.2,10.0.0.4,10.0.0.5,10.0.0.6")

	srv1.PopulateKV(data)
	gia, err := g.generateFloatingIP(ipUrl, "0")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.2,10.0.0.4,10.0.0.5,10.0.0.6", gia.Pool)
	assert.True(t, gia.IsIp)
	ips := strings.Split(gia.Pool, ",")
	assert.Len(t, ips, 4)
}
