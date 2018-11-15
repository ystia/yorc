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
	"path"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
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

	wfSteps, err := BuildWorkFlow(kv, deploymentID, wfName)
	require.Nil(t, err)
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

	wfSteps, err := BuildWorkFlow(kv, deploymentID, wfName)
	require.Nil(t, err)
	step := wfSteps["stepName"]
	require.NotNil(t, step)

	require.Equal(t, "nodeName", step.Target)
	require.Equal(t, "stepName", step.Name)
	require.Len(t, step.Activities, 1)

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

	steps, err := BuildWorkFlow(kv, deploymentID, wfName)
	require.Nil(t, err, "oups")
	require.Len(t, steps, 6)
}
