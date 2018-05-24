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

package workflow

import (
	"context"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/registry"

	"github.com/ystia/yorc/helper/consulutil"

	"path"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
)

func testReadStepFromConsulFailing(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	log.SetDebug(true)

	t.Log("Registering Key")
	// Create a test key/value pair
	wfName := "wf_" + path.Base(t.Name())
	srv1.SetKV(t, wfName+"/steps/stepName/activities/0/delegate", []byte("install"))

	step, err := readStep(kv, wfName+"/steps/", "stepName", nil)
	t.Log(err)
	require.Nil(t, step)
	require.Error(t, err)
}

func testReadStepFromConsul(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()

	t.Log("Registering Key")
	// Create a test key/value pair
	wfName := "wf_" + path.Base(t.Name())
	data := make(map[string][]byte)
	data[wfName+"/steps/stepName/activities/0/delegate"] = []byte("install")
	data[wfName+"/steps/stepName/activities/1/set-state"] = []byte("installed")
	data[wfName+"/steps/stepName/activities/2/call-operation"] = []byte("script.sh")
	data[wfName+"/steps/stepName/target"] = []byte("nodeName")

	data[wfName+"/steps/Some_other_inline/activities/0/inline"] = []byte("my_custom_wf")

	srv1.PopulateKV(t, data)

	visitedMap := make(map[string]*visitStep)
	step, err := readStep(kv, wfName+"/steps/", "stepName", visitedMap)
	require.Nil(t, err)
	require.Equal(t, "nodeName", step.Target)
	require.Equal(t, "stepName", step.Name)
	require.Len(t, step.Activities, 3)
	require.Contains(t, step.Activities, delegateActivity{delegate: "install"})
	require.Contains(t, step.Activities, setStateActivity{state: "installed"})
	require.Contains(t, step.Activities, callOperationActivity{operation: "script.sh"})
	require.Len(t, visitedMap, 1)
	require.Contains(t, visitedMap, "stepName")

	visitedMap = make(map[string]*visitStep)
	step, err = readStep(kv, wfName+"/steps/", "Some_other_inline", visitedMap)
	require.Nil(t, err)
	require.Equal(t, "", step.Target)
	require.Equal(t, "Some_other_inline", step.Name)
	require.Len(t, step.Activities, 1)
	require.Contains(t, step.Activities, inlineActivity{inline: "my_custom_wf"})
	require.Len(t, visitedMap, 1)
	require.Contains(t, visitedMap, "Some_other_inline")
}

func testReadStepWithNext(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()

	t.Log("Registering Key")
	// Create a test key/value pair
	wfName := "wf_" + path.Base(t.Name())
	data := make(map[string][]byte)
	data[wfName+"/steps/stepName/activities/0/delegate"] = []byte("install")
	data[wfName+"/steps/stepName/next/downstream"] = []byte("")
	data[wfName+"/steps/stepName/target"] = []byte("nodeName")

	data[wfName+"/steps/downstream/activities/0/call-operation"] = []byte("script.sh")
	data[wfName+"/steps/downstream/target"] = []byte("downstream")

	srv1.PopulateKV(t, data)

	visitedMap := make(map[string]*visitStep)
	step, err := readStep(kv, wfName+"/steps/", "stepName", visitedMap)
	require.Nil(t, err)
	require.Equal(t, "nodeName", step.Target)
	require.Equal(t, "stepName", step.Name)
	require.Len(t, step.Activities, 1)

	require.Len(t, visitedMap, 2)
	require.Contains(t, visitedMap, "stepName")
	require.Contains(t, visitedMap, "downstream")

	require.Equal(t, 0, visitedMap["stepName"].refCount)
	require.Equal(t, 1, visitedMap["downstream"].refCount)
}

func testReadWorkFlowFromConsul(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()

	t.Log("Registering Keys")
	// Create a test key/value pair
	wfName := "wf_" + path.Base(t.Name())
	data := make(map[string][]byte)

	data[wfName+"/steps/step11/activities/0/delegate"] = []byte("install")
	data[wfName+"/steps/step11/next/step10"] = []byte("")
	data[wfName+"/steps/step11/next/step12"] = []byte("")
	data[wfName+"/steps/step11/target"] = []byte("nodeName")

	data[wfName+"/steps/step10/activities/0/delegate"] = []byte("install")
	data[wfName+"/steps/step10/next/step13"] = []byte("")
	data[wfName+"/steps/step10/target"] = []byte("nodeName")

	data[wfName+"/steps/step12/activities/0/delegate"] = []byte("install")
	data[wfName+"/steps/step12/next/step13"] = []byte("")
	data[wfName+"/steps/step12/target"] = []byte("nodeName")

	data[wfName+"/steps/step13/activities/0/delegate"] = []byte("install")
	data[wfName+"/steps/step13/target"] = []byte("nodeName")

	data[wfName+"/steps/step15/activities/0/inline"] = []byte("inception")

	data[wfName+"/steps/step20/activities/0/delegate"] = []byte("install")
	data[wfName+"/steps/step20/target"] = []byte("nodeName")

	srv1.PopulateKV(t, data)

	steps, err := readWorkFlowFromConsul(kv, wfName)
	require.Nil(t, err, "oups")
	require.Len(t, steps, 6)
}

type mockExecutor struct {
	delegateCalled bool
	callOpsCalled  bool
	errorsDelegate bool
	errorsCallOps  bool
}

func (m *mockExecutor) ExecDelegate(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	m.delegateCalled = true
	if m.errorsDelegate {
		return errors.New("Failed required for mock")
	}
	return nil
}
func (m *mockExecutor) ExecOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
	m.callOpsCalled = true
	if m.errorsCallOps {
		return errors.New("Failed required for mock")
	}
	return nil
}

