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

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
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
	switch taskType {
	case TaskTypeDeploy, TaskTypeUnDeploy, TaskTypePurge, TaskTypeScaleIn, TaskTypeScaleOut, TaskTypeCustomWorkflow, TaskTypeAddNodes, TaskTypeRemoveNodes:
		return true
	default:
		return false
	}
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
//
// Deprecated: Prefer deployments.GetDeploymentTaskList() instead if possible
func GetTasksIdsForTarget(targetID string) ([]string, error) {
	tasksKeys, err := consulutil.GetKeys(consulutil.TasksPrefix + "/")
	if err != nil {
		return nil, err
	}
	tasks := make([]string, 0)
	for _, taskKey := range tasksKeys {
		exist, value, err := consulutil.GetStringValue(path.Join(taskKey, "targetId"))
		if err != nil {
			return nil, err
		}
		if exist && value == targetID {
			tasks = append(tasks, path.Base(taskKey))
		}
	}
	return tasks, nil
}

// GetTaskErrorMessage retrieves the task related error message if any.
//
// If no error message is found, an empty string is returned instead
func GetTaskErrorMessage(taskID string) (string, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.TasksPrefix, taskID, "errorMessage"))
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if exist {
		return value, nil
	}
	return "", nil
}

// SetTaskErrorMessage sets a task related error message.
//
// Set task error message even if it already contains a value.
// For a better control on gracefully setting this error message use CheckAndSetTaskErrorMessage
func SetTaskErrorMessage(taskID, errorMessage string) error {
	return consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "errorMessage"), errorMessage)
}

// CheckAndSetTaskErrorMessage sets a task related error message.
//
// This function check for an existing message and overwrite it only if requested.
func CheckAndSetTaskErrorMessage(taskID, errorMessage string, overwriteExisting bool) error {
	keyPath := path.Join(consulutil.TasksPrefix, taskID, "errorMessage")
	kvp := &api.KVPair{Key: keyPath, Value: []byte(errorMessage)}
	kv := consulutil.GetKV()
	for {
		existingKey, meta, err := kv.Get(keyPath, nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if existingKey != nil && len(existingKey.Value) > 0 && !overwriteExisting {
			return nil
		}
		if existingKey != nil {
			kvp.ModifyIndex = meta.LastIndex
		}
		set, _, err := consulutil.GetKV().CAS(kvp, nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if set {
			return nil
		}
	}
}

// GetTaskResultSet retrieves the task related resultSet in json string format
//
// If no resultSet is found, nil is returned instead
func GetTaskResultSet(taskID string) (string, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.TasksPrefix, taskID, "resultSet"))
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if exist {
		return value, nil
	}
	return "", nil
}

// GetTaskStatus retrieves the TaskStatus of a task
func GetTaskStatus(taskID string) (TaskStatus, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.TasksPrefix, taskID, "status"))
	if err != nil {
		return TaskStatusFAILED, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist || value == "" {
		// Check if task exists to return specific error
		ok, err := TaskExists(taskID)
		if err != nil {
			return TaskStatusFAILED, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if ok {
			return TaskStatusFAILED, errors.Errorf("Missing status for task with id %q", taskID)
		}
		return TaskStatusFAILED, errors.WithStack(taskNotFound{taskID: taskID})
	}
	statusInt, err := strconv.Atoi(value)
	if err != nil {
		return TaskStatusFAILED, errors.Wrapf(err, "Invalid task status:")
	}
	if statusInt < 0 || statusInt > int(TaskStatusCANCELED) {
		return TaskStatusFAILED, errors.Errorf("Invalid status for task with id %q: %q", taskID, value)
	}
	return TaskStatus(statusInt), nil
}

// GetTaskType retrieves the TaskType of a task
func GetTaskType(taskID string) (TaskType, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.TasksPrefix, taskID, "type"))
	if err != nil {
		return TaskTypeDeploy, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist || value == "" {
		return TaskTypeDeploy, errors.Errorf("Missing type for task with id %q", taskID)
	}
	typeInt, err := strconv.Atoi(value)
	if err != nil {
		return TaskTypeDeploy, errors.Wrapf(err, "Invalid task type:")
	}
	if typeInt < 0 || typeInt > int(TaskTypeRemoveNodes) {
		return TaskTypeDeploy, errors.Errorf("Invalid type for task with id %q: %q", taskID, value)
	}
	return TaskType(typeInt), nil
}

