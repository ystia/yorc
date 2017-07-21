package kubernetesutil

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"novaforge.bull.com/starlings-janus/janus/config"
)

var clientset *kubernetes.Clientset

func InitClientSet(cfg config.Configuration) error {
	conf, err := clientcmd.BuildConfigFromFlags(cfg.KubemasterIp, "")
	if err != nil {
		return errors.Wrap(err, "Failed to build kubernetes config")
	}
	clientset, err = kubernetes.NewForConfig(conf)
	if err != nil {
		return errors.Wrap(err, "Failed to create kubernetes clientset from config")
	}

	return nil
}

func GetClientSet() *kubernetes.Clientset {
	return clientset
}
