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
	"context"
	"encoding/json"
	"github.com/matryer/resync"
	"github.com/pkg/errors"
	"path"
	"strings"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage/internal/consul"
	"github.com/ystia/yorc/v4/storage/internal/file"
	"github.com/ystia/yorc/v4/storage/store"
	"github.com/ystia/yorc/v4/storage/types"
)

const consulStoreImpl = "consul"

const fileStoreWithCacheImpl = "fileCache"

const fileStoreWithCacheAndEncryptionImpl = "cipherFileCache"

var once resync.Once
var stores map[types.StoreType]store.Store

// LoadStores reads/saves stores configuration and load store implementations in mem.
// The store config needs to provide store for all defined types. ie. deployments, logs and events.
// The stores config is saved once and can be reset if storage.reset is true.
func LoadStores(cfg config.Configuration) error {
	//time.Sleep(10 * time.Second)
	var err error
	// load stores once
	once.Do(func() {
		var cfgStores []config.Store
		var init bool
		// load stores config from Consul if already present or save them from configuration
		init, cfgStores, err = getConfigStores(cfg)
		if err != nil {
			return
		}

		// load stores implementations
		stores = make(map[types.StoreType]store.Store, 0)
		for _, configStore := range cfgStores {
			for _, storeTypeName := range configStore.Types {
				st, _ := types.ParseStoreType(storeTypeName)
				if _, ok := stores[st]; !ok {
					log.Printf("Using store with name:%q, implementation:%q for type: %q", configStore.Name, configStore.Implementation, storeTypeName)
					err = createStoreImpl(cfg, configStore, st)
					if err != nil {
						return
					}

					// Handle Consul data migration for log/event stores
					if configStore.MigrateDataFromConsul && init && configStore.Implementation != consulStoreImpl {
						err = migrateData(configStore.Name, st, stores[st])
						if err != nil {
							return
						}
					}
				}
			}
		}
	})

	if err != nil {
		clearConfigStore()
	}
	return err
}

func getConfigStores(cfg config.Configuration) (bool, []config.Store, error) {
	consulClient, err := cfg.GetConsulClient()
	if err != nil {
		return false, nil, err
	}
	lock, err := consulutil.AcquireConsulLock(consulClient, ".lock_stores", 0)
	if err != nil {
		return false, nil, err
	}
	defer lock.Unlock()

	kvps, _, err := consulClient.KV().List(consulutil.StoresPrefix, nil)
	if err != nil {
		return false, nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	// Get config Store from Consul if reset is false and exists any store
	if !cfg.Storage.Reset && len(kvps) > 0 {
		log.Debugf("Found %d stores already saved", len(kvps))
		configStores := make([]config.Store, len(kvps))
		for _, kvp := range kvps {
			name := path.Base(kvp.Key)
			configStore := new(config.Store)
			err = json.Unmarshal(kvp.Value, configStore)
			if err != nil {
				return false, nil, errors.Wrapf(err, "failed to unmarshal store with name:%q", name)
			}
			configStores = append(configStores, *configStore)
		}
		return false, configStores, nil
	}
	configStores, err := initConfigStores(cfg)
	return true, configStores, err
}

// Initialize config stores in Consul
func initConfigStores(cfg config.Configuration) ([]config.Store, error) {
	cfgStores, err := checkAndBuildConfigStores(cfg)
	if err != nil {
		return nil, err
	}

	if err := clearConfigStore(); err != nil {
		return nil, err
	}
	// Save stores config in Consul
	for _, configStore := range cfgStores {
		err := consulutil.StoreConsulKeyWithJSONValue(path.Join(consulutil.StoresPrefix, configStore.Name), configStore)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to save store %s in consul", configStore.Name)
		}
		log.Debugf("Save store config with name:%q", configStore.Name)
	}
	return cfgStores, nil
}

// Clear config stores in Consul
func clearConfigStore() error {
	return consulutil.Delete(consulutil.StoresPrefix, true)
}

// Build default config stores if no custom config provided
func buildDefaultConfigStores() []config.Store {
	log.Print("Default config stores is set")
	cfgStores := make([]config.Store, 0)

	// File with cache store for deployments
	fileStoreWithCache := config.Store{
		Name:           "defaultFileStoreWithCache",
		Implementation: fileStoreWithCacheImpl,
		Types:          []string{types.StoreTypeDeployment.String()},
	}
	cfgStores = append(cfgStores, fileStoreWithCache)

	// Consul store for logs and events
	consulStore := config.Store{
		Name:           "defaultConsulStore",
		Implementation: consulStoreImpl,
		Types:          []string{types.StoreTypeLog.String(), types.StoreTypeEvent.String()},
	}
	cfgStores = append(cfgStores, consulStore)
	return cfgStores
}

