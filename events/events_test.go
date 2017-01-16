package events

import (
	"path"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

func TestGroupedEventParallel(t *testing.T) {
	srv1 := testutil.NewTestServerConfig(t, nil)
	defer srv1.Stop()

	config := api.DefaultConfig()
	config.Address = srv1.HTTPAddr

	client, err := api.NewClient(config)
	assert.Nil(t, err)

	kv := client.KV()

	consulutil.InitConsulPublisher(500, kv)
	t.Run("groupEvent", func(t *testing.T) {
		t.Run("TestConsulPubSubStatusChange", func(t *testing.T) {
			ConsulPubSubStatusChange(t, kv)
		})
		t.Run("TestConsulPubSubNewEvents", func(t *testing.T) {
			ConsulPubSubNewEvents(t, kv)
		})
		t.Run("TestConsulPubSubNewEventsTimeout", func(t *testing.T) {
			ConsulPubSubNewEventsTimeout(t, kv)
		})
		t.Run("TestConsulPubSubNewEventsWithIndex", func(t *testing.T) {
			ConsulPubSubNewEventsWithIndex(t, kv)
		})
		t.Run("TestConsulPubSubNewNodeEvents", func(t *testing.T) {
			ConsulPubSubNewNodeEvents(t, kv)
		})
	})
}

func ConsulPubSubStatusChange(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := "test1"
	pub := NewPublisher(kv, deploymentID)

	var testData = []struct {
		node     string
		instance string
		status   string
	}{
		{"node1", "0", "initial"},
		{"node2", "0", "initial"},
		{"node1", "0", "created"},
		{"node1", "0", "started"},
		{"node2", "0", "created"},
		{"node3", "0", "initial"},
		{"node2", "0", "configured"},
		{"node3", "0", "created"},
		{"node2", "0", "started"},
		{"node3", "0", "error"},
	}

	ids := make([]string, 0)
	for _, tc := range testData {
		id, err := pub.StatusChange(tc.node, tc.instance, tc.status)
		assert.Nil(t, err)
		ids = append(ids, id)
	}
	prefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "events")
	kvps, _, err := kv.List(prefix, nil)
	assert.Nil(t, err)
	assert.Len(t, kvps, len(testData))

	for index, kvp := range kvps {
		assert.Equal(t, ids[index], strings.TrimPrefix(kvp.Key, prefix+"/"))
		tc := testData[index]
		assert.Equal(t, tc.node+"\n"+tc.status+"\n"+tc.instance, string(kvp.Value))
	}
}

func ConsulPubSubNewEvents(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := "test2"
	pub := NewPublisher(kv, deploymentID)
	sub := NewSubscriber(kv, deploymentID)

	nodeName := "node1"
	instance := "0"
	nodeStatus := "error"

	ready := make(chan struct{})

	go func() {
		i, err := GetLogsEventsIndex(kv, deploymentID)
		require.Nil(t, err)
		ready <- struct{}{}
		events, _, err := sub.NewEvents(i, 5*time.Minute)
		assert.Nil(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, events[0].Node, nodeName)
		assert.Equal(t, events[0].Status, nodeStatus)
		assert.Equal(t, events[0].Instance, instance)
	}()
	<-ready
	_, err := pub.StatusChange(nodeName, instance, nodeStatus)
	assert.Nil(t, err)
}

func ConsulPubSubNewEventsTimeout(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := "test3"
	sub := NewSubscriber(kv, deploymentID)

	timeout := 25 * time.Millisecond

	t1 := time.Now()
	events, _, err := sub.NewEvents(1, timeout)
	t2 := time.Now()
	assert.Nil(t, err)
	require.Len(t, events, 0)
	assert.WithinDuration(t, t1, t2, timeout+50*time.Millisecond)
}

func ConsulPubSubNewEventsWithIndex(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := "test4"
	pub := NewPublisher(kv, deploymentID)
	sub := NewSubscriber(kv, deploymentID)

	var testData = []struct {
		node     string
		instance string
		status   string
	}{
		{"node1", "0", "initial"},
		{"node1", "1", "initial"},
		{"node1", "0", "creating"},
		{"node1", "1", "creating"},
	}

	for _, tc := range testData {
		_, err := pub.StatusChange(tc.node, tc.instance, tc.status)
		assert.Nil(t, err)
	}

	events, lastIdx, err := sub.NewEvents(1, 5*time.Minute)
	assert.Nil(t, err)
	require.Len(t, events, 4)
	for index, event := range events {
		assert.Equal(t, testData[index].node, event.Node)
		assert.Equal(t, testData[index].instance, event.Instance)
		assert.Equal(t, testData[index].status, event.Status)
	}

	testData = []struct {
		node     string
		instance string
		status   string
	}{
		{"node1", "0", "created"},
		{"node1", "0", "configuring"},
		{"node1", "0", "configured"},
	}

	for _, tc := range testData {
		_, err = pub.StatusChange(tc.node, tc.instance, tc.status)
		assert.Nil(t, err)
	}

	events, lastIdx, err = sub.NewEvents(lastIdx, 5*time.Minute)
	assert.Nil(t, err)
	require.Len(t, events, 3)
	require.NotZero(t, lastIdx)

	for index, event := range events {
		assert.Equal(t, testData[index].node, event.Node)
		assert.Equal(t, testData[index].instance, event.Instance)
		assert.Equal(t, testData[index].status, event.Status)
	}
}

func ConsulPubSubNewNodeEvents(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := "test5"
	pub := NewPublisher(kv, deploymentID)

	nodeName := "node1"
	instance := "0"
	nodeStatus := "error"

	_, err := pub.StatusChange(nodeName, instance, nodeStatus)
	assert.Nil(t, err)

}
