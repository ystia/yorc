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
	"fmt"
	"path"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tosca"
)

var JSONvalidDeployment = `
{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
     "name": "test-deploy"
  },
  "spec": {
     "replicas": 3,
     "template": {
      "metadata": {
       "labels": {
        "app": "yorc"
       }
      },
      "spec": {
       "containers": [
        {
           "name": "yorc-container",
           "image": "ystia/yorc:3.0.2",
           "env": [
            {
             "name": "POD_IP",
             "valueFrom": {
              "fieldRef": {
                 "fieldPath": "status.podIP"
              }
             }
            },
            {
             "name": "NAMESPACE",
             "valueFrom": {
              "fieldRef": {
                 "fieldPath": "metadata.namespace"
              }
             }
            },
            {
             "name": "POD_NAME",
             "valueFrom": {
              "fieldRef": {
                 "fieldPath": "metadata.name"
              }
             }
            },
            {
             "name": "YORC_LOG",
             "value": "DEBUG"
            }
           ]
        }
       ]
      }
     }
  }
 }`

var JSONvalidStatefulSet = `
{
	"metadata" : {
	  "name" : "test-sts"
	},
	"apiVersion" : "apps/v1",
	"kind" : "StatefulSet",
	"spec" : {
	  "template" : {
		"metadata" : {
		  "labels" : {
			"app" : "yorcdeployment-973d85c6b920"
		  }
		},
		"spec" : {
		  "volumes" : [ {
			"name" : "volume",
			"persistentVolumeClaim" : { }
		  } ],
		  "containers" : [ {
			"image" : "ystia/yorc:3.0.2",
			"name" : "yorc--1429866314",
			"resources" : {
			  "requests" : {
				"memory" : 128000000,
				"cpu" : 0.3
			  }
			},
			"ports" : [ {
			  "name" : "yorc-server",
			  "containerPort" : 8800
			}, {
			  "name" : "consul-ui",
			  "containerPort" : 8500
			} ],
			"env" : [ {
			  "name" : "YORC_LOG",
			  "value" : "NO_DEBUG"
			} ]
		  } ]
		}
	  },
	  "replicas" : 3
	}
  }
`

var JSONvalidPVC = `
{
  "apiVersion" : "v1",
  "kind" : "PersistentVolumeClaim",
  "metadata" : {
    "name" : "test-pvc"
    },
  "spec" : {
    "resources" : {
    "requests" : {
      "storage" : 5000000000
    }
    },
    "accessModes" : [ "ReadWriteOnce" ]
  }
  }
 `

var JSONvalidService = `
 {
  "apiVersion" : "v1",
  "kind" : "Service",
  "metadata" : {
    "name" : "test-service"
    },
  "spec" : {
    "selector" : {
    "app" : "yorcdeployment-1623552477"
    },
    "ports" : [ {
    "port" : 8800,
    "name" : "yorc-server",
    "targetPort" : "yorc-server"
    }, {
    "port" : 8500,
    "name" : "consul-ui",
	"targetPort" : "consul-ui",
	"nodePort": 8505
    } ],
    "type" : "NodePort"
  }
  }
 `

var JSONinvalidService = `
{
	"apiVersion" : "v1",
	"kind" : MissingQuoteService",
	"metadata" : {
	  "name" : "yorc-yorcdeployment-service-1116022612"
	  },
	"spec" : {
	  "selector" : {
	  "app" : "yorcdeployment-1623552477"
	  },
	  "ports" : [ {
	  "port" : 8800,
	  "name" : "yorc-server",
	  "targetPort" : "yorc-server"
	  }, {
	  "port" : 8500,
	  "name" : "consul-ui",
	  "targetPort" : "consul-ui"
	  } ],
	  "type" : "NodePort"
	}
	}
   `

type testResource struct {
	K8sObj        yorcK8sObject
	rSpec         string
	resourceGroup string
}

func getSupportedResourceAndJSON() []testResource {
	supportedRes := []testResource{
		{
			&yorcK8sDeployment{},
			JSONvalidDeployment,
			"deployments",
		},
		{
			&yorcK8sPersistentVolumeClaim{},
			JSONvalidPVC,
			"persistentvolumeclaims",
		},
		{
			&yorcK8sService{},
			JSONvalidService,
			"services",
		},
		{
			&yorcK8sStatefulSet{},
			JSONvalidStatefulSet,
			"statefulsets",
		},
	}
	return supportedRes
}

