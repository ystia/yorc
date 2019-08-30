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
	"strings"
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

func Test_getK8sResourceNamespace(t *testing.T) {
	depID := "DepID"
	nsName := "my-namespace"
	type args struct {
		deploymentID string
		k8sResource  yorcK8sObject
	}
	tests := []struct {
		name      string
		args      args
		namespace string
		provided  bool
	}{
		{
			"Test no ns provided",
			args{depID, &yorcK8sDeployment{}},
			strings.ToLower(depID),
			false,
		},
		{
			"Test ns provided",
			args{depID, &yorcK8sService{ObjectMeta: metav1.ObjectMeta{Name: "pvcTest", Namespace: nsName}}},
			nsName,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := getK8sResourceNamespace(tt.args.deploymentID, tt.args.k8sResource)
			if got != tt.namespace {
				t.Errorf("getK8sResourceNamespace() namespace = %v, want %v", got, tt.namespace)
			}
			if got1 != tt.provided {
				t.Errorf("getK8sResourceNamespace() provided = %v, want %v", got1, tt.provided)
			}
		})
	}
}
