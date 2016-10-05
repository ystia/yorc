package tasks

import (
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/log"
	"testing"
)

func TestGroupedTaskParallel(t *testing.T) {
	t.Run("groupTask", func(t *testing.T) {
		t.Run("generateOSBSVolumeSizeConvert", readStepFromConsul)
		t.Run("Test_generateOSBSVolumeSizeConvertError", readStepFromConsulFailing)
		t.Run("Test_generateOSBSVolumeMissingSize", readStepWithNext)
		t.Run("Test_generateOSBSVolumeCheckOptionalValues", testReadWorkFlowFromConsul)
	})
}

func readStepFromConsulFailing(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	require.Nil(t, err)

	kv := client.KV()

	t.Log("Registering Key")
	// Create a test key/value pair
	srv1.SetKV("wf/steps/stepName/activity/delegate", []byte("install"))

	step, err := readStep(kv, "wf/steps/", "stepName", nil)
	t.Log(err)
	require.Nil(t, step)
	require.Error(t, err)
}

func readStepFromConsul(t *testing.T) {
	t.Parallel()
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	require.Nil(t, err)

	kv := client.KV()

	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	data["wf/steps/stepName/activity/delegate"] = []byte("install")
	data["wf/steps/stepName/activity/set-state"] = []byte("installed")
	data["wf/steps/stepName/activity/operation"] = []byte("script.sh")
	data["wf/steps/stepName/node"] = []byte("nodeName")

	srv1.PopulateKV(data)
	//kv.Put(&api.KVPair{Key: "/wf/steps/stepName/activity/delegate", Value:[]byte("install")}, nil)

	visitedMap := make(map[string]*visitStep)
	step, err := readStep(kv, "wf/steps/", "stepName", visitedMap)
	require.Nil(t, err)
	require.Equal(t, "nodeName", step.Node)
	require.Equal(t, "stepName", step.Name)
	require.Len(t, step.Activities, 3)
	require.Contains(t, step.Activities, DelegateActivity{delegate: "install"})
	require.Contains(t, step.Activities, SetStateActivity{state: "installed"})
	require.Contains(t, step.Activities, CallOperationActivity{operation: "script.sh"})

	require.Len(t, visitedMap, 1)
	require.Contains(t, visitedMap, "stepName")
}

func readStepWithNext(t *testing.T) {
	t.Parallel()
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	require.Nil(t, err)

	kv := client.KV()

	t.Log("Registering Key")
	// Create a test key/value pair
	data := make(map[string][]byte)
	data["wf/steps/stepName/activity/delegate"] = []byte("install")
	data["wf/steps/stepName/next/downstream"] = []byte("")
	data["wf/steps/stepName/node"] = []byte("nodeName")

	data["wf/steps/downstream/activity/operation"] = []byte("script.sh")
	data["wf/steps/downstream/node"] = []byte("downstream")

	srv1.PopulateKV(data)
	//kv.Put(&api.KVPair{Key: "/wf/steps/stepName/activity/delegate", Value:[]byte("install")}, nil)

	visitedMap := make(map[string]*visitStep)
	step, err := readStep(kv, "wf/steps/", "stepName", visitedMap)
	require.Nil(t, err)
	require.Equal(t, "nodeName", step.Node)
	require.Equal(t, "stepName", step.Name)
	require.Len(t, step.Activities, 1)

	require.Len(t, visitedMap, 2)
	require.Contains(t, visitedMap, "stepName")
	require.Contains(t, visitedMap, "downstream")

	require.Equal(t, 0, visitedMap["stepName"].refCount)
	require.Equal(t, 1, visitedMap["downstream"].refCount)
}

func testReadWorkFlowFromConsul(t *testing.T) {
	t.Parallel()
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	require.Nil(t, err)

	kv := client.KV()

	t.Log("Registering Keys")
	// Create a test key/value pair
	data := make(map[string][]byte)

	data["wf/steps/step11/activity/delegate"] = []byte("install")
	data["wf/steps/step11/next/step10"] = []byte("")
	data["wf/steps/step11/next/step12"] = []byte("")
	data["wf/steps/step11/node"] = []byte("nodeName")

	data["wf/steps/step10/activity/delegate"] = []byte("install")
	data["wf/steps/step10/next/step13"] = []byte("")
	data["wf/steps/step10/node"] = []byte("nodeName")

	data["wf/steps/step12/activity/delegate"] = []byte("install")
	data["wf/steps/step12/next/step13"] = []byte("")
	data["wf/steps/step12/node"] = []byte("nodeName")

	data["wf/steps/step13/activity/delegate"] = []byte("install")
	data["wf/steps/step13/node"] = []byte("nodeName")

	data["wf/steps/step20/activity/delegate"] = []byte("install")
	data["wf/steps/step20/node"] = []byte("nodeName")

	srv1.PopulateKV(data)
	//kv.Put(&api.KVPair{Key: "/wf/steps/stepName/activity/delegate", Value:[]byte("install")}, nil)

	steps, err := readWorkFlowFromConsul(kv, "wf")
	require.Nil(t, err)
	require.Len(t, steps, 5)

}
