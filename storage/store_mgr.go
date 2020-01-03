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

var stores map[types.StoreType]store.Store

var defaultStores map[types.StoreType]store.Store

func loadDefaultStores(cfg config.Configuration) error {
	var err error
	defaultStores = make(map[types.StoreType]store.Store, 0)
	for _, typ := range types.StoreTypeNames() {
		st, _ := types.ParseStoreType(typ)
		switch st {
		case types.StoreTypeDeployment:
			rootDirectory := path.Join(cfg.WorkingDirectory, "store")
			defaultStores[st], err = file.NewStore(rootDirectory)
			if err != nil {
				return err
			}
		default:
			defaultStores[st] = consul.NewStore()
		}
	}
	return nil
}

// LoadStores fetch all store implementations found in plugins (ie for deployments, logs and events storage
// If no external store is found, default store is used
func LoadStores(cfg config.Configuration) error {
	// Load default stores
	loadDefaultStores(cfg)

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

	for _, typ := range types.StoreTypeNames() {
		st, _ := types.ParseStoreType(typ)
		if _, ok := stores[st]; !ok {
			log.Printf("Using default store for type: %q.", typ)
			stores[st] = defaultStores[st]
		}
	}

	return nil
}

// GetStore returns the store related to a defined store type
func GetStore(typ types.StoreType) store.Store {
	return stores[typ]
}
