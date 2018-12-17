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
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

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
	k8sScaleOperation
)

type dockerConfigEntry struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
	Auth     string `json:"auth"`
}

type execution struct {
	kv           *api.KV
	cfg          config.Configuration
	deploymentID string
	taskID       string
	taskType     tasks.TaskType
	nodeName     string
	operation    prov.Operation
	nodeType     string

	// Bellow params are used in deprecated functions
	envInputs      []*operations.EnvInput
	secretRepoName string
}

func newExecution(kv *api.KV, cfg config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) (*execution, error) {
	taskType, err := tasks.GetTaskType(kv, taskID)
	if err != nil {
		return nil, err
	}

	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}

	return &execution{
		kv:           kv,
		cfg:          cfg,
		deploymentID: deploymentID,
		nodeName:     nodeName,
		operation:    operation,
		envInputs:    make([]*operations.EnvInput, 0),
		taskID:       taskID,
		taskType:     taskType,
		nodeType:     nodeType,
	}, nil
}

func (e *execution) execute(ctx context.Context, clientset kubernetes.Interface) error {

	if e.nodeType == "yorc.nodes.kubernetes.api.types.JobResource" {
		return e.executeJobOperation(ctx, clientset)
	}

	// TODO is there any reason for recreating a new generator for each execution?
	generator := newGenerator(e.kv, e.cfg)
	instances, err := tasks.GetInstances(e.kv, e.taskID, e.deploymentID, e.nodeName)
	nbInstances := int32(len(instances))

	// Supporting both fully qualified and short standard operation names, ie.
	// - tosca.interfaces.node.lifecycle.standard.operation
	// or
	// - standard.operation
	operationName := strings.TrimPrefix(strings.ToLower(e.operation.Name),
		"tosca.interfaces.node.lifecycle.")
	switch operationName {
	case "standard.create":
		return e.manageKubernetesResource(ctx, clientset, generator, k8sCreateOperation)
	case "standard.configure":
		log.Printf("Voluntary bypassing operation %s", e.operation.Name)
		return nil
	case "standard.start":
		if e.taskType == tasks.TaskTypeScaleOut {
			log.Println("Scale up node !")
			err = e.scaleNode(ctx, clientset, tasks.TaskTypeScaleOut, nbInstances)
		} else {
			log.Println("Deploy node !")
			err = e.deployNode(ctx, clientset, generator, nbInstances)
		}
		if err != nil {
			return err
		}
		return e.checkNode(ctx, clientset, generator)
	case "standard.stop":
		if e.taskType == tasks.TaskTypeScaleIn {
			log.Println("Scale down node !")
			return e.scaleNode(ctx, clientset, tasks.TaskTypeScaleIn, nbInstances)
		}
		return e.uninstallNode(ctx, clientset)
	case "standard.delete":
		return e.manageKubernetesResource(ctx, clientset, generator, k8sDeleteOperation)
	case "org.alien4cloud.management.clustercontrol.scale":
		return e.manageKubernetesResource(ctx, clientset, generator, k8sScaleOperation)
	default:
		return errors.Errorf("Unsupported operation %q", e.operation.Name)
	}

}

func (e *execution) manageKubernetesResource(ctx context.Context, clientset kubernetes.Interface, generator *k8sGenerator, op k8sResourceOperation) error {
	rSpec, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.nodeName, "resource_spec")
	if err != nil {
		return err
	}

	if rSpec == nil {
		return errors.Errorf("no resource_spec defined for node %q", e.nodeName)
	}
	switch e.nodeType {
	case deploymentResourceType:
		return e.manageDeploymentResource(ctx, clientset, generator, op, rSpec.RawString())
	case serviceResourceType:
		return e.manageServiceResource(ctx, clientset, generator, op, rSpec.RawString())
	default:
		return errors.Errorf("Unsupported k8s resource type %q", e.nodeType)
	}
}

