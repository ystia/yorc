package kubernetes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/prov/operations"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

// An EnvInput represent a TOSCA operation input
//
// This element is exported in order to be used by text.Template but should be consider as internal

type execution interface {
	execute(ctx context.Context) error
}

type executionScript struct {
	*executionCommon
}

type dockerConfigEntry struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
	Auth     string `json:"auth"`
}

type executionCommon struct {
	kv             *api.KV
	cfg            config.Configuration
	deploymentID   string
	taskID         string
	taskType       tasks.TaskType
	NodeName       string
	Operation      prov.Operation
	NodeType       string
	Description    string
	EnvInputs      []*operations.EnvInput
	VarInputsNames []string
	Repositories   map[string]string
	SecretRepoName string
}

func newExecution(kv *api.KV, cfg config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) (execution, error) {
	taskType, err := tasks.GetTaskType(kv, taskID)
	if err != nil {
		return nil, err
	}

	execCommon := &executionCommon{kv: kv,
		cfg:            cfg,
		deploymentID:   deploymentID,
		NodeName:       nodeName,
		Operation:      operation,
		VarInputsNames: make([]string, 0),
		EnvInputs:      make([]*operations.EnvInput, 0),
		taskID:         taskID,
		taskType:       taskType,
	}

	return execCommon, execCommon.resolveOperation()
}

func (e *executionCommon) resolveOperation() error {
	var err error
	e.NodeType, err = deployments.GetNodeType(e.kv, e.deploymentID, e.NodeName)
	return err
}

func (e *executionCommon) execute(ctx context.Context) (err error) {
	ctx = context.WithValue(ctx, "generator", NewGenerator(e.kv, e.cfg))
	instances, err := tasks.GetInstances(e.kv, e.taskID, e.deploymentID, e.NodeName)
	nbInstances := int32(len(instances))
	switch strings.ToLower(e.Operation.Name) {
	case "tosca.interfaces.node.lifecycle.standard.delete",
		"tosca.interfaces.node.lifecycle.standard.configure":
		log.Printf("Voluntary bypassing operation %s", e.Operation.Name)
		return nil
	case "tosca.interfaces.node.lifecycle.standard.start":
		if e.taskType == tasks.ScaleUp {
			log.Println("##### Scale up node !")
			err = e.scaleNode(ctx, tasks.ScaleUp, nbInstances)
		} else {
			log.Println("##### Deploy node !")
			err = e.deployNode(ctx, nbInstances)
		}
		if err != nil {
			return err
		}
		return e.checkNode(ctx)
	case "tosca.interfaces.node.lifecycle.standard.stop":
		if e.taskType == tasks.ScaleDown {
			log.Println("##### Scale down node !")
			return e.scaleNode(ctx, tasks.ScaleDown, nbInstances)
		}
		return e.uninstallNode(ctx)
	default:
		return errors.Errorf("Unsupported operation %q", e.Operation.Name)
	}

}

func (e *executionCommon) parseEnvInputs() []v1.EnvVar {
	var data []v1.EnvVar

	for _, val := range e.EnvInputs {
		tmp := v1.EnvVar{Name: val.Name, Value: val.Value}
		data = append(data, tmp)
	}

	return data
}

func (e *executionCommon) checkRepository(ctx context.Context) error {
	clientset := ctx.Value("clientset").(*kubernetes.Clientset)
	generator := ctx.Value("generator").(*K8sGenerator)

	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	repoName, err := deployments.GetOperationImplementationRepository(e.kv, e.deploymentID, e.NodeType, e.Operation.Name)
	if err != nil {
		return err
	}
	repoURL, err := deployments.GetRepositoryURLFromName(e.kv, e.deploymentID, repoName)
	if repoURL == deployments.DockerHubURL {
		return nil
	}

	//Generate a new secret
	var byteD []byte
	var dockercfgAuth dockerConfigEntry

	if tokenType, _ := deployments.GetRepositoryTokenTypeFromName(e.kv, e.deploymentID, repoName); tokenType == "password" {
		token, user, err := deployments.GetRepositoryTokenUserFromName(e.kv, e.deploymentID, repoName)
		if err != nil {
			return err
		}
		dockercfgAuth.Username = user
		dockercfgAuth.Password = token
		dockercfgAuth.Auth = base64.StdEncoding.EncodeToString([]byte(user + ":" + token))
		dockercfgAuth.Email = "test@test.com"
	}

	dockerCfg := map[string]dockerConfigEntry{repoURL: dockercfgAuth}

	repoName = strings.ToLower(repoName)
	byteD, err = json.Marshal(dockerCfg)
	if err != nil {
		return err
	}

	_, err = generator.CreateNewRepoSecret(clientset, namespace, repoName, byteD)
	e.SecretRepoName = repoName

	if err != nil {
		return err
	}

	return nil
}

