// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
)

//Interface to implement for new supported objects in K8s
type yorcK8sObject interface {
	// Operations for yorc kubernetes objects
	createResource(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) error
	deleteResource(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) error
	scaleResource(ctx context.Context, e *execution, clientset kubernetes.Interface, namespace string) error
	setAttributes(ctx context.Context, e *execution) error
	// Return a boolean telling if the resource is correctly deployed on K8s and error message if necessary
	isSuccessfullyDeployed(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) (bool, error)
	// Return if the specified resource is correctly deleted
	isSuccessfullyDeleted(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) (bool, error)
	// unmarshal the resourceSpec into struct
	unmarshalResource(ctx context.Context, e *execution, deploymentID string, clientset kubernetes.Interface, rSpec string) error
	streamLogs(ctx context.Context, deploymentID string, clientset kubernetes.Interface)
	getObjectMeta() metav1.ObjectMeta
	// Implem of the stringer interface
	fmt.Stringer

	getObjectRuntime() runtime.Object
}

// Supported k8s resources
type yorcK8sPersistentVolumeClaim corev1.PersistentVolumeClaim
type yorcK8sService corev1.Service
type yorcK8sDeployment v1.Deployment
type yorcK8sStatefulSet v1.StatefulSet

/*
	----------------------------------------------
	| 			PersistentVolumeClaim			 |
	----------------------------------------------
*/
//Implem of yorcK8sObject interface for PersistentVolumeClaim
func (yorcPVC *yorcK8sPersistentVolumeClaim) unmarshalResource(ctx context.Context, e *execution, deploymentID string, clientset kubernetes.Interface, rSpec string) error {
	return json.Unmarshal([]byte(rSpec), &yorcPVC)
}

func (yorcPVC *yorcK8sPersistentVolumeClaim) getObjectMeta() metav1.ObjectMeta {
	return yorcPVC.ObjectMeta
}

func (yorcPVC *yorcK8sPersistentVolumeClaim) createResource(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) error {
	pvc := corev1.PersistentVolumeClaim(*yorcPVC)
	_, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, &pvc, metav1.CreateOptions{})
	return err
}

func (yorcPVC *yorcK8sPersistentVolumeClaim) deleteResource(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) error {
	pvc := corev1.PersistentVolumeClaim(*yorcPVC)
	return clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, pvc.Name, metav1.DeleteOptions{})
}

func (yorcPVC *yorcK8sPersistentVolumeClaim) scaleResource(ctx context.Context, e *execution, clientset kubernetes.Interface, namespace string) error {
	return errors.New("Scale operation is not supported by PersistentVolumeClaims")
}

func (yorcPVC *yorcK8sPersistentVolumeClaim) setAttributes(ctx context.Context, e *execution) error {
	return nil
}

func (yorcPVC *yorcK8sPersistentVolumeClaim) isSuccessfullyDeployed(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) (bool, error) {
	pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, yorcPVC.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	if pvc == nil {
		return false, nil
	}
	if pvc.Status.Phase == corev1.ClaimBound {
		return true, nil
	}
	return false, nil
}

func (yorcPVC *yorcK8sPersistentVolumeClaim) isSuccessfullyDeleted(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) (bool, error) {
	_, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, yorcPVC.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (yorcPVC *yorcK8sPersistentVolumeClaim) String() string {
	return "YorcPersistentVolumeClaim"
}

func (yorcPVC *yorcK8sPersistentVolumeClaim) getObjectRuntime() runtime.Object {
	pvc := corev1.PersistentVolumeClaim(*yorcPVC)
	return &pvc
}

func (yorcPVC *yorcK8sPersistentVolumeClaim) streamLogs(ctx context.Context, deploymentID string, clientset kubernetes.Interface) {
}

/*
	----------------------------------------------
	| 				Deployment					 |
	----------------------------------------------
*/
func (yorcDep *yorcK8sDeployment) unmarshalResource(ctx context.Context, e *execution, deploymentID string, clientset kubernetes.Interface, rSpec string) error {
	err := json.Unmarshal([]byte(rSpec), &yorcDep)
	if err != nil {
		return err
	}
	ns, _ := getNamespace(e.deploymentID, yorcDep.ObjectMeta)
	rSpec, err = replaceServiceIPInResourceSpec(ctx, clientset, e.deploymentID, e.nodeName, ns, rSpec)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(rSpec), &yorcDep)
}

func (yorcDep *yorcK8sDeployment) getObjectMeta() metav1.ObjectMeta {
	return yorcDep.ObjectMeta
}

