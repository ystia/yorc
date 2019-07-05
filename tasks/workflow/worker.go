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
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/events"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/helper/metricsutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/prov"
	"github.com/ystia/yorc/v3/prov/operations"
	"github.com/ystia/yorc/v3/prov/scheduling"
	"github.com/ystia/yorc/v3/registry"
	"github.com/ystia/yorc/v3/tasks"
	"github.com/ystia/yorc/v3/tasks/workflow/builder"
	"github.com/ystia/yorc/v3/tosca"
)

// worker concern is to execute a task execution received from the dispatcher
// It has to poll cancellation flags to stop execution
// In case of workflow task, it runs a workflow step as a task execution and needs to update related task status, and register next step tasks execution if its execution is successful
// It has to release related execution Consul lock and mark the execution to delete
// If workflow is done, the task and deployment statuses need to be updated
// If an error occurred during the step execution, task status need to be updated
// Each done step will register the next ones. In case of join, the last done of previous steps will register the next step
type worker struct {
	workerPool   chan chan *taskExecution
	TaskChannel  chan *taskExecution
	shutdownCh   chan struct{}
	consulClient *api.Client
	cfg          config.Configuration
}

func newWorker(workerPool chan chan *taskExecution, shutdownCh chan struct{}, consulClient *api.Client, cfg config.Configuration) worker {
	return worker{
		workerPool:   workerPool,
		TaskChannel:  make(chan *taskExecution),
		shutdownCh:   shutdownCh,
		consulClient: consulClient,
		cfg:          cfg,
	}
}

// Start method starts the run loop for the worker, listening for a quit channel in
// case we need to stop it
func (w *worker) Start() {
	go func() {
		for {
			// register the current worker into the worker queue.
			w.workerPool <- w.TaskChannel
			select {
			case task := <-w.TaskChannel:
				// we have received a work request.
				log.Debugf("Worker got Task Execution with id %s", task.taskID)
				w.handleExecution(task)

			case <-w.shutdownCh:
				// we have received a signal to stop
				log.Debugln("Worker received shutdown signal. Exiting...")
				return
			}
		}
	}()
}

func setNodeStatus(ctx context.Context, kv *api.KV, taskID, deploymentID, nodeName, status string) error {
	instancesIDs, err := tasks.GetInstances(kv, taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}

	for _, id := range instancesIDs {
		// Publish status change event
		err := deployments.SetInstanceStateStringWithContextualLogs(ctx, kv, deploymentID, nodeName, id, status)
		if err != nil {
			return err
		}
	}
	return nil
}

func getOperationExecutor(kv *api.KV, deploymentID, artifact string) (prov.OperationExecutor, error) {
	reg := registry.GetRegistry()

	exec, originalErr := reg.GetOperationExecutor(artifact)
	if originalErr == nil {
		return exec, nil
	}
	// Try to get an executor for artifact parent type but return the original error if we do not found any executors
	parentArt, err := deployments.GetParentType(kv, deploymentID, artifact)
	if err != nil {
		return nil, err
	}
	if parentArt != "" {
		exec, err := getOperationExecutor(kv, deploymentID, parentArt)
		if err == nil {
			return exec, nil
		}
	}
	return nil, originalErr
}

