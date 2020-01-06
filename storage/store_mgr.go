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
	"github.com/ystia/yorc/v4/storage/store"
	"path"
	"path/filepath"
	"plugin"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage/internal/consul"
	"github.com/ystia/yorc/v4/storage/internal/file"
	"github.com/ystia/yorc/v4/storage/types"
)

const consulStore = "consul"

const fileStoreWithCache = "fileWithCache"

const fileStoreWithCacheAndEncryption = "fileWithCacheAndEncryption"

var stores map[types.StoreType]store.Store

func loadConfigStores(cfg config.Configuration) error {
	// Define default config stores
	if cfg.Stores == nil {
		cfg.Stores = make(config.DynamicMap)
		cfg.Stores[types.StoreTypeDeployment.String()] = fileStoreWithCache
		cfg.Stores[types.StoreTypeLog.String()] = fileStoreWithCacheAndEncryption
		cfg.Stores[types.StoreTypeEvent.String()] = consulStore
	}

	defaultStores := make(map[types.StoreType]store.Store, 0)
	for _, storeTypeName := range types.StoreTypeNames() {
		st, _ := types.ParseStoreType(storeTypeName)
		if _, ok := stores[st]; !ok {
			log.Printf("Using default store %q for type: %q.", cfg.Stores[storeTypeName], storeTypeName)
			if _, defaultOk := defaultStores[st]; !defaultOk {
				storeImpl, cast := cfg.Stores[storeTypeName].(string)
				if !cast {
					return errors.Errorf("failed to cast to string store implementation from stores config with store=%+v", cfg.Stores)
				}

				err := loadStoreImpl(cfg, storeImpl, st, defaultStores)
				if err != nil {
					return err
				}
			}
			stores[st] = defaultStores[st]
		}
	}
	return nil
}

func loadStoreImpl(cfg config.Configuration, storeImpl string, storeType types.StoreType, defaultStores map[types.StoreType]store.Store) error {
	var err error
	switch storeImpl {
	case fileStoreWithCache:
		rootDirectory := path.Join(cfg.WorkingDirectory, "store")
		defaultStores[storeType], err = file.NewStore(rootDirectory, true, false)
		if err != nil {
			return err
		}
	case fileStoreWithCacheAndEncryption:
		rootDirectory := path.Join(cfg.WorkingDirectory, "store")
		defaultStores[storeType], err = file.NewStore(rootDirectory, true, true)
		if err != nil {
			return err
		}
	case consulStore:
		defaultStores[storeType] = consul.NewStore()
	default:
		log.Printf("[WARNING] unknown store type:%q. This will be ignored.", storeType)
	}
	return nil
}

// LoadStores fetch all store implementations found in plugins (ie for deployments, logs and events storage
// If no external store is found, default store is used
func LoadStores(cfg config.Configuration) error {
	stores = make(map[types.StoreType]store.Store, 0)
	pluginsPath := cfg.PluginsDirectory
	if pluginsPath == "" {
		pluginsPath = config.DefaultPluginDir
	}
	pluginPath, err := filepath.Abs(pluginsPath)
	if err != nil {
		return errors.Wrap(err, "Failed to explore plugins directory")
	}

	storePlugins, err := filepath.Glob(filepath.Join(pluginPath, "store_*.so"))
	if err != nil {
		return err
	}

	for _, f := range storePlugins {
		p, err := plugin.Open(f)
		if err != nil {
			return errors.Wrapf(err, "Failed to load plugin from file:%q", f)
		}

		symStore, err := p.Lookup("Store")
		if err != nil {
			return err
		}

		store, ok := symStore.(store.Store)
		if !ok {
			return errors.Errorf("A store type is expected from Go plugin Symbol from file:%q", f)
		}

		log.Printf("Loaded store from plugin file:%q with store types: %v", f, store.Types())
		for _, typ := range store.Types() {
			if _, ok := stores[typ]; ok {
				log.Printf("[WARNING] a store already exists for type: %q. This will be ignored.", typ)
			}
			stores[typ] = store
		}
	}

	// Load config stores if no plugin brings the store type
	return loadConfigStores(cfg)
}

// GetStore returns the store related to a defined store type
func GetStore(typ types.StoreType) store.Store {
	return stores[typ]
}
