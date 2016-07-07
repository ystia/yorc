package tasks

import (
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadStepFromConsulFailing(t *testing.T) {
	//t.Parallel()
	srv1 := testutil.NewTestServerConfig(t, nil)
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

	kv := client.KV()

	t.Log("Registering Key")
	// Create a test key/value pair
	srv1.SetKV("wf/steps/stepName/activity/delegate", []byte("install"))

	step, err := readStep(kv, "wf/steps/", "stepName", nil)
	t.Log(err)
	assert.Nil(t, step)
	assert.Error(t, err)
}

func TestReadStepFromConsul(t *testing.T) {
	//t.Parallel()
	srv1 := testutil.NewTestServerConfig(t, nil)
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

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
	assert.Nil(t, err)
	assert.Equal(t, "nodeName", step.Node)
	assert.Equal(t, "stepName", step.Name)
	assert.Len(t, step.Activities, 3)
	assert.Contains(t, step.Activities, DelegateActivity{delegate: "install"})
	assert.Contains(t, step.Activities, SetStateActivity{state: "installed"})
	assert.Contains(t, step.Activities, CallOperationActivity{operation: "script.sh"})

	assert.Len(t, visitedMap, 1)
	assert.Contains(t, visitedMap, "stepName")
}

func TestReadStepWithNext(t *testing.T) {
	//t.Parallel()
	srv1 := testutil.NewTestServerConfig(t, nil)
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

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
	assert.Nil(t, err)
	assert.Equal(t, "nodeName", step.Node)
	assert.Equal(t, "stepName", step.Name)
	assert.Len(t, step.Activities, 1)

	assert.Len(t, visitedMap, 2)
	assert.Contains(t, visitedMap, "stepName")
	assert.Contains(t, visitedMap, "downstream")

	assert.Equal(t, 0, visitedMap["stepName"].refCount)
	assert.Equal(t, 1, visitedMap["downstream"].refCount)
}

func TestReadWorkFlowFromConsul(t *testing.T) {
	//t.Parallel()
	srv1 := testutil.NewTestServerConfig(t, nil)
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

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
	assert.Nil(t, err)
	assert.Len(t, steps, 2)
	for _, step := range steps {
		stepName := step.Name
		switch {
		case stepName == "step11":

		case stepName == "step20":
		default:
			assert.Fail(t, "Unexpected root step")
		}
	}
}
