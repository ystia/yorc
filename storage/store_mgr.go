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
	"fmt"
	"math/rand"
	"path"
	"strings"
	"time"

	"github.com/matryer/resync"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage/internal/consul"
	"github.com/ystia/yorc/v4/storage/internal/file"
	"github.com/ystia/yorc/v4/storage/store"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/storage/internal/elastic"
)

const consulStoreImpl = "consul"

const elasticStoreImpl = "elastic"

const fileStoreWithCacheImpl = "fileCache"

const fileStoreWithCacheAndEncryptionImpl = "cipherFileCache"

const defaultRelativeRootDir = "store"

const defaultBlockingQueryTimeout = "5m0s"

// Default num counters for file cache (100 000)
// NumCounters is the number of 4-bit access counters to keep for admission and eviction
// We've seen good performance in setting this to 10x the number of items you expect to keep in the cache when full.
const defaultCacheNumCounters = "1e5"

// MaxCost can be used to denote the max size in bytes
const defaultCacheMaxCost = "1e7"

const defaultCacheBufferItems = "64"

var once resync.Once

// stores implementations provided with GetStore(types.StoreType)
var stores map[types.StoreType]store.Store

// default config stores loaded at init
var defaultConfigStores map[string]config.Store

func completePropertiesWithDefault(cfg config.Configuration, props config.DynamicMap) config.DynamicMap {
	complete := config.DynamicMap{}

	for k, v := range props {
		complete[k] = v
	}
	// Complete properties with string values as it's stored in Consul KV
	if !complete.IsSet("root_dir") {
		complete["root_dir"] = path.Join(cfg.WorkingDirectory, defaultRelativeRootDir)
	}
	if !complete.IsSet("blocking_query_default_timeout") {
		complete["blocking_query_default_timeout"] = defaultBlockingQueryTimeout
	}
	if !complete.IsSet("cache_num_counters") {
		complete["cache_num_counters"] = defaultCacheNumCounters
	}
	if !complete.IsSet("cache_max_cost") {
		complete["cache_max_cost"] = defaultCacheMaxCost
	}
	if !complete.IsSet("cache_buffer_items") {
		complete["cache_buffer_items"] = defaultCacheBufferItems
	}
	return complete
}

func initDefaultConfigStores(cfg config.Configuration) {
	defaultConfigStores = make(map[string]config.Store, 0)
	props := completePropertiesWithDefault(cfg, cfg.Storage.DefaultProperties)
	// File with cache store for deployments
	fileStoreWithCache := config.Store{
		Name:           "defaultFileStoreWithCache",
		Implementation: fileStoreWithCacheImpl,
		Types:          []string{types.StoreTypeDeployment.String()},
		Properties:     props,
	}
	defaultConfigStores[fileStoreWithCache.Name] = fileStoreWithCache

	// Consul store for both logs and events
	consulStore := config.Store{
		Name:           "defaultConsulStore",
		Implementation: consulStoreImpl,
		Types:          []string{types.StoreTypeLog.String(), types.StoreTypeEvent.String()},
	}
	defaultConfigStores[consulStore.Name] = consulStore

	// Consul store for logs only
	consulStoreLog := config.Store{
		Name:           "defaultConsulStore" + types.StoreTypeLog.String(),
		Implementation: consulStoreImpl,
		Types:          []string{types.StoreTypeLog.String()},
	}
	defaultConfigStores[consulStoreLog.Name] = consulStore

	// Consul store for events only
	consulStoreEvent := config.Store{
		Name:           "defaultConsulStore" + types.StoreTypeEvent.String(),
		Implementation: consulStoreImpl,
		Types:          []string{types.StoreTypeEvent.String()},
	}
	defaultConfigStores[consulStoreEvent.Name] = consulStoreEvent
}

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
	lock, err := consulutil.AcquireLock(consulClient, ".lock_stores", 0)
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

func getDefaultConfigStores(cfg config.Configuration) []config.Store {
	if defaultConfigStores == nil {
		initDefaultConfigStores(cfg)
	}

	return []config.Store{
		defaultConfigStores["defaultFileStoreWithCache"],
		defaultConfigStores["defaultConsulStore"],
	}
}

