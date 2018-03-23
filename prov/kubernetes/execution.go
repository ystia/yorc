// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubernetes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/prov/operations"
	"github.com/ystia/yorc/tasks"
)

const deploymentResourceType string = "yorc.nodes.kubernetes.api.types.DeploymentResource"
const serviceResourceType string = "yorc.nodes.kubernetes.api.types.ServiceResource"

type k8sResourceOperation int

const (
	k8sCreateOperation k8sResourceOperation = iota
	k8sDeleteOperation
)

type dockerConfigEntry struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
	Auth     string `json:"auth"`
}

type execution struct {
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
	clientSet      *kubernetes.Clientset
	generator      *k8sGenerator
}

func newExecution(kv *api.KV, cfg config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation, clientSet *kubernetes.Clientset) (*execution, error) {
	taskType, err := tasks.GetTaskType(kv, taskID)
	if err != nil {
		return nil, err
	}

	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}

	exec := &execution{kv: kv,
		cfg:            cfg,
		deploymentID:   deploymentID,
		NodeName:       nodeName,
		Operation:      operation,
		VarInputsNames: make([]string, 0),
		EnvInputs:      make([]*operations.EnvInput, 0),
		taskID:         taskID,
		taskType:       taskType,
		NodeType:       nodeType,
		clientSet:      clientSet,
		generator:      newGenerator(kv, cfg),
	}

	return exec, nil
}

func (e *execution) execute(ctx context.Context) error {
	instances, err := tasks.GetInstances(e.kv, e.taskID, e.deploymentID, e.NodeName)
	if err != nil {
		return nil
	}
	nbInstances := int32(len(instances))

	switch strings.ToLower(e.Operation.Name) {
	case "standard.create":
		return e.manageKubernetesResource(ctx, k8sCreateOperation)
	case "standard.configure":
		log.Printf("Voluntary bypassing operation %s", e.Operation.Name)
		return nil
	case "standard.start":
		if e.taskType == tasks.ScaleOut {
			log.Println("##### Scale up node !")
			err = e.scaleNode(ctx, tasks.ScaleOut, nbInstances)
		} else {
			log.Println("##### Deploy node !")
			err = e.deployNode(ctx, nbInstances)
		}
		if err != nil {
			return err
		}
		return e.checkNode(ctx)
	case "standard.stop":
		if e.taskType == tasks.ScaleIn {
			log.Println("##### Scale down node !")
			return e.scaleNode(ctx, tasks.ScaleIn, nbInstances)
		}
		return e.uninstallNode(ctx)
	case "standard.delete":
		return e.manageKubernetesResource(ctx, k8sDeleteOperation)
	default:
		return errors.Errorf("Unsupported operation %q", e.Operation.Name)
	}
}

func (e *execution) manageKubernetesResource(ctx context.Context, op k8sResourceOperation) error {
	_, rSpec, err := deployments.GetNodeProperty(e.kv, e.deploymentID, e.NodeName, "resource_spec")
	if err != nil {
		return err
	}
	switch e.NodeType {
	case deploymentResourceType:
		return e.manageDeploymentResource(ctx, op, rSpec)
	case serviceResourceType:
		return e.manageServiceResource(ctx, op, rSpec)
	default:
		return errors.Errorf("Unsupported k8s resource type %q", e.NodeType)
	}
}

func (e *execution) manageDeploymentResource(ctx context.Context, operationType k8sResourceOperation, rSpec string) (err error) {
	var deploymentRepr v1beta1.Deployment
	if rSpec == "" {
		return errors.Errorf("Missing mandatory resource_spec property for node %s", e.NodeName)
	}
	// Unmarshal JSON to k8s data structs
	if err = json.Unmarshal([]byte(rSpec), &deploymentRepr); err != nil {
		log.Printf("Try to manage k8S resource %s", rSpec)
		return errors.Errorf("The resource-spec JSON unmarshaling failed: %s", err)
	}

	// TODO manage Namespace creation
	// Get it from matadata, or generate it using deploymentID
	// (Synchronize with Alien)
	//namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	//namespace := deploymentRepresentation.ObjectMeta.Namespace
	namespace := "default"

	switch operationType {
	case k8sCreateOperation:
		// Create Deployment k8s resource
		deployment, err := e.clientSet.ExtensionsV1beta1().Deployments(namespace).Create(&deploymentRepr)
		if err != nil {
			return err
		}
		log.Printf("k8s Deployment %s created", deployment.Name)
	case k8sDeleteOperation:
		// Delete Deployment k8s resource
		var deploymentName string
		deploymentName = deploymentRepr.Name
		log.Printf("Delete k8s Deployment %s", deploymentName)

		err = e.clientSet.ExtensionsV1beta1().Deployments(namespace).Delete(deploymentName, nil)
		if err != nil {
			return err
		}
		log.Printf("k8s Deployment %s deleted", deploymentName)
	default:
		return errors.Errorf("Unsupported operation on k8s resource")
	}

	return nil
}

