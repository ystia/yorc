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

type mockActionOperator struct {
	execactionCalled     bool
	ctx                  context.Context
	conf                 config.Configuration
	taskID, deploymentID string
	action               *prov.Action
	contextCancelled     bool
	lof                  events.LogOptionalFields
}

func (m *mockActionOperator) ExecAction(ctx context.Context, conf config.Configuration, taskID, deploymentID string, action *prov.Action) (deregister bool, err error) {

	m.execactionCalled = true
	m.ctx = ctx
	m.conf = conf
	m.taskID = taskID
	m.deploymentID = deploymentID
	m.action = action
	m.lof, _ = events.FromContext(ctx)

	go func() {
		<-m.ctx.Done()
		m.contextCancelled = true
	}()
	if m.deploymentID == "TestCancel" {
		<-m.ctx.Done()
	}
	if m.deploymentID == "TestFailure" {
		return false, NewRPCError(errors.New("a failure occurred during plugin exec operation"))
	}
	return true, nil
}

func createMockActionOperatorClient(t *testing.T) (*mockActionOperator, *plugin.RPCClient) {
	mock := new(mockActionOperator)
	client, _ := plugin.TestPluginRPCConn(
		t,
		map[string]plugin.Plugin{
			ActionPluginName: &ActionPlugin{F: func() prov.ActionOperator {
				return mock
			}},
		},
		nil)
	return mock, client
}

func setupExecActionTestEnv(t *testing.T) (*mockActionOperator, *plugin.RPCClient,
	prov.ActionOperator, prov.Action, events.LogOptionalFields, context.Context) {

	mock, client := createMockActionOperatorClient(t)

	raw, err := client.Dispense(ActionPluginName)
	require.Nil(t, err)

	plugin := raw.(prov.ActionOperator)
	testOperation := prov.Operation{
		Name:                   "testOperationName",
		ImplementationArtifact: "tosca.artifacts.Implementation.Bash",
		RelOp: prov.RelationshipOperation{
			IsRelationshipOperation: true,
			RequirementIndex:        "1",
			TargetNodeName:          "AnotherNode",
		},
	}
	testAsyncOperation := prov.AsyncOperation{
		DeploymentID: "testDeploymentID",
		TaskID:       "testTaskID",
		ExecutionID:  "testExecutionID",
		WorkflowName: "testWorkflowName",
		StepName:     "testStepName",
		NodeName:     "testNodeName",
		Operation:    testOperation,
	}
	testData := map[string]string{
		"data1": "value1",
		"data2": "value2",
	}
	testAction := prov.Action{
		ID:             "testActionID",
		ActionType:     "testActionType",
		AsyncOperation: testAsyncOperation,
		Data:           testData,
	}
	lof := events.LogOptionalFields{
		events.WorkFlowID:    "testWF",
		events.InterfaceName: "action",
		events.OperationName: "testOperationName",
	}
	ctx := events.NewContext(context.Background(), lof)

	return mock, client, plugin, testAction, lof, ctx

}
func TestActionOperatorExecAction(t *testing.T) {
	t.Parallel()
	mock, client, plugin, action, lof, ctx := setupExecActionTestEnv(t)
	defer client.Close()
	deregister, err := plugin.ExecAction(
		ctx,
		config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
		"testTaskID", "testDeploymentID", &action)
	require.NoError(t, err, "Failed to cal plugin ExecAction")
	require.True(t, mock.execactionCalled)
	require.Equal(t, "test", mock.conf.Consul.Address)
	require.Equal(t, "testdc", mock.conf.Consul.Datacenter)
	require.Equal(t, "testTaskID", mock.taskID)
	require.Equal(t, "testDeploymentID", mock.deploymentID)
	require.Equal(t, action, *mock.action)
	require.True(t, deregister, "Wrong return value for ExecAction")
	assert.Equal(t, lof, mock.lof)
}

func TestActionOperatorExecActionWrongContext(t *testing.T) {
	t.Parallel()
	_, client, plugin, action, _, _ := setupExecActionTestEnv(t)
	defer client.Close()
	var wrongContext context.Context
	_, err := plugin.ExecAction(
		wrongContext,
		config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
		"testTaskID", "testDeploymentID", &action)
	require.Error(t, err, "Expected an error calling ExecAction with wrong context")

}

func TestActionOperatorExecActionWithFailure(t *testing.T) {
	t.Parallel()
	mock, client, plugin, action, _, ctx := setupExecActionTestEnv(t)
	defer client.Close()
	deregister, err := plugin.ExecAction(
		ctx,
		config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
		"testTaskID", "TestFailure", &action)
	require.True(t, mock.execactionCalled)
	require.Error(t, err, "An error was expected during executing plugin operation")
	require.EqualError(t, err, "a failure occurred during plugin exec operation")
	require.False(t, deregister, "Wrong return value for ExecAction")
}

func TestActionOperatorExecActionWithCancel(t *testing.T) {
	t.Parallel()
	mock, client := createMockActionOperatorClient(t)
	defer client.Close()

	raw, err := client.Dispense(ActionPluginName)
	require.NoError(t, err, "Unexpected error calling client.Dispense() for Action Plugin")

	plugin := raw.(prov.ActionOperator)
	lof := events.LogOptionalFields{
		events.WorkFlowID:    "testWF",
		events.InterfaceName: "delegate",
		events.OperationName: "myTest",
	}
	ctx := events.NewContext(context.Background(), lof)
	ctx, cancelF := context.WithCancel(ctx)
	go func() {
		_, err := plugin.ExecAction(
			ctx,
			config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
			"testTaskID", "TestCancel", &prov.Action{})
		require.NoError(t, err)
	}()
	cancelF()
	// Wait for cancellation signal to be dispatched
	time.Sleep(50 * time.Millisecond)
	require.True(t, mock.contextCancelled, "Context not cancelled")
}

func TestActionOperatorGetActionTypes(t *testing.T) {
	mock := new(mockActionOperator)
	client, _ := plugin.TestPluginRPCConn(
		t,
		map[string]plugin.Plugin{
			ActionPluginName: &ActionPlugin{
				F: func() prov.ActionOperator {
					return mock
				},
				ActionTypes: []string{"TestActionType1", "TestActionType2"}},
		},
		nil)
	defer client.Close()
	raw, err := client.Dispense(ActionPluginName)
	require.NoError(t, err, "Unexpected error calling client.Dispense() for Action Plugin")
	plugin := raw.(ActionOperator)

	actionTypes, err := plugin.GetActionTypes()
	require.NoError(t, err, "Unexpected error calling plugin.GetActionTypes()")
	require.Len(t, actionTypes, 2)
	require.Contains(t, actionTypes, "TestActionType1")
	require.Contains(t, actionTypes, "TestActionType2")

}
