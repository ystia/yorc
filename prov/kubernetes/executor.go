package kubernetes

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

type defaultExecutor struct {
	clientset *kubernetes.Clientset
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

	e.clientset, err = initClientSet(conf)
	if err != nil {
		return err
	}

	new_ctx := context.WithValue(ctx, "clientset", e.clientset)

	return exec.execute(new_ctx)
}

func initClientSet(cfg config.Configuration) (*kubernetes.Clientset, error) {
	kubConf := cfg.Infrastructures["kubernetes"]
	kubMasterIP := kubConf.GetString("master_url")

	if kubMasterIP == "" {
		return nil, errors.New(`Missing or invalid mandatory parameter master_url in the "kubernetes" infrastructure configuration`)
	}
	conf, err := clientcmd.BuildConfigFromFlags(kubMasterIP, "")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to build kubernetes config")
	}

	conf.TLSClientConfig.Insecure = kubConf.GetBool("insecure")
	conf.TLSClientConfig.CAFile = kubConf.GetString("ca_file")
	conf.TLSClientConfig.CertFile = kubConf.GetString("cert_file")
	conf.TLSClientConfig.KeyFile = kubConf.GetString("key_file")

	clientset, err := kubernetes.NewForConfig(conf)
	return clientset, errors.Wrap(err, "Failed to create kubernetes clientset from config")
}
