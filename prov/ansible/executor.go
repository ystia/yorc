package ansible

import (
	"context"
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/log"
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
		if IsOperationNotImplemented(err) {
			log.Printf("Volontary bypassing error: %s. This is a deprecated feature please update your topology", err.Error())
			// TODO log it in deployment logs
			return nil
		}
		return err
	}
	return exec.execute(ctx)
}
