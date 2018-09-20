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

package collector

import (
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/tasks"
	"path"
	"strconv"
	"testing"
)

func populateKV(t *testing.T, srv *testutil.TestServer) {
	srv.PopulateKV(t, map[string][]byte{
		consulutil.TasksPrefix + "/t12/status":   []byte("3"),
		consulutil.TasksPrefix + "/t12/type":     []byte("5"),
		consulutil.TasksPrefix + "/t12/targetId": []byte("id"),
	})
}

func testResumeTask(t *testing.T, client *api.Client) {
	kv := client.KV()
	testCollector := NewCollector(client)
	type args struct {
		kv     *api.KV
		taskID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"ResumeTask", args{kv, "t12"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testCollector.ResumeTask(tt.args.taskID); (err != nil) != tt.wantErr {
				t.Errorf("ResumeTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, tt.args.taskID, "status"), nil)
			if err != nil {
				t.Errorf("Unexpected Consul communication error: %v", err)
				return
			}
			if kvp == nil {
				t.Error("status missing")
				return
			}

			status, err := strconv.Atoi(string(kvp.Value))
			if err != nil {
				t.Errorf("Invalid task status:")
			}

			if tasks.TaskStatus(status) != tasks.TaskStatusINITIAL {
				t.Errorf("status not set to \"INITIAL\" but to:%s", tasks.TaskStatus(status))
				return
			}
		})
	}
}
