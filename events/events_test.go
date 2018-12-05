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
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"context"
	"github.com/stretchr/testify/assert"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/testutil"
	"path"
	"strings"
)

func testConsulPubSubStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	ctx := context.Background()
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
		id, err := PublishAndLogInstanceStatusChange(ctx, kv, deploymentID, tc.node, tc.instance, tc.status)
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
		res := toStatusChangeMap(t, string(kvp.Value))
		assert.Equal(t, tc.node, res[ENodeID.String()], "unexpected node value for statusChange")
		assert.Equal(t, tc.status, res[EStatus.String()], "unexpected status value for statusChange")
		assert.Equal(t, deploymentID, res[EDeploymentID.String()], "unexpected deploymentID value for statusChange")
		assert.Equal(t, tc.instance, res[EInstanceID.String()], "unexpected instance value for statusChange")
	}
}

func toStatusChangeMap(t *testing.T, statusChange string) map[string]string {
	var data map[string]string
	err := json.Unmarshal([]byte(statusChange), &data)
	require.Nil(t, err)
	return data
}

func testConsulPubSubNewEvents(t *testing.T, kv *api.KV) {
	// Do not run this test in // as it cause some concurrency issue
	// t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	ctx := context.Background()
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

		event := toStatusChangeMap(t, string(events[0]))
		assert.Equal(t, event[ENodeID.String()], nodeName)
		assert.Equal(t, event[EStatus.String()], nodeStatus)
		assert.Equal(t, event[EInstanceID.String()], instance)
	}()
	<-ready
	_, err := PublishAndLogInstanceStatusChange(ctx, kv, deploymentID, nodeName, instance, nodeStatus)
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
	ctx := context.Background()
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
		_, err := PublishAndLogInstanceStatusChange(ctx, kv, deploymentID, tc.node, tc.instance, tc.status)
		assert.Nil(t, err)
	}

	rawEvents, lastIdx, err := StatusEvents(kv, deploymentID, 1, 5*time.Minute)
	assert.Nil(t, err)
	require.Len(t, rawEvents, 4)
	for index, event := range rawEvents {
		evt := toStatusChangeMap(t, string(event))
		assert.Equal(t, testData[index].node, evt[ENodeID.String()])
		assert.Equal(t, testData[index].instance, evt[EInstanceID.String()])
		assert.Equal(t, testData[index].status, evt[EStatus.String()])
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
		_, err = PublishAndLogInstanceStatusChange(ctx, kv, deploymentID, tc.node, tc.instance, tc.status)
		assert.Nil(t, err)
	}

	rawEvents, lastIdx, err = StatusEvents(kv, deploymentID, lastIdx, 5*time.Minute)
	assert.Nil(t, err)
	require.Len(t, rawEvents, 3)
	require.NotZero(t, lastIdx)

	for index, rawEvent := range rawEvents {
		event := toStatusChangeMap(t, string(rawEvent))
		assert.Equal(t, testData[index].node, event[ENodeID.String()])
		assert.Equal(t, testData[index].instance, event[EInstanceID.String()])
		assert.Equal(t, testData[index].status, event[EStatus.String()])
	}
}

func testConsulPubSubNewNodeEvents(t *testing.T, kv *api.KV) {
	t.Parallel()
	ctx := context.Background()
	deploymentID := testutil.BuildDeploymentID(t)

	nodeName := "node1"
	instance := "0"
	nodeStatus := "error"

	_, err := PublishAndLogInstanceStatusChange(ctx, kv, deploymentID, nodeName, instance, nodeStatus)
	assert.Nil(t, err)

}

func testconsulDeploymentStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	ctx := context.Background()
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
			got, err := PublishAndLogDeploymentStatusChange(ctx, tt.args.kv, deploymentID, tt.args.status)
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
		event := toStatusChangeMap(t, string(kvp.Value))
		assert.Equal(t, tc.args.status, event[EStatus.String()])
	}

}

func testconsulCustomCommandStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	ctx := context.Background()
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
			got, err := PublishAndLogCustomCommandStatusChange(ctx, tt.args.kv, deploymentID, tt.args.taskID, tt.args.status)
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
		event := toStatusChangeMap(t, string(kvp.Value))
		assert.Equal(t, tc.args.status, event[EStatus.String()])
		assert.Equal(t, tc.args.taskID, event[ETaskID.String()])
	}

}

func testconsulScalingStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	ctx := context.Background()
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
			got, err := PublishAndLogScalingStatusChange(ctx, tt.args.kv, deploymentID, tt.args.taskID, tt.args.status)
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
		event := toStatusChangeMap(t, string(kvp.Value))
		assert.Equal(t, tc.args.status, event[EStatus.String()])
		assert.Equal(t, tc.args.taskID, event[ETaskID.String()])
	}

}

func testconsulWorkflowStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	ctx := context.Background()
	deploymentID := testutil.BuildDeploymentID(t)
	type args struct {
		kv           *api.KV
		taskID       string
		status       string
		workflowName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"TestStatusInitial", args{kv, "1", "initial", "install"}, false},
		{"TestStatusDepInProgress", args{kv, "1", "running", "install"}, false},
		{"TestStatusDepDeployed", args{kv, "2", "initial", "install"}, false},
		{"TestStatusDepDeployed", args{kv, "2", "running", "install"}, false},
		{"TestStatusDepDeployed", args{kv, "1", "done", "install"}, false},
		{"TestStatusDepDeployed", args{kv, "2", "done", "install"}, false},
	}
	ids := make([]string, 0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PublishAndLogWorkflowStatusChange(ctx, tt.args.kv, deploymentID, tt.args.taskID, tt.args.workflowName, tt.args.status)
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
		event := toStatusChangeMap(t, string(kvp.Value))
		assert.Equal(t, tc.args.status, event[EStatus.String()])
		assert.Equal(t, tc.args.taskID, event[ETaskID.String()])
		assert.Equal(t, tc.args.workflowName, event[EWorkflowID.String()])
	}
}

func testconsulWorkflowStepStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	type args struct {
		kv     *api.KV
		taskID string
		status string
		wfInfo *WorkflowStepInfo
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"TestStatusInitial", args{kv, "1", "initial", &WorkflowStepInfo{WorkflowName: "install", InstanceName: "0", StepName: "step1", NodeName: "node1"}}, false},
		{"TestStatusDepInProgress", args{kv, "1", "running", &WorkflowStepInfo{WorkflowName: "install", InstanceName: "0", StepName: "step1", NodeName: "node1"}}, false},
		{"TestStatusDepDeployed", args{kv, "2", "initial", &WorkflowStepInfo{WorkflowName: "install", InstanceName: "0", StepName: "step1", NodeName: "node1"}}, false},
		{"TestStatusDepDeployed", args{kv, "2", "running", &WorkflowStepInfo{WorkflowName: "install", InstanceName: "0", StepName: "step1", NodeName: "node1"}}, false},
		{"TestStatusDepDeployed", args{kv, "1", "done", &WorkflowStepInfo{WorkflowName: "install", InstanceName: "0", StepName: "step1", NodeName: "node1"}}, false},
		{"TestStatusDepDeployed", args{kv, "2", "done", &WorkflowStepInfo{WorkflowName: "install", InstanceName: "0", StepName: "step1", NodeName: "node1"}}, false},
	}
	ids := make([]string, 0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PublishAndLogWorkflowStepStatusChange(context.Background(), tt.args.kv, deploymentID, tt.args.taskID, tt.args.wfInfo, tt.args.status)
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
		event := toStatusChangeMap(t, string(kvp.Value))
		assert.Equal(t, tc.args.status, event[EStatus.String()])
		assert.Equal(t, tc.args.taskID, event[ETaskID.String()])
		assert.Equal(t, "install", event[EWorkflowID.String()])
		assert.Equal(t, "0", event[EInstanceID.String()])
		assert.Equal(t, "step1", event[EWorkflowStepID.String()])
		assert.Equal(t, "node1", event[ENodeID.String()])
	}
}

func testconsulAlienTaskStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	type args struct {
		kv              *api.KV
		taskID          string
		taskExecutionID string
		status          string
		wfInfo          *WorkflowStepInfo
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"TestStatusInitial", args{kv, "1", "33", "initial", &WorkflowStepInfo{WorkflowName: "install", InstanceName: "0", StepName: "step1", NodeName: "node1"}}, false},
		{"TestStatusDepInProgress", args{kv, "1", "33", "running", &WorkflowStepInfo{WorkflowName: "install", InstanceName: "0", StepName: "step1", NodeName: "node1"}}, false},
		{"TestStatusDepDeployed", args{kv, "2", "33", "initial", &WorkflowStepInfo{WorkflowName: "install", InstanceName: "0", StepName: "step1", NodeName: "node1"}}, false},
		{"TestStatusDepDeployed", args{kv, "2", "33", "running", &WorkflowStepInfo{WorkflowName: "install", InstanceName: "0", StepName: "step1", NodeName: "node1"}}, false},
		{"TestStatusDepDeployed", args{kv, "1", "33", "done", &WorkflowStepInfo{WorkflowName: "install", InstanceName: "0", StepName: "step1", NodeName: "node1"}}, false},
		{"TestStatusDepDeployed", args{kv, "2", "33", "done", &WorkflowStepInfo{WorkflowName: "install", InstanceName: "0", StepName: "step1", NodeName: "node1"}}, false},
	}
	ids := make([]string, 0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PublishAndLogAlienTaskStatusChange(context.Background(), tt.args.kv, deploymentID, tt.args.taskID, tt.args.taskExecutionID, tt.args.wfInfo, tt.args.status)
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
		event := toStatusChangeMap(t, string(kvp.Value))
		assert.Equal(t, tc.args.status, event[EStatus.String()])
		assert.Equal(t, tc.args.taskID, event[ETaskID.String()])
		assert.Equal(t, tc.args.taskExecutionID, event[ETaskExecutionID.String()])
		assert.Equal(t, "install", event[EWorkflowID.String()])
		assert.Equal(t, "0", event[EInstanceID.String()])
		assert.Equal(t, "step1", event[EWorkflowStepID.String()])
		assert.Equal(t, "node1", event[ENodeID.String()])
	}
}