func (e *execution) replaceServiceIPInDeploymentSpec(ctx context.Context, clientset kubernetes.Interface, namespace, rSpec string) (string, error) {
	serviceDepsLookups, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.nodeName, "service_dependency_lookups")
	if err != nil {
		return rSpec, err
	}
	if serviceDepsLookups != nil && serviceDepsLookups.RawString() != "" {
		for _, srvLookup := range strings.Split(serviceDepsLookups.RawString(), ",") {
			srvLookupArgs := strings.SplitN(srvLookup, ":", 2)
			srvPlaceholder := "${" + srvLookupArgs[0] + "}"
			if !strings.Contains(rSpec, srvPlaceholder) || len(srvLookupArgs) != 2 {
				// No need to make an API call if there is no placeholder to replace
				// Alien set services lookups on all nodes
				continue
			}
			srvName := srvLookupArgs[1]
			srv, err := clientset.CoreV1().Services(namespace).Get(srvName, metav1.GetOptions{})
			if err != nil {
				return rSpec, errors.Wrapf(err, "failed to retrieve ClusterIP for service %q", srvName)
			}
			if srv.Spec.ClusterIP == "" || srv.Spec.ClusterIP == "None" {
				// Not supported
				return rSpec, errors.Wrapf(err, "failed to retrieve ClusterIP for service %q, (value=%q)", srvName, srv.Spec.ClusterIP)
			}
			rSpec = strings.Replace(rSpec, srvPlaceholder, srv.Spec.ClusterIP, -1)
		}
	}
	return rSpec, nil
}

func (e *execution) manageDeploymentResource(ctx context.Context, clientset kubernetes.Interface, generator *k8sGenerator, operationType k8sResourceOperation, rSpec string) (err error) {
	if rSpec == "" {
		return errors.Errorf("Missing mandatory resource_spec property for node %s", e.nodeName)
	}

	// Unmarshal JSON to k8s data structs
	var deploymentRepr v1beta1.Deployment
	if err = json.Unmarshal([]byte(rSpec), &deploymentRepr); err != nil {
		return errors.Errorf("The resource-spec JSON unmarshaling failed: %s", err)
	}

	// Get the namespace if provided. Otherwise, the namespace is generated using the default yorc policy
	objectMeta := deploymentRepr.ObjectMeta
	var namespaceName string
	var namespaceProvided bool
	namespaceName, namespaceProvided = getNamespace(e.deploymentID, objectMeta)

	switch operationType {
	case k8sCreateOperation:
		if !namespaceProvided {
			err = createNamespaceIfMissing(e.deploymentID, namespaceName, clientset)
			if err != nil {
				return err
			}
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("k8s Namespace %s created", namespaceName)
		}
		// Update resource_spec with actual reference to used services, if necessary
		rSpec, err = e.replaceServiceIPInDeploymentSpec(ctx, clientset, namespaceName, rSpec)
		if err = json.Unmarshal([]byte(rSpec), &deploymentRepr); err != nil {
			return errors.Errorf("The resource-spec JSON unmarshaling failed: %s", err)
		}
		if err != nil {
			return err
		}
		// Create Deployment k8s resource
		deployment, err := clientset.ExtensionsV1beta1().Deployments(namespaceName).Create(&deploymentRepr)
		if err != nil {
			return err
		}

		streamDeploymentLogs(ctx, e.deploymentID, clientset, deployment)

		err = waitForDeploymentCompletion(ctx, e.deploymentID, clientset, deployment)
		if err != nil {
			return err
		}
		err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.nodeName, "replicas", fmt.Sprint(*deployment.Spec.Replicas))
		if err != nil {
			return err
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("k8s Deployment %s created in namespace %s", deployment.Name, namespaceName)

	case k8sDeleteOperation:
		// Delete Deployment k8s resource
		var deploymentName string
		deploymentName = deploymentRepr.Name
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("Delete k8s Deployment %s", deploymentName)

		deployment, err := clientset.ExtensionsV1beta1().Deployments(namespaceName).Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		streamDeploymentLogs(ctx, e.deploymentID, clientset, deployment)

		deletePolicy := metav1.DeletePropagationForeground
		var gracePeriod int64 = 5
		if err = clientset.ExtensionsV1beta1().Deployments(namespaceName).Delete(deploymentName, &metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod, PropagationPolicy: &deletePolicy}); err != nil {
			return err
		}

		// TODO make timeout configurable
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		err = waitForDeploymentDeletion(ctx, clientset, deployment)
		if err != nil {
			return err
		}

		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("k8s Deployment %s deleted", deploymentName)

		// Delete namespace if it was not provided
		if !namespaceProvided {
			// Check if other deployments exist in the namespace
			// In that case nothing to do
			nbDeployments, err := deploymentsInNamespace(clientset, namespaceName)
			if err != nil {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("Cannot delete %s k8s Namespace", namespaceName)
				return err
			}
			if nbDeployments > 0 {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("Do not delete %s namespace as %d deployments exist", namespaceName, nbDeployments)
			} else {
				err = deleteNamespace(namespaceName, clientset)
				if err != nil {
					events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("Cannot delete %s k8s Namespace", namespaceName)
					return err
				}
			}
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("k8s Namespace %s deleted", namespaceName)
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("k8s Namespace %s deleted", namespaceName)
	case k8sScaleOperation:
		expectedInstances, err := tasks.GetTaskInput(e.kv, e.taskID, "EXPECTED_INSTANCES")
		if err != nil {
			return err
		}
		r, err := strconv.ParseInt(expectedInstances, 10, 32)
		if err != nil {
			return errors.Wrapf(err, "failed to parse EXPECTED_INSTANCES: %q parameter as integer", expectedInstances)
		}
		replicas := int32(r)
		deploymentRepr.Spec.Replicas = &replicas

		deployment, err := clientset.ExtensionsV1beta1().Deployments(namespaceName).Update(&deploymentRepr)
		if err != nil {
			return errors.Wrap(err, "failed to update kubernetes deployment for scaling")
		}
		streamDeploymentLogs(ctx, e.deploymentID, clientset, deployment)

		err = waitForDeploymentCompletion(ctx, e.deploymentID, clientset, deployment)
		if err != nil {
			return err
		}
		err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.nodeName, "replicas", expectedInstances)
		if err != nil {
			return err
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("k8s Deployment %s scaled to %s instances in namespace %s", deployment.Name, expectedInstances, namespaceName)
	default:
		return errors.Errorf("Unsupported operation on k8s resource")
	}

	return nil
}

