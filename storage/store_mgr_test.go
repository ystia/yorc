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
	"encoding/json"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/storage/store"
	"github.com/ystia/yorc/v4/storage/types"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulStoragePackageTests(t *testing.T) {
	cfg := store.SetupTestConfig(t)
	srv, _ := store.NewTestConsulInstance(t, &cfg)
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
		t.Run("testLoadStoresWithPartialStorageConfig2", func(t *testing.T) {
			testLoadStoresWithPartialStorageConfig2(t, srv, cfg)
		})
		t.Run("testLoadStoresWithMissingPassphraseForCipherFileCache", func(t *testing.T) {
			testLoadStoresWithMissingPassphraseForCipherFileCache(t, srv, cfg)
		})
		t.Run("testLoadStoresWithMissingMandatoryParameters", func(t *testing.T) {
			testLoadStoresWithMissingMandatoryParameters(t, srv, cfg)
		})
		t.Run("testLoadStoresWithGeneratedName", func(t *testing.T) {
			testLoadStoresWithGeneratedName(t, srv, cfg)
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

	defaultStores := getDefaultConfigStores(cfg)
	require.NotNil(t, defaultStores)
	require.Len(t, defaultStores, 2)

	for k, v := range MapStores {
		key := path.Base(k)
		switch key {
		case "defaultConsulStore":
			s := new(config.Store)
			err = json.Unmarshal(v, s)
			if !reflect.DeepEqual(*s, defaultStores[1]) {
				t.Errorf("LoadStores() = %v, want %v", v, defaultStores[1])
			}
		case "defaultFileStoreWithCache":
			s := new(config.Store)
			err = json.Unmarshal(v, s)
			if !reflect.DeepEqual(*s, defaultStores[0]) {
				t.Errorf("LoadStores() = %v, want %v", v, defaultStores[0])
			}
		default:
			t.Errorf("unexpected key:%q", key)
		}
	}

}

func testLoadStoresWithPartialStorageConfig2(t *testing.T, srv1 *testutil.TestServer, cfg config.Configuration) {
	// Reset once to allow reload config
	once.Reset()

	deploymentID := t.Name()

	obj := map[string]string{
		"key1": "content1",
		"key2": "content2",
		"key3": "content3",
	}
	b, err := json.Marshal(obj)
	require.NoError(t, err)

	logKeys := []string{path.Join(consulutil.LogsPrefix, deploymentID, "log0001"),
		path.Join(consulutil.LogsPrefix, deploymentID, "log0002"),
		path.Join(consulutil.LogsPrefix, deploymentID, "log0003")}

	eventKeys := []string{path.Join(consulutil.EventsPrefix, deploymentID, "event0001"),
		path.Join(consulutil.LogsPrefix, deploymentID, "event0002"),
		path.Join(consulutil.LogsPrefix, deploymentID, "event0003")}

	keys := append(logKeys, eventKeys...)
	for _, key := range keys {
		srv1.PopulateKV(t, map[string][]byte{key: b})
	}

	srv1.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.LogsPrefix, deploymentID, "log0001"):     b,
		path.Join(consulutil.LogsPrefix, deploymentID, "log0002"):     b,
		path.Join(consulutil.LogsPrefix, deploymentID, "log0003"):     b,
		path.Join(consulutil.EventsPrefix, deploymentID, "event0001"): b,
		path.Join(consulutil.EventsPrefix, deploymentID, "event0002"): b,
		path.Join(consulutil.EventsPrefix, deploymentID, "event0003"): b,
	})

	myStore := config.Store{
		Name:                  "myPersonalStore",
		MigrateDataFromConsul: true,
		Implementation:        "file",
		Types:                 []string{"Log", "Event"},
	}

	cfg.Storage = config.Storage{
		Reset:  true,
		Stores: []config.Store{myStore},
	}

	err = LoadStores(cfg)
	require.NoError(t, err)

	// Check custom configuration + default complement has been saved in Consul
	MapStores, err := consulutil.List(consulutil.StoresPrefix)
	require.NoError(t, err)
	require.NotNil(t, MapStores)
	require.Len(t, MapStores, 2)

	defaultStores := getDefaultConfigStores(cfg)
	require.NotNil(t, defaultStores)
	require.Len(t, defaultStores, 2)

	for k, v := range MapStores {
		key := path.Base(k)
		switch key {
		case "myPersonalStore":
			s := new(config.Store)
			err = json.Unmarshal(v, s)
			require.NoError(t, err)

			require.Equal(t, myStore.Implementation, s.Implementation)
			require.Equal(t, myStore.Name, s.Name)
			require.Equal(t, myStore.MigrateDataFromConsul, s.MigrateDataFromConsul)
			require.Equal(t, myStore.Types, s.Types)
			require.Equal(t, path.Join(cfg.WorkingDirectory, defaultRelativeRootDir), s.Properties["root_dir"])
			require.Equal(t, defaultCacheNumCounters, s.Properties["cache_num_counters"])
			require.Equal(t, defaultCacheMaxCost, s.Properties["cache_max_cost"])
			require.Equal(t, defaultCacheBufferItems, s.Properties["cache_buffer_items"])
		case "defaultFileStoreWithCache":
			s := new(config.Store)
			err = json.Unmarshal(v, s)
			if !reflect.DeepEqual(*s, defaultStores[0]) {
				t.Errorf("LoadStores() = %v, want %v", *s, defaultStores[0])
			}
		default:
			t.Errorf("unexpected key:%q", key)
		}
	}

	value := new(map[string]string)
	for _, key := range logKeys {
		exist, err := GetStore(types.StoreTypeLog).Get(key, value)
		require.NoError(t, err)
		require.True(t, exist)
		require.NotNil(t, value)
		val := *value
		require.Equal(t, "content1", val["key1"])
	}
	for _, key := range eventKeys {
		exist, err := GetStore(types.StoreTypeEvent).Get(key, value)
		require.NoError(t, err)
		require.True(t, exist)
		require.NotNil(t, value)
		val := *value
		require.Equal(t, "content1", val["key1"])
	}
}

