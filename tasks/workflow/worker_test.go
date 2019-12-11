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
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/registry"
	"github.com/ystia/yorc/v4/tasks"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
)

// Test deployment ID
const deploymentID string = "Test.Env"

type mockInfraUsageCollector struct {
	getUsageInfoCalled bool
	ctx                context.Context
	conf               config.Configuration
	taskID             string
	infraName          string
	locationName       string
	contextCancelled   bool
	lof                events.LogOptionalFields
}

func (m *mockInfraUsageCollector) GetUsageInfo(ctx context.Context, conf config.Configuration, taskID, infraName, locationName string,
	params map[string]string) (map[string]interface{}, error) {

	var err error
	if locationName != "testLocation" {
		err = fmt.Errorf("Expected location testLocation got %q", locationName)
	} else if params["myparam"] != "testValue" {
		err = fmt.Errorf("Expected param myparam with value testValue, got %q", params["myparam"])
	}
	return nil, err
}

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
	defer func() {
		if myTaskExecution.finalFunction != nil {
			require.NoError(t, myTaskExecution.finalFunction())
		}
	}()
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

func testRunPurgeFails(t *testing.T, srv *testutil.TestServer, client *api.Client) {
	populateKV(t, srv)
	myWorker := &worker{
		// nil expected to make call fail
		consulClient: nil,
		cfg: config.Configuration{
			// Ensure we are not deleting filesystem files elsewhere
			WorkingDirectory: "./testdata/work/",
		},
	}

	deploymentID := "TestEnv2"

	srv.PopulateKV(t, testData(deploymentID))
	var myTaskExecution taskExecution
	myTaskExecution.cc = client
	// This execution corresponds to the purge task
	// Set targetID to the Test deployment ID
	myTaskExecution.targetID = deploymentID
	myTaskExecution.taskID = "t3"

	err := myWorker.runPurge(context.Background(), &myTaskExecution)
	require.Error(t, err)
	require.NotNil(t, myTaskExecution.finalFunction)
	myTaskExecution.finalFunction()

	// Test that KV contains expected resultData
	// Task status should be failed
	status, err := tasks.GetTaskStatus(myTaskExecution.taskID)
	require.NoError(t, err)
	require.Equal(t, tasks.TaskStatusFAILED, status, "task status not set to failed")

	// Deployment status should be UNDEPLOYMENT_FAILED
	depStatus, err := deployments.GetDeploymentStatus(context.Background(), myTaskExecution.targetID)
	require.NoError(t, err)
	require.Equal(t, deployments.UNDEPLOYMENT_FAILED, depStatus)

	// One event with value containing "deploymentId":"Test-Env","status":"purged"
	kvps, _, err := consulutil.GetKV().List(consulutil.EventsPrefix+"/"+deploymentID, nil)
	require.True(t, len(kvps) == 5)

	kvps, _, err = consulutil.GetKV().List(consulutil.PurgedDeploymentKVPrefix+"/"+deploymentID, nil)
	require.True(t, len(kvps) == 0)
}

func testRunQueryInfraUsage(t *testing.T, srv *testutil.TestServer, client *api.Client) {

	mock := new(mockInfraUsageCollector)
	var reg = registry.GetRegistry()
	reg.RegisterInfraUsageCollector("myInfraName", mock, "mock")

	myWorker := &worker{
		consulClient: client,
		cfg: config.Configuration{
			// Ensure we are not deleting filesystem files elsewhere
			WorkingDirectory: "./testdata/work/",
		},
	}
	var myTaskExecution taskExecution
	myTaskExecution.cc = client
	myTaskExecution.targetID = "infra_usage:myInfraName"
	myTaskExecution.taskID = "tQuery"
	err := myWorker.runQuery(context.Background(), &myTaskExecution)
	require.NoError(t, err, "Unexpected error returned by runQuery()")
}

// Construct key/value to initialise KV before running test
func testData(deploymentID string) map[string][]byte {
	return map[string][]byte{
		// Add Test deployment
		consulutil.DeploymentKVPrefix + "/" + deploymentID + "/status": []byte(deployments.INITIAL.String()),
		// deploy task
		consulutil.TasksPrefix + "/t1/targetId": []byte(deploymentID),
		consulutil.TasksPrefix + "/t1/type":     []byte("0"),
		// undeploy task
		consulutil.TasksPrefix + "/t2/targetId": []byte(deploymentID),
		consulutil.TasksPrefix + "/t2/type":     []byte("1"),
		// purge task
		consulutil.TasksPrefix + "/t3/targetId": []byte(deploymentID),
		consulutil.TasksPrefix + "/t3/type":     []byte("4"),
		consulutil.TasksPrefix + "/t3/status":   []byte(strconv.Itoa(int(tasks.TaskStatusRUNNING))),
		// query task
		consulutil.TasksPrefix + "/tQuery/targetId":          []byte("infra_usage:myInfraName"),
		consulutil.TasksPrefix + "/tQuery/type":              []byte("7"),
		consulutil.TasksPrefix + "/tQuery/data/locationName": []byte("testLocation"),
		consulutil.TasksPrefix + "/tQuery/data/myparam":      []byte("testValue"),

		// some events
		// event should have "deploymentId":"Test-Env" and "type":"anyType but not purge"
		consulutil.EventsPrefix + "/" + deploymentID + "/e1": []byte("aaaa"),
		consulutil.EventsPrefix + "/" + deploymentID + "/e2": []byte("bbbb"),
		// some logs
		consulutil.LogsPrefix + "/" + deploymentID + "/l1": []byte("xxxx"),
		consulutil.LogsPrefix + "/" + deploymentID + "/l2": []byte("yyyy"),
	}
}
