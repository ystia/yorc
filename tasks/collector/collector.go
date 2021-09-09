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

package collector

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tasks/workflow/builder"
)

// A Collector is responsible for registering new tasks/workflows/executions
// A task is an execution for non workflow task
// A task is n executions for workflow task with n equal to the steps number

// Collector concern is
// - register in Consul the task (/tasks/)
// - register in Consul the workflow if task is workflow (/workflows/)
// - register in Consul the executions for initial steps if task is workflow (/executions/)
// - register in Consul the execution for task if non workflow task
type Collector struct {
	consulClient *api.Client
}

// NewCollector creates a Collector
func NewCollector(consulClient *api.Client) *Collector {
	return &Collector{consulClient: consulClient}
}

// RegisterTaskWithData register a new Task of a given type with some data
//
// The task id is returned.
func (c *Collector) RegisterTaskWithData(targetID string, taskType tasks.TaskType, data map[string]string) (string, error) {
	return c.registerTask(targetID, taskType, data)
}

// RegisterTask register a new Task of a given type.
//
// The task id is returned.
// Basically this is a shorthand for RegisterTaskWithData(targetID, taskType, nil)
func (c *Collector) RegisterTask(targetID string, taskType tasks.TaskType) (string, error) {
	return c.RegisterTaskWithData(targetID, taskType, nil)
}

// ResumeTask allows to resume a task previously failed
func (c *Collector) ResumeTask(ctx context.Context, taskID string) error {
	taskType, err := tasks.GetTaskType(taskID)
	if err != nil {
		return errors.Wrapf(err, "Failed to resume task with taskID;%q", taskID)
	}
	targetID, err := tasks.GetTaskTarget(taskID)
	if err != nil {
		return errors.Wrapf(err, "Failed to resume task with taskID;%q", taskID)
	}
	var workflowName string
	if tasks.IsWorkflowTask(taskType) {
		workflowName, err = tasks.GetTaskData(taskID, "workflowName")
		if err != nil {
			return errors.Wrapf(err, "Failed to resume task with taskID;%q", taskID)
		}
	}

	// Set task status to initial and blank error message
	taskPath := path.Join(consulutil.TasksPrefix, taskID)
	taskOps := api.KVTxnOps{
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(taskPath, "status"),
			Value: []byte(strconv.Itoa(int(tasks.TaskStatusINITIAL))),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(taskPath, "errorMessage"),
			Value: []byte(""),
		},
		&api.KVTxnOp{
			Verb: api.KVDelete,
			Key:  path.Join(taskPath, ".errorFlag"),
		},
		&api.KVTxnOp{
			Verb: api.KVDelete,
			Key:  path.Join(taskPath, ".cancelFlag"),
		},
	}
	// Set deployment status to initial for some task types
	switch taskType {
	case tasks.TaskTypeDeploy, tasks.TaskTypeUnDeploy, tasks.TaskTypeScaleIn, tasks.TaskTypeScaleOut, tasks.TaskTypePurge:
		taskOps = append(taskOps, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(consulutil.DeploymentKVPrefix, targetID, "status"),
			Value: []byte(deployments.INITIAL.String()),
		})
	}

	// Get operations to register executions
	err = c.prepareForRegistration(ctx, taskOps, taskType, taskID, targetID, workflowName, false)
	if err != nil {
		return err
	}

	tasks.EmitTaskEventWithContextualLogs(ctx, targetID, taskID, taskType, workflowName, tasks.TaskStatusINITIAL.String())
	return nil
}

