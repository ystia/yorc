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
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage/internal"
	"github.com/ystia/yorc/v4/storage/types"
	"path/filepath"
	"plugin"
)

var stores map[types.StoreType]Store

var defaultStore Store

func init() {
	defaultStore = internal.NewStore()
}

// LoadStores fetch all store implementations found in plugins (ie for deployments, logs and events storage
// If no external store is found, default store is used
func LoadStores() error {
	stores = make(map[types.StoreType]Store, 0)
	storePlugins, err := filepath.Glob("plugins/store_*.so")
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

		store, ok := symStore.(Store)
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
			stores[st] = defaultStore
		}
	}

	return nil
}

func GetStore(typ types.StoreType) Store {
	return stores[typ]
}
