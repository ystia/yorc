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

package upgradeschema

import (
	"os"
	"path"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/testutil"
)

func TestUpgradeTo131(t *testing.T) {
	cfg := testutil.SetupTestConfig(t)
	srv, client := testutil.NewTestConsulInstance(t, &cfg)
	defer func() {
		srv.Stop()
		os.RemoveAll(cfg.WorkingDirectory)
	}()

	srv.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, "dep1", "status"): []byte("INITIAL"),
		path.Join(consulutil.DeploymentKVPrefix, "dep2", "status"): []byte("INITIAL"),
		path.Join(consulutil.TasksPrefix, "id1", "targetId"):       []byte("dep1"),
		path.Join(consulutil.TasksPrefix, "id2", "targetId"):       []byte("dep1"),
		path.Join(consulutil.TasksPrefix, "id3", "targetId"):       []byte("notexist"),
		path.Join(consulutil.TasksPrefix, "id4", "targetId"):       []byte("dep2"),
	})

	err := UpgradeTo131(cfg, client.KV(), nil)
	assert.NilError(t, err)
	tasks := srv.ListKV(t, path.Join(consulutil.DeploymentKVPrefix, "dep1", "tasks"))
	assert.DeepEqual(t, tasks, []string{"_yorc/deployments/dep1/tasks/id1", "_yorc/deployments/dep1/tasks/id2"})
	tasks = srv.ListKV(t, path.Join(consulutil.DeploymentKVPrefix, "dep2", "tasks"))
	assert.DeepEqual(t, tasks, []string{"_yorc/deployments/dep2/tasks/id4"})

}