func (e *executionCommon) scaleNode(ctx context.Context, scaleType tasks.TaskType, nbInstances int32) error {
	clientset := ctx.Value("clientset")

	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	deployment, err := (clientset.(*kubernetes.Clientset)).ExtensionsV1beta1().Deployments(namespace).Get(strings.ToLower(e.cfg.ResourcesPrefix+e.NodeName), metav1.GetOptions{})

	replica := *deployment.Spec.Replicas
	if scaleType == tasks.ScaleUp {
		replica = replica + nbInstances
	} else if scaleType == tasks.ScaleDown {
		replica = replica - nbInstances
	}

	deployment.Spec.Replicas = &replica
	_, err = (clientset.(*kubernetes.Clientset)).ExtensionsV1beta1().Deployments(namespace).Update(deployment)
	if err != nil {
		return errors.Wrap(err, "Failed to scale deployment")
	}

	return nil
}

func (e *executionCommon) deployNode(ctx context.Context, nbInstances int32) error {
	clientset := ctx.Value("clientset")
	generator := ctx.Value("generator").(*K8sGenerator)

	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	err = generator.CreateNamespaceIfMissing(e.deploymentID, namespace, clientset.(*kubernetes.Clientset))
	if err != nil {
		return err
	}

	err = e.checkRepository(ctx)
	if err != nil {
		return err
	}

	e.EnvInputs, e.VarInputsNames, err = operations.ResolveInputs(e.kv, e.deploymentID, e.NodeName, e.taskID, e.Operation)
	if err != nil {
		return err
	}
	inputs := e.parseEnvInputs()

	deployment, service, err := generator.GenerateDeployment(e.deploymentID, e.NodeName, e.Operation.Name, e.NodeType, e.SecretRepoName, inputs, nbInstances)
	if err != nil {
		return err
	}

	_, err = (clientset.(*kubernetes.Clientset)).ExtensionsV1beta1().Deployments(namespace).Create(&deployment)

	if err != nil {
		return errors.Wrap(err, "Failed to create deployment")
	}

	if service.Name != "" {
		serv, err := (clientset.(*kubernetes.Clientset)).CoreV1().Services(namespace).Create(&service)
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
		for _, val := range serv.Spec.Ports {
			log.Printf("%s : %s: %d:%d mapped to %d", serv.Name, val.Name, val.Port, val.TargetPort.IntVal, val.NodePort)
		}

		// Legacy
		err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.NodeName, "ip_address", service.Name)
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}

		err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.NodeName, "k8s_service_name", service.Name)
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
		// TODO check that it is a good idea to use it as endpoint ip_address
		err = deployments.SetCapabilityAttributeForAllInstances(e.kv, e.deploymentID, e.NodeName, "endpoint", "ip_address", service.Name)
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
	}

	return nil
}

func (e *executionCommon) checkNode(ctx context.Context) error {
	clientset := ctx.Value("clientset")

	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	deploymentReady := false
	var available int32 = -1

	for !deploymentReady {
		deployment, err := (clientset.(*kubernetes.Clientset)).ExtensionsV1beta1().Deployments(namespace).Get(strings.ToLower(e.cfg.ResourcesPrefix+e.NodeName), metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "Failed fetch deployment")
		}
		if available != deployment.Status.AvailableReplicas {
			available = deployment.Status.AvailableReplicas
			log.Printf("Deployment %s : %d pod available of %d", e.NodeName, available, *deployment.Spec.Replicas)
		}

		if deployment.Status.AvailableReplicas == *deployment.Spec.Replicas {
			deploymentReady = true
		} else {
			selector := ""
			for key, val := range deployment.Spec.Selector.MatchLabels {
				if selector != "" {
					selector += ","
				}
				selector += key + "=" + val
			}
			//log.Printf("selector: %s", selector)
			pods, _ := (clientset.(*kubernetes.Clientset)).CoreV1().Pods(namespace).List(
				metav1.ListOptions{
					LabelSelector: selector,
				})

			// We should always have only 1 pod (as the Replica is set to 1)
			for _, podItem := range pods.Items {

				//log.Printf("Check pod %s", podItem.Name)
				err := e.checkPod(ctx, podItem.Name)
				if err != nil {
					return err
				}

			}
		}

		time.Sleep(2 * time.Second)
	}

	return nil
}

