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
	"testing"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
	expected := "v0.0.0-master+$Format:%H$"
	if v != expected {
		t.Fatal("getVersion expected " + expected + ", got " + v)
	}
}

func TestGetExternalIPAdress(t *testing.T) {
	k8s := newTestSimpleK8s()
	nodeExtIP := "1.2.3.4"
	nodeName := "testNode"
	node := newTestSimpleNode(nodeName, nodeExtIP)
	ctx := context.Background()
	k8s.clientset.CoreV1().Nodes().Create(ctx, &node, metav1.CreateOptions{})
	ip, err := getExternalIPAdress(ctx, k8s.clientset, nodeName)
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
	ctx := context.Background()
	k8s.clientset.CoreV1().Nodes().Create(ctx, &node, metav1.CreateOptions{})
	ip, err := getExternalIPAdress(ctx, k8s.clientset, "randomNodeName")
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
	ctx := context.Background()
	k8s.clientset.CoreV1().Nodes().Create(ctx, &node, metav1.CreateOptions{})
	ip, err := getExternalIPAdress(ctx, k8s.clientset, "testNode")
	if err == nil || ip != "" {
		t.Fatal("Getting a node with no externalIP should raise error and return empty ip")
	}
}

/*
	Return a map of mocked k8s objects corresponding to their resource in

Currently support only : StatefulSet, PVC, Service and Deployment
*/
func mockSupportedResources() map[string]runtime.Object {
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "pvcTest", Namespace: "test-ns"}}
	dep := &v1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "deploymentTest", Namespace: "test-ns"}}
	sts := &v1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "statefulSetTest", Namespace: "test-ns"},
		Spec: v1.StatefulSetSpec{}}
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "serviceTest", Namespace: "test-ns"}}
	supportedRes := make(map[string]runtime.Object)
	supportedRes["persistentvolumeclaims"] = pvc
	supportedRes["deployments"] = dep
	supportedRes["statefulsets"] = sts
	supportedRes["services"] = svc
	return supportedRes
}

/*
	React to get on k8s objects. After 2 get if deleteObj boolean is set to true then it fakely delete it. If the boolean is set to false,

then it marked the object as succefully present or deployed for pods controllers.
If the API continue to receive GET, raise error by signaling the errorChan
*/
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

func fakeObjectCompletion(k8sObject yorcK8sObject, errorChan chan struct{}) k8stesting.ReactionFunc {
	getCount := 0
	return func(action k8stesting.Action) (bool, runtime.Object, error) {
		if action.GetVerb() != "get" {
			close(errorChan)
		}
		getCount++
		if getCount > 5 {
			close(errorChan)
		} else if getCount > 2 {
			//Mark obj deployed

			switch obj := k8sObject.(type) {
			case *yorcK8sDeployment:
				obj.Status.AvailableReplicas = *obj.Spec.Replicas
				return true, obj.getObjectRuntime(), nil
			case *yorcK8sService:
				return true, obj.getObjectRuntime(), nil
			case *yorcK8sPersistentVolumeClaim:
				obj.Status.Phase = corev1.ClaimBound
				return true, obj.getObjectRuntime(), nil
			case *yorcK8sStatefulSet:
				obj.Status.ReadyReplicas = *obj.Spec.Replicas
				return true, obj.getObjectRuntime(), nil
			default:
				close(errorChan)
			}
			return true, nil, nil
		}
		return true, nil, nil
	}
}

func fakeObjectScale(k8sObject yorcK8sObject, errorChan chan struct{}) k8stesting.ReactionFunc {
	getCount := 0
	return func(action k8stesting.Action) (bool, runtime.Object, error) {
		if action.GetVerb() != "get" {
			close(errorChan)
		}
		getCount++
		if getCount > 5 {
			close(errorChan)
		} else if getCount > 2 {
			//Mark obj deployed

			switch obj := k8sObject.(type) {
			case *yorcK8sDeployment:
				obj.Status.AvailableReplicas = *obj.Spec.Replicas
				return true, obj.getObjectRuntime(), nil
			case *yorcK8sStatefulSet:
				obj.Status.ReadyReplicas = *obj.Spec.Replicas
				return true, obj.getObjectRuntime(), nil
			default:
				close(errorChan)
			}
			return true, nil, nil
		}
		return true, nil, nil
	}
}

func fakeObjectDeletion(k8sObject yorcK8sObject, errorChan chan struct{}) k8stesting.ReactionFunc {
	getCount := 0
	return func(action k8stesting.Action) (bool, runtime.Object, error) {
		if action.GetVerb() != "get" {
			close(errorChan)
		}
		getCount++
		if getCount > 5 {
			close(errorChan)
		} else if getCount > 1 {
			//Mark obj removed
			return true, nil, apierrors.NewNotFound(action.GetResource().GroupResource(), action.GetResource().Resource)
			//return true, nil, nil
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
			got, got1 := getNamespace(tt.args.deploymentID, tt.args.k8sResource.getObjectMeta())
			if got != tt.namespace {
				t.Errorf("getK8sResourceNamespace() namespace = %v, want %v", got, tt.namespace)
			}
			if got1 != tt.provided {
				t.Errorf("getK8sResourceNamespace() provided = %v, want %v", got1, tt.provided)
			}
		})
	}
}

