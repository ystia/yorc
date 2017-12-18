package workflow

import (
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/log"
	"path"
)

func testReadStepFromConsulFailing(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	log.SetDebug(true)

	t.Log("Registering Key")
	// Create a test key/value pair
	wfName := "wf_" + path.Base(t.Name())
	srv1.SetKV(t, wfName+"/steps/stepName/activity/delegate", []byte("install"))

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
	data[wfName+"/steps/stepName/activity/delegate"] = []byte("install")
	data[wfName+"/steps/stepName/activity/set-state"] = []byte("installed")
	data[wfName+"/steps/stepName/activity/call-operation"] = []byte("script.sh")
	data[wfName+"/steps/stepName/target"] = []byte("nodeName")

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
}

func testReadStepWithNext(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()

	t.Log("Registering Key")
	// Create a test key/value pair
	wfName := "wf_" + path.Base(t.Name())
	data := make(map[string][]byte)
	data[wfName+"/steps/stepName/activity/delegate"] = []byte("install")
	data[wfName+"/steps/stepName/next/downstream"] = []byte("")
	data[wfName+"/steps/stepName/target"] = []byte("nodeName")

	data[wfName+"/steps/downstream/activity/call-operation"] = []byte("script.sh")
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

	data[wfName+"/steps/step11/activity/delegate"] = []byte("install")
	data[wfName+"/steps/step11/next/step10"] = []byte("")
	data[wfName+"/steps/step11/next/step12"] = []byte("")
	data[wfName+"/steps/step11/target"] = []byte("nodeName")

	data[wfName+"/steps/step10/activity/delegate"] = []byte("install")
	data[wfName+"/steps/step10/next/step13"] = []byte("")
	data[wfName+"/steps/step10/target"] = []byte("nodeName")

	data[wfName+"/steps/step12/activity/delegate"] = []byte("install")
	data[wfName+"/steps/step12/next/step13"] = []byte("")
	data[wfName+"/steps/step12/target"] = []byte("nodeName")

	data[wfName+"/steps/step13/activity/delegate"] = []byte("install")
	data[wfName+"/steps/step13/target"] = []byte("nodeName")

	data[wfName+"/steps/step20/activity/delegate"] = []byte("install")
	data[wfName+"/steps/step20/target"] = []byte("nodeName")

	srv1.PopulateKV(t, data)

	steps, err := readWorkFlowFromConsul(kv, wfName)
	require.Nil(t, err, "oups")
	require.Len(t, steps, 5)
}
