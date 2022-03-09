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

package workflow

import (
	"context"
	"path"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tasks/workflow/builder"
)

const (
	// Task data providing the parent worflow name for an inline workflow
	taskDataParentWorkflowName = "parentWorkflowName"
	// Task data providing the parent step name for an inline workflow
	taskDataParentStepName = "parentStepName"
	// Task data providing the parent task ID
	taskDataParentTaskID = "parentTaskID"
	// Task data providing the deployment ID
	taskDataDeploymentID = "deploymentID"
	// Task data providing the current workflow name
	taskDataWorkflowName = "workflowName"
)

// createWorkflowStepsOperations returns Consul transactional KV operations for initiating workflow execution
func createWorkflowStepsOperations(taskID string, steps []*step) (api.KVTxnOps, error) {
	ops := make(api.KVTxnOps, 0)
	var stepOps api.KVTxnOps
	for _, step := range steps {
		// Add execution key for initial steps only
		u, err := uuid.NewV4()
		if err != nil {
			return ops, errors.Wrapf(err, "Failed to generate a UUID")
		}
		execID := u.String()
		log.Debugf("Register task execution with ID:%q, taskID:%q and step:%q", execID, taskID, step.Name)
		stepExecPath := path.Join(consulutil.ExecutionsTaskPrefix, execID)
		stepOps = api.KVTxnOps{
			&api.KVTxnOp{
				Verb:  api.KVSet,
				Key:   path.Join(stepExecPath, "taskID"),
				Value: []byte(taskID),
			},
			&api.KVTxnOp{
				Verb:  api.KVSet,
				Key:   path.Join(stepExecPath, "step"),
				Value: []byte(step.Name),
			},
			&api.KVTxnOp{
				Verb:  api.KVSet,
				Key:   path.Join(consulutil.TasksPrefix, taskID, ".runningExecutions", execID),
				Value: []byte(""),
			},
		}
		ops = append(ops, stepOps...)
		log.Debugf("Will store runningExecutions with id %q in txn for task %q", execID, taskID)
	}
	return ops, nil
}

func getCallOperationsFromStep(s *step) []string {
	ops := make([]string, 0)
	for _, a := range s.Activities {
		if a.Type() == builder.ActivityTypeCallOperation {
			ops = append(ops, a.Value())
		}
	}
	return ops
}

func updateTaskStatusAccordingToWorkflowStatus(ctx context.Context, deploymentID, taskID, workflowName string) (tasks.TaskStatus, error) {
	log.Debugf("Updating task status according to workflow status. Deployment %q, taskID %q, workflow %q", deploymentID, taskID, workflowName)
	hasCancelledFlag, err := tasks.TaskHasCancellationFlag(taskID)
	if err != nil {
		return tasks.TaskStatusFAILED, errors.Wrapf(err, "Failed to retrieve workflow step statuses with TaskID:%q", taskID)
	}
	hasErrorFlag, err := tasks.TaskHasErrorFlag(taskID)
	if err != nil {
		return tasks.TaskStatusFAILED, errors.Wrapf(err, "Failed to retrieve workflow step statuses with TaskID:%q", taskID)
	}

	status := tasks.TaskStatusDONE
	if hasCancelledFlag {
		status = tasks.TaskStatusCANCELED
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).Registerf("Workflow %q canceled", workflowName)
	} else if hasErrorFlag {
		status = tasks.TaskStatusFAILED
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).Registerf("Workflow %q ended in error", workflowName)
	} else {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).Registerf("Workflow %q ended successfully", workflowName)
	}
	// Assume that step in error has stored the error reason on task
	return status, errors.Wrapf(checkAndSetTaskStatus(ctx, deploymentID, taskID, status, nil), "Failed to update task status to %q with TaskID: %q", status, taskID)
}

