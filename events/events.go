package events

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"encoding/json"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

// InstanceStatusChange publishes a status change for a given instance of a given node
//
// InstanceStatusChange returns the published event id
func InstanceStatusChange(kv *api.KV, deploymentID, nodeName, instance, status string) (string, error) {
	id, err := storeStatusUpdateEvent(kv, deploymentID, InstanceStatusChangeType, nodeName+"\n"+status+"\n"+instance)
	if err != nil {
		return "", err
	}
	//TODO add log Optional fields
	SimpleLogEntry(INFO, deploymentID).RegisterAsString(fmt.Sprintf("Status for node %q, instance %q changed to %q", nodeName, instance, status))
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
	//TODO add log Optional fields
	SimpleLogEntry(INFO, deploymentID).RegisterAsString(fmt.Sprintf("Status for deployment %q changed to %q", deploymentID, status))
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
	//TODO add log Optional fields
	SimpleLogEntry(INFO, deploymentID).RegisterAsString(fmt.Sprintf("Status for custom-command %q changed to %q", taskID, status))
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
	//TODO add log Optional fields
	SimpleLogEntry(INFO, deploymentID).RegisterAsString(fmt.Sprintf("Status for scaling task %q changed to %q", taskID, status))
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
	//TODO add log Optional fields
	SimpleLogEntry(INFO, deploymentID).RegisterAsString(fmt.Sprintf("Status for workflow task %q changed to %q", taskID, status))
	return id, nil
}

// Create a KVPair corresponding to an event and put it to Consul under the event prefix,
// in a sub-tree corresponding to its deployment
// The eventType goes to the KVPair's Flags field
func storeStatusUpdateEvent(kv *api.KV, deploymentID string, eventType StatusUpdateType, data string) (string, error) {
	now := time.Now().Format(time.RFC3339Nano)
	eventsPrefix := path.Join(consulutil.EventsPrefix, deploymentID)
	p := &api.KVPair{Key: path.Join(eventsPrefix, now), Value: []byte(data), Flags: uint64(eventType)}
	_, err := kv.Put(p, nil)
	if err != nil {
		return "", err
	}
	return now, nil
}

// StatusEvents return a list of events (StatusUpdate instances) for all, or a given deployment
func StatusEvents(kv *api.KV, deploymentID string, waitIndex uint64, timeout time.Duration) ([]StatusUpdate, uint64, error) {
	events := make([]StatusUpdate, 0)

	var eventsPrefix string
	var depIDProvided bool
	if deploymentID != "" {
		// the returned list of events must correspond to the provided deploymentID
		eventsPrefix = path.Join(consulutil.EventsPrefix, deploymentID)
		depIDProvided = true
	} else {
		// the returned list of events must correspond to all the deployments
		eventsPrefix = path.Join(consulutil.EventsPrefix)
		depIDProvided = false
	}

	kvps, qm, err := kv.List(eventsPrefix, &api.QueryOptions{WaitIndex: waitIndex, WaitTime: timeout})
	if err != nil || qm == nil {
		return events, 0, err
	}
	for _, kvp := range kvps {
		if kvp.ModifyIndex <= waitIndex {
			continue
		}
		var eventTimestamp string
		if depIDProvided {
			eventTimestamp = strings.TrimPrefix(kvp.Key, eventsPrefix+"/")
		} else {
			depIDAndTimestamp := strings.Split(strings.TrimPrefix(kvp.Key, eventsPrefix+"/"), "/")
			deploymentID = depIDAndTimestamp[0]
			eventTimestamp = depIDAndTimestamp[1]
		}

		values := strings.Split(string(kvp.Value), "\n")
		eventType := StatusUpdateType(kvp.Flags)

		switch eventType {
		case InstanceStatusChangeType:
			if len(values) != 3 {
				return events, qm.LastIndex, errors.Errorf("Unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
			}
			events = append(events, StatusUpdate{Timestamp: eventTimestamp, Type: eventType.String(), Node: values[0], Status: values[1], Instance: values[2], DeploymentID: deploymentID})
		case DeploymentStatusChangeType:
			if len(values) != 1 {
				return events, qm.LastIndex, errors.Errorf("Unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
			}
			events = append(events, StatusUpdate{Timestamp: eventTimestamp, Type: eventType.String(), Status: values[0], DeploymentID: deploymentID})
		case CustomCommandStatusChangeType, ScalingStatusChangeType, WorkflowStatusChangeType:
			if len(values) != 2 {
				return events, qm.LastIndex, errors.Errorf("Unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
			}
			events = append(events, StatusUpdate{Timestamp: eventTimestamp, Type: eventType.String(), TaskID: values[0], Status: values[1], DeploymentID: deploymentID})
		default:
			return events, qm.LastIndex, errors.Errorf("Unsupported event type %d for event %q", kvp.Flags, kvp.Key)
		}
	}
	return events, qm.LastIndex, nil
}

// LogsEvents allows to return logs from Consul KV storage for all, or a given deployment
func LogsEvents(kv *api.KV, deploymentID string, waitIndex uint64, timeout time.Duration) ([]json.RawMessage, uint64, error) {
	logs := make([]json.RawMessage, 0)

	var logsPrefix string
	if deploymentID != "" {
		// the returned list of logs must correspond to the provided deploymentID
		logsPrefix = path.Join(consulutil.LogsPrefix, deploymentID)
	} else {
		// the returned list of logs must correspond to all the deployments
		logsPrefix = path.Join(consulutil.LogsPrefix)
	}
	kvps, qm, err := kv.List(logsPrefix, &api.QueryOptions{WaitIndex: waitIndex, WaitTime: timeout})
	if err != nil || qm == nil {
		return logs, 0, err
	}
	log.Debugf("Found %d logs before accessing index[%q]", len(kvps), strconv.FormatUint(qm.LastIndex, 10))
	for _, kvp := range kvps {
		if kvp.ModifyIndex <= waitIndex {
			continue
		}

		logs = append(logs, kvp.Value)
	}
	log.Debugf("Found %d logs after index", len(logs))
	return logs, qm.LastIndex, nil
}

// GetStatusEventsIndex returns the latest index of InstanceStatus events for a given deployment
func GetStatusEventsIndex(kv *api.KV, deploymentID string) (uint64, error) {
	_, qm, err := kv.Get(path.Join(consulutil.EventsPrefix, deploymentID), nil)
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
	_, qm, err := kv.Get(path.Join(consulutil.LogsPrefix, deploymentID), nil)
	if err != nil {
		return 0, err
	}
	if qm == nil {
		return 0, errors.New("Failed to retrieve last index for logs")
	}
	return qm.LastIndex, nil
}