func (e *execution) manageServiceResource(ctx context.Context, clientset kubernetes.Interface, generator *k8sGenerator, operationType k8sResourceOperation, rSpec string) (err error) {
	var serviceRepr apiv1.Service
	if rSpec == "" {
		return errors.Errorf("Missing mandatory resource_spec property for node %s", e.nodeName)
	}

	// Unmarshal JSON to k8s data structs
	if err = json.Unmarshal([]byte(rSpec), &serviceRepr); err != nil {
		return errors.Errorf("The resource-spec JSON unmarshaling failed: %s", err)
	}

	// Get the namespace if provided. Otherwise, the namespace is generated using the default yorc policy
	objectMeta := serviceRepr.ObjectMeta
	var namespace string
	namespace, _ = getNamespace(e.deploymentID, objectMeta)

	switch operationType {
	case k8sCreateOperation:
		// Create Service k8s resource
		service, err := clientset.CoreV1().Services(namespace).Create(&serviceRepr)
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}

		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("k8s Service %s created in namespace %s", service.Name, namespace)
		node, err := getHealthyNode(clientset)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, e.deploymentID).Registerf("Not able to find an healthy node")
		}
		h, err := getExternalIPAdress(clientset, node)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, e.deploymentID).Registerf("Error getting external ip of node %s", node)
		}
		for _, val := range service.Spec.Ports {
			if val.NodePort != 0 {
				str := fmt.Sprintf("http://%s:%d", h, val.NodePort)
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("%s : %s: %d:%d mapped to %s", service.Name, val.Name, val.Port, val.TargetPort.IntVal, str)
				err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.nodeName, "k8s_service_url", str)
				if err != nil {
					return errors.Wrap(err, "Failed to set attribute")
				}
				err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.nodeName, "node_port", strconv.Itoa(int(val.NodePort)))
				if err != nil {
					return errors.Wrap(err, "Failed to set attribute")
				}
			}
		}
	case k8sDeleteOperation:
		// Delete Deployment k8s resource
		var serviceName string
		serviceName = serviceRepr.Name
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("Delete k8s Service %s", serviceName)

		err = clientset.CoreV1().Services(namespace).Delete(serviceName, nil)
		if err != nil {
			return errors.Wrap(err, "Failed to delete service")
		}

		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("k8s Service %s deleted!", serviceName)
	default:
		return errors.Errorf("Unsupported operation on k8s resource")
	}
	return nil
}

