package kubernetes

import (
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/prov"

	"context"
)

type defaultExecutor struct {
	clientset *kubernetes.Clientset
}

// NewExecutor returns an Executor
func NewExecutor() prov.OperationExecutor {
	return &defaultExecutor{}
}

func (e *defaultExecutor) ExecOperation(ctx context.Context, kv *api.KV, cfg config.Configuration, taskID, deploymentID, nodeName, operation string) error {
	exec, err := newExecution(kv, cfg, taskID, deploymentID, nodeName, operation)
	if err != nil {
		return err
	}

	err = e.initClientSet(cfg)
	if err != nil {
		return err
	}

	new_ctx := context.WithValue(ctx, "clientset", e.clientset)

	return exec.execute(new_ctx)
}

func (e *defaultExecutor) initClientSet(cfg config.Configuration) error {
	var clientset *kubernetes.Clientset
	conf, err := clientcmd.BuildConfigFromFlags(cfg.KubeMasterIp, "")
	if err != nil {
		return errors.Wrap(err, "Failed to build kubernetes config")
	}
	clientset, err = kubernetes.NewForConfig(conf)
	if err != nil {
		return errors.Wrap(err, "Failed to create kubernetes clientset from config")
	}

	e.clientset = clientset
	return nil
}
