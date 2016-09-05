package ansible

import (
	"context"
	"github.com/hashicorp/consul/api"
)

type Executor interface {
	ExecOperation(ctx context.Context, deploymentId, nodeName, operation string) error
}

type defaultExecutor struct {
	kv *api.KV
}

func NewExecutor(kv *api.KV) Executor {
	return &defaultExecutor{kv: kv}
}

func (e *defaultExecutor) ExecOperation(ctx context.Context, deploymentId, nodeName, operation string) error {
	exec, err := newExecution(e.kv, deploymentId, nodeName, operation)
	if err != nil {
		return err
	}
	return exec.execute(ctx)
}
