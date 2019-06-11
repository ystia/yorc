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
	"encoding/json"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/log"
)

// PublishAndLogAttributeValueChange publishes a value change for a given attribute instance of a given node and log this change into the log API
//
// PublishAndLogAttributeValueChange returns the published event id
func PublishAndLogAttributeValueChange(ctx context.Context, deploymentID, nodeName, instanceName, attributeName, value, status string) (string, error) {
	ctx = AddLogOptionalFields(ctx, LogOptionalFields{NodeID: nodeName, InstanceID: instanceName})

	info := buildInfoFromContext(ctx)
	info[ENodeID] = nodeName
	info[EInstanceID] = instanceName
	info[EAttributeName] = attributeName
	info[EAttributeValue] = value
	e, err := newStatusChange(StatusChangeTypeAttributeValue, info, deploymentID, status)
	if err != nil {
		return "", err
	}
	id, err := e.register()
	if err != nil {
		return "", err
	}
	WithContextOptionalFields(ctx).NewLogEntry(LogLevelINFO, deploymentID).Registerf("Attribute %q value for node %q, instance %q changed to %q", attributeName, nodeName, instanceName, value)
	return id, nil
}

// PublishAndLogMapAttributeValueChange publishes a map attribute/value change for a given attribute instance of a given node and log this change into the log API
// This function doesn't return any published event id
func PublishAndLogMapAttributeValueChange(ctx context.Context, deploymentID, nodeName, instanceName string, attributesValues map[string]string, status string) error {
	for attr, attrVal := range attributesValues {
		_, err := PublishAndLogAttributeValueChange(ctx, deploymentID, nodeName, instanceName, attr, attrVal, status)
		if err != nil {
			return err
		}
	}
	return nil
}

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
	info := buildInfoFromContext(ctx)
	info[ENodeID] = nodeName
	info[EInstanceID] = instance
	e, err := newStatusChange(StatusChangeTypeInstance, info, deploymentID, strings.ToLower(status))
	if err != nil {
		return "", err
	}
	id, err := e.register()
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
	e, err := newStatusChange(StatusChangeTypeDeployment, nil, deploymentID, strings.ToLower(status))
	if err != nil {
		return "", err
	}
	id, err := e.register()
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
	info := buildInfoFromContext(ctx)
	info[ETaskID] = taskID
	e, err := newStatusChange(StatusChangeTypeCustomCommand, info, deploymentID, strings.ToLower(status))
	if err != nil {
		return "", err
	}
	id, err := e.register()
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
	info := buildInfoFromContext(ctx)
	info[ETaskID] = taskID
	e, err := newStatusChange(StatusChangeTypeScaling, info, deploymentID, strings.ToLower(status))
	if err != nil {
		return "", err
	}
	id, err := e.register()
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
func WorkflowStatusChange(kv *api.KV, deploymentID, taskID, workflowName, status string) (string, error) {
	return PublishAndLogWorkflowStatusChange(nil, kv, deploymentID, taskID, workflowName, status)
}

// PublishAndLogWorkflowStepStatusChange publishes a status change for a workflow step execution and log this change into the log API
//
// PublishAndLogWorkflowStepStatusChange returns the published event id
func PublishAndLogWorkflowStepStatusChange(ctx context.Context, kv *api.KV, deploymentID, taskID string, wfStepInfo *WorkflowStepInfo, status string) (string, error) {
	if ctx == nil {
		ctx = NewContext(context.Background(), LogOptionalFields{ExecutionID: taskID})
	}
	if wfStepInfo == nil {
		return "", errors.Errorf("WorkflowStep information  param must be provided")
	}
	info := buildInfoFromContext(ctx)
	info[ETaskID] = taskID
	info[EInstanceID] = wfStepInfo.InstanceName
	info[EWorkflowID] = wfStepInfo.WorkflowName
	info[ENodeID] = wfStepInfo.NodeName
	info[EWorkflowStepID] = wfStepInfo.StepName
	info[EOperationName] = wfStepInfo.OperationName
	info[ETargetNodeID] = wfStepInfo.TargetNodeID
	info[ETargetInstanceID] = wfStepInfo.TargetInstanceID
	e, err := newStatusChange(StatusChangeTypeWorkflowStep, info, deploymentID, strings.ToLower(status))
	if err != nil {
		return "", err
	}
	id, err := e.register()
	if err != nil {
		return "", err
	}
	WithContextOptionalFields(ctx).NewLogEntry(LogLevelINFO, deploymentID).Registerf("Status for workflow step %q changed to %q", wfStepInfo.StepName, status)
	return id, nil
}

