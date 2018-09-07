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
	"path"
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/registry"
)

func testBuildStep(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()

	t.Log("Registering Key")
	// Create a test key/value pair
	deploymentID := "dep_" + path.Base(t.Name())
	wfName := "wf_" + path.Base(t.Name())
	prefix := path.Join("_yorc/deployments", deploymentID, "workflows", wfName)
	data := make(map[string][]byte)
	data[prefix+"/steps/stepName/activities/0/delegate"] = []byte("install")
	data[prefix+"/steps/stepName/activities/1/set-state"] = []byte("installed")
	data[prefix+"/steps/stepName/activities/2/call-operation"] = []byte("script.sh")
	data[prefix+"/steps/stepName/target"] = []byte("nodeName")

	data[prefix+"/steps/Some_other_inline/activities/0/inline"] = []byte("my_custom_wf")

	srv1.PopulateKV(t, data)

	wfSteps, err := buildWorkFlow(kv, deploymentID, wfName)
	require.Nil(t, err)
	step := wfSteps["stepName"]
	require.NotNil(t, step)
	require.Equal(t, "nodeName", step.target)
	require.Equal(t, "stepName", step.name)
	require.Len(t, step.activities, 3)
	require.Contains(t, step.activities, delegateActivity{delegate: "install"})
	require.Contains(t, step.activities, setStateActivity{state: "installed"})
	require.Contains(t, step.activities, callOperationActivity{operation: "script.sh"})

	step = wfSteps["Some_other_inline"]
	require.NotNil(t, step)
	require.Equal(t, "", step.target)
	require.Equal(t, "Some_other_inline", step.name)
	require.Len(t, step.activities, 1)
	require.Contains(t, step.activities, inlineActivity{inline: "my_custom_wf"})
}

func testBuildStepWithNext(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()

	t.Log("Registering Key")
	// Create a test key/value pair
	deploymentID := "dep_" + path.Base(t.Name())
	wfName := "wf_" + path.Base(t.Name())
	prefix := path.Join("_yorc/deployments", deploymentID, "workflows", wfName)
	data := make(map[string][]byte)
	data[prefix+"/steps/stepName/activities/0/delegate"] = []byte("install")
	data[prefix+"/steps/stepName/next/downstream"] = []byte("")
	data[prefix+"/steps/stepName/target"] = []byte("nodeName")

	data[prefix+"/steps/downstream/activities/0/call-operation"] = []byte("script.sh")
	data[prefix+"/steps/downstream/target"] = []byte("downstream")

	srv1.PopulateKV(t, data)

	wfSteps, err := buildWorkFlow(kv, deploymentID, wfName)
	require.Nil(t, err)
	step := wfSteps["stepName"]
	require.NotNil(t, step)

	require.Equal(t, "nodeName", step.target)
	require.Equal(t, "stepName", step.name)
	require.Len(t, step.activities, 1)

}

func testBuildWorkFlow(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()
	t.Log("Registering Keys")

	// Create a test key/value pair
	deploymentID := "dep_" + path.Base(t.Name())
	wfName := "wf_" + path.Base(t.Name())
	prefix := path.Join("_yorc/deployments", deploymentID, "workflows", wfName)
	data := make(map[string][]byte)

	data[prefix+"/steps/step11/activities/0/delegate"] = []byte("install")
	data[prefix+"/steps/step11/next/step10"] = []byte("")
	data[prefix+"/steps/step11/next/step12"] = []byte("")
	data[prefix+"/steps/step11/target"] = []byte("nodeName")

	data[prefix+"/steps/step10/activities/0/delegate"] = []byte("install")
	data[prefix+"/steps/step10/next/step13"] = []byte("")
	data[prefix+"/steps/step10/target"] = []byte("nodeName")

	data[prefix+"/steps/step12/activities/0/delegate"] = []byte("install")
	data[prefix+"/steps/step12/next/step13"] = []byte("")
	data[prefix+"/steps/step12/target"] = []byte("nodeName")

	data[prefix+"/steps/step13/activities/0/delegate"] = []byte("install")
	data[prefix+"/steps/step13/target"] = []byte("nodeName")

	data[prefix+"/steps/step15/activities/0/inline"] = []byte("inception")

	data[prefix+"/steps/step20/activities/0/delegate"] = []byte("install")
	data[prefix+"/steps/step20/target"] = []byte("nodeName")

	srv1.PopulateKV(t, data)

	steps, err := buildWorkFlow(kv, deploymentID, wfName)
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
func (m *mockExecutor) ExecAsyncOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
	return errors.New("Asynchronous operation is not yet handled by this executor")
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

func testRunStep(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
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

			wfSteps, err := buildWorkFlow(kv, deploymentID, tt.args.workflowName)
			require.Nil(t, err)
			s := wfSteps[tt.args.stepName]
			require.NotNil(t, s)

			s.next = nil
			s.t = &taskExecution{id: "taskExecutionID", taskID: "taskID", targetID: deploymentID}
			srv1.SetKV(t, path.Join(consulutil.WorkflowsPrefix, s.t.taskID, "WFNode_create"), []byte("initial"))
			srv1.SetKV(t, path.Join(consulutil.WorkflowsPrefix, s.t.taskID, "Compute_install"), []byte("initial"))
			err = s.run(context.Background(), config.Configuration{}, kv, deploymentID, tt.args.bypassErrors, tt.args.workflowName, &worker{})
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
				if mah.taskID != s.t.taskID {
					t.Errorf("step.run() pre_activity_hook.taskID = %v, want %v", mah.taskID, s.t.id)
				}
				if mah.target != s.target {
					t.Errorf("step.run() pre_activity_hook.target = %v, want %v", mah.target, s.target)
				}
				if mah.activity != s.activities[0] {
					t.Errorf("step.run() pre_activity_hook.activity = %v, want %v", mah.activity, s.activities[0])
				}
			}
			for _, mah := range postAH {
				require.NotZero(t, mah.deploymentID, "postActivityHook not called")
				if mah.deploymentID != deploymentID {
					t.Errorf("step.run() post_activity_hook.deploymentID = %v, want %v", mah.deploymentID, deploymentID)
				}
				if mah.taskID != s.t.taskID {
					t.Errorf("step.run() post_activity_hook.taskID = %v, want %v", mah.taskID, s.t.id)
				}
				if mah.target != s.target {
					t.Errorf("step.run() post_activity_hook.target = %v, want %v", mah.target, s.target)
				}
				if mah.activity != s.activities[0] {
					t.Errorf("step.run() post_activity_hook.activity = %v, want %v", mah.activity, s.activities[0])
				}
			}
		})
	}
}

func clearActivityHooks() {
	preActivityHooks = make([]ActivityHook, 0)
	postActivityHooks = make([]ActivityHook, 0)
}