// Below code is for legacy way of managing Kubernetes containers

// Deprecated
func (e *execution) parseEnvInputs() []apiv1.EnvVar {
	var data []apiv1.EnvVar

	for _, val := range e.envInputs {
		tmp := apiv1.EnvVar{Name: val.Name, Value: val.Value}
		data = append(data, tmp)
	}

	return data
}
func (e *execution) checkRepository(ctx context.Context, clientset kubernetes.Interface, generator *k8sGenerator) error {
	namespace, err := defaultNamespace(e.deploymentID)

	if err != nil {
		return err
	}

	repoName, err := deployments.GetOperationImplementationRepository(e.kv, e.deploymentID, e.operation.ImplementedInNodeTemplate, e.nodeType, e.operation.Name)
	if err != nil {
		return err
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(repoName, metav1.GetOptions{})
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

	_, err = generator.createNewRepoSecret(clientset, namespace, repoName, byteD)
	e.secretRepoName = repoName

	if err != nil {
		return err
	}

	return nil
}

func (e *execution) scaleNode(ctx context.Context, clientset kubernetes.Interface, scaleType tasks.TaskType, nbInstances int32) error {
	namespace, err := defaultNamespace(e.deploymentID)
	if err != nil {
		return err
	}

	deployment, err := clientset.ExtensionsV1beta1().Deployments(namespace).Get(strings.ToLower(e.cfg.ResourcesPrefix+e.nodeName), metav1.GetOptions{})

	replica := *deployment.Spec.Replicas
	if scaleType == tasks.TaskTypeScaleOut {
		replica = replica + nbInstances
	} else if scaleType == tasks.TaskTypeScaleIn {
		replica = replica - nbInstances
	}

	deployment.Spec.Replicas = &replica
	_, err = clientset.ExtensionsV1beta1().Deployments(namespace).Update(deployment)
	if err != nil {
		return errors.Wrap(err, "Failed to scale deployment")
	}

	return nil
}

func (e *execution) deployNode(ctx context.Context, clientset kubernetes.Interface, generator *k8sGenerator, nbInstances int32) error {
	namespace, err := defaultNamespace(e.deploymentID)
	if err != nil {
		return err
	}

	err = createNamespaceIfMissing(e.deploymentID, namespace, clientset)
	if err != nil {
		return err
	}

	err = e.checkRepository(ctx, clientset, generator)
	if err != nil {
		return err
	}

	e.envInputs, _, err = operations.ResolveInputs(e.kv, e.deploymentID, e.nodeName, e.taskID, e.operation)
	if err != nil {
		return err
	}
	inputs := e.parseEnvInputs()

	deployment, service, err := generator.generateDeployment(e.deploymentID, e.nodeName, e.operation, e.nodeType, e.secretRepoName, inputs, nbInstances)
	if err != nil {
		return err
	}

	_, err = clientset.ExtensionsV1beta1().Deployments(namespace).Create(&deployment)

	if err != nil {
		return errors.Wrap(err, "Failed to create deployment")
	}

	if service.Name != "" {
		serv, err := clientset.CoreV1().Services(namespace).Create(&service)
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
		var s string
		node, err := getHealthyNode(clientset)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, e.deploymentID).Registerf("Not able to find an healthy node")
		}
		h, err := getExternalIPAdress(clientset, node)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, e.deploymentID).Registerf("Error getting external ip of node %s", node)
		}
		for _, val := range serv.Spec.Ports {
			str := fmt.Sprintf("http://%s:%d", h, val.NodePort)

			log.Printf("%s : %s: %d:%d mapped to %s", serv.Name, val.Name, val.Port, val.TargetPort.IntVal, str)

			s = fmt.Sprintf("%s %d ==> %s \n", s, val.Port, str)

			if val.NodePort != 0 {
				// The service is accessible to an external IP address through
				// this port. Updating the corresponding public endpoints
				// kubernetes port mapping
				err := e.updatePortMappingPublicEndpoints(val.Port, h, val.NodePort)
				if err != nil {
					return errors.Wrap(err, "Failed to update endpoint capabilities port mapping")
				}
			}
		}
		err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.nodeName, "k8s_service_url", s)
		if err != nil {
			return errors.Wrap(err, "Failed to set attribute")
		}

		// Legacy
		err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.nodeName, "ip_address", service.Name)
		if err != nil {
			return errors.Wrap(err, "Failed to set attribute")
		}

		err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.nodeName, "k8s_service_name", service.Name)
		if err != nil {
			return errors.Wrap(err, "Failed to set attribute")
		}
		// TODO check that it is a good idea to use it as endpoint ip_address
		err = deployments.SetCapabilityAttributeForAllInstances(e.kv, e.deploymentID, e.nodeName, "endpoint", "ip_address", service.Name)
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