// GetTaskTarget retrieves the targetID of a task
func GetTaskTarget(taskID string) (string, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.TasksPrefix, taskID, "targetId"))
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist || value == "" {
		return "", errors.Errorf("Missing targetId for task with id %q", taskID)
	}
	return value, nil
}

// GetTaskCreationDate retrieves the creationDate of a task
func GetTaskCreationDate(taskID string) (time.Time, error) {
	exist, value, err := consulutil.GetValue(path.Join(consulutil.TasksPrefix, taskID, "creationDate"))
	creationDate := time.Time{}
	if err != nil {
		return creationDate, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist || len(value) == 0 {
		return creationDate, errors.Errorf("Missing creationDate for task with id %q", taskID)
	}
	err = creationDate.UnmarshalBinary(value)
	if err != nil {
		return creationDate, errors.Wrapf(err, "Failed to get task creationDate for task with id %q", taskID)
	}
	return creationDate, nil
}

// TaskExists checks if a task with the given taskID exists
func TaskExists(taskID string) (bool, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.TasksPrefix, taskID, "targetId"))
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist || value == "" {
		return false, nil
	}
	return true, nil
}

// IsStepRegistrationInProgress checks if a task registration is still in progress,
// in which case it should not yet be executed
func IsStepRegistrationInProgress(taskID string) (bool, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.TasksPrefix, taskID, stepRegistrationInProgressKey))
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist || value == "" {
		return false, nil
	}
	return true, nil
}

// StoreOperations stores operations related to a task through a transaction
// splitting this transaction if needed, in which case it will create a key
// notifying a registration is in progress
func StoreOperations(taskID string, operations api.KVTxnOps) error {

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
	return consulutil.ExecuteSplittableTransaction(operations, preOpSplit, postOpSplit)
}

// CancelTask marks a task as Canceled
func CancelTask(taskID string) error {
	return consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, ".canceledFlag"), "true")
}

// DeleteTask allows to delete a stored task
func DeleteTask(taskID string) error {
	return consulutil.Delete(path.Join(consulutil.TasksPrefix, taskID)+"/", true)
}

// TargetHasLivingTasks checks if a targetID has associated tasks in status INITIAL or RUNNING and returns the id and status of the first one found
//
// The last argument specifies tasks types which should be ignored.
//
// Deprecated: Prefer HasLivingTasks() instead
func TargetHasLivingTasks(targetID string, tasksTypesToIgnore []TaskType) (bool, string, string, error) {
	taskIDs, err := GetTasksIdsForTarget(targetID)
	if err != nil {
		return false, "", "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return HasLivingTasks(taskIDs, tasksTypesToIgnore)
}

// HasLivingTasks checks if the tasks list contains tasks in status INITIAL or RUNNING and returns the id and status of the first one found
//
// The last argument specifies tasks types which should be ignored.
func HasLivingTasks(taskIDs []string, tasksTypesToIgnore []TaskType) (bool, string, string, error) {
	for _, taskID := range taskIDs {

		tStatus, err := GetTaskStatus(taskID)
		if err != nil {
			return false, "", "", err
		}

		if tStatus == TaskStatusINITIAL || tStatus == TaskStatusRUNNING {

			// Check if this task type should be ignored
			tType, err := GetTaskType(taskID)
			if err != nil {
				return false, "", "", err
			}
			if !isInTaskTypeSlice(tType, tasksTypesToIgnore) {
				return true, taskID, tStatus.String(), nil
			}
		}
	}
	return false, "", "", nil
}

// GetTaskInput retrieves inputs for tasks
func GetTaskInput(taskID, inputName string) (string, error) {
	return GetTaskData(taskID, path.Join("inputs", inputName))
}

// GetTaskOutput retrieves a specified output with name outputName for tasks
func GetTaskOutput(taskID, outputName string) (string, error) {
	return GetTaskData(taskID, path.Join("outputs", outputName))
}

// GetTaskOutputs retrieves all outputs for tasks
func GetTaskOutputs(taskID string) (map[string]string, error) {
	outputs := make(map[string]string)
	kvs, err := consulutil.List(path.Join(consulutil.TasksPrefix, taskID, "data", "outputs") + "/")
	if err != nil {
		return nil, err
	}
	for outputName, outputValue := range kvs {
		outputs[path.Base(outputName)] = string(outputValue)
	}
	return outputs, nil
}

// GetTaskData retrieves data for tasks
func GetTaskData(taskID, dataName string) (string, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.TasksPrefix, taskID, "data", dataName))
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist {
		return "", errors.WithStack(taskDataNotFound{name: dataName, taskID: taskID})
	}
	return value, nil
}

