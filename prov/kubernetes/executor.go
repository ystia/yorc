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
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"novaforge.bull.com/starlings-janus/janus/tosca"
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

	instances, err := tasks.GetInstances(kv, taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}

	op := strings.ToLower(delegateOperation)
	switch {
	case op == "install":
		err = e.installNode(ctx, kv, cfg, deploymentID, nodeName, instances)
	case op == "uninstall":
		err = e.uninstallNode(ctx, kv, cfg, deploymentID, nodeName, instances)
	default:
		return errors.Errorf("Unsupported operation %q", delegateOperation)
	}

	return err
}

func (e *defaultExecutor) installNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, instances []string) error {
	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateCreating)
		if err != nil {
			return err
		}
	}

	err := e.initClientSet(cfg)
	if err != nil {
		return err
	}

	generator := NewGenerator(kv, cfg)

	namespace, err := getNamespace(kv,deploymentID, nodeName)
	if err != nil {
		return err
	}

	namespace = strings.ToLower(namespace)
	err = generator.CreateNamespaceIfMissing(deploymentID, namespace, e.clientset)
	if err != nil {
		return err
	}

	pod, err := generator.GeneratePod(deploymentID, nodeName)
	if err != nil {
		return err
	}

	_, err = e.clientset.CoreV1().Pods(namespace).Create(&pod)
	if err != nil {
		for _, instance := range instances {
			err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateError)
			if err != nil {
				return err
			}
		}
		return errors.Wrap(err, "Failed to create pod")
	}

	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateStarted)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *defaultExecutor) uninstallNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string, instances []string) error {
	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateDeleting)
		if err != nil {
			return err
		}
	}
	err := e.initClientSet(cfg)
	if err != nil {
		return err
	}

	namespace, err := getNamespace(kv,deploymentID, nodeName)
	if err != nil {
		return err
	}

	err = e.clientset.CoreV1().Pods(strings.ToLower(namespace)).Delete(strings.ToLower(cfg.ResourcesPrefix + nodeName), &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	for _, instance := range instances {
		err := deployments.SetInstanceState(kv, deploymentID, nodeName, instance, tosca.NodeStateDeleted)
		if err != nil {
			return err
		}
	}
	return nil
}

func getNamespace(kv *api.KV, deploymentID, nodeName string) (string, error) {
	found, namespace, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "namespace")
	if err != nil {
		return "", err
	}
	if !found || namespace == "" {
		return deployments.GetDeploymentTemplateName(kv, deploymentID)
	}
	return namespace, nil
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