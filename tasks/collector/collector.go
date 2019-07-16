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

	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/events"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/tasks"
	"github.com/ystia/yorc/v3/tasks/workflow/builder"
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
func (c *Collector) ResumeTask(taskID string) error {
	taskType, err := tasks.GetTaskType(c.consulClient.KV(), taskID)
	if err != nil {
		return errors.Wrapf(err, "Failed to resume task with taskID;%q", taskID)
	}
	targetID, err := tasks.GetTaskTarget(c.consulClient.KV(), taskID)
	if err != nil {
		return errors.Wrapf(err, "Failed to resume task with taskID;%q", taskID)
	}
	var workflowName string
	if tasks.IsWorkflowTask(taskType) {
		workflowName, err = tasks.GetTaskData(c.consulClient.KV(), taskID, "workflowName")
		if err != nil {
			return errors.Wrapf(err, "Failed to resume task with taskID;%q", taskID)
		}
	}

	// Set task status to initial
	taskPath := path.Join(consulutil.TasksPrefix, taskID)
	taskOps := api.KVTxnOps{
		&api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(taskPath, "status"),
			Value: []byte(strconv.Itoa(int(tasks.TaskStatusINITIAL))),
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
	err = c.prepareForRegistration(taskOps, taskType, taskID, targetID, workflowName, false)
	if err != nil {
		return err
	}

	tasks.EmitTaskEventWithContextualLogs(nil, c.consulClient.KV(), targetID, taskID, taskType, workflowName, tasks.TaskStatusINITIAL.String())
	return nil
}

func (c *Collector) registerTask(targetID string, taskType tasks.TaskType, data map[string]string) (string, error) {
	// First check if other tasks are running for this target before creating a new one except for Action tasks
	if tasks.TaskTypeAction != taskType {
		hasLivingTask, livingTaskID, livingTaskStatus, err := tasks.TargetHasLivingTasks(c.consulClient.KV(), targetID)
		if err != nil {
			return "", err
		} else if hasLivingTask {
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

	err = c.prepareForRegistration(taskOps, taskType, taskID, targetID, workflowName, true)
	if err != nil {
		return "", err
	}

	ctx := events.NewContext(context.Background(), events.LogOptionalFields{
		events.WorkFlowID:  workflowName,
		events.ExecutionID: taskID,
	})

	if taskType == tasks.TaskTypeUnDeploy || taskType == tasks.TaskTypePurge {
		status, err := deployments.GetDeploymentStatus(c.consulClient.KV(), targetID)
		if err != nil {
			return "", err
		}
		if status != deployments.UNDEPLOYED {
			// Set the deployment status to undeployment in progress right now as the task was registered
			// But we don't know when it will be processed. This will prevent someone to retry to undeploy as
			// the status stay in deployed. Doing this will be an error as the task for undeploy is already registered.
			deployments.SetDeploymentStatus(ctx, c.consulClient.KV(), targetID, deployments.UNDEPLOYMENT_IN_PROGRESS)
		}
	}
	tasks.EmitTaskEventWithContextualLogs(ctx, c.consulClient.KV(), targetID, taskID, taskType, workflowName, tasks.TaskStatusINITIAL.String())
	return taskID, nil
}

func (c *Collector) prepareForRegistration(operations api.KVTxnOps, taskType tasks.TaskType, taskID, targetID, workflowName string, registerWorkflow bool) error {
	// Register step tasks for each step in case of workflow
	// Add executions for each initial steps
	if workflowName != "" {
		stepOps, err := builder.BuildInitExecutionOperations(c.consulClient.KV(), targetID, taskID, workflowName, registerWorkflow)
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

	err := tasks.StoreOperations(c.consulClient.KV(), taskID, operations)
	if err != nil {
		return errors.Wrapf(err, "Failed to register task with targetID:%q, taskType:%q due to error %s",
			targetID, taskType.String(), err.Error())
	}

	log.Debugf("Registered task %s type %q for target %s\n", taskID, taskType.String(), targetID)

	return nil
}
