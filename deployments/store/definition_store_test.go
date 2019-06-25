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

package store

import (
	"context"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/log"
)

// TestRunDefinitionStoreTests aims to run a max of tests on store functions
func TestRunDefinitionStoreTests(t *testing.T) {
	log.Printf("TestRunDefinitionStoreTests")
	srv, _ := newTestConsulInstance(t)
	defer srv.Stop()

	t.Run("StoreTests", func(t *testing.T) {
		t.Run("TestTypesPath", func(t *testing.T) {
			storeSomething()
			testTypesPath(t)
		})
	})
}

// newTestConsulInstance creates and configures Conul instance for tests
// can't use util functions from testutil package in order to avoid import cycles
func newTestConsulInstance(t *testing.T) (*testutil.TestServer, *api.Client) {
	log.Printf("Try to create consul instance")
	logLevel := "debug"
	if isCI, ok := os.LookupEnv("CI"); ok && isCI == "true" {
		logLevel = "warn"
	}
	log.Printf("Log level ok")
	cb := func(c *testutil.TestServerConfig) {
		c.Args = []string{"-ui"}
		c.LogLevel = logLevel
	}
	srv1, err := testutil.NewTestServerConfig(cb)
	if err != nil {
		t.Fatalf("Failed to create consul server: %v", err)
	}
	log.Printf("TestServerConfig ok")

	cfg := config.Configuration{
		Consul: config.Consul{
			Address:        srv1.HTTPAddr,
			PubMaxRoutines: config.DefaultConsulPubMaxRoutines,
		},
	}

	client, err := cfg.GetNewConsulClient()
	assert.Nil(t, err)

	kv := client.KV()
	consulutil.InitConsulPublisher(cfg.Consul.PubMaxRoutines, kv)
	return srv1, client
}

// storeSomething allows to store some type definitions to consul under consulutil.CommonsTypesKVPrefix
func storeSomething() {
	prefix := consulutil.CommonsTypesKVPrefix + "/"
	resources := getSomeResourcesToStore(2)
	ctx := context.Background()
	storeDefinition(ctx, prefix, resources)
}

// storeDefinition uses the consulStore from the context
func storeDefinition(ctx context.Context, origin string, resDefinitions map[string]string) {
	ctx, _, consulStore := consulutil.WithContext(ctx)
	for name, value := range resDefinitions {
		consulStore.StoreConsulKeyAsString(path.Join(origin, name), value)
	}
}

// getSomeResourcesToStore creates a map for resource definitiaons to store
// for each of the nbres respurces, construct 2 resource names corresponding to 2 versions of the resource
func getSomeResourcesToStore(nbres int) map[string]string {
	res := make(map[string]string, nbres)
	value1 := "abc"
	value2 := "xyz"
	for i := 0; i < nbres; i++ {
		resName := "test_types" + strconv.Itoa(i)
		resName1 := resName + "/" + "1.1.1"
		res[resName1] = value1
		resName2 := resName + "/" + "2.1.1"
		res[resName2] = value2
	}
	return res
}

// restTypePath ais to test getLatestCommonsTypesPaths
// WIP
func testTypesPath(t *testing.T) {
	log.Printf("Try to execute testTypesPath")
	log.SetDebug(true)

	paths, err := getLatestCommonsTypesPaths()

	log.Printf("Length of paths is " + string(len(paths)))

	require.Nil(t, err)
	require.Len(t, paths, 0)
}
