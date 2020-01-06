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
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/storage/store"
	"os"
	"testing"
)

func TestFileStoreWithCache(t *testing.T) {
	rootDir := "./.work_" + t.Name()
	defer func() {
		err := os.RemoveAll(rootDir)
		require.NoError(t, err, "failed to remove test working directory:%q", rootDir)
	}()
	fileStore, err := NewStore("testStoreID", rootDir, true, false)
	require.NoError(t, err, "failed to instantiate new store")
	store.CommonStoreTest(t, fileStore)
}

func TestFileStoreTypesWithCache(t *testing.T) {
	rootDir := "./.work_" + t.Name()
	defer func() {
		err := os.RemoveAll(rootDir)
		require.NoError(t, err, "failed to remove test working directory:%q", rootDir)
	}()
	fileStore, err := NewStore("testStoreID", rootDir, true, false)
	require.NoError(t, err, "failed to instantiate new store")
	store.CommonStoreTestAllTypes(t, fileStore)
}