func (c *Collector) registerTask(targetID string, taskType tasks.TaskType, data map[string]string) (string, error) {
	// First check if other tasks are running for this target before creating a new one except for Action tasks
	if tasks.IsDeploymentRelatedTask(taskType) {

		tasksTypesToIgnore := []tasks.TaskType{
			tasks.TaskTypeQuery, tasks.TaskTypeAction,
		}
		// Concurrent executions of custom commands and custom workflows are
		// allowed, so the registration of a custom command or custom workflow task
		// is allowed if another custom command or custom workflow task is on-going.
		// Adding the corresponding task types to the list of types to ignore
		// when checking if a task is already registered for this target.
		if taskType == tasks.TaskTypeCustomCommand ||
			taskType == tasks.TaskTypeCustomWorkflow {

			tasksTypesToIgnore = append(tasksTypesToIgnore, tasks.TaskTypeCustomCommand,
				tasks.TaskTypeCustomWorkflow)
		}

		taskList, err := deployments.GetDeploymentTaskList(context.Background(), targetID)
		if err != nil {
			return "", err
		}
		hasLivingTask, livingTaskID, livingTaskStatus, err :=
			tasks.HasLivingTasks(taskList, tasksTypesToIgnore)
		if err != nil {
			return "", err
		}
		if hasLivingTask {
			return "", tasks.NewAnotherLivingTaskAlreadyExistsError(livingTaskID, targetID, livingTaskStatus)
		}
	}

	taskID := fmt.Sprint(uuid.NewV4())
	taskPath := path.Join(consulutil.TasksPrefix, taskID)
	creationDate, err := time.Now().MarshalBinary()
	if err != nil {
		return "", errors.Wrap(err, "Failed to generate task creation date")
	}

	taskOps := api.KVTxnOps{
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(taskPath, "targetId"),
			Value: []byte(targetID),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(taskPath, "status"),
			Value: []byte(strconv.Itoa(int(tasks.TaskStatusINITIAL))),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(taskPath, "type"),
			Value: []byte(strconv.Itoa(int(taskType))),
		},
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(taskPath, "creationDate"),
			Value: creationDate,
		},
	}

	if tasks.IsDeploymentRelatedTask(taskType) {
		// We store tasks references under deployment to speedup access to task of a given deployment
		// see https://github.com/ystia/yorc/issues/671
		taskOps = append(taskOps, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(consulutil.DeploymentKVPrefix, targetID, "tasks", taskID),
			Flags: 1,
		})
	}

	if data != nil {
		for k, v := range data {
			taskOps = append(taskOps, &api.KVTxnOp{
				Verb:  api.KVSet,
				Key:   path.Join(taskPath, "data", k),
				Value: []byte(v),
			})
		}
	}
	var workflowName string
	if tasks.IsWorkflowTask(taskType) {
		var ok bool
		workflowName, ok = data["workflowName"]
		if !ok {
			return "", errors.Errorf("Workflow name can't be retrieved from data :%v", data)
		}
	}
	ctx := events.NewContext(context.Background(), events.LogOptionalFields{
		events.WorkFlowID:  workflowName,
		events.ExecutionID: taskID,
	})

	err = c.prepareForRegistration(ctx, taskOps, taskType, taskID, targetID, workflowName, true)
	if err != nil {
		return "", err
	}

	if taskType == tasks.TaskTypeUnDeploy || taskType == tasks.TaskTypePurge {
		status, err := deployments.GetDeploymentStatus(ctx, targetID)
		if err != nil {
			return "", err
		}
		if status != deployments.UNDEPLOYED {
			// Set the deployment status to undeployment in progress right now as the task was registered
			// But we don't know when it will be processed. This will prevent someone to retry to undeploy as
			// the status stay in deployed. Doing this will be an error as the task for undeploy is already registered.
			deployments.SetDeploymentStatus(ctx, targetID, deployments.UNDEPLOYMENT_IN_PROGRESS)
		}
	}
	tasks.EmitTaskEventWithContextualLogs(ctx, targetID, taskID, taskType, workflowName, tasks.TaskStatusINITIAL.String())
	return taskID, nil
}

func (c *Collector) prepareForRegistration(ctx context.Context, operations api.KVTxnOps, taskType tasks.TaskType, taskID, targetID, workflowName string, registerWorkflow bool) error {
	// Register step tasks for each step in case of workflow
	// Add executions for each initial steps
	if workflowName != "" {
		stepOps, err := builder.BuildInitExecutionOperations(ctx, targetID, taskID, workflowName, registerWorkflow)
		if err != nil {
			return err
		}
		operations = append(operations, stepOps...)
	} else {
		// Add execution key for non workflow task
		execID := fmt.Sprint(uuid.NewV4())
		stepExecPath := path.Join(consulutil.ExecutionsTaskPrefix, execID)
		operations = append(operations, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(stepExecPath, "taskID"),
			Value: []byte(taskID),
		})
	}

	err := tasks.StoreOperations(taskID, operations)
	if err != nil {
		return errors.Wrapf(err, "Failed to register task with targetID:%q, taskType:%q due to error %s",
			targetID, taskType.String(), err.Error())
	}

	log.Debugf("Registered task %s type %q for target %s\n", taskID, taskType.String(), targetID)

	return nil
}