// GetAllTaskData returns all registered data for a task
func GetAllTaskData(taskID string) (map[string]string, error) {
	dataPrefix := path.Join(consulutil.TasksPrefix, taskID, "data")
	// Appending a final "/" here is not necessary as there is no other keys starting with "data" prefix
	kvs, err := consulutil.List(dataPrefix)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	data := make(map[string]string, len(kvs))
	for key, value := range kvs {
		if value != nil {
			data[strings.TrimPrefix(key, dataPrefix+"/")] = string(value)
		}
	}

	return data, nil
}

// SetTaskData sets a data in the task's context
func SetTaskData(taskID, dataName, dataValue string) error {
	return consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, taskID, "data", dataName), dataValue)
}

// SetTaskDataList sets a list of data into the task's context
func SetTaskDataList(taskID string, data map[string]string) error {
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
func GetInstances(ctx context.Context, taskID, deploymentID, nodeName string) ([]string, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.TasksPrefix, taskID, "data/nodes", nodeName))
	if err != nil {
		return nil, err
	}
	if !exist || value == "" {
		return deployments.GetNodeInstancesIds(ctx, deploymentID, nodeName)
	}
	return strings.Split(value, ","), nil
}

// GetTaskRelatedNodes returns the list of nodes that are specifically targeted by this task
//
// Currently it only appens for scaling tasks
func GetTaskRelatedNodes(taskID string) ([]string, error) {
	nodes, err := consulutil.GetKeys(path.Join(consulutil.TasksPrefix, taskID, "data/nodes"))
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for i := range nodes {
		nodes[i] = path.Base(nodes[i])
	}
	return nodes, nil
}

// IsTaskRelatedNode checks if the given nodeName is declared as a task related node
func IsTaskRelatedNode(taskID, nodeName string) (bool, error) {
	exist, _, err := consulutil.GetStringValue(path.Join(consulutil.TasksPrefix, taskID, "data/nodes", nodeName))
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return exist, nil
}

// EmitTaskEventWithContextualLogs emits a task event based on task type
func EmitTaskEventWithContextualLogs(ctx context.Context, deploymentID, taskID string, taskType TaskType, workflowName, status string) (string, error) {
	if ctx == nil {
		ctx = events.NewContext(context.Background(), events.LogOptionalFields{events.ExecutionID: taskID})
	}
	switch taskType {
	case TaskTypeCustomCommand:
		return events.PublishAndLogCustomCommandStatusChange(ctx, deploymentID, taskID, strings.ToLower(status))
	case TaskTypeCustomWorkflow, TaskTypeDeploy, TaskTypeUnDeploy, TaskTypePurge, TaskTypeAddNodes, TaskTypeRemoveNodes:
		return events.PublishAndLogWorkflowStatusChange(ctx, deploymentID, taskID, workflowName, strings.ToLower(status))
	case TaskTypeScaleIn, TaskTypeScaleOut:
		return events.PublishAndLogScalingStatusChange(ctx, deploymentID, taskID, strings.ToLower(status))
	}
	return "", errors.Errorf("unsupported task type: %s", taskType)
}

// GetTaskRelatedSteps returns the steps of the related workflow
func GetTaskRelatedSteps(taskID string) ([]TaskStep, error) {
	steps := make([]TaskStep, 0)
	kvs, err := consulutil.List(path.Join(consulutil.WorkflowsPrefix, taskID) + "/")
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	for key, value := range kvs {
		steps = append(steps, TaskStep{Name: path.Base(key), Status: string(value)})
	}
	return steps, nil
}

