package events

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

type Publisher interface {
	StatusChange(nodeName, instance, status string) (string, error)
}

type Subscriber interface {
	NewEvents(waitIndex uint64, timeout time.Duration) ([]deployments.Event, uint64, error)
	LogsEvents(filter string, waitIndex uint64, timeout time.Duration) ([]deployments.Logs, uint64, error)
}

type consulPubSub struct {
	kv           *api.KV
	deploymentID string
}

func NewPublisher(kv *api.KV, deploymentID string) Publisher {
	return &consulPubSub{kv: kv, deploymentID: deploymentID}
}

func NewSubscriber(kv *api.KV, deploymentID string) Subscriber {
	return &consulPubSub{kv: kv, deploymentID: deploymentID}
}

func (cp *consulPubSub) StatusChange(nodeName, instance, status string) (string, error) {
	now := time.Now().Format(time.RFC3339Nano)
	eventsPrefix := path.Join(consulutil.DeploymentKVPrefix, cp.deploymentID, "events")
	err := consulutil.StoreConsulKeyAsString(path.Join(eventsPrefix, now), nodeName+"\n"+status+"\n"+instance)
	if err != nil {
		return "", err
	}
	deployments.LogInConsul(cp.kv, cp.deploymentID, fmt.Sprintf("Status for node %q, instance %q changed to %q", nodeName, instance, status))
	return now, nil
}

func (cp *consulPubSub) NewEvents(waitIndex uint64, timeout time.Duration) ([]deployments.Event, uint64, error) {

	eventsPrefix := path.Join(consulutil.DeploymentKVPrefix, cp.deploymentID, "events")
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
		if len(values) != 3 {
			return events, qm.LastIndex, fmt.Errorf("Unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
		}
		events = append(events, deployments.Event{Timestamp: eventTimestamp, Node: values[0], Status: values[1], Instance: values[2]})
	}

	log.Debugf("Found %d events after filtering", len(events))
	return events, qm.LastIndex, nil
}

func (cp *consulPubSub) LogsEvents(filter string, waitIndex uint64, timeout time.Duration) ([]deployments.Logs, uint64, error) {

	eventsPrefix := path.Join(consulutil.DeploymentKVPrefix, cp.deploymentID, "logs")
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

func GetEventsIndex(kv *api.KV, deploymentID string) (uint64, error) {
	_, qm, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "events"), nil)
	if err != nil {
		return 0, err
	}
	if qm == nil {
		return 0, errors.New("Failed to retrieve last index for events")
	}
	return qm.LastIndex, nil
}

func GetLogsEventsIndex(kv *api.KV, deploymentID string) (uint64, error) {
	_, qm, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "logs"), nil)
	if err != nil {
		return 0, err
	}
	if qm == nil {
		return 0, errors.New("Failed to retrieve last index for logs")
	}
	return qm.LastIndex, nil
}
