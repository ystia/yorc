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

package events

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/testutil"
)

func testConsulPubSubStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)

	var testData = []struct {
		node     string
		instance string
		status   string
	}{
		{"node1", "0", "initial"},
		{"node2", "0", "initial"},
		{"node1", "0", "created"},
		{"node1", "0", "started"},
		{"node2", "0", "created"},
		{"node3", "0", "initial"},
		{"node2", "0", "configured"},
		{"node3", "0", "created"},
		{"node2", "0", "started"},
		{"node3", "0", "error"},
	}

	ids := make([]string, 0)
	for _, tc := range testData {
		id, err := InstanceStatusChange(kv, deploymentID, tc.node, tc.instance, tc.status)
		assert.Nil(t, err)
		ids = append(ids, id)
	}
	prefix := path.Join(consulutil.EventsPrefix, deploymentID)
	kvps, _, err := kv.List(prefix, nil)
	assert.Nil(t, err)
	assert.Len(t, kvps, len(testData))

	for index, kvp := range kvps {
		assert.Equal(t, ids[index], strings.TrimPrefix(kvp.Key, prefix+"/"))
		tc := testData[index]
		assert.Equal(t, tc.node+"\n"+tc.status+"\n"+tc.instance, string(kvp.Value))
	}
}

func testConsulPubSubNewEvents(t *testing.T, kv *api.KV) {
	// Do not run this test in // as it cause some concurrency issue
	// t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)

	nodeName := "node1"
	instance := "0"
	nodeStatus := "error"

	ready := make(chan struct{})

	go func() {
		i, err := GetStatusEventsIndex(kv, deploymentID)
		require.Nil(t, err)
		ready <- struct{}{}
		events, _, err := StatusEvents(kv, deploymentID, i, 5*time.Minute)
		assert.Nil(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, events[0].Node, nodeName)
		assert.Equal(t, events[0].Status, nodeStatus)
		assert.Equal(t, events[0].Instance, instance)
	}()
	<-ready
	_, err := InstanceStatusChange(kv, deploymentID, nodeName, instance, nodeStatus)
	assert.Nil(t, err)
}

func testConsulPubSubNewEventsTimeout(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)

	timeout := 25 * time.Millisecond

	t1 := time.Now()
	events, _, err := StatusEvents(kv, deploymentID, 1, timeout)
	t2 := time.Now()
	assert.Nil(t, err)
	require.Len(t, events, 0)
	assert.WithinDuration(t, t1, t2, timeout+50*time.Millisecond)
}

func testConsulPubSubNewEventsWithIndex(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)

	var testData = []struct {
		node     string
		instance string
		status   string
	}{
		{"node1", "0", "initial"},
		{"node1", "1", "initial"},
		{"node1", "0", "creating"},
		{"node1", "1", "creating"},
	}

	for _, tc := range testData {
		_, err := InstanceStatusChange(kv, deploymentID, tc.node, tc.instance, tc.status)
		assert.Nil(t, err)
	}

	events, lastIdx, err := StatusEvents(kv, deploymentID, 1, 5*time.Minute)
	assert.Nil(t, err)
	require.Len(t, events, 4)
	for index, event := range events {
		assert.Equal(t, testData[index].node, event.Node)
		assert.Equal(t, testData[index].instance, event.Instance)
		assert.Equal(t, testData[index].status, event.Status)
	}

	testData = []struct {
		node     string
		instance string
		status   string
	}{
		{"node1", "0", "created"},
		{"node1", "0", "configuring"},
		{"node1", "0", "configured"},
	}

	for _, tc := range testData {
		_, err = InstanceStatusChange(kv, deploymentID, tc.node, tc.instance, tc.status)
		assert.Nil(t, err)
	}

	events, lastIdx, err = StatusEvents(kv, deploymentID, lastIdx, 5*time.Minute)
	assert.Nil(t, err)
	require.Len(t, events, 3)
	require.NotZero(t, lastIdx)

	for index, event := range events {
		assert.Equal(t, testData[index].node, event.Node)
		assert.Equal(t, testData[index].instance, event.Instance)
		assert.Equal(t, testData[index].status, event.Status)
	}
}

func testConsulPubSubNewNodeEvents(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)

	nodeName := "node1"
	instance := "0"
	nodeStatus := "error"

	_, err := InstanceStatusChange(kv, deploymentID, nodeName, instance, nodeStatus)
	assert.Nil(t, err)

}

func testconsulDeploymentStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	type args struct {
		kv     *api.KV
		status string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"TestStatusInitial", args{kv, "initial"}, false},
		{"TestStatusDepInProgress", args{kv, "deployment_in_progress"}, false},
		{"TestStatusDepDeployed", args{kv, "deployed"}, false},
	}
	ids := make([]string, 0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeploymentStatusChange(tt.args.kv, deploymentID, tt.args.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeploymentStatusChange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != "" {
				ids = append(ids, got)
			}
		})
	}

	prefix := path.Join(consulutil.EventsPrefix, deploymentID)
	kvps, _, err := kv.List(prefix, nil)
	assert.Nil(t, err)
	assert.Len(t, kvps, len(tests))

	for index, kvp := range kvps {
		assert.Equal(t, ids[index], strings.TrimPrefix(kvp.Key, prefix+"/"))
		tc := tests[index]
		assert.Equal(t, tc.args.status, string(kvp.Value))
	}

}

func testconsulCustomCommandStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	type args struct {
		kv     *api.KV
		taskID string
		status string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"TestStatusInitial", args{kv, "1", "initial"}, false},
		{"TestStatusDepInProgress", args{kv, "1", "running"}, false},
		{"TestStatusDepDeployed", args{kv, "2", "initial"}, false},
		{"TestStatusDepDeployed", args{kv, "2", "running"}, false},
		{"TestStatusDepDeployed", args{kv, "1", "done"}, false},
		{"TestStatusDepDeployed", args{kv, "2", "done"}, false},
	}
	ids := make([]string, 0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CustomCommandStatusChange(tt.args.kv, deploymentID, tt.args.taskID, tt.args.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("CustomCommandStatusChange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != "" {
				ids = append(ids, got)
			}
		})
	}

	prefix := path.Join(consulutil.EventsPrefix, deploymentID)
	kvps, _, err := kv.List(prefix, nil)
	assert.Nil(t, err)
	assert.Len(t, kvps, len(tests))

	for index, kvp := range kvps {
		assert.Equal(t, ids[index], strings.TrimPrefix(kvp.Key, prefix+"/"))
		tc := tests[index]
		assert.Equal(t, tc.args.taskID+"\n"+tc.args.status, string(kvp.Value))
	}

}

func testconsulScalingStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	type args struct {
		kv     *api.KV
		taskID string
		status string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"TestStatusInitial", args{kv, "1", "initial"}, false},
		{"TestStatusDepInProgress", args{kv, "1", "running"}, false},
		{"TestStatusDepDeployed", args{kv, "2", "initial"}, false},
		{"TestStatusDepDeployed", args{kv, "2", "running"}, false},
		{"TestStatusDepDeployed", args{kv, "1", "done"}, false},
		{"TestStatusDepDeployed", args{kv, "2", "done"}, false},
	}
	ids := make([]string, 0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ScalingStatusChange(tt.args.kv, deploymentID, tt.args.taskID, tt.args.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScalingStatusChange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != "" {
				ids = append(ids, got)
			}
		})
	}

	prefix := path.Join(consulutil.EventsPrefix, deploymentID)
	kvps, _, err := kv.List(prefix, nil)
	assert.Nil(t, err)
	assert.Len(t, kvps, len(tests))

	for index, kvp := range kvps {
		assert.Equal(t, ids[index], strings.TrimPrefix(kvp.Key, prefix+"/"))
		tc := tests[index]
		assert.Equal(t, tc.args.taskID+"\n"+tc.args.status, string(kvp.Value))
	}

}

func testconsulWorkflowStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	type args struct {
		kv     *api.KV
		taskID string
		status string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"TestStatusInitial", args{kv, "1", "initial"}, false},
		{"TestStatusDepInProgress", args{kv, "1", "running"}, false},
		{"TestStatusDepDeployed", args{kv, "2", "initial"}, false},
		{"TestStatusDepDeployed", args{kv, "2", "running"}, false},
		{"TestStatusDepDeployed", args{kv, "1", "done"}, false},
		{"TestStatusDepDeployed", args{kv, "2", "done"}, false},
	}
	ids := make([]string, 0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := WorkflowStatusChange(tt.args.kv, deploymentID, tt.args.taskID, tt.args.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("WorkflowStatusChange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != "" {
				ids = append(ids, got)
			}
		})
	}

	prefix := path.Join(consulutil.EventsPrefix, deploymentID)
	kvps, _, err := kv.List(prefix, nil)
	assert.Nil(t, err)
	assert.Len(t, kvps, len(tests))

	for index, kvp := range kvps {
		assert.Equal(t, ids[index], strings.TrimPrefix(kvp.Key, prefix+"/"))
		tc := tests[index]
		assert.Equal(t, tc.args.taskID+"\n"+tc.args.status, string(kvp.Value))
	}

}

