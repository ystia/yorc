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

// StatusEvents return a list of events (StatusUpdate instances) for a givent deploymentID
func StatusEvents(kv *api.KV, deploymentID string, waitIndex uint64, timeout time.Duration) ([]StatusUpdate, uint64, error) {
	events := make([]StatusUpdate, 0)

	if deploymentID != "" {
		return deploymentStatusEvents(kv, deploymentID, waitIndex, timeout)
	}
	// Get the deployment IDs
	depPaths, _, err := kv.Keys(consulutil.DeploymentKVPrefix+"/", "/", nil)
	if err != nil {
		return events, 0, err
	}
	if len(depPaths) == 0 {
		log.Debug("=== no deployments")
		return events, 0, nil
	}

	var lastIndex uint64
	for depIndex, depPath := range depPaths {
		depID := strings.TrimRight(strings.TrimPrefix(depPath, consulutil.DeploymentKVPrefix+"/"), "/ ")
		log.Debugf("=== Found deployment %s with index %d", depID, depIndex)
		depStatusUpdates, depLastIndex, err := deploymentStatusEvents(kv, depID, waitIndex, timeout)
		if err != nil {
			return events, lastIndex, err
		}
		if depLastIndex > lastIndex {
			lastIndex = depLastIndex
		}
		log.Debugf("=== lastIndex = %d", lastIndex)
		for _, depStatusUpdate := range depStatusUpdates {
			events = append(events, depStatusUpdate)
		}
	}
	return events, lastIndex, nil
}

func deploymentStatusEvents(kv *api.KV, deploymentID string, waitIndex uint64, timeout time.Duration) ([]StatusUpdate, uint64, error) {
	deploymentEvents := make([]StatusUpdate, 0)
	deploymentEventsPrefix := path.Join(consulutil.EventsPrefix, deploymentID)
	// Get all the events for a deployment using a QueryOptions with the given waitIndex and with WaitTime corresponding to the given timeout
	kvps, qm, err := kv.List(deploymentEventsPrefix, &api.QueryOptions{WaitIndex: waitIndex, WaitTime: timeout})

	if err != nil || qm == nil {
		return deploymentEvents, 0, err
	}

	for _, kvp := range kvps {

		if kvp.ModifyIndex <= waitIndex {
			continue
		}

		eventTimestamp := strings.TrimPrefix(kvp.Key, deploymentEventsPrefix+"/")
		values := strings.Split(string(kvp.Value), "\n")
		eventType := StatusUpdateType(kvp.Flags)
		switch eventType {
		case InstanceStatusChangeType:
			if len(values) != 3 {
				return deploymentEvents, qm.LastIndex, errors.Errorf("Unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
			}
			deploymentEvents = append(deploymentEvents, StatusUpdate{Timestamp: eventTimestamp, Type: eventType.String(), Node: values[0], Status: values[1], Instance: values[2], DeploymentID: deploymentID})
		case DeploymentStatusChangeType:
			if len(values) != 1 {
				return deploymentEvents, qm.LastIndex, errors.Errorf("Unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
			}
			deploymentEvents = append(deploymentEvents, StatusUpdate{Timestamp: eventTimestamp, Type: eventType.String(), Status: values[0], DeploymentID: deploymentID})
		case CustomCommandStatusChangeType, ScalingStatusChangeType, WorkflowStatusChangeType:
			if len(values) != 2 {
				return deploymentEvents, qm.LastIndex, errors.Errorf("Unexpected event value %q for event %q", string(kvp.Value), kvp.Key)
			}
			deploymentEvents = append(deploymentEvents, StatusUpdate{Timestamp: eventTimestamp, Type: eventType.String(), TaskID: values[0], Status: values[1], DeploymentID: deploymentID})
		default:
			return deploymentEvents, qm.LastIndex, errors.Errorf("Unsupported event type %d for event %q", kvp.Flags, kvp.Key)
		}
	}

	return deploymentEvents, qm.LastIndex, nil
}

// LogsEvents allows to return logs from Consul KV storage for a given deploymentID
func LogsEvents(kv *api.KV, deploymentID string, waitIndex uint64, timeout time.Duration) ([]json.RawMessage, uint64, error) {
	logs := make([]json.RawMessage, 0)

	if deploymentID != "" {
		return deploymentLogsEvents(kv, deploymentID, waitIndex, timeout)
	}

	// Get the deployment IDs
	depPaths, _, err := kv.Keys(consulutil.DeploymentKVPrefix, "/", nil)
	if err != nil {
		return logs, 0, err
	}
	if len(depPaths) == 0 {
		log.Debug("=== no deployments")
		return logs, 0, nil
	}
	var lastIndex uint64
	for _, depPath := range depPaths {
		log.Debugf("=== lastIndex = %d", lastIndex)
		depID := strings.TrimRight(strings.TrimPrefix(depPath, consulutil.DeploymentKVPrefix), "/ ")
		depLogs, depLastIndex, err := deploymentLogsEvents(kv, depID, waitIndex, timeout)
		if err != nil {
			return logs, lastIndex, err
		}
		if depLastIndex > lastIndex {
			lastIndex = depLastIndex
		}
		for _, depLog := range depLogs {
			logs = append(logs, depLog)
		}
	}
	log.Debugf("Found %d logs after index", len(logs))
	return logs, lastIndex, nil
}

func deploymentLogsEvents(kv *api.KV, deploymentID string, waitIndex uint64, timeout time.Duration) ([]json.RawMessage, uint64, error) {
	logs := make([]json.RawMessage, 0)

	logsPrefix := path.Join(consulutil.LogsPrefix, deploymentID)
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
