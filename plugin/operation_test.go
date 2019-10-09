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

package plugin

import (
	"context"
	"errors"
	"testing"
	"time"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/prov"
)

type mockOperationExecutor struct {
	execoperationCalled            bool
	execAsyncOperationCalled       bool
	ctx                            context.Context
	conf                           config.Configuration
	taskID, deploymentID, nodeName string
	operation                      prov.Operation
	contextCancelled               bool
	lof                            events.LogOptionalFields
}

func (m *mockOperationExecutor) ExecAsyncOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation, stepName string) (*prov.Action, time.Duration, error) {
	m.execAsyncOperationCalled = true
	m.ctx = ctx
	m.conf = conf
	m.taskID = taskID
	m.deploymentID = deploymentID
	m.nodeName = nodeName
	m.operation = operation
	m.lof, _ = events.FromContext(ctx)

	go func() {
		<-m.ctx.Done()
		m.contextCancelled = true
	}()
	if m.deploymentID == "TestCancel" {
		<-m.ctx.Done()
	}
	if m.deploymentID == "TestFailure" {
		return nil, 0, NewRPCError(errors.New("a failure occurred during plugin exec async operation"))
	}
	action := prov.Action{
		ID:         "testID",
		ActionType: "testActionType",
		AsyncOperation: prov.AsyncOperation{
			DeploymentID: deploymentID,
			TaskID:       taskID,
			StepName:     stepName,
			NodeName:     nodeName,
			Operation:    operation,
		},
		Data: map[string]string{
			"testDataKey": "testDataValue",
		},
	}
	return &action, 5 * time.Second, nil
}

func (m *mockOperationExecutor) ExecOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
	m.execoperationCalled = true
	m.ctx = ctx
	m.conf = conf
	m.taskID = taskID
	m.deploymentID = deploymentID
	m.nodeName = nodeName
	m.operation = operation
	m.lof, _ = events.FromContext(ctx)

	go func() {
		<-m.ctx.Done()
		m.contextCancelled = true
	}()
	if m.deploymentID == "TestCancel" {
		<-m.ctx.Done()
	}
	if m.deploymentID == "TestFailure" {
		return NewRPCError(errors.New("a failure occurred during plugin exec operation"))
	}
	return nil
}

func createMockOperationExecutorClient(t *testing.T) (*mockOperationExecutor, *plugin.RPCClient) {
	mock := new(mockOperationExecutor)
	client, _ := plugin.TestPluginRPCConn(
		t,
		map[string]plugin.Plugin{
			OperationPluginName: &OperationPlugin{F: func() prov.OperationExecutor {
				return mock
			}},
		},
		nil)
	return mock, client
}

func setupExecOperationTestEnv(t *testing.T) (*mockOperationExecutor, *plugin.RPCClient,
	prov.OperationExecutor, prov.Operation, events.LogOptionalFields, context.Context) {

	mock, client := createMockOperationExecutorClient(t)

	raw, err := client.Dispense(OperationPluginName)
	require.Nil(t, err)

	plugin := raw.(prov.OperationExecutor)
	op := prov.Operation{
		Name:                   "myOps",
		ImplementationArtifact: "tosca.artifacts.Implementation.Bash",
		RelOp: prov.RelationshipOperation{
			IsRelationshipOperation: true,
			RequirementIndex:        "1",
			TargetNodeName:          "AnotherNode",
		},
	}
	lof := events.LogOptionalFields{
		events.WorkFlowID:    "testWF",
		events.InterfaceName: "delegate",
		events.OperationName: "myTest",
	}
	ctx := events.NewContext(context.Background(), lof)

	return mock, client, plugin, op, lof, ctx

}
func TestOperationExecutorExecOperation(t *testing.T) {
	t.Parallel()
	mock, client, plugin, op, lof, ctx := setupExecOperationTestEnv(t)
	defer client.Close()
	err := plugin.ExecOperation(
		ctx,
		config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
		"TestTaskID", "TestDepID", "TestNodeName", op)
	require.Nil(t, err)
	require.True(t, mock.execoperationCalled)
	require.Equal(t, "test", mock.conf.Consul.Address)
	require.Equal(t, "testdc", mock.conf.Consul.Datacenter)
	require.Equal(t, "TestTaskID", mock.taskID)
	require.Equal(t, "TestDepID", mock.deploymentID)
	require.Equal(t, "TestNodeName", mock.nodeName)
	require.Equal(t, op, mock.operation)
	assert.Equal(t, lof, mock.lof)
}