func getScalableResourceAndJSON() []testResource {
	supportedRes := []testResource{
		{
			&yorcK8sDeployment{},
			JSONvalidDeployment,
			"deployments",
		},
		{
			&yorcK8sStatefulSet{},
			JSONvalidStatefulSet,
			"statefulsets",
		},
	}
	return supportedRes
}

func Test_execution_invalid_JSON(t *testing.T) {
	tests := []struct {
		name        string
		k8sResource yorcK8sObject
		rSpec       string
		wantErr     bool
	}{
		{
			"Test no rSpec",
			&yorcK8sDeployment{},
			" ",
			true,
		},
		// {
		// 	"Test wrong rSpec",
		// 	&yorcK8sDeployment{},
		// 	JSONvalidPVC,
		// 	true,
		// },
		{
			"Test invalid JSON rSpec",
			&yorcK8sService{},
			JSONinvalidService,
			true,
		},
	}
	ctx := context.Background()
	deploymentID := "Dep-ID"

	e := &execution{
		deploymentID: deploymentID,
	}
	k8s := newTestSimpleK8s()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.k8sResource.unmarshalResource(ctx, e, deploymentID, k8s.clientset, tt.rSpec); (err != nil) != tt.wantErr {
				t.Errorf("yorcK8sObject.unmarshalResource error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}

func Test_execution_scale_resources(t *testing.T) {
	t.Skip()
	deploymentID := "Dep-ID"
	e := &execution{
		deploymentID: deploymentID,
		taskID:       "Task-ID",
	}
	wantErr := false
	k8s := newTestK8s()
	ctx := context.Background()
	operationType := k8sScaleOperation
	tasks.SetTaskData(e.taskID, "inputs/EXPECTED_INSTANCES", strconv.Itoa(int(3)))

	resources := getScalableResourceAndJSON()

	deployTestResources(ctx, e, k8s, resources)
	var wg sync.WaitGroup
	wg.Add(len(resources))

	for _, testRes := range resources {
		go func(testRes testResource) {
			defer wg.Done()
			errorChan := make(chan struct{})
			okChan := make(chan struct{})
			k8s.clientset.(*fake.Clientset).Fake.AddReactor("get", testRes.resourceGroup, fakeObjectScale(testRes.K8sObj, errorChan))
			t.Run("Test scale resources "+testRes.K8sObj.String(), func(t *testing.T) {
				if err := e.manageKubernetesResource(context.Background(), k8s.clientset, nil, testRes.K8sObj, operationType, true); (err != nil) != wantErr {
					t.Errorf("execution.manageKubernetesResource() error = %v, wantErr %v", err, wantErr)
				}
				close(okChan)
			})
			select {
			case <-errorChan:
				t.Fatal("fatal")
			case <-okChan:
				t.Logf("Scale ok for %s\n", testRes.K8sObj)
			}
		}(testRes)
	}
	if waitTimeout(&wg, 30*time.Second) {
		t.Fatal("timeout exceeded")
	} else {
		fmt.Println("Execution ok")
	}
}

func Test_execution_del_resources(t *testing.T) {
	t.Skip()
	ctx := context.Background()
	deploymentID := "Dep-ID"

	e := &execution{
		deploymentID: deploymentID,
		nodeName:     "testNode",
	}

	testNodeType := tosca.NodeType{
		Type: tosca.Type{
			Base:        tosca.TypeBaseNODE,
			DerivedFrom: "tosca.nodes.Root",
		},
	}
	testNode := tosca.NodeTemplate{
		Type: "fakeType",
	}
	err := storage.GetStore(types.StoreTypeDeployment).Set(ctx, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types/fakeType"), testNodeType)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes/testNode"), testNode)
	require.Nil(t, err)
	wantErr := false
	k8s := newTestK8s()
	operationType := k8sDeleteOperation

	resources := getSupportedResourceAndJSON()

	var wg sync.WaitGroup
	wg.Add(len(resources))

	deployTestResources(ctx, e, k8s, resources)
	for _, testRes := range resources {
		go func(testRes testResource) {
			defer wg.Done()
			errorChan := make(chan struct{})
			okChan := make(chan struct{})
			k8s.clientset.(*fake.Clientset).Fake.AddReactor("get", testRes.resourceGroup, fakeObjectDeletion(testRes.K8sObj, errorChan))
			t.Run("Test delete resource "+testRes.K8sObj.String(), func(t *testing.T) {
				if err := e.manageKubernetesResource(context.Background(), k8s.clientset, nil, testRes.K8sObj, operationType, true); (err != nil) != wantErr {
					t.Errorf("execution.manageKubernetesResource() error = %v, wantErr %v", err, wantErr)
				}
				close(okChan)
			})
			select {
			case <-errorChan:
				t.Fatal("fatal")
			case <-okChan:
				t.Logf("Deletion ok for %s\n", testRes.K8sObj)
			}
		}(testRes)
	}
	if waitTimeout(&wg, 30*time.Second) {
		t.Fatal("timeout exceeded")
	} else {
		fmt.Println("Execution ok")
	}
}

func deployTestResources(ctx context.Context, e *execution, k8s *k8s, resources []testResource) error {
	for _, testRes := range resources {
		testRes.K8sObj.unmarshalResource(ctx, e, e.deploymentID, k8s.clientset, testRes.rSpec)
		if err := testRes.K8sObj.createResource(ctx, e.deploymentID, k8s.clientset, "test-namespace"); err != nil {
			return err
		}
	}
	return nil
}

func Test_execution_create_resource(t *testing.T) {
	t.Skip()
	deploymentID := "Dep-ID"
	e := &execution{
		deploymentID: deploymentID,
	}
	wantErr := false
	k8s := newTestK8s()
	ctx := context.Background()
	operationType := k8sCreateOperation

	resources := getSupportedResourceAndJSON()
	for _, testResource := range resources {
		testResource.K8sObj.unmarshalResource(ctx, e, deploymentID, k8s.clientset, testResource.rSpec)
	}
	var wg sync.WaitGroup
	wg.Add(len(resources))

	for _, testRes := range resources {
		go func(testRes testResource) {
			defer wg.Done()
			errorChan := make(chan struct{})
			okChan := make(chan struct{})
			k8s.clientset.(*fake.Clientset).Fake.AddReactor("get", testRes.resourceGroup, fakeObjectCompletion(testRes.K8sObj, errorChan))
			t.Run("Test resource "+testRes.K8sObj.String(), func(t *testing.T) {
				t.Logf("Testing %s\n", testRes.K8sObj)
				if err := e.manageKubernetesResource(ctx, k8s.clientset, nil, testRes.K8sObj, operationType, true); (err != nil) != wantErr {
					t.Errorf("execution.manageKubernetesResource() error = %v, wantErr %v", err, wantErr)
				}
				close(okChan)
			})
			select {
			case <-errorChan:
				t.Fatal("fatal")
			case <-okChan:
				t.Logf("Execution ok for %s\n", testRes.K8sObj)
			}
		}(testRes)
	}
	if waitTimeout(&wg, 30*time.Second) {
		t.Fatal("timeout exceeded")
	} else {
		fmt.Println("Execution ok")
	}

}

// waitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}

func testExecutionExecuteInvalidOperation(t *testing.T) {
	tests := []struct {
		name    string
		opName  string
		wantErr bool
	}{
		{
			"test standard.create",
			"standard.create",
			true,
		},
		{
			"test standard.delete",
			"standard.delete",
			true,
		},
		{
			"test scale",
			"org.alien4cloud.management.clustercontrol.scale",
			true,
		},
		{
			"test some operation",
			"something",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := prov.Operation{
				Name: tt.opName,
			}
			e := &execution{
				operation: op,
			}

			err := e.executeOperation(nil, nil, nil, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Failed %s : %s", tt.name, err)
			}
		})
	}
}

// func Test_getYorcK8sObject(t *testing.T) {
// 	ctx := context.Background()
// 	k8s := newTestK8s()
// 	tests := []struct {
// 		name                  string
// 		k8sResourceType       string
// 		k8sSimpleResourceType string
// 		wantErr               bool
// 	}{
// 		{
// 			"test k8sDeploymentResourceType",
// 			k8sDeploymentResourceType,
// 			"",
// 			false,
// 		},
// 		{
// 			"test k8sStatefulsetResourceType",
// 			k8sStatefulsetResourceType,
// 			"",
// 			false,
// 		},
// 		{
// 			"test k8sServiceResourceType",
// 			k8sServiceResourceType,
// 			"",
// 			false,
// 		},
// 		{
// 			"test k8sSimpleRessourceType pvc",
// 			k8sSimpleRessourceType,
// 			"pvc",
// 			false,
// 		},
// 		{
// 			"test unsupported k8s simple resource type",
// 			k8sSimpleRessourceType,
// 			"something",
// 			true,
// 		},
// 		{
// 			"test unsupported k8s resource type",
// 			"something",
// 			"",
// 			true,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			e := &execution{
// 				nodeType: tt.k8sResourceType,
// 			}

// 			_, err := e.getYorcK8sObject(ctx, k8s.clientset)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("Failed %s : %s", tt.name, err)
// 			}
// 		})
// 	}
// }

func testExecutionGetExpectedInstances(t *testing.T) {
	deploymentID := "Dep-ID"

	type fields struct {
		deploymentID string
		taskID       string
	}
	tests := []struct {
		name    string
		fields  fields
		data    string
		want    int32
		wantErr bool
	}{
		{
			"task input filled",
			fields{deploymentID, "task-id-1"},
			strconv.Itoa(int(3)),
			3,
			false,
		},
		{
			"task input wrongly filled",
			fields{deploymentID, "task-id-2"},
			"not a integer",
			-1,
			true,
		},
		{
			"task input not filled",
			fields{deploymentID, "task-id-3"},
			"",
			-1,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &execution{
				deploymentID: tt.fields.deploymentID,
				taskID:       tt.fields.taskID,
			}
			tasks.SetTaskData(e.taskID, "inputs/EXPECTED_INSTANCES", tt.data)
			got, err := e.getExpectedInstances()
			if (err != nil) != tt.wantErr {
				t.Errorf("execution.getExpectedInstances() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("execution.getExpectedInstances() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testExecutionManageNamespaceDeletion(t *testing.T) {
	deploymentID := "Dep-ID"
	ctx := context.Background()

	e := &execution{
		deploymentID: deploymentID,
		nodeName:     "testNode",
	}
	testNodeType := tosca.NodeType{
		Type: tosca.Type{
			Base:        tosca.TypeBaseNODE,
			DerivedFrom: "tosca.nodes.Root",
		},
	}
	testNode := tosca.NodeTemplate{
		Type: "fakeType",
		Properties: map[string]*tosca.ValueAssignment{
			"volumeDeletable": &tosca.ValueAssignment{
				Type:  0,
				Value: "true",
			},
		},
	}
	err := storage.GetStore(types.StoreTypeDeployment).Set(ctx, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types/fakeType"), testNodeType)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes/testNode"), testNode)
	require.Nil(t, err)
	//Setup
	// One ns "default", 0 controler
	k8s := newTestSimpleK8s()
	k8s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}}, metav1.CreateOptions{})
	// One ns "test-ns", 1 controler
	k8s1 := newTestSimpleK8s()
	k8s1.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns"}}, metav1.CreateOptions{})
	k8s1.clientset.AppsV1().Deployments("test-ns").Create(ctx, &v1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "deploy"}}, metav1.CreateOptions{})

	type args struct {
		clientset         kubernetes.Interface
		namespaceProvided bool
		namespaceName     string
	}
	tests := []struct {
		name          string
		args          args
		wantNsDeleted bool
		wantErr       bool
	}{
		// Namespace provided -> no deletion
		{
			"NS provided",
			args{clientset: k8s.clientset, namespaceProvided: true, namespaceName: "default"},
			false,
			false,
		},
		// Not provided but controller left -> no deletion
		{
			"NS not provided, controller left",
			args{clientset: k8s1.clientset, namespaceProvided: false, namespaceName: "test-ns"},
			false,
			false,
		},
		// Not provided and 0 controller left -> deletion
		{
			"NS not provided, no controller",
			args{clientset: k8s.clientset, namespaceProvided: false, namespaceName: "default"},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if err := e.manageNamespaceDeletion(ctx, tt.args.clientset, tt.args.namespaceProvided, tt.args.namespaceName); (err != nil) != tt.wantErr {
				t.Errorf("execution.manageNamespaceDeletion() error = %v, wantErr %v", err, tt.wantErr)
			}
			if ns, _ := tt.args.clientset.CoreV1().Namespaces().Get(ctx, tt.args.namespaceName, metav1.GetOptions{}); (ns == nil) != tt.wantNsDeleted {
				t.Errorf("execution.manageNamespaceDeletion() namespace = %v, wantNsDeleted %v", ns, tt.wantNsDeleted)
			}
			//t.Logf("Ns found : %s, err : %s", ns, err)
		})
	}
}