func testLoadStoresWithPartialStorageConfig(t *testing.T, srv1 *testutil.TestServer, cfg config.Configuration) {
	// Reset once to allow reload config
	once.Reset()

	deploymentID := t.Name()

	obj := map[string]string{
		"key1": "content1",
		"key2": "content2",
		"key3": "content3",
	}
	b, err := json.Marshal(obj)
	require.NoError(t, err)

	logKeys := []string{path.Join(consulutil.LogsPrefix, deploymentID, "log0001"),
		path.Join(consulutil.LogsPrefix, deploymentID, "log0002"),
		path.Join(consulutil.LogsPrefix, deploymentID, "log0003")}

	eventKeys := []string{path.Join(consulutil.EventsPrefix, deploymentID, "event0001"),
		path.Join(consulutil.LogsPrefix, deploymentID, "event0002"),
		path.Join(consulutil.LogsPrefix, deploymentID, "event0003")}

	keys := append(logKeys, eventKeys...)
	for _, key := range keys {
		srv1.PopulateKV(t, map[string][]byte{key: b})
	}

	srv1.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.LogsPrefix, deploymentID, "log0001"):     b,
		path.Join(consulutil.LogsPrefix, deploymentID, "log0002"):     b,
		path.Join(consulutil.LogsPrefix, deploymentID, "log0003"):     b,
		path.Join(consulutil.EventsPrefix, deploymentID, "event0001"): b,
		path.Join(consulutil.EventsPrefix, deploymentID, "event0002"): b,
		path.Join(consulutil.EventsPrefix, deploymentID, "event0003"): b,
	})

	myStore := config.Store{
		Name:                  "myPersonalStore",
		MigrateDataFromConsul: true,
		Implementation:        "cipherFileCache",
		Types:                 []string{"Log", "Event"},
		Properties: config.DynamicMap{
			"passphrase": "myverystrongpasswordo32bitlength",
		},
	}

	cfg.Storage = config.Storage{
		Reset:  true,
		Stores: []config.Store{myStore},
	}

	err = LoadStores(cfg)
	require.NoError(t, err)

	// Check custom configuration + default complement has been saved in Consul
	MapStores, err := consulutil.List(consulutil.StoresPrefix)
	require.NoError(t, err)
	require.NotNil(t, MapStores)
	require.Len(t, MapStores, 2)

	defaultStores := getDefaultConfigStores(cfg)
	require.NotNil(t, defaultStores)
	require.Len(t, defaultStores, 2)

	for k, v := range MapStores {
		key := path.Base(k)
		switch key {
		case "myPersonalStore":
			s := new(config.Store)
			err = json.Unmarshal(v, s)
			require.NoError(t, err)

			require.Equal(t, myStore.Implementation, s.Implementation)
			require.Equal(t, myStore.Name, s.Name)
			require.Equal(t, myStore.MigrateDataFromConsul, s.MigrateDataFromConsul)
			require.Equal(t, myStore.Types, s.Types)
			require.Equal(t, myStore.Properties["passphrase"], s.Properties["passphrase"])
			require.Equal(t, path.Join(cfg.WorkingDirectory, defaultRelativeRootDir), s.Properties["root_dir"])
			require.Equal(t, defaultCacheNumCounters, s.Properties["cache_num_counters"])
			require.Equal(t, defaultCacheMaxCost, s.Properties["cache_max_cost"])
			require.Equal(t, defaultCacheBufferItems, s.Properties["cache_buffer_items"])
		case "defaultFileStoreWithCache":
			s := new(config.Store)
			err = json.Unmarshal(v, s)
			if !reflect.DeepEqual(*s, defaultStores[0]) {
				t.Errorf("LoadStores() = %v, want %v", *s, defaultStores[0])
			}
		default:
			t.Errorf("unexpected key:%q", key)
		}
	}

	value := new(map[string]string)
	for _, key := range logKeys {
		exist, err := GetStore(types.StoreTypeLog).Get(key, value)
		require.NoError(t, err)
		require.True(t, exist)
		require.NotNil(t, value)
		val := *value
		require.Equal(t, "content1", val["key1"])
	}
	for _, key := range eventKeys {
		exist, err := GetStore(types.StoreTypeEvent).Get(key, value)
		require.NoError(t, err)
		require.True(t, exist)
		require.NotNil(t, value)
		val := *value
		require.Equal(t, "content1", val["key1"])
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

func testLoadStoresWithMissingMandatoryParameters(t *testing.T, srv1 *testutil.TestServer, cfg config.Configuration) {
	tests := []struct {
		name     string
		myStores []config.Store
	}{
		{"storeWithoutImplementation", []config.Store{{
			Name:  "myStore",
			Types: []string{"Log", "Event"},
		}}},
		{"storeWithoutTypes", []config.Store{{
			Name:           "myStore",
			Implementation: "consul",
		}}},
		{"storeWithoutTypes2", []config.Store{{
			Name:           "myStore",
			Implementation: "consul",
			Types:          []string{},
		}}},
		{"TwoStoreWithTheSameName", []config.Store{{
			Name:           "myStore",
			Implementation: "consul",
			Types:          []string{"Log", "Event"},
		},
			{
				Name:           "myStore",
				Implementation: "consul",
				Types:          []string{"Log", "Event"},
			}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset once to allow reload config
			once.Reset()

			cfg.Storage = config.Storage{
				Reset:  true,
				Stores: tt.myStores,
			}

			err := LoadStores(cfg)
			require.Error(t, err)

			// Check stores config has been cleared in Consul
			MapStores, err := consulutil.List(consulutil.StoresPrefix)
			require.NoError(t, err)
			require.NotNil(t, MapStores)
			require.Len(t, MapStores, 0)
		})
	}

}

func testLoadStoresWithGeneratedName(t *testing.T, srv1 *testutil.TestServer, cfg config.Configuration) {
	tests := []struct {
		name     string
		myStores []config.Store
	}{
		{"storeWithoutName", []config.Store{{
			Name:           "",
			Implementation: "consul",
			Types:          []string{"Log", "Event"},
		},
			{
				Name:           "",
				Implementation: "consul",
				Types:          []string{"Log", "Event"},
			},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset once to allow reload config
			once.Reset()

			cfg.Storage = config.Storage{
				Reset:  true,
				Stores: tt.myStores,
			}

			err := LoadStores(cfg)
			require.NoError(t, err)
			MapStores, err := consulutil.List(consulutil.StoresPrefix)
			require.NoError(t, err)
			require.NotNil(t, MapStores)
			require.Len(t, MapStores, 2)

			for k, _ := range MapStores {
				if !strings.Contains(k, "default") {
					store := new(config.Store)
					err = json.Unmarshal(MapStores[k], store)
					require.True(t, strings.HasPrefix(store.Name, "consulLogEvent-"))
					require.NoError(t, err)
				}
			}
		})
	}
}
