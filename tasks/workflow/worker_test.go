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

package workflow

import (
	"context"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
)

func populateKV(t *testing.T, srv *testutil.TestServer) {
	srv.PopulateKV(t, testData())
}

func testRunPurge(t *testing.T, kv *api.KV, client *api.Client) {
	var myWorker worker
	myWorker.consulClient = client
	var myTaskExecution taskExecution
	myTaskExecution.cc = client
	// This execution corresponds to the purge task
	// Set targetID to the Test deployment ID
	myTaskExecution.targetID = "Test.Env"
	myTaskExecution.taskID = "purgeTask"
	err := myWorker.runPurge(context.Background(), &myTaskExecution)
	if err != nil {
		t.Errorf("TaskExists() error = %v", err)
		return
	}
	// Test that KV contains expected resultData
	// (no more deployment, nor tasks, but one log and one event corresponding to purged status,
	// and also a purged/deploymentID key)
}

// Construct key/value to initialise KV before running test
// TODO :
// add KVs corresponding to
// - one Test deployment
// - several tasks for the Test deployment
// - several events and logs for the Test deployment
func testData() map[string][]byte {
	return map[string][]byte{}
}
