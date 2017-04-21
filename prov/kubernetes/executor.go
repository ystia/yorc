package kubernetes

import (
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/prov"

	"context"
	"strings"
)

type defaultExecutor struct {
	clientset *kubernetes.Clientset
}

// NewExecutor returns an Executor
func NewExecutor() prov.DelegateExecutor {
	return &defaultExecutor{}
}

func (e *defaultExecutor) ExecDelegate(ctx context.Context, kv *api.KV, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(nodeType, "janus.nodes.KubernetesContainer") {
		return errors.Errorf("Unsupported node type '%s' for node '%s'", nodeType, nodeName)
	}

	op := strings.ToLower(delegateOperation)
	switch {
	case op == "install":
		err = e.installNode(ctx, kv, cfg, deploymentID, nodeName)
	case op == "uninstall":
		err = e.uninstallNode(ctx, kv, cfg, deploymentID, nodeName)
	default:
		return errors.Errorf("Unsupported operation %q", delegateOperation)
	}

	return err
}

func (e *defaultExecutor) installNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string) error {

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
	generator := NewGenerator(kv, cfg)

	namespace, err := deployments.GetDeploymentTemplateName(kv, deploymentID)
	if err != nil {
		return err
	}

	namespace = strings.ToLower(namespace)
	err = generator.CreateNamespaceIfMissing(deploymentID, namespace, clientset)
	if err != nil {
		return err
	}

	pod, err := generator.GeneratePod(deploymentID, nodeName)
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Pods(namespace).Create(&pod)
	return errors.Wrap(err, "Failed to create pod")
}

func (e *defaultExecutor) uninstallNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string) error {
	found, namespace, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "namespace")
	if err != nil {
		return err
	}
	if !found || namespace == "" {
		namespace = "default"
	}

	err = e.clientset.CoreV1().Pods(namespace).Delete(strings.ToLower(nodeName), &metav1.DeleteOptions{})
	return err
}
