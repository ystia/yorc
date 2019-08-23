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
	"sync"
	"testing"

	appsv1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

type k8s struct {
	clientset kubernetes.Interface
}

func newTestSimpleK8s() *k8s {
	client := k8s{}
	client.clientset = fake.NewSimpleClientset()
	return &client
}

func newTestK8s() *k8s {
	client := k8s{
		clientset: &fake.Clientset{},
	}
	return &client
}

func namespaceRuntimeObject(namespace string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
}

func newTestSimpleNode(nodeName string, extIP string) corev1.Node {
	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
	}
	if extIP == "" {
		return node
	}
	node.Status.Addresses = []corev1.NodeAddress{{Type: corev1.NodeExternalIP,
		Address: extIP},
	}
	return node
}

func TestGetVersionDefault(t *testing.T) {
	k8s := newTestSimpleK8s()
	v, err := getVersion(k8s.clientset)
	if err != nil {
		t.Fatal("getVersion should not raise an error")
	}
	expected := "v0.0.0-master+$Format:%h$"
	if v != expected {
		t.Fatal("getVersion should return " + expected)
	}
}

func TestGetExternalIPAdress(t *testing.T) {
	k8s := newTestSimpleK8s()
	nodeExtIP := "1.2.3.4"
	nodeName := "testNode"
	node := newTestSimpleNode(nodeName, nodeExtIP)
	k8s.clientset.CoreV1().Nodes().Create(&node)
	ip, err := getExternalIPAdress(k8s.clientset, nodeName)
	if err != nil {
		t.Fatal("should not raise an error when IP is present", err)
	}
	if ip != nodeExtIP {
		t.Fatal("IP returned by function (" + ip + ") should be " + nodeExtIP)
	}
}

func TestGetExternalIPAdressWrongNodeName(t *testing.T) {
	k8s := newTestSimpleK8s()
	nodeExtIP := "1.2.3.4"
	nodeName := "testNode"
	node := newTestSimpleNode(nodeName, nodeExtIP)
	k8s.clientset.CoreV1().Nodes().Create(&node)
	ip, err := getExternalIPAdress(k8s.clientset, "randomNodeName")
	if err == nil || ip != "" {
		t.Fatal("Getting a non existing node should raise an error and return empty string")
	}
}

func TestGetNoneExternalIPAdress(t *testing.T) {
	k8s := newTestSimpleK8s()
	nodeExtIP := ""
	nodeName := "testNode"
	node := newTestSimpleNode(nodeName, nodeExtIP)
	t.Log(node.Status.Addresses)
	k8s.clientset.CoreV1().Nodes().Create(&node)
	ip, err := getExternalIPAdress(k8s.clientset, "testNode")
	if err == nil || ip != "" {
		t.Fatal("Getting a node with no externalIP should raise error and return empty ip")
	}
}

/* DEPRECATED Use more generic test TestWaitForK8sObjectDeletion instead */
func TestWaitForPVCDeletionAndDeleted(t *testing.T) {
	k8s := newTestK8s()
	errorChan := make(chan struct{})
	finishedChan := make(chan struct{})
	getCount := 0
	ctx := context.Background()
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "pvcTest", Namespace: "test-ns"}}
	//Simulate a deletion in progress : wait for 2 get that return the volume and then fakely delete it
	//If the API continue to receive GET, raise error
	k8s.clientset.(*fake.Clientset).Fake.AddReactor("get", "persistentvolumeclaims", func(action k8stesting.Action) (bool, runtime.Object, error) {
		getCount++
		if getCount > 5 {
			close(errorChan)
		} else if getCount > 2 {
			return true, nil, apierrors.NewNotFound(action.GetResource().GroupResource(), action.GetResource().Resource)
		}
		return true, pvc, nil
	})
	go func() {
		err := waitForK8sObjectDeletion(ctx, k8s.clientset, pvc)
		if err != nil {
			t.Logf("Error : %s", err.Error())
			t.Fatal("Deleted pvc should not raise an error")
		}
		close(finishedChan)
	}()
	select {
	case <-errorChan:
		t.Fatal("Function waitForPVCDeletion is still polling API even though it is deleted")
	case <-finishedChan:
		//Wait for test to be well done
	}
}

/* Test function that wait for K8s object deletion.  */
func TestWaitForK8sObjectDeletion(t *testing.T) {
	k8s := newTestK8s()
	ctx := context.Background()
	supportedResources := mockSupportedResources()
	// Waitgroup for parallel execution
	var wg sync.WaitGroup
	wg.Add(len(supportedResources))

	for resourceName, resource := range supportedResources {
		go func(resourceName string, resource runtime.Object) {
			defer wg.Done()
			t.Logf("Testing deletion of %s\n", resourceName)
			errorChan := make(chan struct{})
			finishedChan := make(chan struct{})
			k8s.clientset.(*fake.Clientset).Fake.AddReactor("get", resourceName, fakeGetObjectReaction(resource, errorChan, true))
			go func() {
				err := waitForK8sObjectDeletion(ctx, k8s.clientset, resource)
				if err != nil {
					t.Logf("Error : %s", err.Error())
					t.Fatalf("Deleting %s should not raise an error", resourceName)
				}
				close(finishedChan)
			}()
			select {
			case <-errorChan:
				t.Fatalf("Function is still polling API even though %s it is deleted", resourceName)
			case <-finishedChan:
				//Wait for test to be well done
				t.Logf("OK for %s.\n", resourceName)
			}
		}(resourceName, resource)
	}
	wg.Wait()
}

/* Return a map of mocked k8s objects corresponding to their resource in
Currently support only : StatefulSet, PVC, Service and Deployment */
func mockSupportedResources() map[string]runtime.Object {
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "pvcTest", Namespace: "test-ns"}}
	dep := &v1beta1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "deploymentTest", Namespace: "test-ns"}}
	sts := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "statefulSetTest", Namespace: "test-ns"},
		Spec: appsv1.StatefulSetSpec{}}
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "serviceTest", Namespace: "test-ns"}}
	supportedRes := make(map[string]runtime.Object)
	supportedRes["persistentvolumeclaims"] = pvc
	supportedRes["deployments"] = dep
	supportedRes["statefulsets"] = sts
	supportedRes["services"] = svc
	return supportedRes
}

/*  React to get on k8s objects. After 2 get if deleteObj boolean is set to true then it fakely delete it. If the boolean is set to false,
then it marked the object as succefully present or deployed for pods controllers.
If the API continue to receive GET, raise error by signaling the errorChan */
func fakeGetObjectReaction(k8sObject runtime.Object, errorChan chan struct{}, deleteObj bool) k8stesting.ReactionFunc {
	getCount := 0
	return func(action k8stesting.Action) (bool, runtime.Object, error) {
		// TODO: manage error another way
		if action.GetVerb() != "get" {
			close(errorChan)
		}

		getCount++
		if getCount > 5 {
			close(errorChan)
		} else if getCount > 2 {
			if deleteObj {
				return true, nil, apierrors.NewNotFound(action.GetResource().GroupResource(), action.GetResource().Resource)
			}
			return true, k8sObject, nil

		}
		return true, nil, nil
	}
}
