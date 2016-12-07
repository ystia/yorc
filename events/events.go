package events

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
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
	LogsEvents(filter string, waitIndex uint64, timeout time.Duration) ([]deployments.Logs, uint64, error)
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
	eventsPrefix := path.Join(consulutil.DeploymentKVPrefix, cp.deploymentId, "events", "global")
	eventsNodePrefix := path.Join(consulutil.DeploymentKVPrefix, cp.deploymentId, "events", nodeName)
	eventEntry := &api.KVPair{Key: path.Join(eventsPrefix, now), Value: []byte(nodeName + "\n" + status)}
	eventNodeEntry := &api.KVPair{Key: path.Join(eventsNodePrefix, "status"), Value: []byte(status)}
	if _, err := cp.kv.Put(eventEntry, nil); err != nil {
		return "", err
	}
	if _, err := cp.kv.Put(eventNodeEntry, nil); err != nil {
		return "", err
	}
	deployments.LogInConsul(cp.kv, cp.deploymentId, fmt.Sprintf("Status for node %q changed to %q", nodeName, status))
	return now, nil
}

func (cp *consulPubSub) NewEvents(waitIndex uint64, timeout time.Duration) ([]deployments.Event, uint64, error) {

	eventsPrefix := path.Join(consulutil.DeploymentKVPrefix, cp.deploymentId, "events", "global")
	kvps, qm, err := cp.kv.List(eventsPrefix, &api.QueryOptions{WaitIndex: waitIndex, WaitTime: timeout})
	events := make([]deployments.Event, 0)

	if err != nil || qm == nil {
		return events, 0, err
	}

	log.Debugf("Found %d events before filtering, last index is %q", len(kvps), strconv.FormatUint(qm.LastIndex, 10))

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

func (cp *consulPubSub) LogsEvents(filter string, waitIndex uint64, timeout time.Duration) ([]deployments.Logs, uint64, error) {

	eventsPrefix := path.Join(consulutil.DeploymentKVPrefix, cp.deploymentId, "logs")
	kvps, qm, err := cp.kv.List(eventsPrefix, &api.QueryOptions{WaitIndex: waitIndex, WaitTime: timeout})
	logs := make([]deployments.Logs, 0)
	if err != nil || qm == nil {
		return logs, 0, err
	}
	log.Debugf("Found %d events before filtering, last index is %q", len(kvps), strconv.FormatUint(qm.LastIndex, 10))
	for _, kvp := range kvps {
		if kvp.ModifyIndex <= waitIndex || (filter != "all" && !strings.HasPrefix(strings.TrimPrefix(kvp.Key, eventsPrefix+"/"), filter)) {
			continue
		}

		index := strings.Index(kvp.Key, "__")
		eventTimestamp := kvp.Key[index+2 : len(kvp.Key)]
		logs = append(logs, deployments.Logs{Timestamp: eventTimestamp, Logs: string(kvp.Value)})

	}

	log.Debugf("Found %d events after filtering", len(logs))
	return logs, qm.LastIndex, nil
}

func GetEventsIndex(kv *api.KV, deploymentId string) (uint64, error) {
	_, qm, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentId, "events", "global"), nil)
	if err != nil {
		return 0, err
	}
	if qm == nil {
		return 0, errors.New("Failed to retrieve last index for events")
	}
	return qm.LastIndex, nil
}

func GetLogsEventsIndex(kv *api.KV, deploymentId string) (uint64, error) {
	_, qm, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentId, "logs"), nil)
	if err != nil {
		return 0, err
	}
	if qm == nil {
		return 0, errors.New("Failed to retrieve last index for logs")
	}
	return qm.LastIndex, nil
}
