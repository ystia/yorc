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

package builder

import (
	"context"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
	"path"
	"testing"
	"time"
)

func testBuildStep(t *testing.T, srv1 *testutil.TestServer) {
	t.Parallel()
	ctx := context.Background()
	t.Log("Registering Key")
	// Create a test key/value pair
	deploymentID := "dep_" + path.Base(t.Name())
	wfName := "wf_" + path.Base(t.Name())
	prefix := path.Join("_yorc/deployments", deploymentID, "workflows", wfName)

	wf := tosca.Workflow{Steps: map[string]*tosca.Step{
		"stepName": {
			Target:             "nodeName",
			TargetRelationShip: "",
			Activities: []tosca.Activity{
				{Delegate: &tosca.WorkflowActivity{Workflow: "install"}},
				{SetState: "installed"},
				{CallOperation: &tosca.OperationActivity{Operation: "script.sh"}},
			},
		},
		"Some_other_inline": {
			//Target:             "nodeName",
			Activities: []tosca.Activity{
				{Inline: &tosca.WorkflowActivity{Workflow: "my_custom_wf"}},
			},
		},
	}}
	err := storage.GetStore(types.StoreTypeDeployment).Set(ctx, prefix, wf)
	require.Nil(t, err)
	time.Sleep(1 * time.Second)
	wfSteps, err := BuildWorkFlow(context.Background(), deploymentID, wfName)
	require.Nil(t, err, "BuildWorkflowError: %+v", err)
	step := wfSteps["stepName"]
	require.NotNil(t, step)
	require.Equal(t, "nodeName", step.Target)
	require.Equal(t, "stepName", step.Name)
	require.Len(t, step.Activities, 3)
	require.Contains(t, step.Activities, delegateActivity{delegate: "install"})
	require.Contains(t, step.Activities, setStateActivity{state: "installed"})
	require.Contains(t, step.Activities, callOperationActivity{operation: "script.sh"})

	step = wfSteps["Some_other_inline"]
	require.NotNil(t, step)
	require.Equal(t, "", step.Target)
	require.Equal(t, "Some_other_inline", step.Name)
	require.Len(t, step.Activities, 1)
	require.Contains(t, step.Activities, inlineActivity{inline: "my_custom_wf"})
}

func testBuildStepWithNext(t *testing.T, srv1 *testutil.TestServer) {
	t.Parallel()
	ctx := context.Background()
	t.Log("Registering Key")
	// Create a test key/value pair
	deploymentID := "dep_" + path.Base(t.Name())
	wfName := "wf_" + path.Base(t.Name())
	prefix := path.Join("_yorc/deployments", deploymentID, "workflows", wfName)

	wf := tosca.Workflow{Steps: map[string]*tosca.Step{
		"stepName": {
			Target:    "nodeName",
			OnSuccess: []string{"downstream"},
			Activities: []tosca.Activity{
				{Delegate: &tosca.WorkflowActivity{Workflow: "install"}},
			},
		},
		"downstream": {
			Target: "downstream",
			Activities: []tosca.Activity{
				{CallOperation: &tosca.OperationActivity{Operation: "script.sh"}},
			},
		},
	}}
	err := storage.GetStore(types.StoreTypeDeployment).Set(ctx, prefix, wf)
	require.Nil(t, err)

	wfSteps, err := BuildWorkFlow(context.Background(), deploymentID, wfName)
	require.Nil(t, err)
	step := wfSteps["stepName"]
	require.NotNil(t, step)

	require.Equal(t, "nodeName", step.Target)
	require.Equal(t, "stepName", step.Name)
	require.Len(t, step.Activities, 1)

}

func testBuildStepWithNonExistentNextStep(t *testing.T, srv1 *testutil.TestServer) {
	t.Parallel()
	ctx := context.Background()
	t.Log("Registering Key")
	// Create a test key/value pair
	deploymentID := "dep_" + path.Base(t.Name())
	wfName := "wf_" + path.Base(t.Name())
	prefix := path.Join("_yorc/deployments", deploymentID, "workflows", wfName)

	wf := tosca.Workflow{Steps: map[string]*tosca.Step{
		"stepName": {
			Target:    "nodeName",
			OnSuccess: []string{"nonexistent"},
			Activities: []tosca.Activity{
				{Delegate: &tosca.WorkflowActivity{Workflow: "install"}},
			},
		},
	}}
	err := storage.GetStore(types.StoreTypeDeployment).Set(ctx, prefix, wf)
	require.Nil(t, err)

	wfSteps, err := BuildWorkFlow(context.Background(), deploymentID, wfName)
	require.Errorf(t, err, "non-existent step should raise an error")
	require.EqualError(t, err, "Referenced step with name:\"nonexistent\" doesn't exist in the workflow:\"wf_testBuildStepWithNonExistentNextStep\"")
	require.Nil(t, wfSteps)
}

func testBuildWorkFlow(t *testing.T, srv1 *testutil.TestServer) {
	t.Parallel()
	t.Log("Registering Keys")
	ctx := context.Background()
	// Create a test key/value pair
	deploymentID := "dep_" + path.Base(t.Name())
	wfName := "wf_" + path.Base(t.Name())

	prefix := path.Join("_yorc/deployments", deploymentID, "workflows", wfName)

	wf := tosca.Workflow{Steps: map[string]*tosca.Step{
		"step11": {
			Target:    "nodeName",
			OnSuccess: []string{"step10", "step12"},
			Activities: []tosca.Activity{
				{Delegate: &tosca.WorkflowActivity{Workflow: "install"}},
			},
		},
		"step10": {
			Target:    "nodeName",
			OnSuccess: []string{"step13"},
			Activities: []tosca.Activity{
				{Delegate: &tosca.WorkflowActivity{Workflow: "install"}},
			},
		},
		"step12": {
			Target:    "nodeName",
			OnSuccess: []string{"step13"},
			Activities: []tosca.Activity{
				{Delegate: &tosca.WorkflowActivity{Workflow: "install"}},
			},
		},
		"step13": {
			Target: "nodeName",
			Activities: []tosca.Activity{
				{Delegate: &tosca.WorkflowActivity{Workflow: "install"}},
			},
		},
		"step15": {
			Target: "nodeName",
			Activities: []tosca.Activity{
				{Inline: &tosca.WorkflowActivity{Workflow: "inception"}},
			},
		},
		"step20": {
			Target: "nodeName",
			Activities: []tosca.Activity{
				{Delegate: &tosca.WorkflowActivity{Workflow: "install"}},
			},
		},
	}}
	err := storage.GetStore(types.StoreTypeDeployment).Set(ctx, prefix, wf)
	require.Nil(t, err)

	steps, err := BuildWorkFlow(context.Background(), deploymentID, wfName)
	require.Nil(t, err, "oups")
	require.Len(t, steps, 6)
}
