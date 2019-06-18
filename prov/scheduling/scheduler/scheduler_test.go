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

package scheduler

import (
	"encoding/json"
	"github.com/ystia/yorc/v3/events"
	"github.com/ystia/yorc/v3/tasks"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/prov"
	"github.com/ystia/yorc/v3/prov/scheduling"
)

func testRegisterAction(t *testing.T, client *api.Client) {
	t.Parallel()
	deploymentID := "dep-" + t.Name()
	ti := 1 * time.Second
	actionType := "test-action"
	action := &prov.Action{ActionType: actionType, Data: map[string]string{"key1": "val1", "key2": "val2", "key3": "val3"}}
	id, err := scheduling.RegisterAction(client, deploymentID, ti, action)
	require.Nil(t, err, "Unexpected error while registering action")
	require.NotEmpty(t, id, "id is not expected to be empty")

	// Check action has been registered
	sca, err := defaultScheduler.buildScheduledAction(id)
	require.Nil(t, err, "Unexpected error while building scheduled action from action id")
	require.NotNil(t, sca, "scheduled action is not required to be nil")
	require.Equal(t, actionType, sca.ActionType, "Unexpected value for action type")
	require.Equal(t, id, sca.ID, "Unexpected value for ID")
	require.Equal(t, 3, len(sca.Data), "Unexpected nb of data")
	require.Equal(t, "val1", sca.Data["key1"], "Unexpected value for Data[key1]")
	require.Equal(t, "val2", sca.Data["key2"], "Unexpected value for Data[key2]")
	require.Equal(t, "val3", sca.Data["key3"], "Unexpected value for Data[key3]")
}

func testProceedScheduledAction(t *testing.T, client *api.Client) {
	t.Parallel()
	deploymentID := "dep-" + t.Name()
	ti := 1 * time.Second
	actionType := "test-action"
	action := &prov.Action{ActionType: actionType, Data: map[string]string{"key1": "val1", "key2": "val2", "key3": "val3"}}
	id, err := scheduling.RegisterAction(client, deploymentID, ti, action)
	require.Nil(t, err, "Unexpected error while registering action")
	require.NotEmpty(t, id, "id is not expected to be empty")

	closeCh := make(chan struct{})
	defer close(closeCh)
	go func() {
		var latestIndex uint64
		for {
			select {
			case <-closeCh:
				return
			default:
			}
			kvp, meta, err := client.KV().Get(path.Join(consulutil.SchedulingKVPrefix, "actions", id, "latestTaskID"), &api.QueryOptions{WaitIndex: latestIndex})
			if err != nil {
				t.Logf("%v", err)
				continue
			}
			if latestIndex == meta.LastIndex {
				continue
			}

			// set the related task to done asap to reschedule them
			if kvp != nil && len(kvp.Value) > 0 {
				taskID := string(kvp.Value)
				p := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, taskID, "status"), Value: []byte(strconv.Itoa(int(tasks.TaskStatusDONE)))}
				client.KV().Put(p, nil)
			}

		}
	}()

	var check = func(index int, cpt *int) {
		*cpt++
		// Check related tasks have been created
		keys, _, err := client.KV().Keys(consulutil.TasksPrefix+"/", "/", nil)
		require.Nil(t, err, "Unexpected error while checking actions tasks")
		depTask := 0
		for _, key := range keys {
			kvp, _, err := client.KV().Get(key+"targetId", nil)
			if kvp != nil && string(kvp.Value) == deploymentID {
				depTask++
				kvp, _, err = client.KV().Get(key+"data/actionType", nil)
				require.Nil(t, err, "Unexpected error while getting action type")
				require.NotNil(t, kvp, "kvp is nil for action type")
				require.Equal(t, string(kvp.Value), actionType)

				kvp, _, err = client.KV().Get(key+"data/key1", nil)
				require.Nil(t, err, "Unexpected error while getting key1")
				require.NotNil(t, kvp, "kvp is nil for key1")
				require.Equal(t, string(kvp.Value), "val1")

				kvp, _, err = client.KV().Get(key+"data/key2", nil)
				require.Nil(t, err, "Unexpected error while getting key3")
				require.NotNil(t, kvp, "kvp is nil for key2")
				require.Equal(t, string(kvp.Value), "val2")

				kvp, _, err = client.KV().Get(key+"data/key3", nil)
				require.Nil(t, err, "Unexpected error while getting key3")
				require.NotNil(t, kvp, "kvp is nil for key3")
				require.Equal(t, string(kvp.Value), "val3")
			}
		}
		require.Equal(t, index, depTask, "Unexpected nb of tasks")
	}

	ind := 0
	checkCpt := 0
	ticker := time.NewTicker(ti)
	time.Sleep(2 * time.Second)
	for i := 0; i <= 2; i++ {
		select {
		case <-ticker.C:
			ind++
			check(ind, &checkCpt)
			if ind == 3 {
				ticker.Stop()
			}
		}
	}

	require.Equal(t, checkCpt, 3, "unexpected number of checks done")
}