func TestOperationExecutorExecOperationWithFailure(t *testing.T) {
	t.Parallel()
	mock, client, plugin, op, _, ctx := setupExecOperationTestEnv(t)
	defer client.Close()
	err := plugin.ExecOperation(
		ctx,
		config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
		"TestTaskID", "TestFailure", "TestNodeName", op)
	require.True(t, mock.execoperationCalled)
	require.Error(t, err, "An error was expected during executing plugin operation")
	require.EqualError(t, err, "a failure occurred during plugin exec operation")
}

func TestOperationExecutorExecOperationWithCancel(t *testing.T) {
	t.Parallel()
	mock, client := createMockOperationExecutorClient(t)
	defer client.Close()

	raw, err := client.Dispense(OperationPluginName)
	require.Nil(t, err)

	plugin := raw.(prov.OperationExecutor)
	lof := events.LogOptionalFields{
		events.WorkFlowID:    "testWF",
		events.InterfaceName: "delegate",
		events.OperationName: "myTest",
	}
	ctx := events.NewContext(context.Background(), lof)
	ctx, cancelF := context.WithCancel(ctx)
	go func() {
		err = plugin.ExecOperation(
			ctx,
			config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
			"TestTaskID", "TestCancel", "TestNodeName", prov.Operation{})
		require.Nil(t, err)
	}()
	cancelF()
	// Wait for cancellation signal to be dispatched
	time.Sleep(50 * time.Millisecond)
	require.True(t, mock.contextCancelled, "Context not cancelled")
}

func TestOperationGetSupportedArtifactTypes(t *testing.T) {
	mock := new(mockOperationExecutor)
	client, _ := plugin.TestPluginRPCConn(
		t,
		map[string]plugin.Plugin{
			OperationPluginName: &OperationPlugin{
				F: func() prov.OperationExecutor {
					return mock
				},
				SupportedTypes: []string{"tosca.my.types", "test"}},
		},
		nil)
	defer client.Close()
	raw, err := client.Dispense(OperationPluginName)
	require.Nil(t, err)
	plugin := raw.(OperationExecutor)

	supportedTypes, err := plugin.GetSupportedArtifactTypes()
	require.Nil(t, err)
	require.Len(t, supportedTypes, 2)
	require.Contains(t, supportedTypes, "tosca.my.types")
	require.Contains(t, supportedTypes, "test")
}

func TestOperationExecutorExecAsyncOperation(t *testing.T) {
	t.Parallel()
	mock, client, plugin, op, lof, ctx := setupExecOperationTestEnv(t)
	defer client.Close()
	action, interval, err := plugin.ExecAsyncOperation(
		ctx,
		config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
		"TestTaskID", "TestDepID", "TestNodeName", op, "TestStepName")
	require.NoError(t, err, "Failed to call plugin ExecAsyncOperation")
	require.True(t, mock.execAsyncOperationCalled)
	require.Equal(t, "TestTaskID", action.AsyncOperation.TaskID)
	require.Equal(t, "testDataValue", action.Data["testDataKey"])
	require.Equal(t, "TestNodeName", action.AsyncOperation.NodeName)
	require.Equal(t, op, action.AsyncOperation.Operation)
	require.Equal(t, 5*time.Second, interval)
	assert.Equal(t, lof, mock.lof)
}

func TestOperationExecutorExecAsyncOperationWithFailure(t *testing.T) {
	t.Parallel()
	mock, client, plugin, op, _, ctx := setupExecOperationTestEnv(t)
	defer client.Close()
	_, _, err := plugin.ExecAsyncOperation(
		ctx,
		config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
		"TestTaskID", "TestFailure", "TestNodeName", op, "TestStepName")
	require.True(t, mock.execAsyncOperationCalled)
	require.Error(t, err, "An error was expected calling plugin ExecAsyncOperation")
	require.EqualError(t, err, "a failure occurred during plugin exec async operation")
}

func TestOperationExecutorExecAsyncOperationWithCancel(t *testing.T) {
	t.Parallel()
	mock, client := createMockOperationExecutorClient(t)
	defer client.Close()

	raw, err := client.Dispense(OperationPluginName)
	require.Nil(t, err)

	plugin := raw.(prov.OperationExecutor)
	lof := events.LogOptionalFields{
		events.WorkFlowID:    "testWF",
		events.InterfaceName: "delegate",
		events.OperationName: "myTest",
	}
	ctx := events.NewContext(context.Background(), lof)
	ctx, cancelF := context.WithCancel(ctx)
	go func() {
		_, _, err := plugin.ExecAsyncOperation(
			ctx,
			config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
			"TestTaskID", "TestCancel", "TestNodeName", prov.Operation{}, "TestStepName")
		require.Nil(t, err)
	}()
	cancelF()
	// Wait for cancellation signal to be dispatched
	time.Sleep(50 * time.Millisecond)
	require.True(t, mock.contextCancelled, "Context not cancelled")
}
