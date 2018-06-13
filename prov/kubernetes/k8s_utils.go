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
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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