func (e *execution) manageServiceResource(ctx context.Context, operationType k8sResourceOperation, rSpec string) (err error) {

	var serviceRepr v1.Service
	if rSpec == "" {
		return errors.Errorf("Missing mandatory resource_spec property for node %s", e.NodeName)
	}
	// Unmarshal JSON to k8s data structs
	if err = json.Unmarshal([]byte(rSpec), &serviceRepr); err != nil {
		log.Printf("Try to manage k8s resource %s", rSpec)
		return errors.Errorf("The resource-spec JSON unmarshaling failed: %s", err)
	}

	namespace := "default"

	switch operationType {
	case k8sCreateOperation:
		// Create Service k8s resource
		service, err := e.clientSet.CoreV1().Services(namespace).Create(&serviceRepr)
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}

		log.Printf("k8s Service %s created", service.Name)
	case k8sDeleteOperation:
		// Delete Deployment k8s resource
		var serviceName string
		serviceName = serviceRepr.Name
		log.Printf("Delete k8s Service %s", serviceName)

		err = e.clientSet.CoreV1().Services(namespace).Delete(serviceName, nil)
		if err != nil {
			return errors.Wrap(err, "Failed to delete service")
		}

		log.Printf("k8s Service %s deleted", serviceName)
	default:
		return errors.Errorf("Unsupported operation on k8s resource")
	}
	return nil
}

func (e *execution) parseEnvInputs() []v1.EnvVar {
	var data []v1.EnvVar

	for _, val := range e.EnvInputs {
		tmp := v1.EnvVar{Name: val.Name, Value: val.Value}
		data = append(data, tmp)
	}

	return data
}

func (e *execution) checkRepository(ctx context.Context) error {

	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	repoName, err := deployments.GetOperationImplementationRepository(e.kv, e.deploymentID, e.NodeType, e.Operation.Name)
	if err != nil {
		return err
	}

	secret, err := e.clientSet.CoreV1().Secrets(namespace).Get(repoName, metav1.GetOptions{})
	if err == nil && secret.Name == repoName {
		return nil
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

	_, err = e.generator.createNewRepoSecret(e.clientSet, namespace, repoName, byteD)
	e.SecretRepoName = repoName

	if err != nil {
		return err
	}

	return nil
}

func (e *execution) scaleNode(ctx context.Context, scaleType tasks.TaskType, nbInstances int32) error {

	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	deployment, err := e.clientSet.ExtensionsV1beta1().Deployments(namespace).Get(strings.ToLower(e.cfg.ResourcesPrefix+e.NodeName), metav1.GetOptions{})

	replica := *deployment.Spec.Replicas
	if scaleType == tasks.ScaleOut {
		replica = replica + nbInstances
	} else if scaleType == tasks.ScaleIn {
		replica = replica - nbInstances
	}

	deployment.Spec.Replicas = &replica
	_, err = e.clientSet.ExtensionsV1beta1().Deployments(namespace).Update(deployment)
	if err != nil {
		return errors.Wrap(err, "Failed to scale deployment")
	}

	return nil
}

func (e *execution) deployNode(ctx context.Context, nbInstances int32) error {

	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	err = e.generator.createNamespaceIfMissing(e.deploymentID, namespace, e.clientSet)
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

	deployment, service, err := e.generator.generateDeployment(e.deploymentID, e.NodeName, e.Operation.Name, e.NodeType, e.SecretRepoName, inputs, nbInstances)
	if err != nil {
		return err
	}

	_, err = e.clientSet.ExtensionsV1beta1().Deployments(namespace).Create(&deployment)

	if err != nil {
		return errors.Wrap(err, "Failed to create deployment")
	}

	if service.Name != "" {
		serv, err := e.clientSet.CoreV1().Services(namespace).Create(&service)
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
		var s string
		for _, val := range serv.Spec.Ports {
			kubConf := e.cfg.Infrastructures["kubernetes"]
			kubMasterIP := kubConf.GetString("master_url")
			u, _ := url.Parse(kubMasterIP)
			h := strings.Split(u.Host, ":")
			str := fmt.Sprintf("http://%s:%d", h[0], val.NodePort)

			log.Printf("%s : %s: %d:%d mapped to %s", serv.Name, val.Name, val.Port, val.TargetPort.IntVal, str)

			s = fmt.Sprintf("%s %d ==> %s \n", s, val.Port, str)
		}
		err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.NodeName, "k8s_service_url", s)
		if err != nil {
			return errors.Wrap(err, "Failed to set attribute")
		}

		// Legacy
		err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.NodeName, "ip_address", service.Name)
		if err != nil {
			return errors.Wrap(err, "Failed to set attribute")
		}

		err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.NodeName, "k8s_service_name", service.Name)
		if err != nil {
			return errors.Wrap(err, "Failed to set attribute")
		}
		// TODO check that it is a good idea to use it as endpoint ip_address
		err = deployments.SetCapabilityAttributeForAllInstances(e.kv, e.deploymentID, e.NodeName, "endpoint", "ip_address", service.Name)
		if err != nil {
			return errors.Wrap(err, "Failed to set capability attribute")
		}
	}

	// TODO this is very bad but we need to add a hook in order to undeploy our pods we the tosca node stops
	// It will be better if we have a Kubernetes node type with a default stop implementation that will be inherited by
	// sub components.
	// So let's add an implementation of the stop operation in the node type
	return e.setUnDeployHook()
}

