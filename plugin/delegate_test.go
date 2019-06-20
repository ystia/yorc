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

type mockDelegateExecutor struct {
	execDelegateCalled                                bool
	ctx                                               context.Context
	conf                                              config.Configuration
	taskID, deploymentID, nodeName, delegateOperation string
	contextCancelled                                  bool
	lof                                               events.LogOptionalFields
}

func (m *mockDelegateExecutor) ExecDelegate(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	m.execDelegateCalled = true
	m.ctx = ctx
	m.conf = conf
	m.taskID = taskID
	m.deploymentID = deploymentID
	m.nodeName = nodeName
	m.delegateOperation = delegateOperation
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

func createMockDelegateExecutorClient(t *testing.T) (*mockDelegateExecutor, *plugin.RPCClient) {
	mock := new(mockDelegateExecutor)
	client, _ := plugin.TestPluginRPCConn(
		t,
		map[string]plugin.Plugin{
			DelegatePluginName: &DelegatePlugin{F: func() prov.DelegateExecutor {
				return mock
			}},
		},
		nil)
	return mock, client
}

func setupExecDelegateTestEnv(t *testing.T) (*mockDelegateExecutor, *plugin.RPCClient,
	prov.DelegateExecutor, events.LogOptionalFields, context.Context) {

	mock, client := createMockDelegateExecutorClient(t)
	raw, err := client.Dispense(DelegatePluginName)
	require.Nil(t, err)

	delegate := raw.(prov.DelegateExecutor)

	lof := events.LogOptionalFields{
		events.WorkFlowID:    "testWF",
		events.InterfaceName: "delegate",
		events.OperationName: "myTest",
	}
	ctx := events.NewContext(context.Background(), lof)

	return mock, client, delegate, lof, ctx
}

func TestDelegateExecutorExecDelegate(t *testing.T) {
	t.Parallel()
	mock, client, delegate, lof, ctx := setupExecDelegateTestEnv(t)
	defer client.Close()
	err := delegate.ExecDelegate(
		ctx,
		config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
		"TestTaskID", "TestDepID", "TestNodeName", "TestDelegateOP")
	require.Nil(t, err)
	require.True(t, mock.execDelegateCalled)
	require.Equal(t, "test", mock.conf.Consul.Address)
	require.Equal(t, "testdc", mock.conf.Consul.Datacenter)
	require.Equal(t, "TestTaskID", mock.taskID)
	require.Equal(t, "TestDepID", mock.deploymentID)
	require.Equal(t, "TestNodeName", mock.nodeName)
	require.Equal(t, "TestDelegateOP", mock.delegateOperation)
	assert.Equal(t, lof, mock.lof)
}

func TestDelegateExecutorExecDelegateWithFailure(t *testing.T) {
	t.Parallel()
	_, client, delegate, _, ctx := setupExecDelegateTestEnv(t)
	defer client.Close()
	err := delegate.ExecDelegate(
		ctx,
		config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
		"TestTaskID", "TestFailure", "TestNodeName", "TestDelegateOP")
	require.Error(t, err, "An error was expected during executing plugin operation")
	require.EqualError(t, err, "a failure occurred during plugin exec operation")
}

func TestDelegateExecutorExecDelegateWithCancel(t *testing.T) {
	t.Parallel()
	mock, client := createMockDelegateExecutorClient(t)
	defer client.Close()

	raw, err := client.Dispense(DelegatePluginName)
	require.Nil(t, err)

	delegate := raw.(prov.DelegateExecutor)

	lof := events.LogOptionalFields{
		events.WorkFlowID:    "testWF",
		events.InterfaceName: "delegate",
		events.OperationName: "myTest",
	}
	ctx := events.NewContext(context.Background(), lof)
	ctx, cancelF := context.WithCancel(ctx)
	go func() {
		err = delegate.ExecDelegate(
			ctx,
			config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}},
			"TestTaskID", "TestCancel", "TestNodeName", "TestDelegateOP")
		require.Nil(t, err)
	}()
	cancelF()
	// Wait for cancellation signal to be dispatched
	time.Sleep(50 * time.Millisecond)
	require.True(t, mock.contextCancelled, "Context not cancelled")
}

func TestDelegateGetSupportedTypes(t *testing.T) {
	mock := new(mockDelegateExecutor)
	client, _ := plugin.TestPluginRPCConn(
		t,
		map[string]plugin.Plugin{
			DelegatePluginName: &DelegatePlugin{
				F: func() prov.DelegateExecutor {
					return mock
				},
				SupportedTypes: []string{"tosca.my.types", "test"}},
		},
		nil)
	defer client.Close()
	raw, err := client.Dispense(DelegatePluginName)
	require.Nil(t, err)
	delegateExec := raw.(DelegateExecutor)

	supportedTypes, err := delegateExec.GetSupportedTypes()
	require.Nil(t, err)
	require.Len(t, supportedTypes, 2)
	require.Contains(t, supportedTypes, "tosca.my.types")
	require.Contains(t, supportedTypes, "test")

}