// cleanupScaledDownNodes removes nodes instances from Consul
func (w *worker) cleanupScaledDownNodes(t *taskExecution) error {
	kv := w.consulClient.KV()

	nodes, err := tasks.GetTaskRelatedNodes(kv, t.taskID)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		var instances []string
		instances, err = tasks.GetInstances(kv, t.taskID, t.targetID, node)
		if err != nil {
			return err
		}
		for _, instance := range instances {
			err = deployments.DeleteInstance(kv, t.targetID, node, instance)
			if err != nil {
				return err
			}
			err = deployments.DeleteRelationshipInstance(kv, t.targetID, node, instance)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

func (w *worker) handleExecution(t *taskExecution) {
	log.Debugf("Handle task execution:%+v", t)
	err := t.notifyStart()
	if err != nil {
		log.Printf("%+v", err)
		return
	}
	defer func() {
		// Remove currently processing execution flag
		err := t.notifyEnd()
		if err != nil {
			log.Printf("%+v", err)
		}
		t.delete()
		if err != nil {
			log.Printf("%+v", err)
		}
		t.releaseLock()
		if err != nil {
			log.Printf("%+v", err)
		}
	}()

	if taskStatus, err := t.getTaskStatus(); err != nil && taskStatus == tasks.TaskStatusINITIAL {
		metrics.MeasureSince([]string{"tasks", "wait"}, t.creationDate)
	}

	metrics.MeasureSince([]string{"TaskExecution", "wait"}, t.creationDate)
	kv := w.consulClient.KV()
	// Fill log optional fields for log registration
	wfName, _ := tasks.GetTaskData(kv, t.taskID, "workflowName")
	logOptFields := events.LogOptionalFields{
		events.WorkFlowID:  wfName,
		events.ExecutionID: t.taskID,
	}
	ctx := events.NewContext(context.Background(), logOptFields)
	err = checkAndSetTaskStatus(ctx, t.cc.KV(), t.targetID, t.taskID, tasks.TaskStatusRUNNING)
	if err != nil {
		log.Printf("%+v", err)
		return
	}
	defer func(t *taskExecution, start time.Time) {
		if taskStatus, err := t.getTaskStatus(); err != nil && taskStatus != tasks.TaskStatusRUNNING {
			metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"task", t.targetID, t.taskType.String(), taskStatus.String()}), 1)
			metrics.MeasureSince(metricsutil.CleanupMetricKey([]string{"task", t.targetID, t.taskType.String()}), start)
		}
	}(t, time.Now())

	switch t.taskType {
	case tasks.TaskTypeDeploy:
		err = w.runDeploy(ctx, t)
	case tasks.TaskTypeUnDeploy, tasks.TaskTypePurge:
		err = w.runUndeploy(ctx, t)
	case tasks.TaskTypeScaleOut:
		err = w.runScaleOut(ctx, t)
	case tasks.TaskTypeScaleIn:
		err = w.runScaleIn(ctx, t)
	case tasks.TaskTypeCustomWorkflow:
		err = w.runCustomWorkflow(ctx, t, wfName)
	case tasks.TaskTypeAction:
		err = w.runAction(ctx, t)
	case tasks.TaskTypeQuery, tasks.TaskTypeCustomCommand, tasks.TaskTypeForcePurge:
		// Those kind of task will manage monitoring of taskFailure differently
		err = w.runOneExecutionTask(ctx, t)
	default:
		err = errors.Errorf("Unknown TaskType %d (%s) for TaskExecution with id %q", t.taskType, t.taskType.String(), t.taskID)
	}
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, t.targetID).Registerf("%v", err)
		if log.IsDebug() {
			log.Debugf("%+v", err)
		}
	}
}

func (w *worker) runOneExecutionTask(ctx context.Context, t *taskExecution) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	wasCancelled := new(bool)
	tasks.MonitorTaskCancellation(ctx, w.consulClient.KV(), t.taskID, func() {
		// Mark cancelled
		*wasCancelled = true
		// Actually cancel our context
		cancelFunc()
	})

	var err error
	defer func() {
		kv := w.consulClient.KV()
		if *wasCancelled && err != nil {
			// A Cancel was requested and there was an error
			// let's assume the we failed for this reason
			// while we actually don't know if we encounter another error
			tasks.UpdateTaskStepWithStatus(kv, t.taskID, t.step, tasks.TaskStepStatusCANCELED)
			checkAndSetTaskStatus(ctx, kv, t.targetID, t.taskID, tasks.TaskStatusCANCELED)
		} else if err != nil {
			tasks.UpdateTaskStepWithStatus(kv, t.taskID, t.step, tasks.TaskStepStatusERROR)
			checkAndSetTaskStatus(ctx, kv, t.targetID, t.taskID, tasks.TaskStatusFAILED)
		} else {
			tasks.UpdateTaskStepWithStatus(kv, t.taskID, t.step, tasks.TaskStepStatusDONE)
			checkAndSetTaskStatus(ctx, kv, t.targetID, t.taskID, tasks.TaskStatusDONE)
		}
	}()
	// We do not monitor task failure as there is only one execution

	switch t.taskType {
	case tasks.TaskTypeQuery:
		err = w.runQuery(ctx, t)
	case tasks.TaskTypeCustomCommand:
		ctx, err = w.runCustomCommand(ctx, t)
	case tasks.TaskTypeForcePurge:
		err = w.runPurge(ctx, t)
	default:
		err = errors.Errorf("Unknown TaskType %d (%s) for TaskExecution with id %q", t.taskType, t.taskType.String(), t.taskID)
	}
	return err
}

