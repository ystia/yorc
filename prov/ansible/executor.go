package ansible

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

type defaultExecutor struct {
	r *rand.Rand
}

// NewExecutor returns an Executor
func NewExecutor() prov.OperationExecutor {
	return &defaultExecutor{r: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (e *defaultExecutor) ExecOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName, operation string) error {
	consulClient, err := conf.GetConsulClient()
	if err != nil {
		return err
	}
	kv := consulClient.KV()
	exec, err := newExecution(kv, conf, taskID, deploymentID, nodeName, operation)
	if err != nil {
		if IsOperationNotImplemented(err) {
			log.Printf("Voluntary bypassing error: %s. This is a deprecated feature please update your topology", err.Error())
			events.LogEngineMessage(kv, deploymentID, fmt.Sprintf("Voluntary bypassing error: %s. This is a deprecated feature please update your topology", err.Error()))
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