// Build default config stores for missing one of specified type
func getDefaultConfigStore(cfg config.Configuration, storeTypeName string) config.Store {
	if defaultConfigStores == nil {
		initDefaultConfigStores(cfg)
	}

	var defaultStore config.Store
	switch storeTypeName {
	case types.StoreTypeDeployment.String():
		defaultStore = defaultConfigStores["defaultFileStoreWithCache"]
	case types.StoreTypeEvent.String():
		defaultStore = defaultConfigStores["defaultConsulStore"+types.StoreTypeEvent.String()]
	case types.StoreTypeLog.String():
		defaultStore = defaultConfigStores["defaultFileStoreWithCache"+types.StoreTypeLog.String()]
	}
	return defaultStore
}

func storeExists(stores []config.Store, name string) bool {
	for _, storeItem := range stores {
		if storeItem.Name == name {
			return true
		}
	}
	return false
}

// Check if all stores types are provided by stores config
// If no config is provided, global default config store is added
// If any store type is missing, a related default config store is added
func checkAndBuildConfigStores(cfg config.Configuration) ([]config.Store, error) {
	if cfg.Storage.Stores == nil {
		return getDefaultConfigStores(cfg), nil
	}

	cfgStores := make([]config.Store, 0)
	checkStoreTypes := make([]string, 0)
	checkStoreNames := make([]string, 0)
	for _, cfgStore := range cfg.Storage.Stores {
		if cfgStore.Implementation == "" {
			return nil, errors.Errorf("Missing mandatory property \"implementation\" for store with name:%q", cfgStore.Name)
		}
		if cfgStore.Types == nil || len(cfgStore.Types) == 0 {
			return nil, errors.Errorf("Missing mandatory property \"types\" for store with name:%q", cfgStore.Name)
		}

		if cfgStore.Name == "" {
			rand.Seed(time.Now().UnixNano())
			extra := rand.Intn(100)
			cfgStore.Name = fmt.Sprintf("%s%s-%d", cfgStore.Implementation, strings.Join(cfgStore.Types, ""), extra)
		}

		// Check store name is unique
		if collections.ContainsString(checkStoreNames, cfgStore.Name) {
			return nil, errors.Errorf("At least, 2 different stores have the same name:%q", cfgStore.Name)
		}

		// Complete store properties with default for fileCache implementation
		if cfgStore.Implementation == fileStoreWithCacheAndEncryptionImpl || cfgStore.Implementation == fileStoreWithCacheImpl {
			props := completePropertiesWithDefault(cfg, cfgStore.Properties)
			cfgStore.Properties = props
		}

		checkStoreNames = append(checkStoreNames, cfgStore.Name)

		// Prepare store types check
		// First store type occurrence is taken in account
		for _, storeTypeName := range cfgStore.Types {
			// let's do this case insensitive
			name := strings.ToLower(storeTypeName)
			if !collections.ContainsString(checkStoreTypes, name) {
				checkStoreTypes = append(checkStoreTypes, name)
				// Add the store for this store type if not already added
				if !storeExists(cfgStores, cfgStore.Name) {
					log.Printf("Provided config store will be used for store type:%q.", storeTypeName)
					cfgStores = append(cfgStores, cfgStore)
				}
			}
		}
	}

	// Check each store type has its implementation.
	// Add default if none is provided by config
	for _, storeTypeName := range types.StoreTypeNames() {
		name := strings.ToLower(storeTypeName)
		if !collections.ContainsString(checkStoreTypes, name) {
			log.Printf("Default config store will be used for store type:%q.", storeTypeName)
			cfgStores = append(cfgStores, getDefaultConfigStore(cfg, storeTypeName))
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
	case strings.ToLower(elasticStoreImpl):
		stores[storeType] = elastic.NewStore(cfg, configStore)
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
		return errors.Wrapf(err, "failed to migrate data from Consul for root path:%q in store with name:%q", rootPath, storeName)
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
		return errors.Wrapf(err, "failed to migrate data from Consul for root path:%q in store with name:%q", rootPath, storeName)
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
