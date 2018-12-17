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
	"encoding/json"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/log"
)

func isDeploymentFailed(clientset kubernetes.Interface, deployment *v1beta1.Deployment) (bool, string) {
	for _, c := range deployment.Status.Conditions {
		if c.Type == v1beta1.DeploymentReplicaFailure && c.Status == corev1.ConditionTrue {
			return true, c.Message
		} else if c.Type == v1beta1.DeploymentProgressing && c.Status == corev1.ConditionFalse {
			return true, c.Message
		}
	}
	return false, ""
}

func waitForDeploymentDeletion(ctx context.Context, clientset kubernetes.Interface, deployment *v1beta1.Deployment) error {
	return wait.PollUntil(2*time.Second, func() (bool, error) {
		_, err := clientset.ExtensionsV1beta1().Deployments(deployment.Namespace).Get(deployment.Name, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())

}

func waitForDeploymentCompletion(ctx context.Context, deploymentID string, clientset kubernetes.Interface, deployment *v1beta1.Deployment) error {
	return wait.PollUntil(2*time.Second, func() (bool, error) {
		deployment, err := clientset.ExtensionsV1beta1().Deployments(deployment.Namespace).Get(deployment.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if deployment.Status.AvailableReplicas == *deployment.Spec.Replicas {
			return true, nil
		}

		if failed, msg := isDeploymentFailed(clientset, deployment); failed {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, deploymentID).Registerf("Kubernetes deployment %q failed: %s", deployment.Name, msg)
			return false, errors.Errorf("Kubernetes deployment %q: %s", deployment.Name, msg)
		}
		return false, nil
	}, ctx.Done())
}

func streamDeploymentLogs(ctx context.Context, deploymentID string, clientset kubernetes.Interface, deployment *v1beta1.Deployment) {
	go func() {
		watcher, err := clientset.CoreV1().Events(deployment.Namespace).Watch(metav1.ListOptions{})
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).Registerf("Failed to monitor Kubernetes deployment events: %v", err)
			return
		}
		defer watcher.Stop()
		for {
			select {
			case <-ctx.Done():
				return

			case e, ok := <-watcher.ResultChan():
				if !ok {
					events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).RegisterAsString("Failed to monitor Kubernetes deployment events: watch channel closed")
					return
				}
				if e.Type == watch.Error {
					if status, ok := e.Object.(*metav1.Status); ok {
						events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).Registerf("Failed to monitor Kubernetes deployment events: %s: %s", status.Reason, status.Message)
						return
					}
					events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).Registerf("Failed to monitor Kubernetes deployment events:Received unexpected error: %#v", e.Object)
					return
				}
				if event, ok := e.Object.(*corev1.Event); ok {
					if ok, err := isChildOf(clientset, deployment.UID, referenceFromObjectReference(event.InvolvedObject)); err == nil && ok {
						switch e.Type {
						case watch.Added, watch.Modified:
							level := events.LogLevelDEBUG
							if strings.ToLower(event.Type) == "warning" {
								level = events.LogLevelWARN
							}
							events.WithContextOptionalFields(ctx).NewLogEntry(level, deploymentID).Registerf("%s (source: component: %q, host: %q)", event.Message, event.Source.Component, event.Source.Host)
						case watch.Deleted:
							// Deleted events are silently ignored.
						default:
							events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).Registerf("Unknown watchUpdate.Type: %#v", e.Type)
						}
					}
				} else {
					events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).Registerf("Wrong object received: %v", e)
				}
			}
		}
	}()
}

