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

package tasks

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/events"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/log"
)

type anotherLivingTaskAlreadyExistsError struct {
	taskID   string
	targetID string
	status   string
}

const (
	// stepRegistrationInProgressKey is the Consul key name, whose presence means the
	// new steps are being registered in Consul for a given task
	stepRegistrationInProgressKey = "stepRegistrationInProgress"
)

func (e anotherLivingTaskAlreadyExistsError) Error() string {
	return fmt.Sprintf("Task with id %q and status %q already exists for target %q", e.taskID, e.status, e.targetID)
}

// NewAnotherLivingTaskAlreadyExistsError allows to create a new anotherLivingTaskAlreadyExistsError error
func NewAnotherLivingTaskAlreadyExistsError(taskID, targetID, status string) error {
	return anotherLivingTaskAlreadyExistsError{taskID: taskID, targetID: targetID, status: status}
}

// IsAnotherLivingTaskAlreadyExistsError checks if an error is due to the fact that another task is currently running
// If true, it returns the taskID of the currently running task
func IsAnotherLivingTaskAlreadyExistsError(err error) (bool, string) {
	e, ok := err.(anotherLivingTaskAlreadyExistsError)
	if ok {
		return ok, e.taskID
	}
	return ok, ""
}

// IsWorkflowTask returns true if the task type is related to workflow
func IsWorkflowTask(taskType TaskType) bool {
	return taskType == TaskTypeDeploy || taskType == TaskTypeUnDeploy || taskType == TaskTypePurge || taskType == TaskTypeScaleIn || taskType == TaskTypeScaleOut || taskType == TaskTypeCustomWorkflow
}

type taskDataNotFound struct {
	name   string
	taskID string
}

type taskNotFound struct {
	taskID string
}

func (t taskDataNotFound) Error() string {
	return fmt.Sprintf("Data %q not found for task %q", t.name, t.taskID)
}

func (t taskNotFound) Error() string {
	return fmt.Sprintf("Task %q not found", t.taskID)
}

// IsTaskDataNotFoundError checks if an error is a task data not found error
func IsTaskDataNotFoundError(err error) bool {
	cause := errors.Cause(err)
	_, ok := cause.(taskDataNotFound)
	return ok
}

// IsTaskNotFoundError checks if an error is a task not found error
func IsTaskNotFoundError(err error) bool {
	cause := errors.Cause(err)
	_, ok := cause.(taskNotFound)
	return ok
}

// GetTasksIdsForTarget returns IDs of tasks related to a given targetID
func GetTasksIdsForTarget(kv *api.KV, targetID string) ([]string, error) {
	tasksKeys, _, err := kv.Keys(consulutil.TasksPrefix+"/", "/", nil)
	if err != nil {
		return nil, err
	}
	tasks := make([]string, 0)
	for _, taskKey := range tasksKeys {
		kvp, _, err := kv.Get(path.Join(taskKey, "targetId"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp != nil && len(kvp.Value) > 0 && string(kvp.Value) == targetID {
			tasks = append(tasks, path.Base(taskKey))
		}
	}
	return tasks, nil
}

// GetTaskResultSet retrieves the task related resultSet in json string format
//
// If no resultSet is found, nil is returned instead
func GetTaskResultSet(kv *api.KV, taskID string) (string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "resultSet"), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil {
		return string(kvp.Value), nil
	}
	return "", nil
}

// GetTaskStatus retrieves the TaskStatus of a task
func GetTaskStatus(kv *api.KV, taskID string) (TaskStatus, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "status"), nil)
	if err != nil {
		return TaskStatusFAILED, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		// Check if task exists to return specific error
		ok, err := TaskExists(kv, taskID)
		if err != nil {
			return TaskStatusFAILED, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if ok {
			return TaskStatusFAILED, errors.Errorf("Missing status for task with id %q", taskID)
		}
		return TaskStatusFAILED, errors.WithStack(taskNotFound{taskID: taskID})
	}
	statusInt, err := strconv.Atoi(string(kvp.Value))
	if err != nil {
		return TaskStatusFAILED, errors.Wrapf(err, "Invalid task status:")
	}
	if statusInt < 0 || statusInt > int(TaskStatusCANCELED) {
		return TaskStatusFAILED, errors.Errorf("Invalid status for task with id %q: %q", taskID, string(kvp.Value))
	}
	return TaskStatus(statusInt), nil
}

// GetTaskType retrieves the TaskType of a task
func GetTaskType(kv *api.KV, taskID string) (TaskType, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "type"), nil)
	if err != nil {
		return TaskTypeDeploy, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return TaskTypeDeploy, errors.Errorf("Missing type for task with id %q", taskID)
	}
	typeInt, err := strconv.Atoi(string(kvp.Value))
	if err != nil {
		return TaskTypeDeploy, errors.Wrapf(err, "Invalid task type:")
	}
	if typeInt < 0 || typeInt > int(TaskTypeForcePurge) {
		return TaskTypeDeploy, errors.Errorf("Invalid type for task with id %q: %q", taskID, string(kvp.Value))
	}
	return TaskType(typeInt), nil
}

