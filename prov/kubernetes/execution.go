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
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/tasks"
)

const k8sDeploymentResourceType string = "yorc.nodes.kubernetes.api.types.DeploymentResource"
const k8sStatefulsetResourceType string = "yorc.nodes.kubernetes.api.types.StatefulSetResource"
const k8sServiceResourceType string = "yorc.nodes.kubernetes.api.types.ServiceResource"
const k8sSimpleRessourceType string = "yorc.nodes.kubernetes.api.types.SimpleResource"

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
}

const namespaceCreatedMessage string = "K8's Namespace %s created"
const namespaceDeletedMessage string = "K8's Namespace %s deleted"
const namespaceDeletionFailedMessage string = "Cannot delete K8's Namespace %s"
const unsupportedOperationOnK8sResource string = "Unsupported operation on k8s resource"

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

	rType, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.nodeName, "resource_type")
	if err != nil {
		return err
	}
	if rType == nil {
		return errors.Errorf("No resource_type defined for node %q", e.nodeName)
	}
	// Get K8s object specification
	rSpecProp, err := deployments.GetNodePropertyValue(e.kv, e.deploymentID, e.nodeName, "resource_spec")
	if err != nil {
		return err
	}
	if rSpecProp == nil {
		return errors.Errorf("No resource_spec defined for node %q", e.nodeName)
	}
	rSpec := rSpecProp.RawString()
	// Create Yorc representation of the K8S object
	K8sObj, err := e.getYorcK8sObject(rType.RawString())
	// unmarshal resource spec
	err = K8sObj.unmarshalResource(ctx, e, e.deploymentID, clientset, rSpec)
	if err != nil {
		return errors.Errorf("The resource_spec JSON unmarshaling failed for node %s: %s", e.nodeName, err)
	}

	return e.executeOperation(ctx, generator, clientset, K8sObj, rSpec)

}

func (e *execution) executeOperation(ctx context.Context, generator *k8sGenerator, clientset kubernetes.Interface, K8sObj yorcK8sObject, rSpec string) error {
	// Supporting both fully qualified and short standard operation names, ie.
	// - tosca.interfaces.node.lifecycle.standard.operation
	// or
	// - standard.operation
	operationName := strings.TrimPrefix(strings.ToLower(e.operation.Name),
		"tosca.interfaces.node.lifecycle.")
	switch operationName {
	case "standard.create":
		return e.manageKubernetesResource(ctx, clientset, generator, K8sObj, k8sCreateOperation, rSpec)
	case "standard.delete":
		return e.manageKubernetesResource(ctx, clientset, generator, K8sObj, k8sDeleteOperation, rSpec)
	case "org.alien4cloud.management.clustercontrol.scale":
		return e.manageKubernetesResource(ctx, clientset, generator, K8sObj, k8sScaleOperation, rSpec)
	default:
		return errors.Errorf("Unsupported operation %q", e.operation.Name)
	}
}

// Create yorcK8sObject of appropriate type
func (e *execution) getYorcK8sObject(resourceType string) (yorcK8sObject, error) {
	var K8sObj yorcK8sObject
	switch e.nodeType {
	case k8sDeploymentResourceType:
		K8sObj = &yorcK8sDeployment{}
	case k8sStatefulsetResourceType:
		K8sObj = &yorcK8sStatefulSet{}
	case k8sServiceResourceType:
		K8sObj = &yorcK8sService{}
	case k8sSimpleRessourceType:
		switch resourceType {
		case "pvc":
			K8sObj = &yorcK8sPersistentVolumeClaim{}
		default:
			return nil, errors.Errorf("Unsupported k8s SimpleResource type %q", resourceType)
		}
	default:
		return nil, errors.Errorf("Unsupported k8s resource type %q", e.nodeType)
	}
	return K8sObj, nil
}

