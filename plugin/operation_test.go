package plugin

import (
	"context"
	"testing"
	"time"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/prov"
)

type mockOperationExecutor struct {
	execoperationCalled            bool
	ctx                            context.Context
	conf                           config.Configuration
	taskID, deploymentID, nodeName string
	operation                      prov.Operation
	contextCancelled               bool
}

func (m *mockOperationExecutor) ExecOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
	m.execoperationCalled = true
	m.ctx = ctx
	m.conf = conf
	m.taskID = taskID
	m.deploymentID = deploymentID
	m.nodeName = nodeName
	m.operation = operation

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

func TestOperationExecutorExecOperation(t *testing.T) {
	t.Parallel()
	mock := new(mockOperationExecutor)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		OperationPluginName: &OperationPlugin{F: func() prov.OperationExecutor {
			return mock
		}},
	})
	defer client.Close()

	raw, err := client.Dispense(OperationPluginName)
	require.Nil(t, err)

	plugin := raw.(prov.OperationExecutor)
	op := prov.Operation{
		Name: "myOps",
		ImplementationArtifact: "tosca.artifacts.Implementation.Bash",
		RelOp: prov.RelationshipOperation{
			IsRelationshipOperation: true,
			RequirementIndex:        "1",
			TargetNodeName:          "AnotherNode",
		},
	}
	err = plugin.ExecOperation(context.Background(), config.Configuration{ConsulAddress: "test", ConsulDatacenter: "testdc"}, "TestTaskID", "TestDepID", "TestNodeName", op)
	require.Nil(t, err)
	require.True(t, mock.execoperationCalled)
	require.Equal(t, "test", mock.conf.ConsulAddress)
	require.Equal(t, "testdc", mock.conf.ConsulDatacenter)
	require.Equal(t, "TestTaskID", mock.taskID)
	require.Equal(t, "TestDepID", mock.deploymentID)
	require.Equal(t, "TestNodeName", mock.nodeName)
	require.Equal(t, op, mock.operation)
}

func TestOperationExecutorExecOperationWithFailure(t *testing.T) {
	t.Parallel()
	mock := new(mockOperationExecutor)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		OperationPluginName: &OperationPlugin{F: func() prov.OperationExecutor {
			return mock
		}},
	})
	defer client.Close()

	raw, err := client.Dispense(OperationPluginName)
	require.Nil(t, err)

	plugin := raw.(prov.OperationExecutor)
	op := prov.Operation{
		Name: "myOps",
		ImplementationArtifact: "tosca.artifacts.Implementation.Bash",
		RelOp: prov.RelationshipOperation{
			IsRelationshipOperation: true,
			RequirementIndex:        "1",
			TargetNodeName:          "AnotherNode",
		},
	}
	err = plugin.ExecOperation(context.Background(), config.Configuration{ConsulAddress: "test", ConsulDatacenter: "testdc"}, "TestTaskID", "TestFailure", "TestNodeName", op)
	require.Error(t, err, "An error was expected during executing plugin operation")
}

func TestOperationExecutorExecOperationWithCancel(t *testing.T) {
	t.Parallel()
	mock := new(mockOperationExecutor)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		OperationPluginName: &OperationPlugin{F: func() prov.OperationExecutor {
			return mock
		}},
	})
	defer client.Close()

	raw, err := client.Dispense(OperationPluginName)
	require.Nil(t, err)

	plugin := raw.(prov.OperationExecutor)
	ctx := context.Background()
	ctx, cancelF := context.WithCancel(ctx)
	go func() {
		err = plugin.ExecOperation(ctx, config.Configuration{ConsulAddress: "test", ConsulDatacenter: "testdc"}, "TestTaskID", "TestCancel", "TestNodeName", prov.Operation{})
		require.Nil(t, err)
	}()
	cancelF()
	// Wait for cancellation signal to be dispatched
	time.Sleep(50 * time.Millisecond)
	require.True(t, mock.contextCancelled, "Context not cancelled")
}

func TestOperationGetSupportedArtifactTypes(t *testing.T) {
	mock := new(mockOperationExecutor)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		OperationPluginName: &OperationPlugin{
			F: func() prov.OperationExecutor {
				return mock
			},
			SupportedTypes: []string{"tosca.my.types", "test"}},
	})
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
