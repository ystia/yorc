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
	"strings"
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/ystia/yorc/events"
)

func isDeploymentFailed(clientset *kubernetes.Clientset, deployment *v1beta1.Deployment) (bool, string) {
	for _, c := range deployment.Status.Conditions {
		if c.Type == v1beta1.DeploymentReplicaFailure && c.Status == corev1.ConditionTrue {
			return true, c.Message
		} else if c.Type == v1beta1.DeploymentProgressing && c.Status == corev1.ConditionFalse {
			return true, c.Message
		}
	}
	return false, ""
}

func waitForDeploymentCompletion(ctx context.Context, deploymentID string, clientset *kubernetes.Clientset, deployment *v1beta1.Deployment) error {
	return wait.PollUntil(2*time.Second, func() (bool, error) {
		deployment, err := clientset.ExtensionsV1beta1().Deployments(deployment.Namespace).Get(deployment.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if deployment.Status.AvailableReplicas == *deployment.Spec.Replicas {
			return true, nil
		}

		if failed, msg := isDeploymentFailed(clientset, deployment); failed {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.ERROR, deploymentID).Registerf("Kubernetes deployment %q failed: %s", deployment.Name, msg)
			return false, errors.Errorf("Kubernetes deployment %q: ", deployment.Name, msg)
		}
		return false, nil
	}, ctx.Done())
}

func streamDeploymentLogs(ctx context.Context, deploymentID string, clientset *kubernetes.Clientset, deployment *v1beta1.Deployment) {
	go func() {
		watcher, err := clientset.Events(deployment.Namespace).Watch(metav1.ListOptions{})
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).Registerf("Failed to monitor Kubernetes deployment events: %v", err)
			return
		}
		defer watcher.Stop()
		for {
			select {
			case <-ctx.Done():
				return

			case e, ok := <-watcher.ResultChan():
				if !ok {
					events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).RegisterAsString("Failed to monitor Kubernetes deployment events: watch channel closed")
					return
				}
				if e.Type == watch.Error {
					if status, ok := e.Object.(*metav1.Status); ok {
						events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).Registerf("Failed to monitor Kubernetes deployment events: %s: %s", status.Reason, status.Message)
						return
					}
					events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).Registerf("Failed to monitor Kubernetes deployment events:Received unexpected error: %#v", e.Object)
					return
				}
				if event, ok := e.Object.(*corev1.Event); ok {
					if ok, err := isChildOf(clientset, deployment.UID, referenceFromObjectReference(event.InvolvedObject)); err == nil && ok {
						switch e.Type {
						case watch.Added, watch.Modified:
							level := events.DEBUG
							if strings.ToLower(event.Type) == "warning" {
								level = events.WARN
							}
							events.WithContextOptionalFields(ctx).NewLogEntry(level, deploymentID).Registerf("%s (source: component: %q, host: %q)", event.Message, event.Source.Component, event.Source.Host)
						case watch.Deleted:
							// Deleted events are silently ignored.
						default:
							events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).Registerf("Unknown watchUpdate.Type: %#v", e.Type)
						}
					}
				} else {
					events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).Registerf("Wrong object received: %v", e)
				}
			}
		}
	}()
}

func isChildOf(clientset *kubernetes.Clientset, parent types.UID, ref reference) (bool, error) {
	if ref.UID == parent {
		return true, nil
	}
	var om metav1.Object
	var err error
	switch strings.ToLower(ref.Kind) {
	case "pod":
		om, err = clientset.Pods(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
	case "replicaset":
		om, err = clientset.ReplicaSets(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
	case "deployment":
		om, err = clientset.ExtensionsV1beta1().Deployments(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
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
