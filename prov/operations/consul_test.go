// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package operations

import (
	"os"
	"testing"

	consultests "github.com/hashicorp/consul/sdk/testutil"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulTasksPackageTests(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)

	cfg := testutil.SetupTestConfig(t)
	srv, _ := testutil.NewTestConsulInstance(t, &cfg)
	defer func() {
		srv.Stop()
		os.RemoveAll(cfg.WorkingDirectory)
	}()

	populateKV(t, srv)

	t.Run("TestGetOverlayPath", func(t *testing.T) {
		testGetOverlayPath(t)
	})

}

func populateKV(t *testing.T, srv *consultests.TestServer) {
	srv.PopulateKV(t, map[string][]byte{
		consulutil.TasksPrefix + "/deployTask/targetId": []byte("id1"),
		consulutil.TasksPrefix + "/deployTask/status":   []byte("0"),
		consulutil.TasksPrefix + "/deployTask/type":     []byte("0"), // tasks.TaskTypeDeploy
		consulutil.TasksPrefix + "/removeTask/targetId": []byte("id1"),
		consulutil.TasksPrefix + "/removeTask/status":   []byte("0"),
		consulutil.TasksPrefix + "/removeTask/type":     []byte("11"), // tasks.TaskTypeRemoveNodes
	})
}