type mockActivityHook struct {
	taskID       string
	deploymentID string
	target       string
	activity     Activity
}

func (m *mockActivityHook) hook(ctx context.Context, cfg config.Configuration, taskID, deploymentID, target string, activity Activity) {
	m.target = target
	m.taskID = taskID
	m.deploymentID = deploymentID
	m.activity = activity
}

func testRunWorkflow(t *testing.T, kv *api.KV) {
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := deployments.StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/workflow.yaml")
	require.Nil(t, err)

	mockExecutor := &mockExecutor{}
	registry.GetRegistry().RegisterDelegates([]string{"ystia.yorc.tests.nodes.WFCompute"}, mockExecutor, "tests")
	registry.GetRegistry().RegisterOperationExecutor([]string{"ystia.yorc.tests.artifacts.Implementation.Custom"}, mockExecutor, "tests")

	type args struct {
		workflowName       string
		stepName           string
		errorsDelegate     bool
		errorsCallOps      bool
		bypassErrors       bool
		nbPreActivityHook  int
		nbPostActivityHook int
	}
	tests := []struct {
		name               string
		args               args
		wantDelegateCalled bool
		wantCallOpsCalled  bool
		wantErr            bool
	}{
		{"ExecuteStandardCallOps", args{"install", "WFNode_create", false, false, false, 0, 0}, false, true, false},
		{"ExecuteErrorCallOps", args{"install", "WFNode_create", false, true, false, 0, 0}, false, true, true},
		{"ExecuteBypassErrorCallOps", args{"install", "WFNode_create", false, true, true, 0, 0}, false, true, false},
		{"ExecuteStandardDelegate", args{"install", "Compute_install", false, false, false, 0, 0}, true, false, false},
		{"ExecuteErrorDelegate", args{"install", "Compute_install", true, false, false, 0, 0}, true, false, true},
		{"ExecuteBypassErrorDelegate", args{"install", "Compute_install", true, false, true, 0, 0}, true, false, false},
		{"ExecuteHooksDelegate", args{"install", "Compute_install", false, false, true, 2, 3}, true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearActivityHooks()
			preAH := make([]*mockActivityHook, tt.args.nbPreActivityHook)
			for index := 0; index < tt.args.nbPreActivityHook; index++ {
				preAH[index] = &mockActivityHook{}
				RegisterPreActivityHook(preAH[index].hook)
			}
			postAH := make([]*mockActivityHook, tt.args.nbPostActivityHook)
			for index := 0; index < tt.args.nbPostActivityHook; index++ {
				postAH[index] = &mockActivityHook{}
				RegisterPostActivityHook(postAH[index].hook)
			}
			mockExecutor.callOpsCalled = false
			mockExecutor.errorsCallOps = tt.args.errorsCallOps
			mockExecutor.delegateCalled = false
			mockExecutor.errorsDelegate = tt.args.errorsDelegate
			stepsPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "workflows", tt.args.workflowName, "steps") + "/"

			s, err := readStep(kv, stepsPrefix, tt.args.stepName, make(map[string]*visitStep))
			require.NoError(t, err)

			s.Next = nil
			s.SetTaskID(&task{ID: "taskID", TargetID: deploymentID})
			err = s.run(context.Background(), deploymentID, kv, make(chan error, 10), make(chan struct{}), config.Configuration{}, tt.args.bypassErrors, tt.args.workflowName, worker{})
			if (err != nil) != tt.wantErr {
				t.Errorf("step.run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if mockExecutor.delegateCalled != tt.wantDelegateCalled {
				t.Errorf("step.run() delegateCalled = %v, want %v", mockExecutor.delegateCalled, tt.wantDelegateCalled)
			}
			if mockExecutor.callOpsCalled != tt.wantCallOpsCalled {
				t.Errorf("step.run() delegateCalled = %v, want %v", mockExecutor.callOpsCalled, tt.wantCallOpsCalled)
			}
			for _, mah := range preAH {
				require.NotZero(t, mah.deploymentID, "preActivityHook not called")
				if mah.deploymentID != deploymentID {
					t.Errorf("step.run() pre_activity_hook.deploymentID = %v, want %v", mah.deploymentID, deploymentID)
				}
				if mah.taskID != s.t.ID {
					t.Errorf("step.run() pre_activity_hook.taskID = %v, want %v", mah.taskID, s.t.ID)
				}
				if mah.target != s.Target {
					t.Errorf("step.run() pre_activity_hook.target = %v, want %v", mah.target, s.Target)
				}
				if mah.activity != s.Activities[0] {
					t.Errorf("step.run() pre_activity_hook.activity = %v, want %v", mah.activity, s.Activities[0])
				}
			}
			for _, mah := range postAH {
				require.NotZero(t, mah.deploymentID, "postActivityHook not called")
				if mah.deploymentID != deploymentID {
					t.Errorf("step.run() post_activity_hook.deploymentID = %v, want %v", mah.deploymentID, deploymentID)
				}
				if mah.taskID != s.t.ID {
					t.Errorf("step.run() post_activity_hook.taskID = %v, want %v", mah.taskID, s.t.ID)
				}
				if mah.target != s.Target {
					t.Errorf("step.run() post_activity_hook.target = %v, want %v", mah.target, s.Target)
				}
				if mah.activity != s.Activities[0] {
					t.Errorf("step.run() post_activity_hook.activity = %v, want %v", mah.activity, s.Activities[0])
				}
			}
		})
	}
}

func clearActivityHooks() {
	preActivityHooks = make([]ActivityHook, 0)
	postActivityHooks = make([]ActivityHook, 0)
}