// Check if all stores types are provided by stores config
// If no config is provided, global default config store is added
// If any store type is missing, a related default config store is added
func checkAndBuildConfigStores(cfg config.Configuration) ([]config.Store, error) {
	if cfg.Storage.Stores == nil {
		return buildDefaultConfigStores(), nil
	}

	cfgStores := cfg.Storage.Stores
	checkStoreTypes := make([]string, 0)
	checkStoreNames := make([]string, 0)
	for _, configStore := range cfg.Storage.Stores {
		if configStore.Name == "" {
			return nil, errors.Errorf("Missing mandatory property \"Name\" for store with no name...")
		}
		if configStore.Implementation == "" {
			return nil, errors.Errorf("Missing mandatory property \"implementation\" for store with name:%q", configStore.Name)
		}
		if configStore.Types == nil || len(configStore.Types) == 0 {
			return nil, errors.Errorf("Missing mandatory property \"types\" for store with name:%q", configStore.Name)
		}

		// Check store name is unique
		if collections.ContainsString(checkStoreNames, configStore.Name) {
			return nil, errors.Errorf("At least, 2 different stores have the same name:%q", configStore.Name)
		}
		checkStoreNames = append(checkStoreNames, configStore.Name)

		// Prepare store types check
		for _, storeTypeName := range configStore.Types {
			// let's do this case insensitive
			name := strings.ToLower(storeTypeName)
			if !collections.ContainsString(checkStoreTypes, name) {
				checkStoreTypes = append(checkStoreTypes, name)
			}
		}
	}

	// Check each store type has its implementation.
	// Add default if none is provided by config
	for _, storeTypeName := range types.StoreTypeNames() {
		name := strings.ToLower(storeTypeName)
		if !collections.ContainsString(checkStoreTypes, name) {
			log.Printf("Default config store will be used for store type:%q.", storeTypeName)
			var defaultStore config.Store
			switch storeTypeName {
			case types.StoreTypeDeployment.String():
				defaultStore = config.Store{
					Name:           "defaultFileStoreWithCache",
					Implementation: fileStoreWithCacheImpl,
					Types:          []string{types.StoreTypeDeployment.String()},
				}
			case types.StoreTypeEvent.String():
				defaultStore = config.Store{
					Name:           "defaultConsulStore" + types.StoreTypeEvent.String(),
					Implementation: consulStoreImpl,
					Types:          []string{types.StoreTypeEvent.String()},
				}
			case types.StoreTypeLog.String():
				defaultStore = config.Store{
					Name:           "defaultConsulStore" + types.StoreTypeLog.String(),
					Implementation: consulStoreImpl,
					Types:          []string{types.StoreTypeLog.String()},
				}
			}

			cfgStores = append(cfgStores, defaultStore)
		}
	}
	return cfgStores, nil
}

// Create store implementations
func createStoreImpl(cfg config.Configuration, configStore config.Store, storeType types.StoreType) error {
	var err error
	impl := strings.ToLower(configStore.Implementation)
	switch impl {
	case strings.ToLower(fileStoreWithCacheImpl), strings.ToLower(fileStoreWithCacheAndEncryptionImpl):
		stores[storeType], err = file.NewStore(cfg, configStore.Name, configStore.Properties, true, impl == strings.ToLower(fileStoreWithCacheAndEncryptionImpl))
		if err != nil {
			return err
		}
	case strings.ToLower(consulStoreImpl):
		stores[storeType] = consul.NewStore()
	default:
		log.Printf("[WARNING] unknown store implementation:%q. This will be ignored.", storeType)
	}
	return nil
}

// this allows to migrate log or events from Consul to new store implementations (other than Consul)
func migrateData(storeName string, storeType types.StoreType, storeImpl store.Store) error {

	var rootPath string
	switch storeType {
	case types.StoreTypeLog:
		rootPath = consulutil.LogsPrefix
	case types.StoreTypeEvent:
		rootPath = consulutil.EventsPrefix
	default:
		log.Printf("[WARNING] No migration handled for type:%q (demanded in config for store name:%q)", storeType, storeName)
		return nil
	}
	kvps, _, err := consulutil.GetKV().List(rootPath, nil)
	if err != nil {
		errors.Wrapf(err, "failed to migrate data from Consul for root path:%q in store with name:%q", rootPath, storeName)
	}
	if kvps == nil || len(kvps) == 0 {
		return nil
	}
	keyValues := make([]store.KeyValueIn, 0)
	var value json.RawMessage
	for _, kvp := range kvps {
		value = kvp.Value
		keyValues = append(keyValues, store.KeyValueIn{
			Key:   kvp.Key,
			Value: value,
		})

	}
	err = storeImpl.SetCollection(context.Background(), keyValues)
	if err != nil {
		errors.Wrapf(err, "failed to migrate data from Consul for root path:%q in store with name:%q", rootPath, storeName)
	}

	return consulutil.Delete(rootPath, true)
}

// GetStore returns the store related to a defined store type
func GetStore(tType types.StoreType) store.Store {
	store, ok := stores[tType]
	if !ok {
		log.Panic("Store %q is missing. This is not expected.", tType.String())
	}
	return store
}