// GetTaskTarget retrieves the targetID of a task
func GetTaskTarget(kv *api.KV, taskID string) (string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "targetId"), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", errors.Errorf("Missing targetId for task with id %q", taskID)
	}
	return string(kvp.Value), nil
}

// GetTaskCreationDate retrieves the creationDate of a task
func GetTaskCreationDate(kv *api.KV, taskID string) (time.Time, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "creationDate"), nil)
	creationDate := time.Time{}
	if err != nil {
		return creationDate, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return creationDate, errors.Errorf("Missing creationDate for task with id %q", taskID)
	}
	err = creationDate.UnmarshalBinary(kvp.Value)
	if err != nil {
		return creationDate, errors.Wrapf(err, "Failed to get task creationDate for task with id %q", taskID)
	}
	return creationDate, nil
}

// TaskExists checks if a task with the given taskID exists
func TaskExists(kv *api.KV, taskID string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "targetId"), nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return false, nil
	}
	return true, nil
}

// IsStepRegistrationInProgress checks if a task registration is still in progress,
// in which case it should not yet be executed
func IsStepRegistrationInProgress(kv *api.KV, taskID string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, stepRegistrationInProgressKey), nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return false, nil
	}
	return true, nil
}

// StoreOperations stores operations related to a task through a transaction
// splitting this transaction if needed, in which case it will create a key
// notifying a registration is in progress
func StoreOperations(kv *api.KV, taskID string, operations api.KVTxnOps) error {

	registrationStatusKeyPath := path.Join(consulutil.TasksPrefix, taskID, stepRegistrationInProgressKey)
	preOpSplit := &api.KVTxnOp{
		Verb:  api.KVSet,
		Key:   registrationStatusKeyPath,
		Value: []byte("true"),
	}
	postOpSplit := &api.KVTxnOp{
		Verb: api.KVDelete,
		Key:  registrationStatusKeyPath,
	}
	return consulutil.ExecuteSplittableTransaction(kv, operations, preOpSplit, postOpSplit)
}

// CancelTask marks a task as Canceled
func CancelTask(kv *api.KV, taskID string) error {
	return consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, ".canceledFlag"), "true")
}

// ResumeTask marks a task as Initial to allow it being resumed
//
// Deprecated: use (c *collector.Collector) ResumeTask instead
func ResumeTask(kv *api.KV, taskID string) error {
	return consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "status"), strconv.Itoa(int(TaskStatusINITIAL)))
}

// DeleteTask allows to delete a stored task
func DeleteTask(kv *api.KV, taskID string) error {
	_, err := kv.DeleteTree(path.Join(consulutil.TasksPrefix, taskID), nil)
	return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
}

// TargetHasLivingTasks checks if a targetID has associated tasks in status INITIAL or RUNNING and returns the id and status of the first one found
//
// Only Deploy, UnDeploy, ScaleOut, ScaleIn and Purge task type are considered.
func TargetHasLivingTasks(kv *api.KV, targetID string) (bool, string, string, error) {

	taskIDs, err := GetTasksIdsForTarget(kv, targetID)
	if err != nil {
		return false, "", "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	for _, taskID := range taskIDs {

		tStatus, err := GetTaskStatus(kv, taskID)
		if err != nil {
			return false, "", "", err
		}
		tType, err := GetTaskType(kv, taskID)
		if err != nil {
			return false, "", "", err
		}

		switch tType {
		case TaskTypeDeploy, TaskTypeUnDeploy, TaskTypePurge, TaskTypeScaleIn, TaskTypeScaleOut:
			if tStatus == TaskStatusINITIAL || tStatus == TaskStatusRUNNING {
				return true, taskID, tStatus.String(), nil
			}
		}
	}
	return false, "", "", nil
}

// GetTaskInput retrieves inputs for tasks
func GetTaskInput(kv *api.KV, taskID, inputName string) (string, error) {
	return GetTaskData(kv, taskID, path.Join("inputs", inputName))
}

// GetTaskData retrieves data for tasks
func GetTaskData(kv *api.KV, taskID, dataName string) (string, error) {
	kvP, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "data", dataName), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvP == nil {
		return "", errors.WithStack(taskDataNotFound{name: dataName, taskID: taskID})
	}
	return string(kvP.Value), nil
}

