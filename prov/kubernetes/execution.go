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

	"github.com/pkg/errors"
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

type execution struct {
	cfg          config.Configuration
	deploymentID string
	taskID       string
	taskType     tasks.TaskType
	nodeName     string
	operation    prov.Operation
	nodeType     string
}

const namespaceCreatedMessage string = "K8's Namespace %s created"
const namespaceDeletionFailedMessage string = "Cannot delete K8's Namespace %s"
const unsupportedOperationOnK8sResource string = "Unsupported operation on k8s resource"

func newExecution(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) (*execution, error) {
	taskType, err := tasks.GetTaskType(taskID)
	if err != nil {
		return nil, err
	}

	nodeType, err := deployments.GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}

	return &execution{
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
	generator := newGenerator(e.cfg)

	// Create Yorc representation of the K8S object
	K8sObj, err := e.getYorcK8sObject(ctx, clientset)

	if err != nil {
		return errors.Errorf("The resource_spec JSON unmarshaling failed for node %s: %s", e.nodeName, err)
	}

	return e.executeOperation(ctx, generator, clientset, K8sObj)

}

func (e *execution) executeOperation(ctx context.Context, generator *k8sGenerator, clientset kubernetes.Interface, K8sObj yorcK8sObject) error {
	envSet := true
	if ctx == nil || generator == nil || clientset == nil || K8sObj == nil {
		envSet = false
	}
	// Supporting both fully qualified and short standard operation names, ie.
	// - tosca.interfaces.node.lifecycle.standard.operation
	// or
	// - standard.operation
	operationName := strings.TrimPrefix(strings.ToLower(e.operation.Name),
		"tosca.interfaces.node.lifecycle.")
	switch operationName {
	case "standard.create":
		return e.manageKubernetesResource(ctx, clientset, generator, K8sObj, k8sCreateOperation, envSet)
	case "standard.delete":
		return e.manageKubernetesResource(ctx, clientset, generator, K8sObj, k8sDeleteOperation, envSet)
	case "org.alien4cloud.management.clustercontrol.scale":
		return e.manageKubernetesResource(ctx, clientset, generator, K8sObj, k8sScaleOperation, envSet)
	default:
		return errors.Errorf("Unsupported operation %q", e.operation.Name)
	}
}

// Create yorcK8sObject of appropriate type
func (e *execution) getYorcK8sObject(ctx context.Context, clientset kubernetes.Interface) (yorcK8sObject, error) {

	var K8sObj yorcK8sObject
	switch e.nodeType {
	case k8sDeploymentResourceType:
		K8sObj = &yorcK8sDeployment{}
	case k8sStatefulsetResourceType:
		K8sObj = &yorcK8sStatefulSet{}
	case k8sServiceResourceType:
		K8sObj = &yorcK8sService{}
	case k8sSimpleRessourceType:
		rType, err := e.getResourceType(ctx)
		if err != nil {
			return nil, err
		}
		if rType == "" {
			return nil, errors.Errorf("Not provided resource type for node %s in deployment %s", e.nodeName, e.deploymentID)
		}
		switch rType {
		case "pvc":
			K8sObj = &yorcK8sPersistentVolumeClaim{}
		default:
			return nil, errors.Errorf("Unsupported k8s SimpleResource type %q", rType)
		}
	default:
		return nil, errors.Errorf("Unsupported k8s resource type %q", e.nodeType)
	}

	// Get K8s object specification
	rSpec, err := e.getResourceSpec(ctx)
	if err != nil {
		return nil, err
	}
	if rSpec == "" {
		return nil, errors.Errorf("Not provided resource specification for node %s in deployment %s", e.nodeName, e.deploymentID)
	}
	// unmarshal resource spec
	err = K8sObj.unmarshalResource(ctx, e, e.deploymentID, clientset, rSpec)
	if err != nil {
		return nil, err
	}
	return K8sObj, nil
}

func (e *execution) getResourceType(ctx context.Context) (string, error) {
	rType, err := deployments.GetNodePropertyValue(ctx, e.deploymentID, e.nodeName, "resource_type")
	if err != nil {
		return "", err
	}
	if rType == nil {
		return "", errors.Errorf("No resource_type defined for node %q", e.nodeName)
	}
	return rType.RawString(), nil
}

func (e *execution) getResourceSpec(ctx context.Context) (string, error) {
	rSpecProp, err := deployments.GetNodePropertyValue(ctx, e.deploymentID, e.nodeName, "resource_spec")
	if err != nil {
		return "", err
	}
	if rSpecProp == nil {
		return "", errors.Errorf("No resource_spec defined for node %q", e.nodeName)
	}
	return rSpecProp.RawString(), nil
}

func (e *execution) manageKubernetesResource(ctx context.Context, clientset kubernetes.Interface, generator *k8sGenerator, k8sObject yorcK8sObject,
	operationType k8sResourceOperation, envSet bool) (err error) {
	if !envSet {
		return errors.Errorf("Can't execute operation %q. Environment not set", e.operation.Name)
	}
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
			err = createNamespaceIfMissing(ctx, namespaceName, clientset)
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
		err = waitForYorcK8sObjectCompletion(ctx, e.deploymentID, clientset, k8sObject, namespaceName)
		if err != nil {
			return err
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("%T %s created in namespace %s", k8sObject, k8sObject.getObjectMeta().Name, namespaceName)
		// set attributes
		err = k8sObject.setAttributes(ctx, e)
		if err != nil {
			return err
		}

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
		err = waitForYorcK8sObjectDeletion(ctx, clientset, k8sObject, namespaceName)
		if err != nil {
			return err
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("%T %s deleted in namespace %s", k8sObject, k8sObject.getObjectMeta().Name, namespaceName)
		return e.manageNamespaceDeletion(ctx, clientset, namespaceProvided, namespaceName)

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
		err = waitForYorcK8sObjectCompletion(ctx, e.deploymentID, clientset, k8sObject, namespaceName)
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
	expectedInstances, err := tasks.GetTaskInput(e.taskID, "EXPECTED_INSTANCES")
	if err != nil {
		return -1, err
	}
	r, err := strconv.ParseInt(expectedInstances, 10, 32)
	if err != nil {
		return -1, errors.Wrapf(err, "failed to parse EXPECTED_INSTANCES: %q parameter as integer", expectedInstances)
	}
	return int32(r), nil
}

func (e *execution) manageNamespaceDeletion(ctx context.Context, clientset kubernetes.Interface, namespaceProvided bool, namespaceName string) error {
	if !namespaceProvided { //TODO applicable for all objects ?
		volDeletable, err := deployments.GetBooleanNodeProperty(ctx, e.deploymentID, e.nodeName, "volumeDeletable")
		if err != nil {
			return err
		}
		if !volDeletable {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("Volumes keeped and k8s Namespace %s not deleted", namespaceName)
			return nil
		}
		// Check if other deployments exist in the namespace
		// In that case nothing to do
		nbControllers, err := podControllersInNamespace(ctx, clientset, namespaceName)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(namespaceDeletionFailedMessage, namespaceName)
			return err
		}
		if nbControllers > 0 {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf("Do not delete %s namespace as %d deployments exist", namespaceName, nbControllers)
		} else {
			err = deleteNamespace(ctx, namespaceName, clientset)
			if err != nil {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(namespaceDeletionFailedMessage, namespaceName)
				return err
			}
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("Namespace %s deleted", namespaceName)
		}
	}
	return nil
}
