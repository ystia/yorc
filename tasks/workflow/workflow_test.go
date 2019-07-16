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
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/prov"
	"github.com/ystia/yorc/v3/registry"
	"github.com/ystia/yorc/v3/tasks/workflow/builder"
)

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
func (m *mockExecutor) ExecAsyncOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation, stepName string) (*prov.Action, time.Duration, error) {
	return nil, 0, errors.New("Asynchronous operation is not yet handled by this executor")
}

type mockActivityHook struct {
	taskID       string
	deploymentID string
	target       string
	activity     builder.Activity
}

func (m *mockActivityHook) hook(ctx context.Context, cfg config.Configuration, taskID, deploymentID, target string, activity builder.Activity) {
	m.target = target
	m.taskID = taskID
	m.deploymentID = deploymentID
	m.activity = activity
}

func testRunStep(t *testing.T, srv1 *testutil.TestServer, cc *api.Client) {
	kv := cc.KV()
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

			wfSteps, err := builder.BuildWorkFlow(kv, deploymentID, tt.args.workflowName)
			require.Nil(t, err)
			bs := wfSteps[tt.args.stepName]
			require.NotNil(t, bs)

			bs.Next = nil
			te := &taskExecution{id: "taskExecutionID", taskID: "taskID", targetID: deploymentID}
			s := wrapBuilderStep(bs, cc, te)
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
				if mah.taskID != s.t.taskID {
					t.Errorf("step.run() post_activity_hook.taskID = %v, want %v", mah.taskID, s.t.id)
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

func testRegisterInlineWorkflow(t *testing.T, srv1 *testutil.TestServer, cc *api.Client) {
	kv := cc.KV()
	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	topologyPath := "../../deployments/testdata/inline_workflow.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), kv, deploymentID, topologyPath)
	require.NoError(t, err, "Unexpected error storing %s", topologyPath)

	_, err = builder.BuildWorkFlow(kv, deploymentID, "install")
	require.NoError(t, err, "Unexpected error building workflow for %s", topologyPath)

}