func (yorcDep *yorcK8sDeployment) createResource(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) error {
	deploy := v1.Deployment(*yorcDep)
	_, err := clientset.AppsV1().Deployments(namespace).Create(ctx, &deploy, metav1.CreateOptions{})
	return err
}

func (yorcDep *yorcK8sDeployment) deleteResource(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) error {
	deploy := v1.Deployment(*yorcDep)
	deletePolicy := metav1.DeletePropagationForeground
	var gracePeriod int64 = 5
	return clientset.AppsV1().Deployments(namespace).Delete(ctx, deploy.Name, metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod, PropagationPolicy: &deletePolicy})
}

func (yorcDep *yorcK8sDeployment) scaleResource(ctx context.Context, e *execution, clientset kubernetes.Interface, namespace string) error {
	deploy := v1.Deployment(*yorcDep)
	expectedInstances, err := e.getExpectedInstances()
	if err != nil {
		return err
	}
	deploy.Spec.Replicas = &expectedInstances

	_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, &deploy, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to scale kubernetes deployment")
	}
	return nil
}

func (yorcDep *yorcK8sDeployment) setAttributes(ctx context.Context, e *execution) (err error) {
	return deployments.SetAttributeForAllInstances(ctx, e.deploymentID, e.nodeName, "replicas", fmt.Sprint(*yorcDep.Spec.Replicas))
}

func (yorcDep *yorcK8sDeployment) isSuccessfullyDeployed(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) (bool, error) {
	dep, err := clientset.AppsV1().Deployments(namespace).Get(ctx, yorcDep.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	if dep == nil {
		return false, nil
	}
	if dep.Status.AvailableReplicas == *yorcDep.Spec.Replicas {
		return true, nil
	}

	if failed, msg := isDeploymentFailed(dep); failed {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, deploymentID).Registerf("Kubernetes deployment %q failed: %s", yorcDep.Name, msg)
		return false, errors.Errorf("Kubernetes deployment %q: %s", yorcDep.Name, msg)
	}

	return false, nil
}

func (yorcDep *yorcK8sDeployment) isSuccessfullyDeleted(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) (bool, error) {
	_, err := clientset.AppsV1().Deployments(namespace).Get(ctx, yorcDep.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (yorcDep *yorcK8sDeployment) String() string {
	return "YorcDeployment"
}

func (yorcDep *yorcK8sDeployment) getObjectRuntime() runtime.Object {
	deploy := v1.Deployment(*yorcDep)
	return &deploy
}

func (yorcDep *yorcK8sDeployment) streamLogs(ctx context.Context, deploymentID string, clientset kubernetes.Interface) {
	deploy := v1.Deployment(*yorcDep)
	streamDeploymentLogs(ctx, deploymentID, clientset, &deploy)
}

/*
	----------------------------------------------
	| 				StatefulSet					 |
	----------------------------------------------
*/
func (yorcSts *yorcK8sStatefulSet) unmarshalResource(ctx context.Context, e *execution, deploymentID string, clientset kubernetes.Interface, rSpec string) error {
	err := json.Unmarshal([]byte(rSpec), &yorcSts)
	if err != nil {
		return err
	}
	ns, _ := getNamespace(e.deploymentID, yorcSts.ObjectMeta)
	rSpec, err = replaceServiceIPInResourceSpec(ctx, clientset, e.deploymentID, e.nodeName, ns, rSpec)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(rSpec), &yorcSts)
}

func (yorcSts *yorcK8sStatefulSet) getObjectMeta() metav1.ObjectMeta {
	return yorcSts.ObjectMeta
}

func (yorcSts *yorcK8sStatefulSet) createResource(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) error {
	sts := v1.StatefulSet(*yorcSts)
	_, err := clientset.AppsV1().StatefulSets(namespace).Create(ctx, &sts, metav1.CreateOptions{})
	return err
}

func (yorcSts *yorcK8sStatefulSet) deleteResource(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) error {
	sts := v1.StatefulSet(*yorcSts)
	deletePolicy := metav1.DeletePropagationForeground
	var gracePeriod int64 = 5
	return clientset.AppsV1().StatefulSets(namespace).Delete(ctx, sts.Name, metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod, PropagationPolicy: &deletePolicy})
}