func (e *execution) manageKubernetesResource(ctx context.Context, clientset kubernetes.Interface, generator *k8sGenerator, k8sObject yorcK8sObject, operationType k8sResourceOperation, rSpec string) (err error) {
	/*  Steps :
	get NS
	switch OPtype
	*/
	namespaceName, namespaceProvided := getNamespace(e.deploymentID, k8sObject.getObjectMeta())
	switch operationType {
	case k8sCreateOperation:
		/*
			  Creation steps :
				create ns if missing 	OK
				create Resource   		OK
				(stream logs)			OK
				wait for completion		OK
				set attributes			OK
		*/
		if !namespaceProvided {
			err = createNamespaceIfMissing(e.deploymentID, namespaceName, clientset)
			if err != nil {
				return err
			}
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(namespaceCreatedMessage, namespaceName)
		}
		err := k8sObject.createResource(ctx, e.deploymentID, clientset, namespaceName)
		if err != nil {
			return err
		}

		k8sObject.streamLogs(ctx, e.deploymentID, clientset)
		err = waitForYorcK8sObjectCompletion(ctx, e.deploymentID, clientset, k8sObject)
		if err != nil {
			return err
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("%T %s created in namespace %s", k8sObject, k8sObject.getObjectMeta().Name, namespaceName)
		// set attributes
		err = k8sObject.setAttributes(ctx, e)
		if err != nil {
			return err
		}
		/*
			err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.nodeName, "replicas", fmt.Sprint(*deployment.Spec.Replicas))
			if err != nil {
				return err
			}
		*/
	case k8sDeleteOperation:
		/*
			Deletion steps :
				delete resource				OK
				wait for deletion			OK
				delete ns if not provided	OK
		*/
		k8sObject.streamLogs(ctx, e.deploymentID, clientset)
		err := k8sObject.deleteResource(ctx, e.deploymentID, clientset, namespaceName)
		if err != nil {
			return err
		}
		err = waitForYorcK8sObjectDeletion(ctx, clientset, k8sObject)
		if err != nil {
			return err
		}
		if !namespaceProvided {
			// Check if other deployments exist in the namespace
			// In that case nothing to do
			nbControllers, err := podControllersInNamespace(clientset, namespaceName)
			if err != nil {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(namespaceDeletionFailedMessage, namespaceName)
				return err
			}
			if nbControllers > 0 {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("Do not delete %s namespace as %d deployments exist", namespaceName, nbControllers)
			} else {
				err = deleteNamespace(namespaceName, clientset)
				if err != nil {
					events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(namespaceDeletionFailedMessage, namespaceName)
					return err
				}
			}
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("%T %s deleted in namespace %s", k8sObject, k8sObject.getObjectMeta().Name, namespaceName)
	case k8sScaleOperation:
		/*
			Scale steps :
				Updtade resource		OK
				(stream logs)			OK
				wait for completion		OK
				set attr				OK
		*/
		err := k8sObject.scaleResource(ctx, e, clientset, namespaceName)
		if err != nil {
			return err
		}
		k8sObject.streamLogs(ctx, e.deploymentID, clientset)
		err = waitForYorcK8sObjectCompletion(ctx, e.deploymentID, clientset, k8sObject)
		if err != nil {
			return err
		}
		err = k8sObject.setAttributes(ctx, e)
		if err != nil {
			return err
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("%T %s scaled in namespace %s", k8sObject, k8sObject.getObjectMeta().Name, namespaceName)
	default:
		return errors.Errorf(unsupportedOperationOnK8sResource)
	}
	return nil
}

func (e *execution) getExpectedInstances() (int32, error) {
	expectedInstances, err := tasks.GetTaskInput(e.kv, e.taskID, "EXPECTED_INSTANCES")
	if err != nil {
		return -1, err
	}
	r, err := strconv.ParseInt(expectedInstances, 10, 32)
	if err != nil {
		return -1, errors.Wrapf(err, "failed to parse EXPECTED_INSTANCES: %q parameter as integer", expectedInstances)
	}
	return int32(r), nil
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

/* TO BE DELETED
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
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(namespaceCreatedMessage, namespaceName)
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

		err = waitForK8sObjectCompletion(ctx, e.deploymentID, clientset, deployment)
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
			nbDeployments, err := podControllersInNamespace(clientset, namespaceName)
			if err != nil {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(namespaceDeletionFailedMessage, namespaceName)
				return err
			}
			if nbDeployments > 0 {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("Do not delete %s namespace as %d deployments exist", namespaceName, nbDeployments)
			} else {
				err = deleteNamespace(namespaceName, clientset)
				if err != nil {
					events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(namespaceDeletionFailedMessage, namespaceName)
					return err
				}
			}
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(namespaceDeletedMessage, namespaceName)
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(namespaceDeletedMessage, namespaceName)
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

		err = waitForK8sObjectCompletion(ctx, e.deploymentID, clientset, deployment)
		if err != nil {
			return err
		}
		err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.nodeName, "replicas", expectedInstances)
		if err != nil {
			return err
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("k8s Deployment %s scaled to %s instances in namespace %s", deployment.Name, expectedInstances, namespaceName)
	default:
		return errors.Errorf(unsupportedOperationOnK8sResource)
	}

	return nil
}

func (e *execution) manageStatefulSetResource(ctx context.Context, clientset kubernetes.Interface, generator *k8sGenerator, operationType k8sResourceOperation, rSpec string) (err error) {
	var stfsRepr appsv1.StatefulSet
	if rSpec == "" {
		return errors.Errorf("Missing mandatory resource_spec property for node %s", e.nodeName)
	}
	if err = json.Unmarshal([]byte(rSpec), &stfsRepr); err != nil {
		return errors.Errorf("The resource-spec JSON unmarshaling failed: %s", err)
	}
	objectMeta := stfsRepr.ObjectMeta
	namespace, nsProvided := getNamespace(e.deploymentID, objectMeta)

	switch operationType {
	case k8sCreateOperation:
		if !nsProvided {
			err = createNamespaceIfMissing(e.deploymentID, namespace, clientset)
			if err != nil {
				return err
			}
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(namespaceCreatedMessage, namespace)
		}
		stfs, err := clientset.AppsV1beta1().StatefulSets(namespace).Create(&stfsRepr)
		if err != nil {
			return errors.Wrap(err, "Failed to create statefulSet")
		}
		err = waitForK8sObjectCompletion(ctx, e.deploymentID, clientset, stfs)
		if err != nil {
			return err
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("k8s StatefulSet %s created in namespace %s", stfs.Name, namespace)
	case k8sDeleteOperation:
		var stfsName = stfsRepr.Name
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("Deleting k8s StatefulSet %s", stfsName)
		stfs, err := clientset.AppsV1beta1().StatefulSets(namespace).Get(stfsName, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "StatefuSet %s does not exists", stfs.Name)
		}
		deletePolicy := metav1.DeletePropagationForeground
		err = clientset.AppsV1beta1().StatefulSets(namespace).Delete(stfsName, &metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
		if err != nil {
			return errors.Wrapf(err, "Failed to delete StatefulSet %s", stfsName)
		}
		err = waitForK8sObjectDeletion(ctx, clientset, stfs)
		if err != nil {
			return err
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("k8s StatefulSet %s deleted!", stfsName)
		// Delete namespace if it was not provided and volumes marked as removable
		volDeletable, err := deployments.GetBooleanNodeProperty(e.kv, e.deploymentID, e.nodeName, "volumeDeletable")
		if err != nil {
			return err
		}
		if !volDeletable {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("Volumes keeped and k8s Namespace %s not deleted", namespace)
			return nil
		}
		if !nsProvided {
			nbController, err := podControllersInNamespace(clientset, namespace)
			if err != nil {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(namespaceDeletionFailedMessage, namespace)
				return err
			}
			if nbController > 0 {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("Do not delete %s namespace as %d deployments exist", namespace, nbController)
			} else {
				err = deleteNamespace(namespace, clientset)
				if err != nil {
					events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(namespaceDeletionFailedMessage, namespace)
					return err
				}
			}
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(namespaceDeletedMessage, namespace)
		}

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
		stfsRepr.Spec.Replicas = &replicas

		sts, err := clientset.AppsV1beta1().StatefulSets(namespace).Update(&stfsRepr)
		if err != nil {
			return errors.Wrap(err, "failed to update kubernetes statefulSet for scaling")
		}

		err = waitForK8sObjectCompletion(ctx, e.deploymentID, clientset, sts)
		if err != nil {
			return err
		}
		err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.nodeName, "replicas", expectedInstances)
		if err != nil {
			return err
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("k8s StatefulSet %s scaled to %s instances in namespace %s", sts.Name, expectedInstances, namespace)
	default:
		return errors.Errorf(unsupportedOperationOnK8sResource)
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
		return errors.Errorf(unsupportedOperationOnK8sResource)
	}
	return nil
}

//Manage kubernetes PersistentVolumeClaim
func (e *execution) manageSimpleResourcePVC(ctx context.Context, clientset kubernetes.Interface, generator *k8sGenerator, operationType k8sResourceOperation, rSpec string) (err error) {
	if rSpec == "" {
		return errors.Errorf("Missing mandatory resource_spec property for node %s", e.nodeName)
	}
	var pvcRepr apiv1.PersistentVolumeClaim
	if err = json.Unmarshal([]byte(rSpec), &pvcRepr); err != nil {
		return errors.Errorf("The resource-spec JSON unmarshaling failed: %s", err)
	}
	//Test if ressource request field is filled
	if len(pvcRepr.Spec.Resources.Requests) == 0 {
		return errors.Errorf("Missing mandatory field resource request property for node %s", e.nodeName)
	}
	namespace, nsProvided := getNamespace(e.deploymentID, pvcRepr.ObjectMeta)

	switch operationType {
	case k8sCreateOperation:
		if !nsProvided {
			err = createNamespaceIfMissing(e.deploymentID, namespace, clientset)
			if err != nil {
				return err
			}
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(namespaceCreatedMessage, namespace)
		}
		pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Create(&pvcRepr)
		if err != nil {
			return errors.Wrapf(err, "Failed to create persistent volume claim %s", pvc.Name)
		}
		err = waitForK8sObjectCompletion(ctx, e.deploymentID, clientset, pvc)
		if err != nil {
			return err
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("k8s PVC %s created in namespace %s", pvc.Name, namespace)

	case k8sDeleteOperation:
		var pvcName = pvcRepr.Name
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("Deleting k8s PVC %s", pvcName)
		pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(pvcName, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "Persisent volume claim %s does not exists", pvc.Name)
		}
		err = clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(pvcName, nil)
		if err != nil {
			return errors.Wrapf(err, "Failed to delete persistent volume claim %s", pvcName)
		}
		err = waitForPVCDeletion(ctx, clientset, pvc)
		if err != nil {
			return err
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("k8s PVC %s deleted!", pvcName)
	default:
		return errors.Errorf("Unsupported operation on k8s SimpleResourcePVC")
	}
	return nil
}
*/
