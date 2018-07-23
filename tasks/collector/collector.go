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
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tasks/workflow"
	"path"
	"strconv"
	"strings"
	"time"
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

func (c *Collector) registerTask(targetID string, taskType tasks.TaskType, data map[string]string) (string, error) {
	// First check if other tasks are running for this target before creating a new one
	hasLivingTask, livingTaskID, livingTaskStatus, err := tasks.TargetHasLivingTasks(c.consulClient.KV(), targetID)
	if err != nil {
		return "", err
	} else if hasLivingTask {
		return "", tasks.NewAnotherLivingTaskAlreadyExistsError(livingTaskID, targetID, livingTaskStatus)
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
				Key:   path.Join(taskPath, k),
				Value: []byte(v),
			})
		}
	}

	// Register step tasks for each step in case of workflow
	// Add executions for each initial steps
	if tasks.IsWorkflowTask(taskType) {
		wfName, err := getWfNameFromTaskType(taskType, data)
		if err != nil {
			return "", err
		}
		stepOps, err := workflow.GetWorkflowInitOperations(c.consulClient.KV(), targetID, taskID, wfName)
		if err != nil {
			return "", err
		}
		taskOps = append(taskOps, stepOps...)
	} else {
		// Add execution key for non workflow task
		execID := fmt.Sprint(uuid.NewV4())
		stepExecPath := path.Join(consulutil.ExecutionsTaskPrefix, execID)
		taskOps = append(taskOps, &api.KVTxnOp{
			Verb:  api.KVSet,
			Key:   path.Join(stepExecPath, "taskID"),
			Value: []byte(taskID),
		})
	}

	ok, response, _, err := c.consulClient.KV().Txn(taskOps, nil)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to register task with targetID:%q, taskType", targetID, taskType.String())
	}
	if !ok {
		errs := make([]string, 0)
		for _, e := range response.Errors {
			errs = append(errs, e.What)
		}
		return "", errors.Wrapf(err, "Failed to register task with targetID:%q, taskType due to:%s", targetID, taskType.String(), strings.Join(errs, ", "))
	}

	tasks.EmitTaskEventWithContextualLogs(nil, c.consulClient.KV(), targetID, taskID, taskType, tasks.TaskStatusINITIAL.String())
	return taskID, nil
}

func getWfNameFromTaskType(taskType tasks.TaskType, data map[string]string) (string, error) {
	switch taskType {
	case tasks.TaskTypeDeploy, tasks.TaskTypeScaleOut:
		return "install", nil
	case tasks.TaskTypeUnDeploy, tasks.TaskTypeScaleIn:
		return "uninstall", nil
	case tasks.TaskTypeCustomWorkflow:
		wfName, ok := data["workflowName"]
		if !ok {
			return "", errors.Errorf("Workflow name can't be retrieved from data :%v", data)
		}
		return wfName, nil
	default:
		return "", errors.Errorf("Workflow can't be resolved from task type:%q", tasks.TaskType(taskType))
	}
}
