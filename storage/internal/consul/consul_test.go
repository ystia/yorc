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

package consul

import (
	"github.com/hashicorp/consul/sdk/testutil"
	"os"
	"testing"

	"github.com/ystia/yorc/v4/storage/encoding"
	"github.com/ystia/yorc/v4/storage/store"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulStoragePackageTests(t *testing.T) {
	cfg := store.SetupTestConfig(t)
	srv, _ := store.NewTestConsulInstance(t, &cfg)
	defer func() {
		srv.Stop()
		os.RemoveAll(cfg.WorkingDirectory)
	}()

	t.Run("groupConsulStore", func(t *testing.T) {
		t.Run("testConsulTypes", func(t *testing.T) {
			testTypes(t, srv)
		})
		t.Run("testConsulStore", func(t *testing.T) {
			testStore(t, srv)
		})
	})
}

func testStore(t *testing.T, srv1 *testutil.TestServer) {
	csStore := &consulStore{encoding.JSON}
	store.CommonStoreTest(t, csStore)
}

func testTypes(t *testing.T, srv1 *testutil.TestServer) {
	csStore := &consulStore{encoding.JSON}
	store.CommonStoreTestAllTypes(t, csStore)
}