// PublishAndLogAlienTaskStatusChange publishes a status change for a task execution and log this change into the log API
//
// PublishAndLogAlienTaskStatusChange returns the published event id
func PublishAndLogAlienTaskStatusChange(ctx context.Context, kv *api.KV, deploymentID, taskID, taskExecutionID string, wfStepInfo *WorkflowStepInfo, status string) (string, error) {
	if ctx == nil {
		ctx = NewContext(context.Background(), LogOptionalFields{ExecutionID: taskID, TaskExecutionID: taskExecutionID})
	}
	info := buildInfoFromContext(ctx)
	info[ETaskID] = taskID
	// Warning: Alien task corresponds to what we call taskExecution
	info[ETaskExecutionID] = taskExecutionID
	info[EWorkflowID] = wfStepInfo.WorkflowName
	info[ENodeID] = wfStepInfo.NodeName
	info[EWorkflowStepID] = wfStepInfo.StepName
	info[EInstanceID] = wfStepInfo.InstanceName
	info[EOperationName] = wfStepInfo.OperationName
	info[ETargetNodeID] = wfStepInfo.TargetNodeID
	info[ETargetInstanceID] = wfStepInfo.TargetInstanceID
	e, err := newStatusChange(StatusChangeTypeAlienTask, info, deploymentID, strings.ToLower(status))
	if err != nil {
		return "", err
	}
	id, err := e.register()
	if err != nil {
		return "", err
	}
	WithContextOptionalFields(ctx).NewLogEntry(LogLevelINFO, deploymentID).Registerf("Status for task execution %q changed to %q", taskExecutionID, status)
	return id, nil
}

// PublishAndLogWorkflowStatusChange publishes a status change for a workflow task and log this change into the log API
//
// PublishAndLogWorkflowStatusChange returns the published event id
func PublishAndLogWorkflowStatusChange(ctx context.Context, kv *api.KV, deploymentID, taskID, workflowID, status string) (string, error) {
	if ctx == nil {
		ctx = NewContext(context.Background(), LogOptionalFields{ExecutionID: taskID})
	}
	info := buildInfoFromContext(ctx)
	info[ETaskID] = taskID
	info[EWorkflowID] = workflowID
	e, err := newStatusChange(StatusChangeTypeWorkflow, info, deploymentID, strings.ToLower(status))
	if err != nil {
		return "", err
	}
	id, err := e.register()
	if err != nil {
		return "", err
	}
	WithContextOptionalFields(ctx).NewLogEntry(LogLevelINFO, deploymentID).Registerf("Status for workflow %q changed to %q", workflowID, status)
	return id, nil
}

// StatusEvents return a list of events (StatusUpdate instances) for all, or a given deployment
func StatusEvents(kv *api.KV, deploymentID string, waitIndex uint64, timeout time.Duration) ([]json.RawMessage, uint64, error) {
	events := make([]json.RawMessage, 0)
	var eventsPrefix string
	if deploymentID != "" {
		// the returned list of events must correspond to the provided deploymentID
		eventsPrefix = path.Join(consulutil.EventsPrefix, deploymentID)
	} else {
		// the returned list of events must correspond to all the deployments
		eventsPrefix = path.Join(consulutil.EventsPrefix)
	}

	kvps, qm, err := kv.List(eventsPrefix, &api.QueryOptions{WaitIndex: waitIndex, WaitTime: timeout})
	if err != nil || qm == nil {
		return events, 0, err
	}
	log.Debugf("Found %d events before accessing index[%q]", len(kvps), strconv.FormatUint(qm.LastIndex, 10))
	for _, kvp := range kvps {
		if kvp.ModifyIndex <= waitIndex {
			continue
		}
		events = append(events, kvp.Value)
	}
	log.Debugf("Found %d events after index", len(events))
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

// PurgeDeploymentEvents deletes all events for a given deployment
func PurgeDeploymentEvents(kv *api.KV, deploymentID string) error {
	_, err := kv.DeleteTree(path.Join(consulutil.EventsPrefix, deploymentID), nil)
	return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
}

// PurgeDeploymentLogs deletes all logs for a given deployment
func PurgeDeploymentLogs(kv *api.KV, deploymentID string) error {
	_, err := kv.DeleteTree(path.Join(consulutil.LogsPrefix, deploymentID), nil)
	return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
}

func buildInfoFromContext(ctx context.Context) Info {
	infoUp := make(Info)

	lof, ok := FromContext(ctx)
	if ok {
		for k, v := range lof {
			infType, has := bindEventInfoWithContext(k)
			if has {
				infoUp[infType] = v
			}
		}
	}
	return infoUp
}

func bindEventInfoWithContext(f FieldType) (InfoType, bool) {
	var i InfoType
	has := true
	switch f {
	case WorkFlowID:
		i = EWorkflowID
	case ExecutionID:
		i = ETaskID
	case NodeID:
		i = ENodeID
	case InstanceID:
		i = EInstanceID
	case OperationName:
		i = EOperationName
	case TaskExecutionID:
		i = ETaskExecutionID
	default:
		has = false
	}
	return i, has
}
