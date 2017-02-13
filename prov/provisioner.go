package prov

import (
	"context"

	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/config"
)

// DelegateExecutor is the interface that wraps the ExecDelegate method
//
// ExecDelegate executes the given delegateOperation for given nodeName on the given deploymentID.
// The taskID identifies the task that requested to execute this delegate operation.
// The given ctx may be used to check for cancellation, conf is the server Configuration.
type DelegateExecutor interface {
	ExecDelegate(ctx context.Context, kv *api.KV, conf config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error
}

// OperationExecutor is the interface that wraps the ExecOperation method
//
// ExecOperation executes the given TOSCA operation for given nodeName on the given deploymentID.
// The taskID identifies the task that requested to execute this operation.
// The given ctx may be used to check for cancellation, conf is the server Configuration.
type OperationExecutor interface {
	ExecOperation(ctx context.Context, kv *api.KV, conf config.Configuration, taskID, deploymentID, nodeName, operation string) error
}
