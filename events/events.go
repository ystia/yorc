package events

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

// A Subscriber is used to poll for new StatusChange and LogEntry events
type Subscriber interface {
	StatusEvents(waitIndex uint64, timeout time.Duration) ([]StatusUpdate, uint64, error)
	LogsEvents(filter string, waitIndex uint64, timeout time.Duration) ([]LogEntry, uint64, error)
}

type consulPubSub struct {
	kv           *api.KV
	deploymentID string
}

// NewSubscriber returns an instance of Subscriber
func NewSubscriber(kv *api.KV, deploymentID string) Subscriber {
	return &consulPubSub{kv: kv, deploymentID: deploymentID}
}

// InstanceStatusChange publishes a status change for a given instance of a given node
//
// InstanceStatusChange returns the published event id
func InstanceStatusChange(kv *api.KV, deploymentID, nodeName, instance, status string) (string, error) {
	id, err := storeStatusUpdateEvent(kv, deploymentID, InstanceStatusChangeType, nodeName+"\n"+status+"\n"+instance)
	if err != nil {
		return "", err
	}
	LogEngineMessage(kv, deploymentID, fmt.Sprintf("Status for node %q, instance %q changed to %q", nodeName, instance, status))
	return id, nil
}

// DeploymentStatusChange publishes a status change for a given deployment
//
// DeploymentStatusChange returns the published event id
func DeploymentStatusChange(kv *api.KV, deploymentID, status string) (string, error) {
	id, err := storeStatusUpdateEvent(kv, deploymentID, DeploymentStatusChangeType, status)
	if err != nil {
		return "", err
	}
	LogEngineMessage(kv, deploymentID, fmt.Sprintf("Status for deployment %q changed to %q", deploymentID, status))
	return id, nil
}

// CustomCommandStatusChange publishes a status change for a custom command
//
// CustomCommandStatusChange returns the published event id
func CustomCommandStatusChange(kv *api.KV, deploymentID, taskID, status string) (string, error) {
	id, err := storeStatusUpdateEvent(kv, deploymentID, CustomCommandStatusChangeType, taskID+"\n"+status)
	if err != nil {
		return "", err
	}
	LogEngineMessage(kv, deploymentID, fmt.Sprintf("Status for custom-command %q changed to %q", taskID, status))
	return id, nil
}

// ScalingStatusChange publishes a status change for a scaling task
//
// ScalingStatusChange returns the published event id
func ScalingStatusChange(kv *api.KV, deploymentID, taskID, status string) (string, error) {
	id, err := storeStatusUpdateEvent(kv, deploymentID, ScalingStatusChangeType, taskID+"\n"+status)
	if err != nil {
		return "", err
	}
	LogEngineMessage(kv, deploymentID, fmt.Sprintf("Status for scaling task %q changed to %q", taskID, status))
	return id, nil
}

// WorkflowStatusChange publishes a status change for a workflow task
//
// WorkflowStatusChange returns the published event id
func WorkflowStatusChange(kv *api.KV, deploymentID, taskID, status string) (string, error) {
	id, err := storeStatusUpdateEvent(kv, deploymentID, WorkflowStatusChangeType, taskID+"\n"+status)
	if err != nil {
		return "", err
	}
	LogEngineMessage(kv, deploymentID, fmt.Sprintf("Status for workflow task %q changed to %q", taskID, status))
	return id, nil
}

func storeStatusUpdateEvent(kv *api.KV, deploymentID string, eventType StatusUpdateType, data string) (string, error) {
	now := time.Now().Format(time.RFC3339Nano)
	eventsPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "events")
	p := &api.KVPair{Key: path.Join(eventsPrefix, now), Value: []byte(data), Flags: uint64(eventType)}
	_, err := kv.Put(p, nil)
	if err != nil {
		return "", err
	}
	return now, nil
}