// GetTaskStepStatus returns the step status of the related step name
func GetTaskStepStatus(taskID, stepName string) (TaskStepStatus, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.WorkflowsPrefix, taskID, stepName))
	if err != nil {
		return TaskStepStatusINITIAL, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist || value == "" {
		return TaskStepStatusINITIAL, nil
	}
	return ParseTaskStepStatus(value)
}

// TaskStepExists checks if a task step exists with a stepID and related to a given taskID and returns it
func TaskStepExists(taskID, stepID string) (bool, *TaskStep, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.WorkflowsPrefix, taskID, stepID))
	if err != nil {
		return false, nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist || value == "" {
		return false, nil, nil
	}
	return true, &TaskStep{Name: stepID, Status: value}, nil
}

// UpdateTaskStepStatus allows to update the task step status
func UpdateTaskStepStatus(taskID string, step *TaskStep) error {
	status, err := ParseTaskStepStatus(step.Status)
	if err != nil {
		return err
	}
	return UpdateTaskStepWithStatus(taskID, step.Name, status)
}

// UpdateTaskStepWithStatus allows to update the task step status
func UpdateTaskStepWithStatus(taskID, stepName string, status TaskStepStatus) error {
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
func GetQueryTaskIDs(taskType TaskType, query string, target string) ([]string, error) {
	tasksKeys, err := consulutil.GetKeys(consulutil.TasksPrefix)
	if err != nil {
		return nil, err
	}
	tasks := make([]string, 0)
	for _, taskKey := range tasksKeys {
		id := path.Base(taskKey)
		if ttyp, err := GetTaskType(id); err != nil {
			// Ignore errors
			log.Printf("[WARNING] the task with id:%q won't be listed due to error:%+v", id, err)
			continue
		} else if ttyp != taskType {
			continue
		}

		exist, targetID, err := consulutil.GetStringValue(path.Join(taskKey, "targetId"))
		if err != nil {
			// Ignore errors
			log.Printf("[WARNING] the task with id:%q won't be listed due to error:%+v", id, err)
			continue
		}

		log.Debugf("targetId:%q", targetID)
		if exist && targetID != "" && strings.HasPrefix(targetID, query) && strings.HasSuffix(targetID, target) {
			tasks = append(tasks, id)
		}
	}
	return tasks, nil
}

// TaskHasErrorFlag check if a task has was flagged as error
func TaskHasErrorFlag(taskID string) (bool, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.TasksPrefix, taskID, ".errorFlag"))
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return exist && value == "true", nil
}

// TaskHasCancellationFlag check if a task has was flagged as canceled
func TaskHasCancellationFlag(taskID string) (bool, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.TasksPrefix, taskID, ".canceledFlag"))
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return exist && value == "true", nil
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
func MonitorTaskCancellation(ctx context.Context, taskID string, f func()) {
	monitorTaskFlag(ctx, taskID, ".canceledFlag", []byte("true"), f)
}

// MonitorTaskFailure runs a routine that will constantly check if a given task is tagged as failed.
//
// If so and if the given f function is not nil then f is called and the routine stop itself.
// To stop this routine the given context should be cancelled.
func MonitorTaskFailure(ctx context.Context, taskID string, f func()) {
	monitorTaskFlag(ctx, taskID, ".errorFlag", []byte("true"), f)
}

func monitorTaskFlag(ctx context.Context, taskID, flag string, value []byte, f func()) {
	go func() {
		var lastIndex uint64
		queryMeta := &api.QueryOptions{}
		queryMeta = queryMeta.WithContext(ctx)
		for {
			queryMeta.WaitIndex = lastIndex
			kvp, qMeta, err := consulutil.GetKV().Get(path.Join(consulutil.TasksPrefix, taskID, flag), queryMeta)

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

func isInTaskTypeSlice(taskType TaskType, slice []TaskType) bool {
	for _, val := range slice {
		if val == taskType {
			return true
		}
	}

	return false
}