func isChildOf(clientset kubernetes.Interface, parent types.UID, ref reference) (bool, error) {
	if ref.UID == parent {
		return true, nil
	}
	var om metav1.Object
	var err error
	switch strings.ToLower(ref.Kind) {
	case "pod":
		om, err = clientset.CoreV1().Pods(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
	case "replicaset":
		om, err = clientset.ExtensionsV1beta1().ReplicaSets(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
	case "deployment":
		om, err = clientset.ExtensionsV1beta1().Deployments(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
	case "job":
		om, err = clientset.BatchV1().Jobs(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
	default:
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "Failed to get pod when checking if event is related to our deployment")
	}
	for _, parentRef := range om.GetOwnerReferences() {
		ok, err := isChildOf(clientset, parent, referenceFromOwnerReference(ref.Namespace, parentRef))
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func deploymentsInNamespace(clientset kubernetes.Interface, namespace string) (int, error) {
	var nbDeployments int
	deploymentsList, err := clientset.ExtensionsV1beta1().Deployments(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nbDeployments, err
	}
	for _, dep := range deploymentsList.Items {
		//depName := dep.GetName()
		depNamespace := dep.GetNamespace()
		//events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).Registerf("Found %s deployment in k8s Namespace %s", depName, depNamespace)
		if depNamespace == namespace {
			nbDeployments++
		}
	}
	return nbDeployments, nil
}

type reference struct {
	Kind      string
	UID       types.UID
	Namespace string
	Name      string
}

func referenceFromObjectReference(ref corev1.ObjectReference) reference {
	return reference{ref.Kind, ref.UID, ref.Namespace, ref.Name}
}

func referenceFromOwnerReference(namespace string, ref metav1.OwnerReference) reference {
	return reference{ref.Kind, ref.UID, namespace, ref.Name}
}

// CreateNamespaceIfMissing create a kubernetes namespace (only if missing)
func createNamespaceIfMissing(deploymentID, namespaceName string, clientset kubernetes.Interface) error {
	_, err := clientset.CoreV1().Namespaces().Get(namespaceName, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			_, err := clientset.CoreV1().Namespaces().Create(&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: namespaceName},
			})
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				return errors.Wrap(err, "Failed to create namespace")
			}
		} else {
			return errors.Wrap(err, "Failed to create namespace")
		}
	}
	return nil
}

// deleteNamespace delete a Kubernetes namespaces known by its name
func deleteNamespace(namespaceName string, clientset kubernetes.Interface) error {
	err := clientset.CoreV1().Namespaces().Delete(namespaceName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to delete namespace "+namespaceName)
	}
	return nil
}

// Default k8s namespace policy for Yorc : one namespace for each deployment
func defaultNamespace(deploymentID string) (string, error) {
	return strings.ToLower(deploymentID), nil
}

// Check if a namespace is provided for a deployment.
// Return it if provided ; return one generated with the default policy, if not provided
func getNamespace(deploymentID string, objectMeta metav1.ObjectMeta) (string, bool) {
	var isProvided bool
	var namespace string
	var providedNamespace string
	if &objectMeta != nil {
		providedNamespace = objectMeta.Namespace
	}

	if providedNamespace != "" {
		namespace = providedNamespace
		isProvided = true
	} else {
		namespace, _ = defaultNamespace(deploymentID)
	}

	return namespace, isProvided
}

func deleteJob(ctx context.Context, deploymentID, namespace, jobID string, namespaceProvided bool, clientset kubernetes.Interface) error {
	deleteForeground := metav1.DeletePropagationForeground
	err := clientset.BatchV1().Jobs(namespace).Delete(jobID, &metav1.DeleteOptions{PropagationPolicy: &deleteForeground})
	if err != nil {
		return errors.Wrapf(err, "failed to delete completed job %q", jobID)
	}
	// Delete namespace if it was not provided
	if !namespaceProvided {
		err = deleteNamespace(namespace, clientset)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).Registerf("Cannot delete %s k8s Namespace", namespace)
			return err
		}

		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).Registerf("k8s Namespace %s deleted", namespace)
	}
	return nil
}

type k8sJob struct {
	jobRepr           *batchv1.Job
	namespace         string
	namespaceProvided bool
}

func getJob(kv *api.KV, deploymentID, nodeName string) (*k8sJob, error) {
	rSpec, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "resource_spec")
	if err != nil {
		return nil, err
	}

	if rSpec == nil {
		return nil, errors.Errorf("no resource_spec defined for node %q", nodeName)
	}
	jobRepr := &batchv1.Job{}
	log.Debugf("jobspec: %v", rSpec.RawString())
	// Unmarshal JSON to k8s data structs
	if err = json.Unmarshal([]byte(rSpec.RawString()), jobRepr); err != nil {
		return nil, errors.Wrap(err, "The resource-spec JSON unmarshaling failed")
	}

	job := &k8sJob{jobRepr: jobRepr}
	objectMeta := jobRepr.ObjectMeta
	job.namespace, job.namespaceProvided = getNamespace(deploymentID, objectMeta)

	return job, nil
}

// Return the first healthy node found in the cluster
func getHealthyNode(clientset kubernetes.Interface) (string, error) {
	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return "", errors.Wrap(err, "Failed to get nodes")
	}
	for _, node := range nodes.Items {
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				return node.ObjectMeta.Name, nil
			}
		}
	}
	return "", errors.Wrap(err, "No healthy node found")
}

//Return the external IP of a given node
func getExternalIPAdress(clientset kubernetes.Interface, nodeName string) (string, error) {
	node, err := clientset.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "Failed to get node "+nodeName)
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeExternalIP {
			return addr.Address, nil
		}
	}
	return "", errors.Wrap(err, "Node "+nodeName+" don't have external IP adress")
}