func testProceedScheduledActionWithFirstActionStillRunning(t *testing.T, client *api.Client) {
	t.Parallel()
	deploymentID := "dep-" + t.Name()
	ti := 1 * time.Second
	actionType := "test-action"
	nodeName := "my-node"
	opeName := "my-op"
	interfaceName := "my-inter"
	taskID := "orig-taskID"
	wfName := "my-wf"
	action := &prov.Action{ActionType: actionType, Data: map[string]string{"key1": "val1", "key2": "val2", "key3": "val3"},
		AsyncOperation: prov.AsyncOperation{DeploymentID: deploymentID, NodeName: nodeName, Operation: prov.Operation{Name: interfaceName + "." + opeName}, TaskID: taskID, WorkflowName: wfName}}
	id, err := scheduling.RegisterAction(client, deploymentID, ti, action)

	require.Nil(t, err, "Unexpected error while registering action")
	require.NotEmpty(t, id, "id is not expected to be empty")

	closeCh := make(chan struct{})
	defer close(closeCh)
	go func() {
		var latestIndex uint64
		for {
			select {
			case <-closeCh:
				return
			default:
			}
			kvp, meta, err := client.KV().Get(path.Join(consulutil.SchedulingKVPrefix, "actions", id, "latestTaskID"), &api.QueryOptions{WaitIndex: latestIndex})
			if err != nil {
				t.Logf("%v", err)
				continue
			}
			if latestIndex == meta.LastIndex {
				continue
			}

			// Set the task status to RUNNING in order to not reschedule another task
			if kvp != nil && len(kvp.Value) > 0 {
				taskID := string(kvp.Value)
				p := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, taskID, "status"), Value: []byte(strconv.Itoa(int(tasks.TaskStatusRUNNING)))}
				client.KV().Put(p, nil)
			}

		}
	}()

	var check = func(index int, cpt *int) {
		*cpt++
		// Check related tasks have been created
		keys, _, err := client.KV().Keys(consulutil.TasksPrefix+"/", "/", nil)
		require.Nil(t, err, "Unexpected error while checking actions tasks")
		depTask := 0
		for _, key := range keys {
			kvp, _, err := client.KV().Get(key+"targetId", nil)
			if kvp != nil && string(kvp.Value) == deploymentID {
				depTask++
				kvp, _, err = client.KV().Get(key+"data/actionType", nil)
				require.Nil(t, err, "Unexpected error while getting action type")
				require.NotNil(t, kvp, "kvp is nil for action type")
				require.Equal(t, string(kvp.Value), actionType)

				kvp, _, err = client.KV().Get(key+"data/key1", nil)
				require.Nil(t, err, "Unexpected error while getting key1")
				require.NotNil(t, kvp, "kvp is nil for key1")
				require.Equal(t, string(kvp.Value), "val1")

				kvp, _, err = client.KV().Get(key+"data/key2", nil)
				require.Nil(t, err, "Unexpected error while getting key3")
				require.NotNil(t, kvp, "kvp is nil for key2")
				require.Equal(t, string(kvp.Value), "val2")

				kvp, _, err = client.KV().Get(key+"data/key3", nil)
				require.Nil(t, err, "Unexpected error while getting key3")
				require.NotNil(t, kvp, "kvp is nil for key3")
				require.Equal(t, string(kvp.Value), "val3")
			}
		}
		require.Equal(t, index, depTask, "Unexpected nb of tasks")
	}

	ind := 0
	checkCpt := 0
	ticker := time.NewTicker(ti)
	time.Sleep(2 * time.Second)
	for i := 0; i <= 2; i++ {
		select {
		case <-ticker.C:
			ind++
			// as the task is still running, no other task is created
			check(1, &checkCpt)
			if ind == 3 {
				ticker.Stop()
			}
		}
	}

	require.Equal(t, checkCpt, 3, "unexpected number of checks done")

	logs, _, err := events.LogsEvents(client.KV(), deploymentID, 0, 5*time.Second)
	require.NoError(t, err, "Could not retrieve logs")
	require.Equal(t, true, len(logs) > 0, "expected at least one logged event")

	var data map[string]interface{}
	err = json.Unmarshal(logs[0], &data)
	require.Nil(t, err)
	require.Equal(t, taskID, data["executionId"], "unexpected event executionID")
	require.Equal(t, wfName, data["workflowId"], "unexpected event workflowId")
	require.Equal(t, nodeName, data["nodeId"], "unexpected event nodeId")
	require.Equal(t, interfaceName, data["interfaceName"], "unexpected event interfaceName")
	require.Equal(t, opeName, data["operationName"], "unexpected event operationName")
}