func (yorcSts *yorcK8sStatefulSet) scaleResource(ctx context.Context, e *execution, clientset kubernetes.Interface, namespace string) error {
	sts := v1.StatefulSet(*yorcSts)
	expectedInstances, err := e.getExpectedInstances()
	if err != nil {
		return err
	}
	sts.Spec.Replicas = &expectedInstances

	_, err = clientset.AppsV1().StatefulSets(namespace).Update(ctx, &sts, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to scale kubernetes statefulset")
	}
	return nil
}

func (yorcSts *yorcK8sStatefulSet) setAttributes(ctx context.Context, e *execution) error {
	return deployments.SetAttributeForAllInstances(ctx, e.deploymentID, e.nodeName, "replicas", fmt.Sprint(*yorcSts.Spec.Replicas))
}

func (yorcSts *yorcK8sStatefulSet) isSuccessfullyDeployed(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) (bool, error) {
	stfs, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, yorcSts.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	if stfs == nil {
		return false, nil
	}
	if stfs.Status.ReadyReplicas == *yorcSts.Spec.Replicas {
		return true, nil
	}
	return false, nil
}

func (yorcSts *yorcK8sStatefulSet) isSuccessfullyDeleted(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) (bool, error) {
	_, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, yorcSts.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (yorcSts *yorcK8sStatefulSet) String() string {
	return "YorcStatefulSet"
}

func (yorcSts *yorcK8sStatefulSet) getObjectRuntime() runtime.Object {
	sts := v1.StatefulSet(*yorcSts)
	return &sts
}

func (yorcSts *yorcK8sStatefulSet) streamLogs(ctx context.Context, deploymentID string, clientset kubernetes.Interface) {
	// TODO : stream logs for this controller
}

/*
	----------------------------------------------
	| 					Service					 |
	----------------------------------------------
*/
func (yorcSvc *yorcK8sService) unmarshalResource(ctx context.Context, e *execution, deploymentID string, clientset kubernetes.Interface, rSpec string) error {
	return json.Unmarshal([]byte(rSpec), &yorcSvc)
}

func (yorcSvc *yorcK8sService) getObjectMeta() metav1.ObjectMeta {
	return yorcSvc.ObjectMeta
}

func (yorcSvc *yorcK8sService) createResource(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) error {
	svc := corev1.Service(*yorcSvc)
	_, err := clientset.CoreV1().Services(namespace).Create(ctx, &svc, metav1.CreateOptions{})
	return err
}

func (yorcSvc *yorcK8sService) deleteResource(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) error {
	svc := corev1.Service(*yorcSvc)
	return clientset.CoreV1().Services(namespace).Delete(ctx, svc.Name, metav1.DeleteOptions{})
}

func (yorcSvc *yorcK8sService) scaleResource(ctx context.Context, e *execution, clientset kubernetes.Interface, namespace string) error {
	return errors.New("Scale operation not supported by Services")
}

func (yorcSvc *yorcK8sService) setAttributes(ctx context.Context, e *execution) (err error) {

	for _, val := range yorcSvc.Spec.Ports {
		if val.NodePort != 0 {
			str := fmt.Sprintf("%d", val.NodePort)
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).Registerf("%s : %s: %d:%d mapped to %s", yorcSvc.Name, val.Name, val.Port, val.TargetPort.IntVal, str)
			err = deployments.SetAttributeForAllInstances(ctx, e.deploymentID, e.nodeName, "k8s_service_url", str)
			if err != nil {
				return errors.Wrap(err, "Failed to set attribute")
			}
			err = deployments.SetAttributeForAllInstances(ctx, e.deploymentID, e.nodeName, "node_port", strconv.Itoa(int(val.NodePort)))
			if err != nil {
				return errors.Wrap(err, "Failed to set attribute")
			}
		}
	}
	return nil
}

func (yorcSvc *yorcK8sService) isSuccessfullyDeployed(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) (bool, error) {
	_, err := clientset.CoreV1().Services(namespace).Get(ctx, yorcSvc.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (yorcSvc *yorcK8sService) isSuccessfullyDeleted(ctx context.Context, deploymentID string, clientset kubernetes.Interface, namespace string) (bool, error) {
	_, err := clientset.CoreV1().Services(namespace).Get(ctx, yorcSvc.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (yorcSvc *yorcK8sService) String() string {
	return "YorcService"
}

func (yorcSvc *yorcK8sService) getObjectRuntime() runtime.Object {
	svc := corev1.Service(*yorcSvc)
	return &svc
}

func (yorcSvc *yorcK8sService) streamLogs(ctx context.Context, deploymentID string, clientset kubernetes.Interface) {
}