func (w *worker) runCustomCommand(ctx context.Context, t *taskExecution) (context.Context, error) {
	kv := w.consulClient.KV()
	commandName, err := tasks.GetTaskData(kv, t.taskID, "commandName")
	if err != nil {
		return ctx, errors.Wrap(err, "failed to retrieve custom command name")
	}

	var interfaceName string
	interfaceName, err = tasks.GetTaskData(kv, t.taskID, "interfaceName")
	if err != nil {
		return ctx, errors.Wrap(err, "failed to retrieve custom interface name")
	}
	nodes, err := tasks.GetTaskRelatedNodes(kv, t.taskID)
	if err != nil {
		return ctx, errors.Wrap(err, "failed to retrieve custom command target node")
	}
	if len(nodes) != 1 {
		return ctx, errors.Wrapf(err, "expecting custom command to be related to \"1\" node while it is actually related to \"%d\" nodes", len(nodes))
	}
	nodeName := nodes[0]
	nodeType, err := deployments.GetNodeType(w.consulClient.KV(), t.targetID, nodeName)
	if err != nil {
		return ctx, err
	}
	op, err := operations.GetOperation(ctx, kv, t.targetID, nodeName, interfaceName+"."+commandName, "", "")
	if err != nil {
		err = setNodeStatus(ctx, t.cc.KV(), t.taskID, t.targetID, nodeName, tosca.NodeStateError.String())
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to set status for node %q: %+v", t.targetID, t.taskID, nodeName, err)
		}
		return ctx, errors.Wrapf(err, "Command TaskExecution failed for node %q", nodeName)
	}
	exec, err := getOperationExecutor(kv, t.targetID, op.ImplementationArtifact)
	if err != nil {
		err = setNodeStatus(ctx, t.cc.KV(), t.taskID, t.targetID, nodeName, tosca.NodeStateError.String())
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to set status for node %q: %+v", t.targetID, t.taskID, nodeName, err)
		}
		return ctx, errors.Wrapf(err, "Command TaskExecution failed for node %q", nodeName)
	}

	ctx = operations.SetOperationLogFields(ctx, op)
	ctx = events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.NodeID: nodeName, events.OperationName: op.Name})

	err = func() error {
		defer metrics.MeasureSince(metricsutil.CleanupMetricKey([]string{"executor", "operation", t.targetID, nodeType, op.Name}), time.Now())
		return exec.ExecOperation(ctx, w.cfg, t.taskID, t.targetID, nodeName, op)
	}()
	if err != nil {
		metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "operation", t.targetID, nodeType, op.Name, "failures"}), 1)
		err2 := setNodeStatus(ctx, t.cc.KV(), t.taskID, t.targetID, nodeName, tosca.NodeStateError.String())
		if err2 != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, t.targetID).Registerf("failed to update node %q state to error", nodeName)
		}
		return ctx, err
	}
	metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "operation", t.targetID, nodeType, op.Name, "successes"}), 1)
	return ctx, err
}

