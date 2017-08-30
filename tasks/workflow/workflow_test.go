package workflow

import (
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/log"
	"path"
	"strings"
)

func testReadStepFromConsulFailing(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	log.SetDebug(true)

	t.Log("Registering Key")
	// Create a test key/value pair
	srv1.SetKV(t, "wf/steps/stepName/activity/delegate", []byte("install"))

	step, err := readStep(kv, "wf/steps/", "stepName", nil)
	t.Log(err)
	require.Nil(t, step)
	require.Error(t, err)
}

func testReadStepFromConsul(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()

	t.Log("Registering Key")
	// Create a test key/value pair
	stepName := "step_" + path.Base(t.Name())
	data := make(map[string][]byte)
	data["wf/steps/"+stepName+"/activity/delegate"] = []byte("install")
	data["wf/steps/"+stepName+"/activity/set-state"] = []byte("installed")
	data["wf/steps/"+stepName+"/activity/call-operation"] = []byte("script.sh")
	data["wf/steps/"+stepName+"/node"] = []byte("nodeName")

	srv1.PopulateKV(t, data)

	visitedMap := make(map[string]*visitStep)
	step, err := readStep(kv, "wf/steps/", stepName, visitedMap)
	require.Nil(t, err)
	require.Equal(t, "nodeName", step.Node)
	require.Equal(t, stepName, step.Name)
	require.Len(t, step.Activities, 3)
	require.Contains(t, step.Activities, delegateActivity{delegate: "install"})
	require.Contains(t, step.Activities, setStateActivity{state: "installed"})
	require.Contains(t, step.Activities, callOperationActivity{operation: "script.sh"})

	require.Len(t, visitedMap, 1)
	require.Contains(t, visitedMap, stepName)
}

func testReadStepWithNext(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()

	t.Log("Registering Key")
	// Create a test key/value pair
	stepName := "step_" + path.Base(t.Name())
	data := make(map[string][]byte)
	data["wf/steps/"+stepName+"/activity/delegate"] = []byte("install")
	data["wf/steps/"+stepName+"/next/downstream"] = []byte("")
	data["wf/steps/"+stepName+"/node"] = []byte("nodeName")

	data["wf/steps/downstream/activity/call-operation"] = []byte("script.sh")
	data["wf/steps/downstream/node"] = []byte("downstream")

	srv1.PopulateKV(t, data)

	visitedMap := make(map[string]*visitStep)
	step, err := readStep(kv, "wf/steps/", stepName, visitedMap)
	require.Nil(t, err)
	require.Equal(t, "nodeName", step.Node)
	require.Equal(t, stepName, step.Name)
	require.Len(t, step.Activities, 1)

	require.Len(t, visitedMap, 2)
	require.Contains(t, visitedMap, stepName)
	require.Contains(t, visitedMap, "downstream")

	require.Equal(t, 0, visitedMap[stepName].refCount)
	require.Equal(t, 1, visitedMap["downstream"].refCount)
}

func testReadWorkFlowFromConsul(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()

	t.Log("Registering Keys")
	stepName := "step_" + path.Base(t.Name())
	// Create a test key/value pair
	data := make(map[string][]byte)

	data["wf/steps/"+stepName+"_11/activity/delegate"] = []byte("install")
	data["wf/steps/"+stepName+"_11/next/"+stepName+"_10"] = []byte("")
	data["wf/steps/"+stepName+"_11/next/"+stepName+"_12"] = []byte("")
	data["wf/steps/"+stepName+"_11/node"] = []byte("nodeName")

	data["wf/steps/"+stepName+"_10/activity/delegate"] = []byte("install")
	data["wf/steps/"+stepName+"_10/next/"+stepName+"_13"] = []byte("")
	data["wf/steps/"+stepName+"_10/node"] = []byte("nodeName")

	data["wf/steps/"+stepName+"_12/activity/delegate"] = []byte("install")
	data["wf/steps/"+stepName+"_12/next/"+stepName+"_13"] = []byte("")
	data["wf/steps/"+stepName+"_12/node"] = []byte("nodeName")

	data["wf/steps/"+stepName+"_13/activity/delegate"] = []byte("install")
	data["wf/steps/"+stepName+"_13/node"] = []byte("nodeName")

	data["wf/steps/"+stepName+"_20/activity/delegate"] = []byte("install")
	data["wf/steps/"+stepName+"_20/node"] = []byte("nodeName")

	srv1.PopulateKV(t, data)

	steps, err := readWorkFlowFromConsul(kv, "wf")
	require.Nil(t, err, "oups")

	// Filter steps to get only those of the test
	filteredSteps := make([]*step, 0)
	for _, step := range steps {
		if strings.HasPrefix(step.Name, stepName) {
			filteredSteps = append(filteredSteps, step)
		}
	}
	require.Len(t, filteredSteps, 5)

}
