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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/consulutil"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
)

// Test deployment ID
const deploymentID string = "Test.Env"

func populateKV(t *testing.T, srv *testutil.TestServer) {
	srv.PopulateKV(t, testData(deploymentID))
}

func testRunPurge(t *testing.T, srv *testutil.TestServer, client *api.Client) {
	myWorker := &worker{
		consulClient: client,
		cfg: config.Configuration{
			// Ensure we are not deleting filesystem files elsewhere
			WorkingDirectory: "./testdata/work/",
		},
	}
	var myTaskExecution taskExecution
	myTaskExecution.cc = client
	// This execution corresponds to the purge task
	// Set targetID to the Test deployment ID
	myTaskExecution.targetID = deploymentID
	err := myWorker.runPurge(context.Background(), &myTaskExecution)
	if err != nil {
		t.Errorf("runPurge() error = %v", err)
		return
	}
	// Test that KV contains expected resultData
	// No more deployments
	kvp, _, err := consulutil.GetKV().Get(consulutil.DeploymentKVPrefix, nil)
	require.Nil(t, kvp)
	// No more tasks
	kvp, _, err = consulutil.GetKV().Get(consulutil.TasksPrefix, nil)
	require.Nil(t, kvp)
	// One event with value containing "deploymentId":"Test-Env","status":"purged"
	kvps, _, err := consulutil.GetKV().List(consulutil.EventsPrefix+"/"+deploymentID, nil)
	require.True(t, len(kvps) == 1)
	var eventData map[string]string
	err = json.Unmarshal([]byte(string(kvps[0].Value)), &eventData)
	require.Nil(t, err)
	assert.Equal(t, eventData["deploymentId"], deploymentID)
	assert.Equal(t, eventData["status"], "purged")

	// One log with value containing "content":"Status for deployment \"Test-Env\" changed to \"purged\"
	kvps, _, err = consulutil.GetKV().List(consulutil.LogsPrefix+"/"+deploymentID, nil)
	require.True(t, len(kvps) == 1)
	// One purge
	kvps, _, err = consulutil.GetKV().List(consulutil.PurgedDeploymentKVPrefix+"/"+deploymentID, nil)
	require.True(t, len(kvps) == 1)
}

// Construct key/value to initialise KV before running test
func testData(deploymentID string) map[string][]byte {
	return map[string][]byte{
		// Add Test deployment
		consulutil.DeploymentKVPrefix + "/" + deploymentID + "/status": []byte(deploymentID),
		// deploy task
		consulutil.TasksPrefix + "/t1/targetId": []byte(deploymentID),
		consulutil.TasksPrefix + "/t1/type":     []byte("0"),
		// undeploy task
		consulutil.TasksPrefix + "/t2/targetId": []byte(deploymentID),
		consulutil.TasksPrefix + "/t2/type":     []byte("1"),
		// purge task
		consulutil.TasksPrefix + "/t3/targetId": []byte(deploymentID),
		consulutil.TasksPrefix + "/t3/type":     []byte("4"),
		// some events
		// event should have "deploymentId":"Test-Env" and "type":"anyType but not purge"
		consulutil.EventsPrefix + "/" + deploymentID + "/e1": []byte("aaaa"),
		consulutil.EventsPrefix + "/" + deploymentID + "/e2": []byte("bbbb"),
		// some logs
		consulutil.LogsPrefix + "/" + deploymentID + "/l1":   []byte("xxxx"),
		consulutil.EventsPrefix + "/" + deploymentID + "/l2": []byte("yyyy"),
	}
}