// GetAllTaskData returns all registered data for a task
func GetAllTaskData(kv *api.KV, taskID string) (map[string]string, error) {
	dataPrefix := path.Join(consulutil.TasksPrefix, taskID, "data")
	kvps, _, err := kv.List(dataPrefix, nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	data := make(map[string]string, len(kvps))
	for _, kvp := range kvps {
		if kvp.Value != nil {
			data[strings.TrimPrefix(kvp.Key, dataPrefix+"/")] = string(kvp.Value)
		}
	}

	return data, nil
}

// SetTaskData sets a data in the task's context
func SetTaskData(kv *api.KV, taskID, dataName, dataValue string) error {
	return consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "data", dataName), dataValue)
}

// SetTaskDataList sets a list of data into the task's context
func SetTaskDataList(kv *api.KV, taskID string, data map[string]string) error {
	_, errGrp, store := consulutil.WithContext(context.Background())
	for k, v := range data {
		store.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "data", k), v)
	}
	return errGrp.Wait()
}

// GetInstances retrieve instances in the context of this task.
//
// Basically it checks if a list of instances is defined for this task for example in case of scaling.
// If not found it will returns the result of deployments.GetNodeInstancesIds(kv, deploymentID, nodeName).
func GetInstances(kv *api.KV, taskID, deploymentID, nodeName string) ([]string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "data/nodes", nodeName), nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return deployments.GetNodeInstancesIds(kv, deploymentID, nodeName)
	}
	return strings.Split(string(kvp.Value), ","), nil
}

// GetTaskRelatedNodes returns the list of nodes that are specifically targeted by this task
//
// Currently it only appens for scaling tasks
func GetTaskRelatedNodes(kv *api.KV, taskID string) ([]string, error) {
	nodes, _, err := kv.Keys(path.Join(consulutil.TasksPrefix, taskID, "data/nodes")+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for i := range nodes {
		nodes[i] = path.Base(nodes[i])
	}
	return nodes, nil
}

// IsTaskRelatedNode checks if the given nodeName is declared as a task related node
func IsTaskRelatedNode(kv *api.KV, taskID, nodeName string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, "data/nodes", nodeName), nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return kvp != nil, nil
}

// EmitTaskEvent emits a task event based on task type
//
// Deprecated: use EmitTaskEventWithContextualLogs instead
func EmitTaskEvent(kv *api.KV, deploymentID, taskID string, taskType TaskType, status string) (string, error) {
	return EmitTaskEventWithContextualLogs(nil, kv, deploymentID, taskID, taskType, "unknown", status)
}

// EmitTaskEventWithContextualLogs emits a task event based on task type
func EmitTaskEventWithContextualLogs(ctx context.Context, kv *api.KV, deploymentID, taskID string, taskType TaskType, workflowName, status string) (string, error) {
	if ctx == nil {
		ctx = events.NewContext(context.Background(), events.LogOptionalFields{events.ExecutionID: taskID})
	}
	switch taskType {
	case TaskTypeCustomCommand:
		return events.PublishAndLogCustomCommandStatusChange(ctx, kv, deploymentID, taskID, strings.ToLower(status))
	case TaskTypeCustomWorkflow, TaskTypeDeploy, TaskTypeUnDeploy, TaskTypePurge:
		return events.PublishAndLogWorkflowStatusChange(ctx, kv, deploymentID, taskID, workflowName, strings.ToLower(status))
	case TaskTypeScaleIn, TaskTypeScaleOut:
		return events.PublishAndLogScalingStatusChange(ctx, kv, deploymentID, taskID, strings.ToLower(status))
	}
	return "", errors.Errorf("unsupported task type: %s", taskType)
}

// GetTaskRelatedSteps returns the steps of the related workflow
func GetTaskRelatedSteps(kv *api.KV, taskID string) ([]TaskStep, error) {
	steps := make([]TaskStep, 0)

	kvps, _, err := kv.List(path.Join(consulutil.WorkflowsPrefix, taskID), nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	for _, kvp := range kvps {
		steps = append(steps, TaskStep{Name: path.Base(kvp.Key), Status: string(kvp.Value)})
	}
	return steps, nil
}

// GetTaskStepStatus returns the step status of the related step name
func GetTaskStepStatus(kv *api.KV, taskID, stepName string) (TaskStepStatus, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.WorkflowsPrefix, taskID, stepName), nil)
	if err != nil {
		return TaskStepStatusINITIAL, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return TaskStepStatusINITIAL, nil
	}
	return ParseTaskStepStatus(string(kvp.Value))
}

// TaskStepExists checks if a task step exists with a stepID and related to a given taskID and returns it
func TaskStepExists(kv *api.KV, taskID, stepID string) (bool, *TaskStep, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.WorkflowsPrefix, taskID, stepID), nil)
	if err != nil {
		return false, nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return false, nil, nil
	}
	return true, &TaskStep{Name: stepID, Status: string(kvp.Value)}, nil
}