func testconsulGetStatusEvents(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	ids := make([]string, 5)
	id, err := InstanceStatusChange(kv, deploymentID, "node1", "1", "started")
	require.Nil(t, err)
	ids[0] = id
	id, err = DeploymentStatusChange(kv, deploymentID, "deployed")
	require.Nil(t, err)
	ids[1] = id
	id, err = ScalingStatusChange(kv, deploymentID, "t2", "failed")
	require.Nil(t, err)
	ids[2] = id
	id, err = CustomCommandStatusChange(kv, deploymentID, "t3", "running")
	require.Nil(t, err)
	ids[3] = id
	id, err = WorkflowStatusChange(kv, deploymentID, "t4", "done")
	require.Nil(t, err)
	ids[4] = id

	events, _, err := StatusEvents(kv, deploymentID, 0, 5*time.Minute)
	require.Nil(t, err)
	require.Len(t, events, 5)

	require.Equal(t, InstanceStatusChangeType.String(), events[0].Type)
	require.Equal(t, "node1", events[0].Node)
	require.Equal(t, "1", events[0].Instance)
	require.Equal(t, "started", events[0].Status)
	require.Equal(t, ids[0], events[0].Timestamp)
	require.Equal(t, "", events[0].TaskID)

	require.Equal(t, DeploymentStatusChangeType.String(), events[1].Type)
	require.Equal(t, "", events[1].Node)
	require.Equal(t, "", events[1].Instance)
	require.Equal(t, "deployed", events[1].Status)
	require.Equal(t, ids[1], events[1].Timestamp)
	require.Equal(t, "", events[1].TaskID)

	require.Equal(t, ScalingStatusChangeType.String(), events[2].Type)
	require.Equal(t, "", events[2].Node)
	require.Equal(t, "", events[2].Instance)
	require.Equal(t, "failed", events[2].Status)
	require.Equal(t, ids[2], events[2].Timestamp)
	require.Equal(t, "t2", events[2].TaskID)

	require.Equal(t, CustomCommandStatusChangeType.String(), events[3].Type)
	require.Equal(t, "", events[3].Node)
	require.Equal(t, "", events[3].Instance)
	require.Equal(t, "running", events[3].Status)
	require.Equal(t, ids[3], events[3].Timestamp)
	require.Equal(t, "t3", events[3].TaskID)

	require.Equal(t, WorkflowStatusChangeType.String(), events[4].Type)
	require.Equal(t, "", events[4].Node)
	require.Equal(t, "", events[4].Instance)
	require.Equal(t, "done", events[4].Status)
	require.Equal(t, ids[4], events[4].Timestamp)
	require.Equal(t, "t4", events[4].TaskID)

}

func testconsulGetLogs(t *testing.T, kv *api.KV) {
	t.Parallel()
	myErr := errors.New("MyError")
	deploymentID := testutil.BuildDeploymentID(t)
	prevIndex, err := GetLogsEventsIndex(kv, deploymentID)
	require.Nil(t, err)
	SimpleLogEntry(LogLevelERROR, deploymentID).RegisterAsString(myErr.Error())
	newIndex, err := GetLogsEventsIndex(kv, deploymentID)
	require.Nil(t, err)
	require.True(t, prevIndex < newIndex)
	prevIndex = newIndex
	SimpleLogEntry(LogLevelINFO, deploymentID).RegisterAsString("message1")
	newIndex, err = GetLogsEventsIndex(kv, deploymentID)
	require.Nil(t, err)
	require.True(t, prevIndex < newIndex)
	prevIndex = newIndex
	SimpleLogEntry(LogLevelINFO, deploymentID).RegisterAsString("message2")
	newIndex, err = GetLogsEventsIndex(kv, deploymentID)
	require.Nil(t, err)
	require.True(t, prevIndex < newIndex)
	prevIndex = newIndex
	SimpleLogEntry(LogLevelINFO, deploymentID).RegisterAsString("message3")
	newIndex, err = GetLogsEventsIndex(kv, deploymentID)
	require.Nil(t, err)
	require.True(t, prevIndex < newIndex)
	prevIndex = newIndex
	logs, _, err := LogsEvents(kv, deploymentID, 0, 5*time.Minute)
	require.Nil(t, err)
	require.Len(t, logs, 4)

	require.Equal(t, fmt.Sprintf("%v", myErr), getLogContent(t, logs[0]))
	require.Equal(t, "message1", getLogContent(t, logs[1]))
	require.Equal(t, "message2", getLogContent(t, logs[2]))
	require.Equal(t, "message3", getLogContent(t, logs[3]))

}

func getLogContent(t *testing.T, log []byte) string {
	var data map[string]interface{}
	err := json.Unmarshal(log, &data)
	require.Nil(t, err)
	return data["content"].(string)
}