// Deprecated
func (e *execution) setUnDeployHook() error {
	_, err := deployments.GetNodeTypeImplementingAnOperation(e.kv, e.deploymentID, e.nodeName, "tosca.interfaces.node.lifecycle.standard.stop")
	if err != nil {
		if !deployments.IsOperationNotImplemented(err) {
			return err
		}
		// Set an implementation type
		// TODO this works as long as Alien still add workflow steps for those operations even if there is no real operation defined
		// As this sounds like a hack we do not create a method in the deployments package to do it
		_, errGrp, store := consulutil.WithContext(context.Background())
		opPath := path.Join(consulutil.DeploymentKVPrefix, e.deploymentID, "topology/types", e.nodeType, "interfaces/standard/stop")
		store.StoreConsulKeyAsString(path.Join(opPath, "name"), "stop")
		store.StoreConsulKeyAsString(path.Join(opPath, "implementation/type"), kubernetesArtifactImplementation)
		store.StoreConsulKeyAsString(path.Join(opPath, "implementation/description"), "Auto-generated operation")
		return errGrp.Wait()
	}
	// There is already a custom stop operation defined in the type hierarchy let's use it
	return nil
}

func (e *execution) checkNode(ctx context.Context, clientset kubernetes.Interface, generator *k8sGenerator) error {
	namespace, err := defaultNamespace(e.deploymentID)
	if err != nil {
		return err
	}

	deploymentReady := false
	var available int32 = -1

	for !deploymentReady {
		deployment, err := clientset.ExtensionsV1beta1().Deployments(namespace).Get(strings.ToLower(e.cfg.ResourcesPrefix+e.nodeName), metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "Failed fetch deployment")
		}
		if available != deployment.Status.AvailableReplicas {
			available = deployment.Status.AvailableReplicas
			log.Printf("Deployment %s : %d pod available of %d", e.nodeName, available, *deployment.Spec.Replicas)
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
			pods, _ := clientset.CoreV1().Pods(namespace).List(
				metav1.ListOptions{
					LabelSelector: selector,
				})

			// We should always have only 1 pod (as the Replica is set to 1)
			for _, podItem := range pods.Items {

				//log.Printf("Check pod %s", podItem.Name)
				err := e.checkPod(ctx, clientset, generator, podItem.Name)
				if err != nil {
					return err
				}

			}
		}

		time.Sleep(2 * time.Second)
	}

	return nil
}

