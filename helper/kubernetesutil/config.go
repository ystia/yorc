package kubernetesutil

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"novaforge.bull.com/starlings-janus/janus/config"
)

var clientset *kubernetes.Clientset

func InitClientSet(cfg config.Configuration) error {
	kubMasterIP := cfg.Infrastructures["kubernetes"].GetString("kube_ip")
	if kubMasterIP == "" {
		return errors.New(`Missing or invalid mandatory parameter kube_ip in the "kubernetes" infrastructure configuration`)
	}
	conf, err := clientcmd.BuildConfigFromFlags(kubMasterIP, "")
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