func testconsulGetStatusEvents(t *testing.T, kv *api.KV) {
	t.Parallel()
	ctx := context.Background()
	deploymentID := testutil.BuildDeploymentID(t)
	ids := make([]string, 5)
	id, err := PublishAndLogInstanceStatusChange(ctx, kv, deploymentID, "node1", "1", "started")
	require.Nil(t, err)
	ids[0] = id
	id, err = PublishAndLogDeploymentStatusChange(ctx, kv, deploymentID, "deployed")
	require.Nil(t, err)
	ids[1] = id
	id, err = PublishAndLogScalingStatusChange(ctx, kv, deploymentID, "t2", "failed")
	require.Nil(t, err)
	ids[2] = id
	id, err = PublishAndLogCustomCommandStatusChange(ctx, kv, deploymentID, "t3", "running")
	require.Nil(t, err)
	ids[3] = id
	id, err = PublishAndLogWorkflowStatusChange(ctx, kv, deploymentID, "t4", "install", "done")
	require.Nil(t, err)
	ids[4] = id

	rawEvents, _, err := StatusEvents(kv, deploymentID, 0, 5*time.Minute)
	require.Nil(t, err)
	require.Len(t, rawEvents, 5)

	events := make([]map[string]string, len(rawEvents))
	for i, rawEvent := range rawEvents {
		events[i] = toStatusChangeMap(t, string(rawEvent))
	}

	require.Equal(t, StatusChangeTypeInstance.String(), events[0][EType.String()])
	require.Equal(t, "node1", events[0][ENodeID.String()])
	require.Equal(t, "1", events[0][EInstanceID.String()])
	require.Equal(t, "started", events[0][EStatus.String()])
	require.Equal(t, ids[0], events[0][ETimestamp.String()])
	require.Equal(t, "", events[0][ETaskID.String()])

	require.Equal(t, StatusChangeTypeDeployment.String(), events[1][EType.String()])
	require.Equal(t, "", events[1][ENodeID.String()])
	require.Equal(t, "", events[1][EInstanceID.String()])
	require.Equal(t, "deployed", events[1][EStatus.String()])
	require.Equal(t, ids[1], events[1][ETimestamp.String()])
	require.Equal(t, "", events[1][ETaskID.String()])

	require.Equal(t, StatusChangeTypeScaling.String(), events[2][EType.String()])
	require.Equal(t, "", events[2][ENodeID.String()])
	require.Equal(t, "", events[2][EInstanceID.String()])
	require.Equal(t, "failed", events[2][EStatus.String()])
	require.Equal(t, ids[2], events[2][ETimestamp.String()])
	require.Equal(t, "t2", events[2][ETaskID.String()])

	require.Equal(t, StatusChangeTypeCustomCommand.String(), events[3][EType.String()])
	require.Equal(t, "", events[3][ENodeID.String()])
	require.Equal(t, "", events[3][EInstanceID.String()])
	require.Equal(t, "running", events[3][EStatus.String()])
	require.Equal(t, ids[3], events[3][ETimestamp.String()])
	require.Equal(t, "t3", events[3][ETaskID.String()])

	require.Equal(t, StatusChangeTypeWorkflow.String(), events[4][EType.String()])
	require.Equal(t, "", events[4][ENodeID.String()])
	require.Equal(t, "", events[4][EInstanceID.String()])
	require.Equal(t, "done", events[4][EStatus.String()])
	require.Equal(t, ids[4], events[4][ETimestamp.String()])
	require.Equal(t, "t4", events[4][ETaskID.String()])
	require.Equal(t, "install", events[4][EWorkflowID.String()])
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
