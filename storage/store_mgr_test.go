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

package storage

import (
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/storage/store"
	"os"
	"path"
	"reflect"
	"testing"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulStoragePackageTests(t *testing.T) {
	srv, _, cfg := store.NewTestConsulInstance(t)
	defer func() {
		srv.Stop()
		os.RemoveAll(cfg.WorkingDirectory)
	}()

	t.Run("groupStorage", func(t *testing.T) {
		t.Run("testLoadStoresWithNoStorageConfig", func(t *testing.T) {
			testLoadStoresWithNoStorageConfig(t, srv, cfg)
		})
		t.Run("testLoadStoresWithPartialStorageConfig", func(t *testing.T) {
			testLoadStoresWithPartialStorageConfig(t, srv, cfg)
		})
		t.Run("testLoadStoresWithMissingPassphraseForCipherFileCache", func(t *testing.T) {
			testLoadStoresWithMissingPassphraseForCipherFileCache(t, srv, cfg)
		})
	})
}

func testLoadStoresWithNoStorageConfig(t *testing.T, srv1 *testutil.TestServer, cfg config.Configuration) {
	err := LoadStores(cfg)
	require.NoError(t, err)

	// Check default configuration has been saved in Consul
	MapStores, err := consulutil.List(consulutil.StoresPrefix)
	require.NoError(t, err)
	require.NotNil(t, MapStores)
	require.Len(t, MapStores, 2)

	defaultStores := buildDefaultConfigStores()
	require.NotNil(t, defaultStores)
	require.Len(t, defaultStores, 2)

	for k, v := range MapStores {
		key := path.Base(k)
		switch key {
		case "defaultConsulStore":
			if reflect.DeepEqual(v, defaultStores[1]) {
				t.Errorf("LoadStores() = %v, want %v", v, defaultStores[1])
			}
		case "defaultFileStoreWithCache":
			if reflect.DeepEqual(v, defaultStores[0]) {
				t.Errorf("LoadStores() = %v, want %v", v, defaultStores[0])
			}
		default:
			t.Errorf("unexpected key:%q", key)
		}
	}

}

func testLoadStoresWithPartialStorageConfig(t *testing.T, srv1 *testutil.TestServer, cfg config.Configuration) {
	// Reset once to allow reload config
	once.Reset()

	myStore := config.Store{
		Name:           "myPersonalStore",
		Implementation: "cipherFileCache",
		Types:          []string{"Log", "Event"},
		Properties: config.DynamicMap{
			"passphrase": "myverystrongpasswordo32bitlength",
		},
	}

	cfg.Storage = config.Storage{
		Reset:  true,
		Stores: []config.Store{myStore},
	}

	err := LoadStores(cfg)
	require.NoError(t, err)

	// Check custom configuration + default complement has been saved in Consul
	MapStores, err := consulutil.List(consulutil.StoresPrefix)
	require.NoError(t, err)
	require.NotNil(t, MapStores)
	require.Len(t, MapStores, 2)

	defaultStores := buildDefaultConfigStores()
	require.NotNil(t, defaultStores)
	require.Len(t, defaultStores, 2)

	for k, v := range MapStores {
		key := path.Base(k)
		switch key {
		case "myPersonalStore":
			if reflect.DeepEqual(v, myStore) {
				t.Errorf("LoadStores() = %v, want %v", v, myStore)
			}
		case "defaultFileStoreWithCache":
			if reflect.DeepEqual(v, defaultStores[0]) {
				t.Errorf("LoadStores() = %v, want %v", v, defaultStores[0])
			}
		default:
			t.Errorf("unexpected key:%q", key)
		}
	}
}

func testLoadStoresWithMissingPassphraseForCipherFileCache(t *testing.T, srv1 *testutil.TestServer, cfg config.Configuration) {
	// Reset once to allow reload config
	once.Reset()

	myStore := config.Store{
		Name:           "myPersonalStore",
		Implementation: "cipherFileCache",
		Types:          []string{"Log", "Event"},
		Properties:     config.DynamicMap{
			//			"passphrase": "myverystrongpasswordo32bitlength",
		},
	}

	cfg.Storage = config.Storage{
		Reset:  true,
		Stores: []config.Store{myStore},
	}

	err := LoadStores(cfg)
	require.Error(t, err)

	// Check stores config has been cleared in Consul
	MapStores, err := consulutil.List(consulutil.StoresPrefix)
	require.NoError(t, err)
	require.NotNil(t, MapStores)
	require.Len(t, MapStores, 0)
}