func (e *execution) setUnDeployHook() error {
	_, err := deployments.GetNodeTypeImplementingAnOperation(e.kv, e.deploymentID, e.NodeName, "tosca.interfaces.node.lifecycle.standard.stop")
	if err != nil {
		if !deployments.IsOperationNotImplemented(err) {
			return err
		}
		// Set an implementation type
		// TODO this works as long as Alien still add workflow steps for those operations even if there is no real operation defined
		// As this sounds like a hack we do not create a method in the deployments package to do it
		_, errGrp, store := consulutil.WithContext(context.Background())
		opPath := path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "topology/types", e.NodeType, "interfaces/standard/stop")
		store.StoreConsulKeyAsString(path.Join(opPath, "name"), "stop")
		store.StoreConsulKeyAsString(path.Join(opPath, "implementation/type"), kubernetesArtifactImplementation)
		store.StoreConsulKeyAsString(path.Join(opPath, "implementation/description"), "Auto-generated operation")
		return errGrp.Wait()
	}
	// There is already a custom stop operation defined in the type hierarchy let's use it
	return nil
}

func (e *execution) checkNode(ctx context.Context) error {

	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	deploymentReady := false
	var available int32 = -1

	for !deploymentReady {
		deployment, err := e.clientSet.ExtensionsV1beta1().Deployments(namespace).Get(strings.ToLower(e.cfg.ResourcesPrefix+e.NodeName), metav1.GetOptions{})
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
			pods, _ := e.clientSet.CoreV1().Pods(namespace).List(
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

func (e *execution) checkPod(ctx context.Context, podName string) error {

	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	status := v1.PodUnknown
	latestReason := ""

	for status != v1.PodRunning && latestReason != "ErrImagePull" && latestReason != "InvalidImageName" {
		pod, err := e.clientSet.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})

		if err != nil {
			return errors.Wrap(err, "Failed to fetch pod")
		}

		status = pod.Status.Phase

		if status == v1.PodPending && len(pod.Status.ContainerStatuses) > 0 {
			if pod.Status.ContainerStatuses[0].State.Waiting != nil {
				reason := pod.Status.ContainerStatuses[0].State.Waiting.Reason
				if reason != latestReason {
					latestReason = reason
					log.Printf(pod.Name + " : " + string(pod.Status.Phase) + "->" + reason)
					events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, e.deploymentID).RegisterAsString("Pod status : " + pod.Name + " : " + string(pod.Status.Phase) + " -> " + reason)
				}
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

				events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, e.deploymentID).RegisterAsString("Pod status : " + pod.Name + " : " + string(pod.Status.Phase) + " (" + state + ")")
				if reason == "RunContainerError" {
					logs, err := e.clientSet.CoreV1().Pods(namespace).GetLogs(strings.ToLower(e.cfg.ResourcesPrefix+e.NodeName), &v1.PodLogOptions{}).Do().Raw()
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

func (e *execution) uninstallNode(ctx context.Context) error {
	namespace, err := getNamespace(e.kv, e.deploymentID, e.NodeName)
	if err != nil {
		return err
	}

	if deployment, err := e.clientSet.ExtensionsV1beta1().Deployments(namespace).Get(strings.ToLower(e.cfg.ResourcesPrefix+e.NodeName), metav1.GetOptions{}); err == nil {
		replica := int32(0)
		deployment.Spec.Replicas = &replica
		_, err = e.clientSet.ExtensionsV1beta1().Deployments(namespace).Update(deployment)

		err = e.clientSet.ExtensionsV1beta1().Deployments(namespace).Delete(strings.ToLower(e.cfg.ResourcesPrefix+e.NodeName), &metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrap(err, "Failed to delete deployment")
		}
		log.Printf("Deployment deleted")
	}

	if _, err = e.clientSet.CoreV1().Services(namespace).Get(strings.ToLower(generatePodName(e.cfg.ResourcesPrefix+e.NodeName)), metav1.GetOptions{}); err == nil {
		err = e.clientSet.CoreV1().Services(namespace).Delete(strings.ToLower(generatePodName(e.cfg.ResourcesPrefix+e.NodeName)), &metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrap(err, "Failed to delete service")
		}
		log.Printf("Service deleted")
	}

	if _, err = e.clientSet.CoreV1().Secrets(namespace).Get(e.SecretRepoName, metav1.GetOptions{}); err == nil {
		err = e.clientSet.CoreV1().Secrets(namespace).Delete(e.SecretRepoName, &metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrap(err, "Failed to delete secret")
		}
		log.Printf("Secret deleted")

	}

	if err = e.clientSet.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "Failed to delete namespace")
	}

	_, err = e.clientSet.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})

	log.Printf("Waiting for namespace to be fully deleted")
	for err == nil {
		time.Sleep(2 * time.Second)
		_, err = e.clientSet.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	}

	log.Printf("Namespace deleted !")
	return nil
}

func getNamespace(kv *api.KV, deploymentID, nodeName string) (string, error) {
	return strings.ToLower(deploymentID), nil
}
