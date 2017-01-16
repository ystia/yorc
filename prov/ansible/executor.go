package ansible

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
)

type Executor interface {
	ExecOperation(ctx context.Context, deploymentID, nodeName, operation string, taskID ...string) error
}

type defaultExecutor struct {
	kv *api.KV
	r  *rand.Rand
}

func NewExecutor(kv *api.KV) Executor {
	return &defaultExecutor{kv: kv, r: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (e *defaultExecutor) ExecOperation(ctx context.Context, deploymentID, nodeName, operation string, taskID ...string) error {
	exec, err := newExecution(e.kv, deploymentID, nodeName, operation, taskID...)
	if err != nil {
		if IsOperationNotImplemented(err) {
			log.Printf("Voluntary bypassing error: %s. This is a deprecated feature please update your topology", err.Error())
			deployments.LogInConsul(e.kv, deploymentID, fmt.Sprintf("Voluntary bypassing error: %s. This is a deprecated feature please update your topology", err.Error()))
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
			log.Printf("Deployment: %q, Node: %q, Operation: %q: Caught a retriable error from Ansible: '%s'. Let's retry in few seconds (%d/%d)", deploymentID, nodeName, operation, err, i+1, reties)
			time.Sleep(time.Duration(e.r.Int63n(10)) * time.Second)
		} else {
			log.Printf("Deployment: %q, Node: %q, Operation: %q: Giving up retries for Ansible error: '%s' (%d/%d)", deploymentID, nodeName, operation, err, i+1, reties)
		}
	}

	return err
}
