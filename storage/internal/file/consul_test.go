// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package file

import (
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/storage/store"
	"os"
	"testing"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunFileStoragePackageTests(t *testing.T) {
	srv, _ := newTestConsulInstance(t)
	defer srv.Stop()

	t.Run("groupStorage", func(t *testing.T) {
		t.Run("testConsulTypes", func(t *testing.T) {
			testFileStoreWithEncryption(t, srv)
		})
		t.Run("testConsulStore", func(t *testing.T) {
			testFileStoreTypesWithEncryption(t, srv)
		})
	})
}

// This is a private Consul server instantiation as done in github.com/ystia/yorc/v4/testutil
// This allows avoiding cyclic dependencies with deployments store package
func newTestConsulInstance(t testing.TB) (*testutil.TestServer, *api.Client) {
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

func testFileStoreWithEncryption(t *testing.T, srv1 *testutil.TestServer) {
	rootDir := "./.work_" + t.Name()
	defer func() {
		err := os.RemoveAll(rootDir)
		require.NoError(t, err, "failed to remove test working directory:%q", rootDir)
	}()
	fileStore, err := NewStore(rootDir, true)
	require.NoError(t, err, "failed to instantiate new store")
	store.CommonStoreTest(t, fileStore)
}

func testFileStoreTypesWithEncryption(t *testing.T, srv1 *testutil.TestServer) {
	rootDir := "./.work_" + t.Name()
	defer func() {
		err := os.RemoveAll(rootDir)
		require.NoError(t, err, "failed to remove test working directory:%q", rootDir)
	}()
	fileStore, err := NewStore(rootDir, true)
	require.NoError(t, err, "failed to instantiate new store")
	store.CommonStoreTestAllTypes(t, fileStore)
}