// Get the parent workflow of an inline workflow, returns an empty string if there
// is no parent workflow
func getParentWorkflow(ctx context.Context, t *taskExecution, wfName string) (string, error) {
	parentWorkflow, err := tasks.GetTaskData(t.taskID, taskDataParentWorkflowName)
	if err != nil && !tasks.IsTaskDataNotFoundError(err) {
		return parentWorkflow, err
	}

	return parentWorkflow, nil
}

// Update a parent workflow step for which an inline workflow just finished with
// the status in argument.
// Register next steps depending on the status of the workflow execution
func updateParentWorkflowStepAndRegisterNextSteps(ctx context.Context, t *taskExecution, wfName string, taskStatus tasks.TaskStatus) error {

	// Get the parent step to update
	parentStepName, err := tasks.GetTaskData(t.taskID, taskDataParentStepName)
	if err != nil && !tasks.IsTaskDataNotFoundError(err) {
		return err
	}

	if parentStepName == "" {
		return errors.Errorf("Found no inline step name in task %s data for parent workflow %s", t.taskID, wfName)
	}

	// Get the parent task ID
	parentTaskID, err := tasks.GetTaskData(t.taskID, taskDataParentTaskID)
	if err != nil && !tasks.IsTaskDataNotFoundError(err) {
		return err
	}

	if parentTaskID == "" {
		return errors.Errorf("Found no parent task ID in task %s data for parent workflow %s", t.taskID, wfName)
	}

	stepStatus := tasks.TaskStepStatusDONE
	if taskStatus == tasks.TaskStatusFAILED {
		stepStatus = tasks.TaskStepStatusERROR
	} else if taskStatus == tasks.TaskStatusCANCELED {
		stepStatus = tasks.TaskStepStatusCANCELED
	}
	err = tasks.UpdateTaskStepWithStatus(parentTaskID, parentStepName, stepStatus)
	if err != nil {
		return err
	}

	if stepStatus == tasks.TaskStepStatusCANCELED {
		return err
	}

	if stepStatus == tasks.TaskStepStatusERROR {
		// Check the option continue on error
		continueOnError, err := checkByPassErrors(t, wfName)
		if err != nil || !continueOnError {
			return err
		}
	}

	// Add next steps
	err = registerParentStepNextSteps(ctx, t, parentTaskID, wfName, parentStepName)
	return err

}

func registerParentStepNextSteps(ctx context.Context, t *taskExecution, parentTaskID, wfName, stepName string) error {

	// Get the deployment ID
	deploymentID, err := tasks.GetTaskData(t.taskID, taskDataDeploymentID)
	if err != nil {
		return err
	}

	wfSteps, err := builder.BuildWorkFlow(ctx, deploymentID, wfName)
	if err != nil {
		return errors.Wrapf(err, "Failed to build steps for workflow:%q", wfName)
	}
	if len(wfSteps) == 0 {
		// Nothing to do
		return nil
	}

	bs, ok := wfSteps[stepName]
	if !ok {
		return errors.Errorf("Failed to build step: %q for workflow: %q, unknown step", stepName, wfName)
	}
	t.taskID = parentTaskID
	s := wrapBuilderStep(bs, t.cc, t)
	return s.registerNextSteps(ctx, wfName)

}

func checkByPassErrors(t *taskExecution, wfName string) (bool, error) {
	continueOnError, err := tasks.GetTaskData(t.taskID, "continueOnError")
	if err != nil {
		return false, err
	}
	bypassErrors, err := strconv.ParseBool(continueOnError)
	if err != nil {
		return false, errors.Wrapf(err, "failed to parse \"continueOnError\" flag for workflow:%q", wfName)
	}
	return bypassErrors, nil
}

func storeWorkflowOutputs(ctx context.Context, deploymentID, taskID, workflowName string) error {
	outputs, err := deployments.ResolveWorkflowOutputs(ctx, deploymentID, workflowName)
	if err != nil {
		return err
	}

	for outputName, outputValue := range outputs {
		err = tasks.SetTaskData(taskID, path.Join("outputs", outputName), outputValue.RawString())
		if err != nil {
			return err
		}
	}

	return nil
}
