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
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage/internal/consul"
	"github.com/ystia/yorc/v4/storage/internal/file"
	"github.com/ystia/yorc/v4/storage/store"
	"github.com/ystia/yorc/v4/storage/types"
	"path"
	"strings"
	"sync"
)

const consulStoreImpl = "consul"

const fileStoreWithCacheImpl = "fileWithCache"

const fileStoreWithCacheAndEncryptionImpl = "fileWithCacheAndEncryption"

var once sync.Once
var stores map[types.StoreType]store.Store

// LoadStores reads stores configuration and load store implementations
// The store config needs to provide store for all defined types. ie. deployments, logs and events.
// An example in json:
// "stores": [
// {
//	 "name": "myFileStore",
//	 "implementation": "fileWithCache",
//	 "types":  ["Deployment"]
// },
// {
//	 "name": "myCipherFileStore",
//	 "implementation": "fileWithCache",
//	 "types":  ["Log", "Event"]
//  }]
//
func LoadStores(cfg config.Configuration) error {
	var err error
	// load stores once
	once.Do(func() {
		stores = make(map[types.StoreType]store.Store, 0)
		// Check provided config stores
		ok := checkConfigStores(cfg)

		// Define default config stores if no correct config has been provided
		if cfg.Stores == nil || !ok {
			buildDefaultConfigStores(&cfg)
		}

		for _, configStore := range cfg.Stores {
			for _, storeTypeName := range configStore.Types {
				st, _ := types.ParseStoreType(storeTypeName)
				if _, ok := stores[st]; !ok {
					log.Printf("Using store with name:%q, implementation:%q for type: %q", configStore.Name, configStore.Implementation, storeTypeName)
					err = createStoreImpl(cfg, configStore, st)
					if err != nil {
						return
					}
				}
			}
		}
	})
	return err
}

func buildDefaultConfigStores(cfg *config.Configuration) {
	cfg.Stores = make([]config.Store, 0)

	// File with cache store for deployments
	fileStoreWithCache := config.Store{
		Name:           "defaultFileStoreWithCache",
		Implementation: fileStoreWithCacheImpl,
		Types:          []string{types.StoreTypeDeployment.String()},
	}
	cfg.Stores = append(cfg.Stores, fileStoreWithCache)

	// File with cache and encryption store for logs
	cipherFileStoreWithCache := config.Store{
		Name:           "defaultFileStoreWithCacheAndEncryption",
		Implementation: fileStoreWithCacheAndEncryptionImpl,
		Types:          []string{types.StoreTypeLog.String()},
	}
	cfg.Stores = append(cfg.Stores, cipherFileStoreWithCache)

	// File with cache and encryption store for events
	consulStore := config.Store{
		Name:           "defaultConsul",
		Implementation: consulStoreImpl,
		Types:          []string{types.StoreTypeEvent.String()},
	}
	cfg.Stores = append(cfg.Stores, consulStore)
}

// Check if all stores types are provided by stores config
// If any store type is missing, default config is used
func checkConfigStores(cfg config.Configuration) bool {
	if cfg.Stores == nil || len(cfg.Stores) == 0 {
		return false
	}
	checkStores := make([]string, 0)
	for _, configStore := range cfg.Stores {
		for _, storeTypeName := range configStore.Types {
			// let's do this case insensitive
			name := strings.ToLower(storeTypeName)
			if !collections.ContainsString(checkStores, name) {
				checkStores = append(checkStores, name)
			}
		}
	}

	for _, storeTypeName := range types.StoreTypeNames() {
		name := strings.ToLower(storeTypeName)
		if !collections.ContainsString(checkStores, name) {
			log.Printf("[WARNING] failed to get any config store for type:%q. Default config stores will be used.", storeTypeName)
			return false
		}
	}
	return true
}

func createStoreImpl(cfg config.Configuration, configStore config.Store, storeType types.StoreType) error {
	var err error
	switch configStore.Implementation {
	case fileStoreWithCacheImpl:
		rootDirectory := path.Join(cfg.WorkingDirectory, "store")
		stores[storeType], err = file.NewStore(cfg, configStore.Name, rootDirectory, true, false)
		if err != nil {
			return err
		}
	case fileStoreWithCacheAndEncryptionImpl:
		rootDirectory := path.Join(cfg.WorkingDirectory, "store")
		stores[storeType], err = file.NewStore(cfg, configStore.Name, rootDirectory, true, true)
		if err != nil {
			return err
		}
	case consulStoreImpl:
		stores[storeType] = consul.NewStore()
	default:
		log.Printf("[WARNING] unknown store implementation:%q. This will be ignored.", storeType)
	}
	return nil
}

// GetStore returns the store related to a defined store type
func GetStore(typ types.StoreType) store.Store {
	return stores[typ]
}