func (e *execution) checkPod(ctx context.Context, clientset kubernetes.Interface, generator *k8sGenerator, podName string) error {
	namespace, err := defaultNamespace(e.deploymentID)
	if err != nil {
		return err
	}

	status := apiv1.PodUnknown
	latestReason := ""

	for status != apiv1.PodRunning && latestReason != "ErrImagePull" && latestReason != "InvalidImageName" {
		pod, err := clientset.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})

		if err != nil {
			return errors.Wrap(err, "Failed to fetch pod")
		}

		status = pod.Status.Phase

		if status == apiv1.PodPending && len(pod.Status.ContainerStatuses) > 0 {
			if pod.Status.ContainerStatuses[0].State.Waiting != nil {
				reason := pod.Status.ContainerStatuses[0].State.Waiting.Reason
				if reason != latestReason {
					latestReason = reason
					log.Printf(pod.Name + " : " + string(pod.Status.Phase) + "->" + reason)
					events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).RegisterAsString("Pod status : " + pod.Name + " : " + string(pod.Status.Phase) + " -> " + reason)
				}
			}

		} else {
			ready := true
			cond := apiv1.PodCondition{}
			for _, condition := range pod.Status.Conditions {
				if condition.Status == apiv1.ConditionFalse {
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

				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).RegisterAsString("Pod status : " + pod.Name + " : " + string(pod.Status.Phase) + " (" + state + ")")
				if reason == "RunContainerError" {
					logs, err := clientset.CoreV1().Pods(namespace).GetLogs(strings.ToLower(e.cfg.ResourcesPrefix+e.nodeName), &apiv1.PodLogOptions{}).Do().Raw()
					if err != nil {
						return errors.Wrap(err, "Failed to fetch pod logs")
					}
					podLogs := string(logs)
					log.Printf("Pod failed to start reason : %s --- Message : %s --- Pod logs : %s", reason, message, podLogs)
				}

				log.Printf("Pod failed to start reason : %s --- Message : %s -- condition : %s", reason, message, cond.Message)
			}
		}

		if status != apiv1.PodRunning {
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

func (e *execution) uninstallNode(ctx context.Context, clientset kubernetes.Interface) error {
	namespace, err := defaultNamespace(e.deploymentID)
	if err != nil {
		return err
	}

	if deployment, err := clientset.ExtensionsV1beta1().Deployments(namespace).Get(strings.ToLower(e.cfg.ResourcesPrefix+e.nodeName), metav1.GetOptions{}); err == nil {
		replica := int32(0)
		deployment.Spec.Replicas = &replica
		_, err = clientset.ExtensionsV1beta1().Deployments(namespace).Update(deployment)

		err = clientset.ExtensionsV1beta1().Deployments(namespace).Delete(strings.ToLower(e.cfg.ResourcesPrefix+e.nodeName), &metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrap(err, "Failed to delete deployment")
		}
		log.Printf("Deployment deleted")
	}

	if _, err = clientset.CoreV1().Services(namespace).Get(strings.ToLower(generatePodName(e.cfg.ResourcesPrefix+e.nodeName)), metav1.GetOptions{}); err == nil {
		err = clientset.CoreV1().Services(namespace).Delete(strings.ToLower(generatePodName(e.cfg.ResourcesPrefix+e.nodeName)), &metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrap(err, "Failed to delete service")
		}
		log.Printf("Service deleted")
	}

	if _, err = clientset.CoreV1().Secrets(namespace).Get(e.secretRepoName, metav1.GetOptions{}); err == nil {
		err = clientset.CoreV1().Secrets(namespace).Delete(e.secretRepoName, &metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrap(err, "Failed to delete secret")
		}
		log.Printf("Secret deleted")

	}

	if err = clientset.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "Failed to delete namespace")
	}

	_, err = clientset.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})

	log.Printf("Waiting for namespace to be fully deleted")
	for err == nil {
		time.Sleep(2 * time.Second)
		_, err = clientset.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	}

	log.Printf("Namespace deleted !")
	return nil
}
