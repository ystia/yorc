package events

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
	"path"
	"strconv"
	"strings"
	"time"
)

type Publisher interface {
	StatusChange(nodeName, status string) (string, error)
}

type Subscriber interface {
	NewEvents(waitIndex uint64, timeout time.Duration) ([]deployments.Event, uint64, error)
	NewNodeEvents(nodeName string) (deployments.Status, error)
}

type consulPubSub struct {
	kv           *api.KV
	deploymentId string
}

func NewPublisher(kv *api.KV, deploymentId string) Publisher {
	return &consulPubSub{kv: kv, deploymentId: deploymentId}
}

func NewSubscriber(kv *api.KV, deploymentId string) Subscriber {
	return &consulPubSub{kv: kv, deploymentId: deploymentId}
}

func (cp *consulPubSub) StatusChange(nodeName, status string) (string, error) {
	now := time.Now().Format(time.RFC3339Nano)
	eventsPrefix := path.Join(deployments.DeploymentKVPrefix, cp.deploymentId, "events", "global")
	eventsNodePrefix := path.Join(deployments.DeploymentKVPrefix, cp.deploymentId, "events", nodeName)
	eventEntry := &api.KVPair{Key: path.Join(eventsPrefix, now), Value: []byte(nodeName + "\n" + status)}
	eventNodeEntry := &api.KVPair{Key: path.Join(eventsNodePrefix, "statut"), Value: []byte(status)}
	if _, err := cp.kv.Put(eventEntry, nil); err != nil {
		return "", err
	}
	if _, err := cp.kv.Put(eventNodeEntry, nil); err != nil {
		return "", err
	}
	return now, nil
}

func (cp *consulPubSub) NewEvents(waitIndex uint64, timeout time.Duration) ([]deployments.Event, uint64, error) {

	eventsPrefix := path.Join(deployments.DeploymentKVPrefix, cp.deploymentId, "events", "global")
	kvps, qm, err := cp.kv.List(eventsPrefix, &api.QueryOptions{WaitIndex: waitIndex, WaitTime: timeout})
	events := make([]deployments.Event, 0)
	log.Debugf("Found %d events before filtering, last index is %q", len(kvps), strconv.FormatUint(qm.LastIndex, 10))

	if err != nil {
		return events, 0, err
	}
	for _, kvp := range kvps {
		if kvp.ModifyIndex <= waitIndex {
			continue
		}

		eventTimestamp := strings.TrimPrefix(kvp.Key, eventsPrefix+"/")
		values := strings.Split(string(kvp.Value), "\n")
		if len(values) != 2 {
			return events, qm.LastIndex, fmt.Errorf("Unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
		}
		events = append(events, deployments.Event{Timestamp: eventTimestamp, Node: values[0], Status: values[1]})
	}

	log.Debugf("Found %d events after filtering", len(events))
	return events, qm.LastIndex, nil
}

func (cp *consulPubSub) NewNodeEvents(nodeName string) (deployments.Status, error) {

	eventsPrefix := path.Join(deployments.DeploymentKVPrefix, cp.deploymentId, "events", nodeName, "statut")

	kvp, _, err := cp.kv.Get(eventsPrefix, nil)

	var statut deployments.Status

	if err != nil {
		return statut, err
	}

	values := strings.Split(string(kvp.Value), "\n")
	if len(values) != 1 {
		return statut, fmt.Errorf("Unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
	}

	statut = deployments.Status{Status: values[0]}

	return statut, nil
}
