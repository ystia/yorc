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

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/store"
	"github.com/ystia/yorc/v4/storage/types"
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
	e, err := newStatusChange(ctx, StatusChangeTypeAttributeValue, info, deploymentID, status)
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

// PublishAndLogInstanceStatusChange publishes a status change for a given instance of a given node and log this change into the log API
//
// PublishAndLogInstanceStatusChange returns the published event id
func PublishAndLogInstanceStatusChange(ctx context.Context, deploymentID, nodeName, instance, status string) (string, error) {
	ctx = AddLogOptionalFields(ctx, LogOptionalFields{NodeID: nodeName, InstanceID: instance})
	info := buildInfoFromContext(ctx)
	info[ENodeID] = nodeName
	info[EInstanceID] = instance
	e, err := newStatusChange(ctx, StatusChangeTypeInstance, info, deploymentID, strings.ToLower(status))
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

// PublishAndLogDeploymentStatusChange publishes a status change for a given deployment and log this change into the log API
//
// PublishAndLogDeploymentStatusChange returns the published event id
func PublishAndLogDeploymentStatusChange(ctx context.Context, deploymentID, status string) (string, error) {
	e, err := newStatusChange(ctx, StatusChangeTypeDeployment, nil, deploymentID, strings.ToLower(status))
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

// PublishAndLogCustomCommandStatusChange publishes a status change for a custom command and log this change into the log API
//
// PublishAndLogCustomCommandStatusChange returns the published event id
func PublishAndLogCustomCommandStatusChange(ctx context.Context, deploymentID, taskID, status string) (string, error) {
	if ctx == nil {
		ctx = NewContext(context.Background(), LogOptionalFields{ExecutionID: taskID})
	}
	info := buildInfoFromContext(ctx)
	info[ETaskID] = taskID
	e, err := newStatusChange(ctx, StatusChangeTypeCustomCommand, info, deploymentID, strings.ToLower(status))
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

// PublishAndLogScalingStatusChange publishes a status change for a scaling task and log this change into the log API
//
// PublishAndLogScalingStatusChange returns the published event id
func PublishAndLogScalingStatusChange(ctx context.Context, deploymentID, taskID, status string) (string, error) {
	if ctx == nil {
		ctx = NewContext(context.Background(), LogOptionalFields{ExecutionID: taskID})
	}
	info := buildInfoFromContext(ctx)
	info[ETaskID] = taskID
	e, err := newStatusChange(ctx, StatusChangeTypeScaling, info, deploymentID, strings.ToLower(status))
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

// PublishAndLogWorkflowStepStatusChange publishes a status change for a workflow step execution and log this change into the log API
//
// PublishAndLogWorkflowStepStatusChange returns the published event id
func PublishAndLogWorkflowStepStatusChange(ctx context.Context, deploymentID, taskID string, wfStepInfo *WorkflowStepInfo, status string) (string, error) {
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
	e, err := newStatusChange(ctx, StatusChangeTypeWorkflowStep, info, deploymentID, strings.ToLower(status))
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
func PublishAndLogAlienTaskStatusChange(ctx context.Context, deploymentID, taskID, taskExecutionID string, wfStepInfo *WorkflowStepInfo, status string) (string, error) {
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
	e, err := newStatusChange(ctx, StatusChangeTypeAlienTask, info, deploymentID, strings.ToLower(status))
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
func PublishAndLogWorkflowStatusChange(ctx context.Context, deploymentID, taskID, workflowID, status string) (string, error) {
	if ctx == nil {
		ctx = NewContext(context.Background(), LogOptionalFields{ExecutionID: taskID})
	}
	info := buildInfoFromContext(ctx)
	info[ETaskID] = taskID
	info[EWorkflowID] = workflowID
	e, err := newStatusChange(ctx, StatusChangeTypeWorkflow, info, deploymentID, strings.ToLower(status))
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

func getLogsOrEvents(ctx context.Context, deploymentID string, waitIndex uint64, timeout time.Duration, isEvents bool) ([]json.RawMessage, uint64, error) {
	logsOrEvents := make([]json.RawMessage, 0)

	var pathPrefix string
	var usedStore store.Store
	var data string
	if isEvents {
		pathPrefix = path.Clean(consulutil.EventsPrefix)
		usedStore = storage.GetStore(types.StoreTypeEvent)
		data = "events"
	} else {
		pathPrefix = path.Clean(consulutil.LogsPrefix)
		usedStore = storage.GetStore(types.StoreTypeLog)
		data = "logs"
	}

	if deploymentID != "" {
		// the returned list of logsOrEvents must correspond to the provided deploymentID
		pathPrefix = path.Join(pathPrefix, deploymentID)
	}
	pathPrefix = pathPrefix + "/"
	kvps, lastIndex, err := usedStore.List(ctx, pathPrefix, waitIndex, timeout)
	if err != nil || lastIndex == 0 {
		return logsOrEvents, 0, err
	}

	log.Debugf("Found %d %s before accessing index[%q]", len(kvps), data, strconv.FormatUint(lastIndex, 10))
	for _, kvp := range kvps {
		if kvp.LastModifyIndex <= waitIndex {
			continue
		}
		logsOrEvents = append(logsOrEvents, kvp.RawValue)
	}
	log.Debugf("Found %d %s after index", len(logsOrEvents), data)
	return logsOrEvents, lastIndex, nil
}

// StatusEvents return a list of events (StatusUpdate instances) for all, or a given deployment
func StatusEvents(ctx context.Context, deploymentID string, waitIndex uint64, timeout time.Duration) ([]json.RawMessage, uint64, error) {
	return getLogsOrEvents(ctx, deploymentID, waitIndex, timeout, true)
}

// LogsEvents allows to return logs from Consul KV storage for all, or a given deployment
func LogsEvents(ctx context.Context, deploymentID string, waitIndex uint64, timeout time.Duration) ([]json.RawMessage, uint64, error) {
	return getLogsOrEvents(ctx, deploymentID, waitIndex, timeout, false)
}

// GetStatusEventsIndex returns the latest index of InstanceStatus events for a given deployment
func GetStatusEventsIndex(deploymentID string) (uint64, error) {
	return storage.GetStore(types.StoreTypeEvent).GetLastModifyIndex(path.Join(consulutil.EventsPrefix, deploymentID))
}

// GetLogsEventsIndex returns the latest index of LogEntry events for a given deployment
func GetLogsEventsIndex(deploymentID string) (uint64, error) {
	return storage.GetStore(types.StoreTypeLog).GetLastModifyIndex(path.Join(consulutil.LogsPrefix, deploymentID))
}

// PurgeDeploymentEvents deletes all events for a given deployment
func PurgeDeploymentEvents(ctx context.Context, deploymentID string) error {
	return storage.GetStore(types.StoreTypeEvent).Delete(ctx, path.Join(consulutil.EventsPrefix, deploymentID)+"/", true)
}

// PurgeDeploymentLogs deletes all logs for a given deployment
func PurgeDeploymentLogs(ctx context.Context, deploymentID string) error {
	return storage.GetStore(types.StoreTypeLog).Delete(ctx, path.Join(consulutil.LogsPrefix, deploymentID)+"/", true)
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
