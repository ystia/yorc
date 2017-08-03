package kubernetes

import (
	"k8s.io/client-go/kubernetes"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/prov"

	"context"
	"novaforge.bull.com/starlings-janus/janus/helper/kubernetesutil"
)

type defaultExecutor struct {
	clientset *kubernetes.Clientset
}

// NewExecutor returns an Executor
func NewExecutor() prov.OperationExecutor {
	return &defaultExecutor{}
}

func (e *defaultExecutor) ExecOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
	consulClient, err := conf.GetConsulClient()
	if err != nil {
		return err
	}
	kv := consulClient.KV()
	exec, err := newExecution(kv, conf, taskID, deploymentID, nodeName, operation)
	if err != nil {
		return err
	}

	err = kubernetesutil.InitClientSet(conf)
	if err != nil {
		return err
	}

	e.clientset = kubernetesutil.GetClientSet()

	if err != nil {
		return err
	}

	new_ctx := context.WithValue(ctx, "clientset", e.clientset)

	return exec.execute(new_ctx)
}