func (e *executionCommon) checkPod(ctx context.Context, podName string) error {
	clientset := ctx.Value("clientset")

	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	status := v1.PodUnknown
	latestReason := ""

	for status != v1.PodRunning && latestReason != "ErrImagePull" && latestReason != "InvalidImageName" {
		pod, err := (clientset.(*kubernetes.Clientset)).CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})

		if err != nil {
			return errors.Wrap(err, "Failed to fetch pod")
		}

		status = pod.Status.Phase

		if status == v1.PodPending && len(pod.Status.ContainerStatuses) > 0 {
			reason := pod.Status.ContainerStatuses[0].State.Waiting.Reason
			if reason != latestReason {
				latestReason = reason
				log.Printf(pod.Name + " : " + string(pod.Status.Phase) + "->" + reason)
				events.LogEngineMessage(e.kv, e.deploymentID, "Pod status : "+pod.Name+" : "+string(pod.Status.Phase)+" -> "+reason)
			}
		} else {
			ready := true
			cond := v1.PodCondition{}
			for _, condition := range pod.Status.Conditions {
				if condition.Status == v1.ConditionFalse {
					ready = false
					cond = condition
				}
			}

			if !ready {
				state := ""
				reason := ""
				message := "running"

				if pod.Status.ContainerStatuses[0].State.Waiting != nil {
					state = "waiting"
					reason = pod.Status.ContainerStatuses[0].State.Waiting.Reason
					message = pod.Status.ContainerStatuses[0].State.Waiting.Message
				} else if pod.Status.ContainerStatuses[0].State.Terminated != nil {
					state = "terminated"
					reason = pod.Status.ContainerStatuses[0].State.Terminated.Reason
					message = pod.Status.ContainerStatuses[0].State.Terminated.Message
				}

				log.Printf(pod.Name + " : " + string(pod.Status.Phase))
				events.LogEngineMessage(e.kv, e.deploymentID, "Pod status : "+pod.Name+" : "+string(pod.Status.Phase)+" ("+state+")")

				if reason == "RunContainerError" {
					logs, err := (clientset.(*kubernetes.Clientset)).CoreV1().Pods(namespace).GetLogs(strings.ToLower(e.cfg.ResourcesPrefix+e.NodeName), &v1.PodLogOptions{}).Do().Raw()
					if err != nil {
						return errors.Wrap(err, "Failed to fetch pod logs")
					}
					podLogs := string(logs)
					log.Printf("Pod failed to start reason : %s --- Message : %s --- Pod logs : %s", reason, message, podLogs)
				}

				log.Printf("Pod failed to start reason : %s --- Message : %s -- condition : %s", reason, message, cond.Message)
			}
		}

		if status != v1.PodRunning {
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

func (e *executionCommon) uninstallNode(ctx context.Context) error {
	clientset := ctx.Value("clientset")

	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}
	deployment, err := (clientset.(*kubernetes.Clientset)).ExtensionsV1beta1().Deployments(namespace).Get(strings.ToLower(e.cfg.ResourcesPrefix+e.NodeName), metav1.GetOptions{})

	replica := int32(0)
	deployment.Spec.Replicas = &replica
	_, err = (clientset.(*kubernetes.Clientset)).ExtensionsV1beta1().Deployments(namespace).Update(deployment)

	err = (clientset.(*kubernetes.Clientset)).ExtensionsV1beta1().Deployments(namespace).Delete(strings.ToLower(e.cfg.ResourcesPrefix+e.NodeName), &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to delete deployment")
	}

	err = (clientset.(*kubernetes.Clientset)).CoreV1().Services(namespace).Delete(strings.ToLower(GeneratePodName(e.cfg.ResourcesPrefix+e.NodeName)), &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to delete service")
	}

	return nil
}

func getNamespace(kv *api.KV, deploymentID, nodeName string) (string, error) {
	return strings.ToLower(deploymentID), nil
}
