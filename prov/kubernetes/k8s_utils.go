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

	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
)

func isDeploymentFailed(deployment *v1.Deployment) (bool, string) {
	for _, c := range deployment.Status.Conditions {
		if c.Type == v1.DeploymentReplicaFailure && c.Status == corev1.ConditionTrue {
			return true, c.Message
		} else if c.Type == v1.DeploymentProgressing && c.Status == corev1.ConditionFalse {
			return true, c.Message
		}
	}
	return false, ""
}

func streamDeploymentLogs(ctx context.Context, deploymentID string, clientset kubernetes.Interface, deployment *v1.Deployment) {
	go func() {
		watcher, err := clientset.CoreV1().Events(deployment.Namespace).Watch(ctx, metav1.ListOptions{})
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
					if ok, err := isChildOf(ctx, clientset, deployment.UID, referenceFromObjectReference(event.InvolvedObject)); err == nil && ok {
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

func isChildOf(ctx context.Context, clientset kubernetes.Interface, parent types.UID, ref reference) (bool, error) {
	if ref.UID == parent {
		return true, nil
	}
	var om metav1.Object
	var err error
	switch strings.ToLower(ref.Kind) {
	case "pod":
		om, err = clientset.CoreV1().Pods(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
	case "replicaset":
		om, err = clientset.AppsV1().ReplicaSets(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
	case "deployment":
		om, err = clientset.AppsV1().Deployments(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
	case "job":
		om, err = clientset.BatchV1().Jobs(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
	default:
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "Failed to get pod when checking if event is related to our deployment")
	}
	for _, parentRef := range om.GetOwnerReferences() {
		ok, err := isChildOf(ctx, clientset, parent, referenceFromOwnerReference(ref.Namespace, parentRef))
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

/* Return the number of pod controllers (Deployment and StatefulSet, more in the future) in a specific namespace or -1, err != nil in case of error */
func podControllersInNamespace(ctx context.Context, clientset kubernetes.Interface, namespace string) (int, error) {
	var nbcontrollers int
	deploymentsList, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return -1, err
	}
	stsList, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return -1, err
	}
	nbcontrollers = len(deploymentsList.Items) + len(stsList.Items)
	return nbcontrollers, nil
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
func createNamespaceIfMissing(ctx context.Context, namespaceName string, clientset kubernetes.Interface) error {
	_, err := clientset.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			_, err := clientset.CoreV1().Namespaces().Create(ctx,
				&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName}},
				metav1.CreateOptions{},
			)
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
func deleteNamespace(ctx context.Context, namespaceName string, clientset kubernetes.Interface) error {
	err := clientset.CoreV1().Namespaces().Delete(ctx, namespaceName, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to delete namespace "+namespaceName)
	}
	return nil
}

// Default k8s namespace policy for Yorc : one namespace for each deployment
func defaultNamespace(deploymentID string) string {
	return strings.ToLower(deploymentID)
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
		namespace = defaultNamespace(deploymentID)
	}

	return namespace, isProvided
}

func deleteJob(ctx context.Context, deploymentID, namespace, jobID string, namespaceProvided bool, clientset kubernetes.Interface) error {
	deleteForeground := metav1.DeletePropagationForeground
	err := clientset.BatchV1().Jobs(namespace).Delete(ctx, jobID, metav1.DeleteOptions{PropagationPolicy: &deleteForeground})
	if err != nil {
		return errors.Wrapf(err, "failed to delete completed job %q", jobID)
	}
	// Delete namespace if it was not provided
	if !namespaceProvided {
		err = deleteNamespace(ctx, namespace, clientset)
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

func getJob(ctx context.Context, clientset kubernetes.Interface, deploymentID, nodeName string) (*k8sJob, error) {
	rSpec, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "resource_spec")
	if err != nil {
		return nil, err
	}

	if rSpec == nil {
		return nil, errors.Errorf("no resource_spec defined for node %q", nodeName)
	}
	rSpecString := rSpec.RawString()
	jobRepr := &batchv1.Job{}
	// Unmarshal JSON to k8s data structs just to retrieve namespace
	if err = json.Unmarshal([]byte(rSpecString), jobRepr); err != nil {
		return nil, errors.Wrap(err, "The resource-spec JSON unmarshaling failed")
	}
	namespace, _ := getNamespace(deploymentID, jobRepr.ObjectMeta)
	rSpecString, err = replaceServiceIPInResourceSpec(ctx, clientset, deploymentID, nodeName, namespace, rSpecString)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON to k8s data structs
	if err = json.Unmarshal([]byte(rSpecString), jobRepr); err != nil {
		return nil, errors.Wrap(err, "The resource-spec JSON unmarshaling failed")
	}

	job := &k8sJob{jobRepr: jobRepr}
	objectMeta := jobRepr.ObjectMeta
	job.namespace, job.namespaceProvided = getNamespace(deploymentID, objectMeta)

	return job, nil
}

func replaceServiceDepLookups(ctx context.Context, clientset kubernetes.Interface, namespace, rSpec, serviceDepsLookups string) (string, error) {
	for _, srvLookup := range strings.Split(serviceDepsLookups, ",") {
		srvLookupArgs := strings.SplitN(srvLookup, ":", 2)
		srvPlaceholder := "${" + srvLookupArgs[0] + "}"
		if !strings.Contains(rSpec, srvPlaceholder) || len(srvLookupArgs) != 2 {
			// No need to make an API call if there is no placeholder to replace
			// Alien set services lookups on all nodes
			continue
		}
		srvName := srvLookupArgs[1]
		srv, err := clientset.CoreV1().Services(namespace).Get(ctx, srvName, metav1.GetOptions{})
		if err != nil {
			return rSpec, errors.Wrapf(err, "failed to retrieve ClusterIP for service %q", srvName)
		}
		if srv.Spec.ClusterIP == "" || srv.Spec.ClusterIP == "None" {
			// Not supported
			return rSpec, errors.Errorf("failed to retrieve ClusterIP for service %q, (value=%q)", srvName, srv.Spec.ClusterIP)
		}
		rSpec = strings.Replace(rSpec, srvPlaceholder, srv.Spec.ClusterIP, -1)
	}
	return rSpec, nil
}

func replaceServiceIPInResourceSpec(ctx context.Context, clientset kubernetes.Interface, deploymentID, nodeName, namespace, rSpec string) (string, error) {
	serviceDepsLookups, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "service_dependency_lookups")
	if err != nil || serviceDepsLookups == nil || serviceDepsLookups.RawString() == "" {
		return rSpec, err
	}
	return replaceServiceDepLookups(ctx, clientset, namespace, rSpec, serviceDepsLookups.RawString())
}

// Return the external IP of a given node
func getExternalIPAdress(ctx context.Context, clientset kubernetes.Interface, nodeName string) (string, error) {
	node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "Failed to get node "+nodeName)
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeExternalIP {
			return addr.Address, nil
		}
	}
	return "", errors.New("Node " + nodeName + " don't have external IP adress")
}

func getVersion(clientset kubernetes.Interface) (string, error) {
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return version.String(), nil
}

// Wait for a kubernetes object to be completed. k8sObject is a pointer of a k8s object
func waitForYorcK8sObjectCompletion(ctx context.Context, deploymentID string, clientset kubernetes.Interface, k8sObject yorcK8sObject, namespace string) error {
	return wait.PollUntil(2*time.Second, func() (bool, error) {
		return k8sObject.isSuccessfullyDeployed(ctx, deploymentID, clientset, namespace)
	}, ctx.Done())

}

// Wait for a kubernetes object to be deleted. k8sObject is a pointer of a k8s object
func waitForYorcK8sObjectDeletion(ctx context.Context, clientset kubernetes.Interface, k8sObject yorcK8sObject, namespace string) error {
	return wait.PollUntil(2*time.Second, func() (bool, error) {
		return k8sObject.isSuccessfullyDeleted(ctx, "", clientset, namespace)
	}, ctx.Done())
}