func (w *worker) endAction(ctx context.Context, t *taskExecution, action *prov.Action, wasCancelled bool, actionErr error) {
	if action.AsyncOperation.TaskID == "" {
		return
	}
	defer func() {
		l, err := acquireRunningExecLock(w.consulClient, action.AsyncOperation.TaskID)
		if err != nil {

			return
		}
		defer l.Unlock()
		w.consulClient.KV().Delete(path.Join(consulutil.TasksPrefix, action.AsyncOperation.TaskID, ".runningExecutions", action.ID), nil)
	}()

	// Rebuild the original workflow step
	steps, err := builder.BuildWorkFlow(w.consulClient.KV(), action.AsyncOperation.DeploymentID, action.AsyncOperation.WorkflowName)
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, action.AsyncOperation.DeploymentID).Registerf("%v", err)
		log.Debugf("%+v", err)
		return
	}
	bs, ok := steps[action.AsyncOperation.StepName]
	if !ok {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, action.AsyncOperation.DeploymentID).Registerf("step %q missing", action.AsyncOperation.StepName)
		return
	}

	taskType, err := tasks.GetTaskType(w.consulClient.KV(), action.AsyncOperation.TaskID)
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, action.AsyncOperation.DeploymentID).Registerf("%v", err)
		log.Debugf("%+v", err)
		return
	}

	// This is a fake task execution as the original one was deleted
	s := wrapBuilderStep(bs, w.consulClient, newTaskExecution(
		action.AsyncOperation.ExecutionID,
		action.AsyncOperation.TaskID,
		action.AsyncOperation.DeploymentID,
		action.AsyncOperation.StepName,
		w.consulClient,
		time.Now(),
		taskType,
	))

	defer updateTaskStatusAccordingToWorkflowStatusIfLatest(ctx, w.consulClient, action.AsyncOperation.DeploymentID, action.AsyncOperation.TaskID, action.AsyncOperation.WorkflowName)

	stepStatus := tasks.TaskStepStatusDONE
	if wasCancelled {
		stepStatus = tasks.TaskStepStatusCANCELED
		s.registerOnCancelOrFailureSteps(ctx, action.AsyncOperation.WorkflowName, s.OnCancel)
	} else if actionErr != nil {
		stepStatus = tasks.TaskStepStatusERROR
		tasks.NotifyErrorOnTask(action.AsyncOperation.TaskID)
		s.registerOnCancelOrFailureSteps(ctx, action.AsyncOperation.WorkflowName, s.OnFailure)
	} else {
		// we are not stopped or erroed we just have to reschedule next steps
		err = s.registerNextSteps(ctx, action.AsyncOperation.WorkflowName)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, action.AsyncOperation.DeploymentID).Registerf("Failed to register steps preceded by %q for execution: %v", action.AsyncOperation.StepName, err)
			log.Debugf("%+v", err)
		}
	}

	err = tasks.UpdateTaskStepWithStatus(w.consulClient.KV(), action.AsyncOperation.TaskID, action.AsyncOperation.StepName, stepStatus)
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, action.AsyncOperation.DeploymentID).Registerf("%v", err)
		log.Debugf("%+v", err)
	}

	instances, err := tasks.GetInstances(w.consulClient.KV(), action.AsyncOperation.TaskID, action.AsyncOperation.DeploymentID, action.AsyncOperation.NodeName)
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, action.AsyncOperation.DeploymentID).Registerf("%v", err)
		log.Debugf("%+v", err)
	}
	for _, instanceName := range instances {
		s.publishInstanceRelatedEvents(ctx, w.consulClient.KV(), action.AsyncOperation.DeploymentID, instanceName, action.AsyncOperation.WorkflowStepInfo, stepStatus)
	}
}

func (w *worker) runAction(ctx context.Context, t *taskExecution) error {
	// Delete task at the end of it execution
	defer tasks.DeleteTask(w.consulClient.KV(), t.taskID)
	action := &prov.Action{}
	var err error
	action.Data, err = tasks.GetAllTaskData(w.consulClient.KV(), t.taskID)
	if err != nil {
		return err
	}
	action.ID = action.Data["id"]
	action.ActionType = action.Data["actionType"]
	aos, ok := action.Data["asyncOperation"]
	if ok && strings.TrimSpace(aos) != "" {
		err = json.Unmarshal([]byte(action.Data["asyncOperation"]), &action.AsyncOperation)
		if err != nil {
			return errors.Wrap(err, "failed to read asyncOperation for action")
		}
	}
	wasCancelled := new(bool)
	if action.AsyncOperation.TaskID != "" {
		ctx = operations.SetOperationLogFields(ctx, action.AsyncOperation.Operation)
		ctx = events.AddLogOptionalFields(ctx, events.LogOptionalFields{
			events.ExecutionID: action.AsyncOperation.TaskID,
			events.WorkFlowID:  action.AsyncOperation.WorkflowName,
			events.NodeID:      action.AsyncOperation.NodeName,
		})
		// Monitor parent task for failure & cancellation
		var cf context.CancelFunc
		ctx, cf = context.WithCancel(ctx)
		defer cf()
		tasks.MonitorTaskCancellation(ctx, w.consulClient.KV(), action.AsyncOperation.TaskID, func() {
			*wasCancelled = true
			// Cancel our context
			cf()
			// Unregister this action asap to prevent new schedulings
			scheduling.UnregisterAction(w.consulClient, action.ID)
			tasks.UpdateTaskStepWithStatus(w.consulClient.KV(), action.AsyncOperation.TaskID, action.AsyncOperation.StepName, tasks.TaskStepStatusCANCELED)
		})
		tasks.MonitorTaskFailure(ctx, w.consulClient.KV(), action.AsyncOperation.TaskID, func() {
			// Unregister this action asap to prevent new schedulings
			scheduling.UnregisterAction(w.consulClient, action.ID)

			// Let this action a chance to finish
			go func() {
				select {
				case <-ctx.Done():
					return
				case <-time.After(w.cfg.WfStepGracefulTerminationTimeout):
					// Graceful timeout reached let's cancel this op
					cf()
					tasks.UpdateTaskStepWithStatus(w.consulClient.KV(), action.AsyncOperation.TaskID, action.AsyncOperation.StepName, tasks.TaskStepStatusCANCELED)
				}
			}()

		})
	}
	// Find an actionOperator which match with this actionType
	var reg = registry.GetRegistry()
	operator, err := reg.GetActionOperator(action.ActionType)
	if err != nil {
		return errors.Wrap(err, "Failed to find matching operator")
	}

	deregister, err := operator.ExecAction(ctx, w.cfg, t.taskID, t.targetID, action)
	if deregister || *wasCancelled {
		scheduling.UnregisterAction(w.consulClient, action.ID)
		w.endAction(ctx, t, action, *wasCancelled, err)
	}
	if err != nil {
		return err
	}
	// useless as we will delete the task at the end of the function
	// checkAndSetTaskStatus(t.cc.KV(), t.taskID, tasks.TaskStatusDONE)
	log.Printf("Action with ID:%s successfully executed", action.ID)
	return nil
}

