package events

import (
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	"fmt"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

func TestGroupedEventParallel(t *testing.T) {
	srv1, err := testutil.NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create consul server: %v", err)
	}
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

	kv := client.KV()

	consulutil.InitConsulPublisher(500, kv)
	t.Run("groupEvent", func(t *testing.T) {
		t.Run("TestConsulPubSubStatusChange", func(t *testing.T) {
			ConsulPubSubStatusChange(t, kv)
		})
		t.Run("TestConsulPubSubNewEvents", func(t *testing.T) {
			ConsulPubSubNewEvents(t, kv)
		})
		t.Run("TestConsulPubSubNewEventsTimeout", func(t *testing.T) {
			ConsulPubSubNewEventsTimeout(t, kv)
		})
		t.Run("TestConsulPubSubNewEventsWithIndex", func(t *testing.T) {
			ConsulPubSubNewEventsWithIndex(t, kv)
		})
		t.Run("TestConsulPubSubNewNodeEvents", func(t *testing.T) {
			ConsulPubSubNewNodeEvents(t, kv)
		})
		t.Run("TestDeploymentStatusChange", func(t *testing.T) {
			consulDeploymentStatusChange(t, kv)
		})
		t.Run("TestCustomCommandStatusChange", func(t *testing.T) {
			consulCustomCommandStatusChange(t, kv)
		})
		t.Run("TestScalingStatusChange", func(t *testing.T) {
			consulScalingStatusChange(t, kv)
		})
		t.Run("TestWorkflowStatusChange", func(t *testing.T) {
			consulWorkflowStatusChange(t, kv)
		})
		t.Run("TestGetStatusEvents", func(t *testing.T) {
			consulGetStatusEvents(t, kv)
		})
		t.Run("TestGetLogs", func(t *testing.T) {
			consulGetLogs(t, kv)
		})
	})
}

func ConsulPubSubStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := "test1"

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
	prefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "events")
	kvps, _, err := kv.List(prefix, nil)
	assert.Nil(t, err)
	assert.Len(t, kvps, len(testData))

	for index, kvp := range kvps {
		assert.Equal(t, ids[index], strings.TrimPrefix(kvp.Key, prefix+"/"))
		tc := testData[index]
		assert.Equal(t, tc.node+"\n"+tc.status+"\n"+tc.instance, string(kvp.Value))
	}
}

func ConsulPubSubNewEvents(t *testing.T, kv *api.KV) {
	// Do not run this test in // as it cause some concurrency issue
	// t.Parallel()
	deploymentID := "test2"
	sub := NewSubscriber(kv, deploymentID)

	nodeName := "node1"
	instance := "0"
	nodeStatus := "error"

	ready := make(chan struct{})

	go func() {
		i, err := GetStatusEventsIndex(kv, deploymentID)
		require.Nil(t, err)
		ready <- struct{}{}
		events, _, err := sub.StatusEvents(i, 5*time.Minute)
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

func ConsulPubSubNewEventsTimeout(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := "test3"
	sub := NewSubscriber(kv, deploymentID)

	timeout := 25 * time.Millisecond

	t1 := time.Now()
	events, _, err := sub.StatusEvents(1, timeout)
	t2 := time.Now()
	assert.Nil(t, err)
	require.Len(t, events, 0)
	assert.WithinDuration(t, t1, t2, timeout+50*time.Millisecond)
}

func ConsulPubSubNewEventsWithIndex(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := "test4"
	sub := NewSubscriber(kv, deploymentID)

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

	events, lastIdx, err := sub.StatusEvents(1, 5*time.Minute)
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

	events, lastIdx, err = sub.StatusEvents(lastIdx, 5*time.Minute)
	assert.Nil(t, err)
	require.Len(t, events, 3)
	require.NotZero(t, lastIdx)

	for index, event := range events {
		assert.Equal(t, testData[index].node, event.Node)
		assert.Equal(t, testData[index].instance, event.Instance)
		assert.Equal(t, testData[index].status, event.Status)
	}
}

func ConsulPubSubNewNodeEvents(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := "test5"

	nodeName := "node1"
	instance := "0"
	nodeStatus := "error"

	_, err := InstanceStatusChange(kv, deploymentID, nodeName, instance, nodeStatus)
	assert.Nil(t, err)

}

func consulDeploymentStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := "consulDeploymentStatusChange"
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

	prefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "events")
	kvps, _, err := kv.List(prefix, nil)
	assert.Nil(t, err)
	assert.Len(t, kvps, len(tests))

	for index, kvp := range kvps {
		assert.Equal(t, ids[index], strings.TrimPrefix(kvp.Key, prefix+"/"))
		tc := tests[index]
		assert.Equal(t, tc.args.status, string(kvp.Value))
	}

}

func consulCustomCommandStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := "consulCustomCommandChange"
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

	prefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "events")
	kvps, _, err := kv.List(prefix, nil)
	assert.Nil(t, err)
	assert.Len(t, kvps, len(tests))

	for index, kvp := range kvps {
		assert.Equal(t, ids[index], strings.TrimPrefix(kvp.Key, prefix+"/"))
		tc := tests[index]
		assert.Equal(t, tc.args.taskID+"\n"+tc.args.status, string(kvp.Value))
	}

}

func consulScalingStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := "consulScalingChange"
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

	prefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "events")
	kvps, _, err := kv.List(prefix, nil)
	assert.Nil(t, err)
	assert.Len(t, kvps, len(tests))

	for index, kvp := range kvps {
		assert.Equal(t, ids[index], strings.TrimPrefix(kvp.Key, prefix+"/"))
		tc := tests[index]
		assert.Equal(t, tc.args.taskID+"\n"+tc.args.status, string(kvp.Value))
	}

}

func consulWorkflowStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := "consulWorkflowChange"
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

	prefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "events")
	kvps, _, err := kv.List(prefix, nil)
	assert.Nil(t, err)
	assert.Len(t, kvps, len(tests))

	for index, kvp := range kvps {
		assert.Equal(t, ids[index], strings.TrimPrefix(kvp.Key, prefix+"/"))
		tc := tests[index]
		assert.Equal(t, tc.args.taskID+"\n"+tc.args.status, string(kvp.Value))
	}

}

func Test_consulPubSub_StatusEvents(t *testing.T) {
	type fields struct {
		kv           *api.KV
		deploymentID string
	}
	type args struct {
		waitIndex uint64
		timeout   time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []StatusUpdate
		want1   uint64
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := &consulPubSub{
				kv:           tt.fields.kv,
				deploymentID: tt.fields.deploymentID,
			}
			got, got1, err := cp.StatusEvents(tt.args.waitIndex, tt.args.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("consulPubSub.StatusEvents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("consulPubSub.StatusEvents() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("consulPubSub.StatusEvents() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func consulGetStatusEvents(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := "consulGetStatusEvents"
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

	sub := NewSubscriber(kv, deploymentID)
	events, _, err := sub.StatusEvents(0, 5*time.Minute)
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

func consulGetLogs(t *testing.T, kv *api.KV) {
	t.Parallel()
	myErr := errors.New("MyError")
	deploymentID := "consulGetLogs"
	prevIndex, err := GetLogsEventsIndex(kv, deploymentID)
	require.Nil(t, err)
	LogEngineError(kv, deploymentID, myErr)
	newIndex, err := GetLogsEventsIndex(kv, deploymentID)
	require.Nil(t, err)
	require.True(t, prevIndex < newIndex)
	prevIndex = newIndex
	LogEngineMessage(kv, deploymentID, "message1")
	newIndex, err = GetLogsEventsIndex(kv, deploymentID)
	require.Nil(t, err)
	require.True(t, prevIndex < newIndex)
	prevIndex = newIndex
	LogInfrastructureMessage(kv, deploymentID, "message2")
	newIndex, err = GetLogsEventsIndex(kv, deploymentID)
	require.Nil(t, err)
	require.True(t, prevIndex < newIndex)
	prevIndex = newIndex
	LogSoftwareMessage(kv, deploymentID, "message3")
	newIndex, err = GetLogsEventsIndex(kv, deploymentID)
	require.Nil(t, err)
	require.True(t, prevIndex < newIndex)
	prevIndex = newIndex
	sub := NewSubscriber(kv, deploymentID)
	events, _, err := sub.LogsEvents("all", 0, 5*time.Minute)
	require.Nil(t, err)
	require.Len(t, events, 4)

	require.Equal(t, fmt.Sprintf("%v", myErr), events[0].Logs)
	require.Equal(t, "message1", events[1].Logs)
	require.Equal(t, "message2", events[2].Logs)
	require.Equal(t, "message3", events[3].Logs)

}
