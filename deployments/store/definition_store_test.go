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
	srv, cc := newTestConsulInstance(t)
	defer srv.Stop()

	t.Run("StoreTests", func(t *testing.T) {
		t.Run("TestTypesPath", func(t *testing.T) {
			testTypesPath(t, cc.KV())
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

func storeCommonTypePath(ctx context.Context, t *testing.T, paths []string) {
	_, errGrp, consulStore := consulutil.WithContext(ctx)
	for _, p := range paths {
		consulStore.StoreConsulKeyAsString(path.Join(consulutil.CommonsTypesKVPrefix, p, ".exist"), "1")
	}
	require.NoError(t, errGrp.Wait())
}

// restTypePath ais to test getLatestCommonsTypesPaths
// WIP
func testTypesPath(t *testing.T, kv *api.KV) {
	log.Printf("Try to execute testTypesPath")
	log.SetDebug(true)

	tests := []struct {
		name          string
		existingPaths []string
		want          []string
		wantErr       bool
	}{
		{"NoPath", nil, []string{}, false},
		{"PathSimple", []string{"toto/1.0.0", "zuzu/2.0.0"}, []string{path.Join(consulutil.CommonsTypesKVPrefix, "toto/1.0.0"), path.Join(consulutil.CommonsTypesKVPrefix, "zuzu/2.0.0")}, false},
		{"PathMultiVersion", []string{"toto/1.0.0", "toto/1.0.1", "toto/1.1.1", "zuzu/2.0.0"}, []string{path.Join(consulutil.CommonsTypesKVPrefix, "toto/1.1.1"), path.Join(consulutil.CommonsTypesKVPrefix, "zuzu/2.0.0")}, false},
	}

	for _, tt := range tests {
		_, err := kv.DeleteTree(consulutil.CommonsTypesKVPrefix, nil)
		require.NoError(t, err)
		storeCommonTypePath(context.Background(), t, tt.existingPaths)
		paths, err := getLatestCommonsTypesPaths()
		assert.Equal(t, tt.wantErr, err != nil, "Actual error: %v while expecting error: %v", err, tt.wantErr)
		assert.Equal(t, tt.want, paths)
	}

}