func Test_podControllersInNamespace(t *testing.T) {
	nsName := "my-test-namespace"
	type args struct {
		namespace string
	}
	ctx := context.Background()
	tests := []struct {
		name    string
		setup   func() kubernetes.Interface
		args    args
		want    int
		wantErr bool
	}{
		{
			"Test no controller left",
			func() kubernetes.Interface {
				k8s := newTestSimpleK8s()
				return k8s.clientset
			},
			args{namespace: nsName},
			0, false,
		},
		{
			"Test one deployment left",
			func() kubernetes.Interface {
				k8s := newTestSimpleK8s()
				k8s.clientset.AppsV1().Deployments(nsName).Create(
					ctx,
					&v1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "my-deployment", Namespace: nsName}},
					metav1.CreateOptions{})
				return k8s.clientset
			},
			args{namespace: nsName},
			1, false,
		},
		{
			"Test one deployment & 2 statefulSets left",
			func() kubernetes.Interface {
				k8s := newTestSimpleK8s()
				k8s.clientset.AppsV1().StatefulSets(nsName).Create(
					ctx,
					&v1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "my-statefulset-1", Namespace: nsName}},
					metav1.CreateOptions{})
				k8s.clientset.AppsV1().StatefulSets(nsName).Create(
					ctx,
					&v1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "my-statefulset-2", Namespace: nsName}},
					metav1.CreateOptions{})
				k8s.clientset.AppsV1().Deployments(nsName).Create(
					ctx,
					&v1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "my-deployment", Namespace: nsName}},
					metav1.CreateOptions{})
				return k8s.clientset
			},
			args{namespace: nsName},
			3, false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := podControllersInNamespace(ctx, tt.setup(), tt.args.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("podControllersInNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("podControllersInNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_replaceServiceDepLookups(t *testing.T) {
	specWithoutPlaceHolders := `Something`
	specWithPlaceHolders := `something ${PLACEHOLDER1} something else ${PLACEHOLDER2}`
	placeholderToService := map[string]string{"PLACEHOLDER1": "service1", "PLACEHOLDER2": "service2"}
	serviceToIP := map[string]string{"service1": "10.0.0.1", "service2": "10.0.0.2"}
	replaceMapInPlaceHolders := func(inputString string, placeholderToService, serviceToIP map[string]string) string {
		for p, s := range placeholderToService {
			i, ok := serviceToIP[s]
			if !ok {
				continue
			}
			inputString = strings.ReplaceAll(inputString, "${"+p+"}", i)
		}
		return inputString
	}

	type args struct {
		clientset          kubernetes.Interface
		namespace          string
		rSpec              string
		serviceDepsLookups string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"TestWithoutPlaceHoldersAndServiceDepLookups", args{fake.NewSimpleClientset(), "", specWithoutPlaceHolders, ""}, specWithoutPlaceHolders, false},
		{"TestWithoutPlaceHoldersButWithServiceDepLookups", args{fake.NewSimpleClientset(), "", specWithoutPlaceHolders, "PLACEHOLDER1:service1,PLACEHOLDER2:service2"}, specWithoutPlaceHolders, false},
		{"TestWithPlaceHoldersButWithoutServiceDepLookups", args{fake.NewSimpleClientset(), "", specWithPlaceHolders, ""}, specWithPlaceHolders, false},
		{"TestWithPlaceHoldersAndServiceDepLookups", args{
			func() kubernetes.Interface {
				c := fake.NewSimpleClientset()
				c.PrependReactor("get", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					if !action.Matches("get", "services") {
						return false, nil, nil
					}
					getAction := action.(k8stesting.GetAction)
					if ip, ok := serviceToIP[getAction.GetName()]; ok {
						return true, &corev1.Service{
							Spec: corev1.ServiceSpec{
								ClusterIP: ip,
							},
						}, nil
					}
					return false, nil, nil
				})
				return c
			}(),
			"", specWithPlaceHolders, "PLACEHOLDER1:service1,PLACEHOLDER2:service2"}, replaceMapInPlaceHolders(specWithPlaceHolders, placeholderToService, serviceToIP), false},
		{"TestClusterIPNone", args{
			func() kubernetes.Interface {
				c := fake.NewSimpleClientset()
				c.PrependReactor("get", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					if !action.Matches("get", "services") {
						return false, nil, nil
					}
					return true, &corev1.Service{
						Spec: corev1.ServiceSpec{
							ClusterIP: corev1.ClusterIPNone,
						},
					}, nil
				})
				return c
			}(),
			"", specWithPlaceHolders, "PLACEHOLDER1:service1,PLACEHOLDER2:service2"}, "", true},
		{"TestClusterIPEmpty", args{
			func() kubernetes.Interface {
				c := fake.NewSimpleClientset()
				c.PrependReactor("get", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					if !action.Matches("get", "services") {
						return false, nil, nil
					}
					return true, &corev1.Service{
						Spec: corev1.ServiceSpec{
							ClusterIP: "",
						},
					}, nil
				})
				return c
			}(),
			"", specWithPlaceHolders, "PLACEHOLDER1:service1,PLACEHOLDER2:service2"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replaceServiceDepLookups(context.Background(), tt.args.clientset, tt.args.namespace, tt.args.rSpec, tt.args.serviceDepsLookups)
			if (err != nil) != tt.wantErr {
				t.Errorf("replaceServiceDepLookups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && got != tt.want {
				t.Errorf("replaceServiceDepLookups() = %v, want %v", got, tt.want)
			}
		})
	}
}
