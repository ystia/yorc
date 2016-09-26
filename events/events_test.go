package events

import (
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
	"path"
	"strings"
	"testing"
	"time"
)

func TestGroupedEventParallel(t *testing.T) {
	t.Run("groupEvent", func(t *testing.T) {
		t.Run("TestConsulPubSub_StatusChange", ConsulPubSub_StatusChange)
		t.Run("TestConsulPubSub_NewEvents", ConsulPubSub_NewEvents)
		t.Run("TestConsulPubSub_NewEventsTimeout", ConsulPubSub_NewEventsTimeout)
		t.Run("TestConsulPubSub_NewEventsWithIndex", ConsulPubSub_NewEventsWithIndex)
		t.Run("TestConsulPubSub_NewNodeEvents", ConsulPubSub_NewNodeEvents)
	})
}

func ConsulPubSub_StatusChange(t *testing.T) {
	t.Parallel()
	srv1 := testutil.NewTestServerConfig(t, nil)
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

	kv := client.KV()
	deploymentId := "test1"
	pub := NewPublisher(kv, deploymentId)

	var testData = []struct {
		node   string
		status string
	}{
		{"node1", "initial"},
		{"node2", "initial"},
		{"node1", "created"},
		{"node1", "started"},
		{"node2", "created"},
		{"node3", "initial"},
		{"node2", "configured"},
		{"node3", "created"},
		{"node2", "started"},
		{"node3", "error"},
	}

	ids := make([]string, 0)
	for _, tc := range testData {
		id, err := pub.StatusChange(tc.node, tc.status)
		assert.Nil(t, err)
		ids = append(ids, id)
	}
	prefix := path.Join(deployments.DeploymentKVPrefix, deploymentId, "events", "global")
	kvps, _, err := kv.List(prefix, nil)
	assert.Nil(t, err)
	assert.Len(t, kvps, len(testData))

	for index, kvp := range kvps {
		assert.Equal(t, ids[index], strings.TrimPrefix(kvp.Key, prefix+"/"))
		tc := testData[index]
		assert.Equal(t, tc.node+"\n"+tc.status, string(kvp.Value))
	}
}

func ConsulPubSub_NewEvents(t *testing.T) {
	t.Parallel()
	srv1 := testutil.NewTestServerConfig(t, nil)
	defer srv1.Stop()
	log.SetDebug(true)
	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

	kv := client.KV()
	deploymentId := "test2"
	pub := NewPublisher(kv, deploymentId)
	sub := NewSubscriber(kv, deploymentId)

	nodeName := "node1"
	nodeStatus := "error"

	go func() {
		events, _, err := sub.NewEvents(1, 5*time.Minute)
		assert.Nil(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, events[0].Node, nodeName)
		assert.Equal(t, events[0].Status, nodeStatus)
	}()

	_, err = pub.StatusChange(nodeName, nodeStatus)
	assert.Nil(t, err)
}

func ConsulPubSub_NewEventsTimeout(t *testing.T) {
	t.Parallel()
	srv1 := testutil.NewTestServerConfig(t, nil)
	defer srv1.Stop()
	log.SetDebug(true)
	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

	kv := client.KV()
	deploymentId := "test3"
	sub := NewSubscriber(kv, deploymentId)

	timeout := 25 * time.Millisecond

	t1 := time.Now()
	events, _, err := sub.NewEvents(1, timeout)
	t2 := time.Now()
	assert.Nil(t, err)
	require.Len(t, events, 0)
	assert.WithinDuration(t, t1, t2, timeout+50*time.Millisecond)
}

func ConsulPubSub_NewEventsWithIndex(t *testing.T) {
	t.Parallel()
	srv1 := testutil.NewTestServerConfig(t, nil)
	defer srv1.Stop()
	log.SetDebug(true)
	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

	kv := client.KV()
	deploymentId := "test4"
	pub := NewPublisher(kv, deploymentId)
	sub := NewSubscriber(kv, deploymentId)

	var testData = []struct {
		node   string
		status string
	}{
		{"node1", "initial"},
		{"node1", "creating"},
	}

	for _, tc := range testData {
		_, err := pub.StatusChange(tc.node, tc.status)
		assert.Nil(t, err)
	}

	events, lastIdx, err := sub.NewEvents(1, 5*time.Minute)
	assert.Nil(t, err)
	require.Len(t, events, 2)
	for index, event := range events {
		assert.Equal(t, testData[index].node, event.Node)
		assert.Equal(t, testData[index].status, event.Status)
	}

	testData = []struct {
		node   string
		status string
	}{
		{"node1", "created"},
		{"node1", "configuring"},
		{"node1", "configured"},
	}

	for _, tc := range testData {
		_, err := pub.StatusChange(tc.node, tc.status)
		assert.Nil(t, err)
	}

	events, lastIdx, err = sub.NewEvents(lastIdx, 5*time.Minute)
	assert.Nil(t, err)
	require.Len(t, events, 3)

	for index, event := range events {
		assert.Equal(t, testData[index].node, event.Node)
		assert.Equal(t, testData[index].status, event.Status)
	}
}

func ConsulPubSub_NewNodeEvents(t *testing.T) {
	srv1 := testutil.NewTestServerConfig(t, nil)
	defer srv1.Stop()
	log.SetDebug(true)
	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

	kv := client.KV()
	deploymentId := "test5"
	pub := NewPublisher(kv, deploymentId)

	nodeName := "node1"
	nodeStatus := "error"

	_, err = pub.StatusChange(nodeName, nodeStatus)
	assert.Nil(t, err)

}
