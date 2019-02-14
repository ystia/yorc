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
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/ystia/yorc/v3/log"
)

func testGeneratePoolIP(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()
	log.SetDebug(true)

	indexSuffix := path.Base(t.Name())
	g := osGenerator{}
	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	ipURL := "node/NetworkFIP" + indexSuffix
	data[ipURL+"/type"] = []byte("yorc.nodes.openstack.FloatingIP")
	data[ipURL+"/properties/floating_network_name"] = []byte("Public_Network")

	srv1.PopulateKV(t, data)
	gia, err := g.generateFloatingIP(kv, ipURL, "0")
	assert.Nil(t, err)
	assert.Equal(t, "Public_Network", gia.Pool)
	assert.False(t, gia.IsIP)
}

func testGenerateSingleIP(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()
	log.SetDebug(true)

	indexSuffix := path.Base(t.Name())
	g := osGenerator{}
	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	ipURL := "node/NetworkFIP" + indexSuffix
	data[ipURL+"/type"] = []byte("yorc.nodes.openstack.FloatingIP")
	data[ipURL+"/properties/ip"] = []byte("10.0.0.2")

	srv1.PopulateKV(t, data)
	gia, err := g.generateFloatingIP(kv, ipURL, "0")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.2", gia.Pool)
	assert.True(t, gia.IsIP)
}

func testGenerateMultipleIP(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()
	log.SetDebug(true)

	indexSuffix := path.Base(t.Name())
	g := osGenerator{}
	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	ipURL := "node/NetworkFIP" + indexSuffix
	data[ipURL+"/type"] = []byte("yorc.nodes.openstack.FloatingIP")
	data[ipURL+"/properties/ip"] = []byte("10.0.0.2,10.0.0.4,10.0.0.5,10.0.0.6")

	srv1.PopulateKV(t, data)
	gia, err := g.generateFloatingIP(kv, ipURL, "0")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.2,10.0.0.4,10.0.0.5,10.0.0.6", gia.Pool)
	assert.True(t, gia.IsIP)
	ips := strings.Split(gia.Pool, ",")
	assert.Len(t, ips, 4)
}