func (w *worker) runQuery(ctx context.Context, t *taskExecution) error {
	split := strings.Split(t.targetID, ":")
	if len(split) != 2 {
		return errors.Errorf("unexpected format for targetID: %q", t.targetID)
	}
	query := split[0]
	target := split[1]

	switch query {
	case "infra_usage":
		var reg = registry.GetRegistry()
		collector, err := reg.GetInfraUsageCollector(target)
		if err != nil {
			return err
		}
		res, err := collector.GetUsageInfo(ctx, w.cfg, t.taskID, target)
		if err != nil {
			return err
		}

		// store resultSet as a JSON
		resultPrefix := path.Join(consulutil.TasksPrefix, t.taskID, "resultSet")
		if res != nil {
			jsonRes, err := json.Marshal(res)
			if err != nil {
				return errors.Wrapf(err, "Failed to marshal infra usage info [%+v]", res)
			}
			err = consulutil.StoreConsulKey(resultPrefix, jsonRes)
			if err != nil {
				return errors.Wrap(err, "Failed to store query result")
			}
		}
	default:
		return errors.Errorf("Unknown query: %q", query)
	}
	return nil
}

func (w *worker) makeWorkflowFinalFunction(ctx context.Context, kv *api.KV, deploymentID, taskID, wfName string, successWfStatus, failureWfStatus deployments.DeploymentStatus) func() error {
	return func() error {
		taskStatus, err := updateTaskStatusAccordingToWorkflowStatus(ctx, kv, deploymentID, taskID, wfName)
		if err != nil {
			return err
		}
		wfStatus := successWfStatus
		if taskStatus != tasks.TaskStatusDONE {
			wfStatus = failureWfStatus
		}
		return deployments.SetDeploymentStatus(ctx, kv, deploymentID, wfStatus)
	}
}

func (w *worker) runDeploy(ctx context.Context, t *taskExecution) error {
	err := deployments.SetDeploymentStatus(ctx, w.consulClient.KV(), t.targetID, deployments.DEPLOYMENT_IN_PROGRESS)
	if err != nil {
		return err
	}
	t.finalFunction = w.makeWorkflowFinalFunction(ctx, t.cc.KV(), t.targetID, t.taskID, "install", deployments.DEPLOYED, deployments.DEPLOYMENT_FAILED)

	return w.runWorkflowStep(ctx, t, "install", false)
}

