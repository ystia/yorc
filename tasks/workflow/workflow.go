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
	"fmt"
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"

	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tasks/workflow/builder"
)

// createWorkflowStepsOperations returns Consul transactional KV operations for initiating workflow execution
func createWorkflowStepsOperations(taskID string, steps []*step) api.KVTxnOps {
	ops := make(api.KVTxnOps, 0)
	var stepOps api.KVTxnOps
	for _, step := range steps {
		// Add execution key for initial steps only
		execID := fmt.Sprint(uuid.NewV4())
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
	}
	return ops
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

func updateTaskStatusAccordingToWorkflowStatusIfLatest(ctx context.Context, cc *api.Client, deploymentID, taskID, workflowName string) error {
	l, e, err := numberOfRunningExecutionsForTask(cc, taskID)
	if err != nil {
		return err
	}
	defer l.Unlock()
	if e <= 1 {
		// we are the latest
		_, err := updateTaskStatusAccordingToWorkflowStatus(ctx, cc.KV(), deploymentID, taskID, workflowName)
		return err
	}
	return nil
}

func updateTaskStatusAccordingToWorkflowStatus(ctx context.Context, kv *api.KV, deploymentID, taskID, workflowName string) (tasks.TaskStatus, error) {
	hasCancelledFlag, err := tasks.TaskHasCancellationFlag(kv, taskID)
	if err != nil {
		return tasks.TaskStatusFAILED, errors.Wrapf(err, "Failed to retrieve workflow step statuses with TaskID:%q", taskID)
	}
	hasErrorFlag, err := tasks.TaskHasErrorFlag(kv, taskID)
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
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).Registerf("Workflow %q ended without error", workflowName)
	}

	events.PublishAndLogWorkflowStatusChange(ctx, kv, deploymentID, taskID, workflowName, status.String())
	return status, errors.Wrapf(checkAndSetTaskStatus(kv, taskID, status), "Failed to update task status to %q with TaskID: %q", status, taskID)
}
