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
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
)

// TestRunDefinitionStoreTests aims to run a max of tests on store functions
func TestRunDefinitionStoreTests(t *testing.T) {
	// create consul server and consul client
	cfg := setupTestConfig(t)
	srv, _ := newTestConsulInstance(t, &cfg)
	defer func() {
		srv.Stop()
		os.RemoveAll(cfg.WorkingDirectory)
	}()
	t.Run("StoreTests", func(t *testing.T) {
		t.Run("TestTypesPath", func(t *testing.T) {
			testTypesPath(t)
		})
	})
}

// SetupTestConfig sets working directory configuration
// Warning: You need to defer the working directory removal
// Note: can't use util functions from testutil package in order to avoid import cycles
func setupTestConfig(t testing.TB) config.Configuration {
	workingDir, err := ioutil.TempDir(os.TempDir(), "work")
	assert.Nil(t, err)
	return config.Configuration{
		WorkingDirectory: workingDir,
	}
}

// newTestConsulInstance creates and configures Consul instance
// for testing functions in the store package
// Note: can't use util functions from testutil package in order to avoid import cycles
func newTestConsulInstance(t *testing.T, cfg *config.Configuration) (*testutil.TestServer, *api.Client) {
	logLevel := "debug"
	if isCI, ok := os.LookupEnv("CI"); ok && isCI == "true" {
		logLevel = "warn"
	}
	cb := func(c *testutil.TestServerConfig) {
		c.Args = []string{"-ui"}
		c.LogLevel = logLevel
	}
	srv1, err := testutil.NewTestServerConfig(cb)
	if err != nil {
		t.Fatalf("Failed to create consul server: %v", err)
	}

	cfg.Consul.Address = srv1.HTTPAddr
	cfg.Consul.PubMaxRoutines = config.DefaultConsulPubMaxRoutines

	client, err := cfg.GetNewConsulClient()
	assert.Nil(t, err)

	kv := client.KV()
	consulutil.InitConsulPublisher(cfg.Consul.PubMaxRoutines, kv)

	waitForConsulReadiness(fmt.Sprintf("http://%s", srv1.HTTPAddr))
	// Load stores
	// Load main stores used for deployments, logs, events
	err = storage.LoadStores(*cfg)
	assert.Nil(t, err)
	return srv1, client
}

func storeCommonTypePath(ctx context.Context, t *testing.T, paths []string) {
	// Store someValue (here "1") with key "_yorc/commons_types/some_type/some_version/some_name"
	// Where "some_type/some_value" is one of the existingPath slice element provided in the tests structure,
	// for example "toto/1.0.0"
	someValue := "1"
	for _, p := range paths {
		// Store value "1" with key _yorc/commons_types/toto/1.0.0/.exist
		err := storage.GetStore(types.StoreTypeDeployment).Set(ctx, path.Join(consulutil.CommonsTypesKVPrefix, p), someValue)
		require.NoError(t, err)
	}

}

// Duplicated here because of cyclic imports. Orginally placed in testutil/helper.go
// waits for a known leader and an index of 2 or more to be observed to confirm leader election is done.
// Inspired by : https://github.com/hashicorp/consul/blob/master/sdk/testutil/server.go#L406
func waitForConsulReadiness(consulHTTPEndpoint string) {
	for {
		leader, index, _ := getConsulLeaderAndIndex(consulHTTPEndpoint)
		if leader != "" && index >= 2 {
			return
		}

		<-time.After(2 * time.Second)
	}
}

func getConsulLeaderAndIndex(consulHTTPEndpoint string) (string, int64, error) {
	// Query the API and check the status code.
	resp, err := http.Get(fmt.Sprintf("%s/v1/catalog/nodes", consulHTTPEndpoint))
	if err != nil {
		return "", -1, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", -1, errors.New(resp.Status)
	}
	leader := resp.Header.Get("X-Consul-KnownLeader")
	if leader != "true" {
		return "", -1, errors.Errorf("Consul leader status: %#v", leader)
	}
	index, err := strconv.ParseInt(resp.Header.Get("X-Consul-Index"), 10, 64)
	if err != nil {
		errors.Wrap(err, "bad consul index")
	}
	return leader, index, nil
}

// testTypePath aims to test getLatestCommonsTypesPath by storing some_value with a path constructed by joining :
// - consulutil.CommonsTypesKVPrefix
// - some_type/some_value
// - some_name
func testTypesPath(t *testing.T) {
	log.SetDebug(true)
	ctx := context.Background()
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
		err := storage.GetStore(types.StoreTypeDeployment).Delete(ctx, consulutil.CommonsTypesKVPrefix, true)
		require.NoError(t, err)
		storeCommonTypePath(context.Background(), t, tt.existingPaths)
		paths, err := getLatestCommonsTypesKeyPaths()
		assert.Equal(t, tt.wantErr, err != nil, "Actual error: %v while expecting error: %v", err, tt.wantErr)
		assert.Equal(t, tt.want, paths)
	}

}