func (w *worker) runUndeploy(ctx context.Context, t *taskExecution) error {
	status, err := deployments.GetDeploymentStatus(w.consulClient.KV(), t.targetID)
	if err != nil {
		return err
	}
	if status != deployments.UNDEPLOYED {
		deployments.SetDeploymentStatus(ctx, w.consulClient.KV(), t.targetID, deployments.UNDEPLOYMENT_IN_PROGRESS)
		t.finalFunction = func() error {
			defer func() {
				// in all cases, if purge has been requested, run it at the end
				if t.taskType == tasks.TaskTypePurge {
					err := w.runPurge(ctx, t)
					if err != nil {
						log.Printf("%+v", err)
					}
				}
			}()
			_, err := updateTaskStatusAccordingToWorkflowStatus(ctx, t.cc.KV(), t.targetID, t.taskID, "uninstall")
			if err != nil {
				return err
			}
			// Set it to undeployed anyway
			return deployments.SetDeploymentStatus(ctx, w.consulClient.KV(), t.targetID, deployments.UNDEPLOYED)
		}
		return w.runWorkflowStep(ctx, t, "uninstall", true)
	} else if t.taskType == tasks.TaskTypePurge {
		return w.runPurge(ctx, t)
	}
	return nil
}

func (w *worker) delayPurge(ctx context.Context, t *taskExecution, taskID string) {
	log.Debugf("let's delay purge as a task is still running:%q", taskID)
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state, err := tasks.GetTaskStatus(t.cc.KV(), taskID)
			if err != nil {
				log.Printf("error occurred during delaying purge:%+v", err)
				return
			}
			if state != tasks.TaskStatusRUNNING {
				log.Debugf("stop delaying purge as task:%q is no longer running", taskID)
				return
			}
		}

		select {
		case <-time.After(30 * time.Second):
			state, err := tasks.GetTaskStatus(t.cc.KV(), taskID)
			if err != nil {
				log.Printf("error occurred during delaying purge:%+v", err)
				return
			}
			if state == tasks.TaskStatusRUNNING {
				log.Debugf("Purge delay timeout is reached: we cancel the task with ID:%q", taskID)
				tasks.CancelTask(t.cc.KV(), taskID)
			}
		}
	}
}

