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
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/testutil"
)

func testsExecJob(t *testing.T, kv *api.KV) {

	t.Run("testExecutionCancelJob", func(t *testing.T) {
		testExecutionCancelJob(t, kv)
	})
}

func jobRuntimeObject(id, namespace string) *batchv1.Job {
	return &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: namespace}}
}

func testExecutionCancelJob(t *testing.T, kv *api.KV) {

	deploymentID := testutil.BuildDeploymentID(t)
	err := deployments.StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/JobCompute.yml")
	require.Nil(t, err, "Failed to parse testdata/JobCompute.yml definition")

	nodeName := "ContainerJobUnit_Resource"
	defNamespace := defaultNamespace(deploymentID)
	taskJobID := "containerjobunit--74546570"
	instanceJobID := "instanceProvidedID"
	err = tasks.SetTaskData(kv, "t1", nodeName+"-jobId", taskJobID)
	require.NoError(t, err)

	err = deployments.SetInstanceAttribute(deploymentID, nodeName, "0", "job_id", instanceJobID)
	require.NoError(t, err, "Failed to set job_id attribute")
	type fields struct {
		taskID   string
		nodeName string
	}
	tests := []struct {
		name            string
		fields          fields
		initialElements []runtime.Object
		wantErr         bool
	}{
		{"TestCancelJobWithIDFromTask", fields{"t1", nodeName}, []runtime.Object{jobRuntimeObject(taskJobID, defNamespace), namespaceRuntimeObject(defNamespace)}, false},
		{"TestCancelJobWithIDFromInstanceAttribute", fields{"t2", nodeName}, []runtime.Object{jobRuntimeObject(instanceJobID, defNamespace), namespaceRuntimeObject(defNamespace)}, false},
		{"TestCancelJobWithoutID", fields{"t2", "ContainerJobUnit_Resource2"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClientSet := fake.NewSimpleClientset(tt.initialElements...)
			e := &execution{
				kv:           kv,
				deploymentID: deploymentID,
				taskID:       tt.fields.taskID,
				nodeName:     tt.fields.nodeName,
			}
			if err := e.cancelJob(context.Background(), fakeClientSet); (err != nil) != tt.wantErr {
				t.Errorf("execution.cancelJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
