package plugin

import (
	"context"
	"testing"
	"time"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

type mockDelegateExecutor struct {
	execDelegateCalled                                bool
	ctx                                               context.Context
	conf                                              config.Configuration
	taskID, deploymentID, nodeName, delegateOperation string
	contextCancelled                                  bool
}

func (m *mockDelegateExecutor) ExecDelegate(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	m.execDelegateCalled = true
	m.ctx = ctx
	m.conf = conf
	m.taskID = taskID
	m.deploymentID = deploymentID
	m.nodeName = nodeName
	m.delegateOperation = delegateOperation

	go func() {
		<-m.ctx.Done()
		m.contextCancelled = true
	}()
	if m.deploymentID == "TestCancel" {
		<-m.ctx.Done()
	}
	return nil
}

func TestDelegateExecutorExecDelegate(t *testing.T) {
	t.Parallel()
	mock := new(mockDelegateExecutor)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		DelegatePluginName: &DelegatePlugin{F: func() prov.DelegateExecutor {
			return mock
		}},
	})
	defer client.Close()

	raw, err := client.Dispense(DelegatePluginName)
	require.Nil(t, err)

	delegate := raw.(prov.DelegateExecutor)

	err = delegate.ExecDelegate(context.Background(), config.Configuration{ConsulAddress: "test", ConsulDatacenter: "testdc"}, "TestTaskID", "TestDepID", "TestNodeName", "TestDelegateOP")
	require.Nil(t, err)
	require.True(t, mock.execDelegateCalled)
	require.Equal(t, "test", mock.conf.ConsulAddress)
	require.Equal(t, "testdc", mock.conf.ConsulDatacenter)
	require.Equal(t, "TestTaskID", mock.taskID)
	require.Equal(t, "TestDepID", mock.deploymentID)
	require.Equal(t, "TestNodeName", mock.nodeName)
	require.Equal(t, "TestDelegateOP", mock.delegateOperation)
}

func TestDelegateExecutorExecDelegateWithCancel(t *testing.T) {
	t.Parallel()
	mock := new(mockDelegateExecutor)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		DelegatePluginName: &DelegatePlugin{F: func() prov.DelegateExecutor {
			return mock
		}},
	})
	defer client.Close()

	raw, err := client.Dispense(DelegatePluginName)
	require.Nil(t, err)

	delegate := raw.(prov.DelegateExecutor)
	ctx := context.Background()
	ctx, cancelF := context.WithCancel(ctx)
	go func() {
		err = delegate.ExecDelegate(ctx, config.Configuration{ConsulAddress: "test", ConsulDatacenter: "testdc"}, "TestTaskID", "TestCancel", "TestNodeName", "TestDelegateOP")
		require.Nil(t, err)
	}()
	cancelF()
	// Wait for cancellation signal to be dispatched
	time.Sleep(50 * time.Millisecond)
	require.True(t, mock.contextCancelled, "Context not cancelled")
}

func TestDelegateGetSupportedTypes(t *testing.T) {
	mock := new(mockDelegateExecutor)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		DelegatePluginName: &DelegatePlugin{
			F: func() prov.DelegateExecutor {
				return mock
			},
			SupportedTypes: []string{"tosca.my.types", "test"}},
	})
	defer client.Close()
	raw, err := client.Dispense(DelegatePluginName)
	require.Nil(t, err)
	delagateExec := raw.(DelegateExecutor)

	supportedTypes, err := delagateExec.GetSupportedTypes()
	require.Nil(t, err)
	require.Len(t, supportedTypes, 2)
	require.Contains(t, supportedTypes, "tosca.my.types")
	require.Contains(t, supportedTypes, "test")

}
