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
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/storage/store"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunFileStoragePackageTests(t *testing.T) {
	cfg := config.Configuration{
		WorkingDirectory: "./work",
	}

	defer func() {
		os.RemoveAll(cfg.WorkingDirectory)
	}()

	t.Run("groupFileStore", func(t *testing.T) {
		t.Run("testFileStoreWithEncryption", func(t *testing.T) {
			testFileStoreWithEncryption(t, cfg)
		})
		t.Run("testFileStoreTypesWithEncryption", func(t *testing.T) {
			testFileStoreTypesWithEncryption(t, cfg)
		})
		t.Run("testFileStoreWithEncryptionWithoutSecretKeyProvided", func(t *testing.T) {
			testFileStoreWithEncryptionWithoutSecretKeyProvided(t, cfg)
		})
		t.Run("testFileStoreWithCache", func(t *testing.T) {
			testFileStoreWithCache(t, cfg)
		})
		t.Run("testFileStoreTypesWithCache", func(t *testing.T) {
			testFileStoreTypesWithCache(t, cfg)
		})
	})
}

func testFileStoreWithEncryptionWithoutSecretKeyProvided(t *testing.T, cfg config.Configuration) {
	_, err := NewStore(cfg, "testStoreID", nil, false, true)
	require.Error(t, err, "expected error as secrek key is missing")
}

func testFileStoreWithEncryption(t *testing.T, cfg config.Configuration) {
	props := config.DynamicMap{
		"passphrase": "myverystrongpasswordo32bitlength",
		"root_dir":   path.Join(cfg.WorkingDirectory, t.Name()),
	}
	fileStore, err := NewStore(cfg, "testStoreID", props, false, true)
	require.NoError(t, err, "failed to instantiate new store")
	store.CommonStoreTest(t, fileStore)
}

func testFileStoreTypesWithEncryption(t *testing.T, cfg config.Configuration) {
	props := config.DynamicMap{
		"passphrase": "myverystrongpasswordo32bitlength",
		"root_dir":   path.Join(cfg.WorkingDirectory, t.Name()),
	}
	fileStore, err := NewStore(cfg, "testStoreID", props, false, true)
	require.NoError(t, err, "failed to instantiate new store")
	store.CommonStoreTestAllTypes(t, fileStore)
}

func testFileStoreWithCache(t *testing.T, cfg config.Configuration) {
	props := config.DynamicMap{
		"root_dir": path.Join(cfg.WorkingDirectory, t.Name()),
	}
	fileStore, err := NewStore(cfg, "testStoreID", props, true, false)
	require.NoError(t, err, "failed to instantiate new store")
	store.CommonStoreTest(t, fileStore)
}

func testFileStoreTypesWithCache(t *testing.T, cfg config.Configuration) {
	props := config.DynamicMap{
		"root_dir": path.Join(cfg.WorkingDirectory, t.Name()),
	}
	fileStore, err := NewStore(cfg, "testStoreID", props, true, false)
	require.NoError(t, err, "failed to instantiate new store")
	store.CommonStoreTestAllTypes(t, fileStore)
}
