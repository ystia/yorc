// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package events

import (
	"context"
	"path"
	"strconv"
	"strings"
	"time"

	"encoding/json"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
)

// InstanceStatusChange publishes a status change for a given instance of a given node
//
// InstanceStatusChange returns the published event id
//
// Deprecated: Use PublishAndLogInstanceStatusChange instead
func InstanceStatusChange(kv *api.KV, deploymentID, nodeName, instance, status string) (string, error) {
	return PublishAndLogInstanceStatusChange(nil, kv, deploymentID, nodeName, instance, status)
}

// PublishAndLogInstanceStatusChange publishes a status change for a given instance of a given node and log this change into the log API
//
// PublishAndLogInstanceStatusChange returns the published event id
func PublishAndLogInstanceStatusChange(ctx context.Context, kv *api.KV, deploymentID, nodeName, instance, status string) (string, error) {
	ctx = AddLogOptionalFields(ctx, LogOptionalFields{NodeID: nodeName, InstanceID: instance})

	id, err := storeStatusUpdateEvent(kv, deploymentID, InstanceStatusChangeType, nodeName+"\n"+status+"\n"+instance)
	if err != nil {
		return "", err
	}
	WithContextOptionalFields(ctx).NewLogEntry(LogLevelINFO, deploymentID).Registerf("Status for node %q, instance %q changed to %q", nodeName, instance, status)
	return id, nil
}

// DeploymentStatusChange publishes a status change for a given deployment
//
// DeploymentStatusChange returns the published event id
//
// Deprecated: use PublishAndLogDeploymentStatusChange instead
func DeploymentStatusChange(kv *api.KV, deploymentID, status string) (string, error) {
	return PublishAndLogDeploymentStatusChange(context.Background(), kv, deploymentID, status)
}

// PublishAndLogDeploymentStatusChange publishes a status change for a given deployment and log this change into the log API
//
// PublishAndLogDeploymentStatusChange returns the published event id
func PublishAndLogDeploymentStatusChange(ctx context.Context, kv *api.KV, deploymentID, status string) (string, error) {
	id, err := storeStatusUpdateEvent(kv, deploymentID, DeploymentStatusChangeType, status)
	if err != nil {
		return "", err
	}
	WithContextOptionalFields(ctx).NewLogEntry(LogLevelINFO, deploymentID).Registerf("Status for deployment %q changed to %q", deploymentID, status)
	return id, nil
}

// CustomCommandStatusChange publishes a status change for a custom command
//
// CustomCommandStatusChange returns the published event id
//
// Deprecated: use PublishAndLogCustomCommandStatusChange instead
func CustomCommandStatusChange(kv *api.KV, deploymentID, taskID, status string) (string, error) {
	return PublishAndLogCustomCommandStatusChange(nil, kv, deploymentID, taskID, status)
}

// PublishAndLogCustomCommandStatusChange publishes a status change for a custom command and log this change into the log API
//
// PublishAndLogCustomCommandStatusChange returns the published event id
func PublishAndLogCustomCommandStatusChange(ctx context.Context, kv *api.KV, deploymentID, taskID, status string) (string, error) {
	if ctx == nil {
		ctx = NewContext(context.Background(), LogOptionalFields{ExecutionID: taskID})
	}
	id, err := storeStatusUpdateEvent(kv, deploymentID, CustomCommandStatusChangeType, taskID+"\n"+status)
	if err != nil {
		return "", err
	}
	WithContextOptionalFields(ctx).NewLogEntry(LogLevelINFO, deploymentID).Registerf("Status for custom-command %q changed to %q", taskID, status)
	return id, nil
}

// ScalingStatusChange publishes a status change for a scaling task
//
// ScalingStatusChange returns the published event id
//
// Deprecated: use PublishAndLogScalingStatusChange instead
func ScalingStatusChange(kv *api.KV, deploymentID, taskID, status string) (string, error) {
	return PublishAndLogScalingStatusChange(nil, kv, deploymentID, taskID, status)
}

// PublishAndLogScalingStatusChange publishes a status change for a scaling task and log this change into the log API
//
// PublishAndLogScalingStatusChange returns the published event id
func PublishAndLogScalingStatusChange(ctx context.Context, kv *api.KV, deploymentID, taskID, status string) (string, error) {
	if ctx == nil {
		ctx = NewContext(context.Background(), LogOptionalFields{ExecutionID: taskID})
	}
	id, err := storeStatusUpdateEvent(kv, deploymentID, ScalingStatusChangeType, taskID+"\n"+status)
	if err != nil {
		return "", err
	}
	WithContextOptionalFields(ctx).NewLogEntry(LogLevelINFO, deploymentID).Registerf("Status for scaling task %q changed to %q", taskID, status)
	return id, nil
}

// WorkflowStatusChange publishes a status change for a workflow task
//
// WorkflowStatusChange returns the published event id
//
// Deprecated: use PublishAndLogWorkflowStatusChange instead
func WorkflowStatusChange(kv *api.KV, deploymentID, taskID, status string) (string, error) {
	return PublishAndLogWorkflowStatusChange(nil, kv, deploymentID, taskID, status)
}

// PublishAndLogWorkflowStatusChange publishes a status change for a workflow task and log this change into the log API
//
// PublishAndLogWorkflowStatusChange returns the published event id
func PublishAndLogWorkflowStatusChange(ctx context.Context, kv *api.KV, deploymentID, taskID, status string) (string, error) {
	if ctx == nil {
		ctx = NewContext(context.Background(), LogOptionalFields{ExecutionID: taskID})
	}
	id, err := storeStatusUpdateEvent(kv, deploymentID, WorkflowStatusChangeType, taskID+"\n"+status)
	if err != nil {
		return "", err
	}
	WithContextOptionalFields(ctx).NewLogEntry(LogLevelINFO, deploymentID).Registerf("Status for workflow task %q changed to %q", taskID, status)
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
