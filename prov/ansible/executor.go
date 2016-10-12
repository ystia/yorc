package ansible

import (
	"context"
	"github.com/hashicorp/consul/api"
	"math/rand"
	"novaforge.bull.com/starlings-janus/janus/log"
	"time"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"fmt"
)

type Executor interface {
	ExecOperation(ctx context.Context, deploymentId, nodeName, operation string) error
}

type defaultExecutor struct {
	kv *api.KV
	r  *rand.Rand
}

func NewExecutor(kv *api.KV) Executor {
	return &defaultExecutor{kv: kv, r: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (e *defaultExecutor) ExecOperation(ctx context.Context, deploymentId, nodeName, operation string) error {
	exec, err := newExecution(e.kv, deploymentId, nodeName, operation)
	if err != nil {
		if IsOperationNotImplemented(err) {
			log.Printf("Volontary bypassing error: %s. This is a deprecated feature please update your topology", err.Error())
			deployments.LogInConsul(e.kv, deploymentId, fmt.Sprintf("Volontary bypassing error: %s. This is a deprecated feature please update your topology", err.Error()))
			return nil
		}
		return err
	}
	reties := 5
	for i := 0; i < reties; i++ {
		err = exec.execute(ctx, i != 0)
		if err == nil {
			return nil
		}
		if !IsRetriable(err) {
			return err
		}
		if i < reties {
			log.Printf("Deployment: %q, Node: %q, Operation: %q: Caught a retriable error from Ansible: '%s'. Let's retry in few seconds (%d/%d)", deploymentId, nodeName, operation, err, i+1, reties)
			time.Sleep(time.Duration(e.r.Int63n(10)) * time.Second)
		} else {
			log.Printf("Deployment: %q, Node: %q, Operation: %q: Giving up retries for Ansible error: '%s' (%d/%d)", deploymentId, nodeName, operation, err, i+1, reties)
		}
	}

	return err
}