// UpdateTaskStepStatus allows to update the task step status
func UpdateTaskStepStatus(kv *api.KV, taskID string, step *TaskStep) error {
	status, err := ParseTaskStepStatus(step.Status)
	if err != nil {
		return err
	}
	return UpdateTaskStepWithStatus(kv, taskID, step.Name, status)
}

// UpdateTaskStepWithStatus allows to update the task step status
func UpdateTaskStepWithStatus(kv *api.KV, taskID, stepName string, status TaskStepStatus) error {
	return consulutil.StoreConsulKeyAsString(path.Join(consulutil.WorkflowsPrefix, taskID, stepName), status.String())
}

// CheckTaskStepStatusChange checks if a status change is allowed
func CheckTaskStepStatusChange(before, after string) (bool, error) {
	if before == after {
		return false, errors.New("Final and initial status are identical: nothing to do")
	}
	stBefore, err := ParseTaskStepStatus(before)
	if err != nil {
		return false, err
	}
	stAfter, err := ParseTaskStepStatus(after)
	if err != nil {
		return false, err
	}

	if stBefore != TaskStepStatusERROR || stAfter != TaskStepStatusDONE {
		return false, nil
	}
	return true, nil
}

// GetQueryTaskIDs returns an array of taskID query-typed, optionally filtered by query and target
func GetQueryTaskIDs(kv *api.KV, taskType TaskType, query string, target string) ([]string, error) {
	tasksKeys, _, err := kv.Keys(consulutil.TasksPrefix+"/", "/", nil)
	if err != nil {
		return nil, err
	}
	tasks := make([]string, 0)
	for _, taskKey := range tasksKeys {
		id := path.Base(taskKey)
		if ttyp, err := GetTaskType(kv, id); err != nil {
			// Ignore errors
			log.Printf("[WARNING] the task with id:%q won't be listed due to error:%+v", id, err)
			continue
		} else if ttyp != taskType {
			continue
		}

		kvp, _, err := kv.Get(path.Join(taskKey, "targetId"), nil)
		if err != nil {
			// Ignore errors
			log.Printf("[WARNING] the task with id:%q won't be listed due to error:%+v", id, err)
			continue
		}
		targetID := string(kvp.Value)
		log.Debugf("targetId:%q", targetID)
		if kvp != nil && len(kvp.Value) > 0 && strings.HasPrefix(targetID, query) && strings.HasSuffix(targetID, target) {
			tasks = append(tasks, id)
		}
	}
	return tasks, nil
}

// TaskHasErrorFlag check if a task has was flagged as error
func TaskHasErrorFlag(kv *api.KV, taskID string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, ".errorFlag"), nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return kvp != nil && string(kvp.Value) == "true", nil
}

// TaskHasCancellationFlag check if a task has was flagged as canceled
func TaskHasCancellationFlag(kv *api.KV, taskID string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, ".canceledFlag"), nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return kvp != nil && string(kvp.Value) == "true", nil
}

// NotifyErrorOnTask sets a flag that is used to notify task executors that a part of the task failed.
//
// MonitorTaskFailure can be used to be notified.
func NotifyErrorOnTask(taskID string) error {
	return consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, ".errorFlag"), "true")
}

// MonitorTaskCancellation runs a routine that will constantly check if a given task is requested to be cancelled
//
// If so and if the given f function is not nil then f is called and the routine stop itself.
// To stop this routine the given context should be cancelled.
func MonitorTaskCancellation(ctx context.Context, kv *api.KV, taskID string, f func()) {
	monitorTaskFlag(ctx, kv, taskID, ".canceledFlag", []byte("true"), f)
}

// MonitorTaskFailure runs a routine that will constantly check if a given task is tagged as failed.
//
// If so and if the given f function is not nil then f is called and the routine stop itself.
// To stop this routine the given context should be cancelled.
func MonitorTaskFailure(ctx context.Context, kv *api.KV, taskID string, f func()) {
	monitorTaskFlag(ctx, kv, taskID, ".errorFlag", []byte("true"), f)
}

func monitorTaskFlag(ctx context.Context, kv *api.KV, taskID, flag string, value []byte, f func()) {
	go func() {
		var lastIndex uint64
		for {
			select {
			case <-ctx.Done():
				log.Debugf("Task monitoring for flag %s exit", flag)
				return
			default:
			}

			kvp, qMeta, err := kv.Get(path.Join(consulutil.TasksPrefix, taskID, flag), &api.QueryOptions{WaitIndex: lastIndex})

			select {
			case <-ctx.Done():
				log.Debugf("Task monitoring for flag %s exit", flag)
				return
			default:
			}

			if qMeta != nil {
				lastIndex = qMeta.LastIndex
			}

			if err == nil && kvp != nil {
				if bytes.Equal(kvp.Value, value) {
					log.Debugf("Task monitoring detected flag %s", flag)
					if f != nil {
						f()
					}
					return
				}
			}
		}
	}()
}