func (cp *consulPubSub) StatusEvents(waitIndex uint64, timeout time.Duration) ([]StatusUpdate, uint64, error) {

	eventsPrefix := path.Join(consulutil.DeploymentKVPrefix, cp.deploymentID, "events")
	kvps, qm, err := cp.kv.List(eventsPrefix, &api.QueryOptions{WaitIndex: waitIndex, WaitTime: timeout})
	events := make([]StatusUpdate, 0)

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
		eventType := StatusUpdateType(kvp.Flags)
		switch eventType {
		case InstanceStatusChangeType:
			if len(values) != 3 {
				return events, qm.LastIndex, errors.Errorf("Unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
			}
			events = append(events, StatusUpdate{Timestamp: eventTimestamp, Type: eventType.String(), Node: values[0], Status: values[1], Instance: values[2]})
		case DeploymentStatusChangeType:
			if len(values) != 1 {
				return events, qm.LastIndex, errors.Errorf("Unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
			}
			events = append(events, StatusUpdate{Timestamp: eventTimestamp, Type: eventType.String(), Status: values[0]})
		case CustomCommandStatusChangeType, ScalingStatusChangeType, WorkflowStatusChangeType:
			if len(values) != 2 {
				return events, qm.LastIndex, errors.Errorf("Unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
			}
			events = append(events, StatusUpdate{Timestamp: eventTimestamp, Type: eventType.String(), TaskID: values[0], Status: values[1]})
		default:
			return events, qm.LastIndex, errors.Errorf("Unsupported event type %d for event %q", kvp.Flags, kvp.Key)
		}

	}

	log.Debugf("Found %d events after filtering", len(events))
	return events, qm.LastIndex, nil
}

func (cp *consulPubSub) LogsEvents(filter string, waitIndex uint64, timeout time.Duration) ([]LogEntry, uint64, error) {

	eventsPrefix := path.Join(consulutil.DeploymentKVPrefix, cp.deploymentID, "logs")
	kvps, qm, err := cp.kv.List(eventsPrefix, &api.QueryOptions{WaitIndex: waitIndex, WaitTime: timeout})
	logs := make([]LogEntry, 0)
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
		logs = append(logs, LogEntry{Timestamp: eventTimestamp, Logs: string(kvp.Value)})

	}

	log.Debugf("Found %d events after filtering", len(logs))
	return logs, qm.LastIndex, nil
}

// GetStatusEventsIndex returns the latest index of InstanceStatus events for a given deployment
func GetStatusEventsIndex(kv *api.KV, deploymentID string) (uint64, error) {
	_, qm, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "events"), nil)
	if err != nil {
		return 0, err
	}
	if qm == nil {
		return 0, errors.New("Failed to retrieve last index for events")
	}
	return qm.LastIndex, nil
}

// GetLogsEventsIndex returns the latest index of LogEntry events for a given deployment
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

// LogEngineMessage stores a engine log message
func LogEngineMessage(kv *api.KV, deploymentID, message string) {
	logInConsulAsString(kv, deploymentID, EngineLogPrefix, message)
}

// LogEngineError stores an engine error message.
//
// Basically it's a shortcut for:
//	LogEngineMessage(kv, deploymentID, fmt.Sprintf("%v", err))
func LogEngineError(kv *api.KV, deploymentID string, err error) {
	LogEngineMessage(kv, deploymentID, fmt.Sprintf("%v", err))
}

// LogSoftwareMessage stores a software provisioning log message
func LogSoftwareMessage(kv *api.KV, deploymentID, message string) {
	logInConsulAsString(kv, deploymentID, SoftwareLogPrefix, message)
}

// LogInfrastructureMessage stores a infrastructure provisioning log message
func LogInfrastructureMessage(kv *api.KV, deploymentID, message string) {
	logInConsulAsString(kv, deploymentID, InfraLogPrefix, message)
}

func logInConsulAsString(kv *api.KV, deploymentID, logType, message string) {
	logInConsul(kv, deploymentID, logType, []byte(message))
}

func logInConsul(kv *api.KV, deploymentID, logType string, message []byte) {
	if kv == nil || deploymentID == "" {
		log.Panic("Can't use LogInConsul function without KV or deployment ID")
	}
	err := consulutil.StoreConsulKey(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "logs", logType+"__"+time.Now().Format(time.RFC3339Nano)), message)
	if err != nil {
		log.Printf("Failed to publish log in consul for deployment %q: %+v", deploymentID, err)
	}
}