func (w *worker) runPurge(ctx context.Context, t *taskExecution) error {
	kv := w.consulClient.KV()
	tasksList, err := tasks.GetTasksIdsForTarget(kv, t.targetID)
	if err != nil {
		return err
	}
	for _, tid := range tasksList {
		if tid != t.taskID {
			state, _ := tasks.GetTaskStatus(t.cc.KV(), tid)

			if state == tasks.TaskStatusRUNNING {
				w.delayPurge(ctx, t, tid)
			}
			err = tasks.DeleteTask(kv, tid)
			if err != nil {
				return err
			}
		}
		_, err = kv.DeleteTree(path.Join(consulutil.WorkflowsPrefix, tid), nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
	}

	// Delete deployment tree
	_, err = kv.DeleteTree(path.Join(consulutil.DeploymentKVPrefix, t.targetID), nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	// Delete events tree corresponding to the deployment TaskExecution
	err = events.PurgeDeploymentEvents(kv, t.targetID)
	if err != nil {
		return err
	}
	// Delete logs tree corresponding to the deployment
	err = events.PurgeDeploymentLogs(kv, t.targetID)
	if err != nil {
		return err
	}
	overlayPath := filepath.Join(w.cfg.WorkingDirectory, "deployments", t.targetID)
	err = os.RemoveAll(overlayPath)
	if err != nil {
		return errors.Wrapf(err, "failed to remove deployments artifacts stored on disk: %q", overlayPath)
	}
	// Now cleanup: mark it as done so nobody will try to run it, clear the processing lock and finally delete the TaskExecution.
	checkAndSetTaskStatus(ctx, t.cc.KV(), t.targetID, t.taskID, tasks.TaskStatusDONE)
	err = tasks.DeleteTask(kv, t.taskID)
	if err != nil {
		return err
	}
	return deployments.TagDeploymentAsPurged(ctx, t.cc, t.targetID)
}

func (w *worker) runScaleOut(ctx context.Context, t *taskExecution) error {
	status, err := deployments.GetDeploymentStatus(w.consulClient.KV(), t.targetID)
	if err != nil {
		return err
	}
	if status == deployments.DEPLOYED {
		nodeName, err := tasks.GetTaskData(t.cc.KV(), t.taskID, "nodeName")
		if err != nil {
			return errors.Wrap(err, "failed to retrieve scale out node name")
		}

		var instancesDelta int
		instancesDeltaStr, err := tasks.GetTaskData(t.cc.KV(), t.taskID, "instancesDelta")
		if err != nil {
			return errors.Wrap(err, "failed to retrieve scale out instancesDelta")
		}
		if instancesDelta, err = strconv.Atoi(instancesDeltaStr); err != nil {
			return errors.Wrap(err, "failed to convert instancesDelta into int")
		}

		// Create and store related node instances for scaling
		instancesByNodes, err := deployments.CreateNewNodeStackInstances(t.cc.KV(), t.targetID, nodeName, instancesDelta)
		if err != nil {
			return errors.Wrap(err, "failed to create new nodes instances in topology")
		}
		data := make(map[string]string)
		for scalableNode, nodeInstances := range instancesByNodes {
			data[path.Join("nodes", scalableNode)] = nodeInstances
		}
		err = tasks.SetTaskDataList(t.cc.KV(), t.taskID, data)
		if err != nil {
			return errors.Wrap(err, "failed to set task data with nodes for scaling out")
		}

		err = deployments.SetDeploymentStatus(ctx, w.consulClient.KV(), t.targetID, deployments.SCALING_IN_PROGRESS)
		if err != nil {
			return err
		}
	}

	t.finalFunction = w.makeWorkflowFinalFunction(ctx, t.cc.KV(), t.targetID, t.taskID, "install", deployments.DEPLOYED, deployments.DEPLOYMENT_FAILED)
	return w.runWorkflowStep(ctx, t, "install", false)
}

func (w *worker) runScaleIn(ctx context.Context, t *taskExecution) error {
	err := deployments.SetDeploymentStatus(ctx, w.consulClient.KV(), t.targetID, deployments.SCALING_IN_PROGRESS)
	if err != nil {
		return err
	}

	classicFinalFn := w.makeWorkflowFinalFunction(ctx, t.cc.KV(), t.targetID, t.taskID, "uninstall", deployments.DEPLOYED, deployments.DEPLOYMENT_FAILED)
	t.finalFunction = func() error {
		err := w.cleanupScaledDownNodes(t)
		if err != nil {
			return err
		}
		return classicFinalFn()
	}

	return w.runWorkflowStep(ctx, t, "uninstall", true)
}

func (w *worker) runCustomWorkflow(ctx context.Context, t *taskExecution, wfName string) error {
	kv := w.consulClient.KV()
	if wfName == "" {
		return errors.New("workflow name missing")
	}
	continueOnError, err := tasks.GetTaskData(kv, t.taskID, "continueOnError")
	if err != nil {
		return err
	}
	bypassErrors, err := strconv.ParseBool(continueOnError)
	if err != nil {
		return errors.Wrap(err, "failed to parse \"continueOnError\" flag for custom workflow")
	}
	t.finalFunction = func() error {
		_, err := updateTaskStatusAccordingToWorkflowStatus(ctx, t.cc.KV(), t.targetID, t.taskID, wfName)
		return err
	}

	return w.runWorkflowStep(ctx, t, wfName, bypassErrors)
}

// bool return indicates if the workflow is done
func (w *worker) runWorkflowStep(ctx context.Context, t *taskExecution, workflowName string, continueOnError bool) error {
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, t.targetID).RegisterAsString(fmt.Sprintf("Start processing workflow step %s:%s", workflowName, t.step))
	wfSteps, err := builder.BuildWorkFlow(w.consulClient.KV(), t.targetID, workflowName)
	if err != nil {
		return errors.Wrapf(err, "Failed to build step:%q for workflow:%q", t.step, workflowName)
	}
	bs, ok := wfSteps[t.step]
	if !ok {
		return errors.Errorf("Failed to build step: %q for workflow: %q, unknown step", t.step, workflowName)
	}
	s := wrapBuilderStep(bs, w.consulClient, t)
	err = s.run(ctx, w.cfg, w.consulClient.KV(), t.targetID, continueOnError, workflowName, w)
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, t.targetID).RegisterAsString(fmt.Sprintf("Error '%+v' happened in workflow %q.", err, workflowName))
		return errors.Wrapf(err, "The workflow %s step %s ended with error:%+v", workflowName, t.step, err)
	}
	if !s.Async {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, t.targetID).RegisterAsString(fmt.Sprintf("DeploymentID:%q, Workflow:%q, step:%q ended without error", t.targetID, workflowName, t.step))
		return s.registerNextSteps(ctx, workflowName)
	}
	// If we are asynchronous then no the workflow is not done
	return nil
}