func testProceedScheduledActionWithBadStatusError(t *testing.T, client *api.Client) {
	t.Parallel()
	deploymentID := "dep-" + t.Name()
	ti := 1 * time.Second
	actionType := "test-action"
	action := &prov.Action{ActionType: actionType, Data: map[string]string{"key1": "val1", "key2": "val2", "key3": "val3"}}
	id, err := scheduling.RegisterAction(client, deploymentID, ti, action)
	require.Nil(t, err, "Unexpected error while registering action")
	require.NotEmpty(t, id, "id is not expected to be empty")

	closeCh := make(chan struct{})
	defer close(closeCh)
	go func() {
		var latestIndex uint64
		for {
			select {
			case <-closeCh:
				return
			default:
			}
			kvp, meta, err := client.KV().Get(path.Join(consulutil.SchedulingKVPrefix, "actions", id, "latestTaskID"), &api.QueryOptions{WaitIndex: latestIndex})
			if err != nil {
				t.Logf("%v", err)
				continue
			}
			if latestIndex == meta.LastIndex {
				continue
			}

			// Set the task status to RUNNING in order to not reschedule another task
			if kvp != nil && len(kvp.Value) > 0 {
				taskID := string(kvp.Value)
				p := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, taskID, "status"), Value: []byte("BAD")}
				client.KV().Put(p, nil)
			}

		}
	}()

	var check = func(index int, cpt *int) {
		*cpt++
		// Check related tasks have been created
		keys, _, err := client.KV().Keys(consulutil.TasksPrefix+"/", "/", nil)
		require.Nil(t, err, "Unexpected error while checking actions tasks")
		depTask := 0
		for _, key := range keys {
			kvp, _, err := client.KV().Get(key+"targetId", nil)
			if kvp != nil && string(kvp.Value) == deploymentID {
				depTask++
				kvp, _, err = client.KV().Get(key+"data/actionType", nil)
				require.Nil(t, err, "Unexpected error while getting action type")
				require.NotNil(t, kvp, "kvp is nil for action type")
				require.Equal(t, string(kvp.Value), actionType)

				kvp, _, err = client.KV().Get(key+"data/key1", nil)
				require.Nil(t, err, "Unexpected error while getting key1")
				require.NotNil(t, kvp, "kvp is nil for key1")
				require.Equal(t, string(kvp.Value), "val1")

				kvp, _, err = client.KV().Get(key+"data/key2", nil)
				require.Nil(t, err, "Unexpected error while getting key3")
				require.NotNil(t, kvp, "kvp is nil for key2")
				require.Equal(t, string(kvp.Value), "val2")

				kvp, _, err = client.KV().Get(key+"data/key3", nil)
				require.Nil(t, err, "Unexpected error while getting key3")
				require.NotNil(t, kvp, "kvp is nil for key3")
				require.Equal(t, string(kvp.Value), "val3")
			}
		}
		require.Equal(t, index, depTask, "Unexpected nb of tasks")
	}

	ind := 0
	checkCpt := 0
	ticker := time.NewTicker(ti)
	time.Sleep(2 * time.Second)
	for i := 0; i <= 2; i++ {
		select {
		case <-ticker.C:
			ind++
			// as the proceed returns an error, the scheduler will stop and only one task will be created
			check(1, &checkCpt)
			if ind == 3 {
				ticker.Stop()
			}
		}
	}

	require.Equal(t, checkCpt, 3, "unexpected number of checks done")
}

func testUnregisterAction(t *testing.T, client *api.Client) {
	t.Parallel()
	deploymentID := "dep-" + t.Name()
	ti := 1 * time.Second
	actionType := "test-action"
	action := &prov.Action{ActionType: actionType, Data: map[string]string{"key1": "val1", "key2": "val2", "key3": "val3"}}
	id, err := scheduling.RegisterAction(client, deploymentID, ti, action)
	require.Nil(t, err, "Unexpected error while registering action")
	require.NotEmpty(t, id, "id is not expected to be empty")

	err = scheduling.UnregisterAction(client, id)
	require.Nil(t, err, "Unexpected error while unregistering action")

	kvp, _, err := client.KV().Get(path.Join(consulutil.SchedulingKVPrefix, "actions", id, ".unregisterFlag"), nil)
	require.Nil(t, err, "Unexpected error while getting flag for removal")
	require.NotNil(t, kvp, "kvp is nil")
	require.Equal(t, "true", string(kvp.Value), "unregisterFlag is not set to true")
}
