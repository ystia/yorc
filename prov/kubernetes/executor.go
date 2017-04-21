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
	"log"
	"k8s.io/client-go/pkg/api/v1"
	"time"
	"novaforge.bull.com/starlings-janus/janus/events"
)

type defaultExecutor struct {
	clientset *kubernetes.Clientset
}

// NewExecutor returns an Executor
func NewExecutor() prov.OperationExecutor {
	return &defaultExecutor{}
}

func (e *defaultExecutor) ExecOperation(ctx context.Context, kv *api.KV, cfg config.Configuration, taskID, deploymentID, nodeName, operation string) error {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(nodeType, "janus.nodes.KubernetesContainer") {
		return errors.Errorf("Unsupported node type '%s' for node '%s'", nodeType, nodeName)
	}

	if err != nil {
		return err
	}

	op := strings.ToLower(operation)
	switch {
	case op == "tosca.interfaces.node.lifecycle.standard.delete":
		log.Printf("Voluntary bypassing operation %s", operation)
		err = nil
	case op == "tosca.interfaces.node.lifecycle.standard.configure":
		err = e.installNode(ctx, kv, cfg, deploymentID, nodeName)
	case op == "tosca.interfaces.node.lifecycle.standard.start":
		err = e.checkNode(ctx, kv, cfg, deploymentID, nodeName)
	case op == "tosca.interfaces.node.lifecycle.standard.stop":
		err = e.uninstallNode(ctx, kv, cfg, deploymentID, nodeName)
	default:
		return errors.Errorf("Unsupported operation %q", operation)
	}

	return err
}

func (e *defaultExecutor) checkNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string) error {
	err := e.initClientSet(cfg)
	if err != nil {
		return err
	}

	namespace, err := getNamespace(kv,deploymentID, nodeName)
	if err != nil {
		return err
	}

	pod := &v1.Pod{}
	status := v1.PodUnknown
	latestReason := ""

	for status != v1.PodRunning && latestReason != "ErrImagePull" {
		pod, err = e.clientset.CoreV1().Pods(strings.ToLower(namespace)).Get(strings.ToLower(cfg.ResourcesPrefix + nodeName), metav1.GetOptions{})

		if err != nil {
			return errors.Wrap(err, "Failed to fetch pod")
		}

		status = pod.Status.Phase

		if status == v1.PodPending && len(pod.Status.ContainerStatuses) > 0{
			reason := pod.Status.ContainerStatuses[0].State.Waiting.Reason
			if reason != latestReason {
				latestReason = reason
				log.Printf(string(pod.Status.Phase) + "->" + reason)
				events.LogEngineMessage(kv, deploymentID, "Pod status : " + string(pod.Status.Phase)+" -> " + reason)
			}
		} else {
			log.Printf(string(pod.Status.Phase))
			events.LogEngineMessage(kv, deploymentID, "Pod status : " + string(pod.Status.Phase))
		}

		time.Sleep(2 * time.Second)
	}

	ready := true
	for _, condition := range pod.Status.Conditions {
		if condition.Status == v1.ConditionFalse {
			ready = false
		}
	}

	if !ready {
		reason := pod.Status.ContainerStatuses[0].State.Waiting.Reason
		message := pod.Status.ContainerStatuses[0].State.Waiting.Message

		if reason == "RunContainerError" {
			logs, err := e.clientset.CoreV1().Pods(strings.ToLower(namespace)).GetLogs(strings.ToLower(cfg.ResourcesPrefix + nodeName), &v1.PodLogOptions{}).Do().Raw()
			if err != nil {
				return errors.Wrap(err, "Failed to fetch pod logs")
			}
			podLogs := string(logs)
			return errors.Errorf("Pod failed to start reason : %s --- Message : %s --- Pod logs : %s", reason, message, podLogs)
		}

		return errors.Errorf("Pod failed to start reason : %s --- Message : %s", reason, message)
	}

	return nil
}

func (e *defaultExecutor) installNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string) error {
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
		return errors.Wrap(err, "Failed to create pod")
	}
	return nil
}

func (e *defaultExecutor) uninstallNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName string) error {
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