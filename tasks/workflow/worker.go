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

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/helper/metricsutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/prov/operations"
	"github.com/ystia/yorc/prov/scheduling"
	"github.com/ystia/yorc/registry"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tasks/workflow/builder"
	"github.com/ystia/yorc/tosca"
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
				log.Printf("Worker received shutdown signal. Exiting...")
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

func (w *worker) monitorTaskCancellation(ctx context.Context, cancelFunc context.CancelFunc, t *taskExecution, workflowName string) {
	go func() {
		var lastIndex uint64
		for {
			kvp, qMeta, err := w.consulClient.KV().Get(path.Join(consulutil.TasksPrefix, t.taskID, ".canceledFlag"), &api.QueryOptions{WaitIndex: lastIndex})

			select {
			case <-ctx.Done():
				log.Debugln("Task cancellation monitoring exit")
				return
			default:
			}

			if qMeta != nil {
				lastIndex = qMeta.LastIndex
			}

			if err == nil && kvp != nil {
				if strings.ToLower(string(kvp.Value)) == "true" {
					log.Debugln("Task cancellation requested.")
					checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusCANCELED)
					events.PublishAndLogWorkflowStatusChange(ctx, w.consulClient.KV(), t.targetID, t.taskID, workflowName, tasks.TaskStatusCANCELED.String())
					cancelFunc()
					return
				}
			}
		}
	}()
}

// monitorTaskFailure allows to poll task status error flag for a workflow task in order to terminate current step in other one has failed
func (w *worker) monitorTaskFailure(ctx context.Context, cancelFunc context.CancelFunc, t *taskExecution) {
	if t.step != "" {
		go func() {
			var lastIndex uint64
			for {
				kvp, qMeta, err := w.consulClient.KV().Get(path.Join(consulutil.TasksPrefix, t.taskID, ".errorFlag"), &api.QueryOptions{WaitIndex: lastIndex})
				select {
				case <-ctx.Done():
					log.Debugln("Task failure monitoring exit")
					return
				default:
				}

				if qMeta != nil {
					lastIndex = qMeta.LastIndex
				}

				if err == nil && kvp != nil {
					log.Debugln("[MONITOR TASK FAILURE] Task failure has been detected.")
					cancelFunc()
					err = t.deleteTaskErrorFlag(ctx, kvp, lastIndex)
					if err != nil {
						log.Debugf("Error flag couldn't have been deleted due to:%v", err)
					}
					return
				}
			}
		}()
	}
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

func (w *worker) checkAndSetDeploymentStatus(ctx context.Context, deploymentID string, finalStatus deployments.DeploymentStatus) error {
	depStatus, err := deployments.GetDeploymentStatus(w.consulClient.KV(), deploymentID)
	if err != nil {
		return err
	}

	if depStatus == deployments.DEPLOYMENT_FAILED && finalStatus != deployments.DEPLOYMENT_FAILED {
		mess := fmt.Sprintf("Can't set deployment status with deploymentID:%q to %q because status is already:%q", deploymentID, finalStatus.String(), deployments.DEPLOYMENT_FAILED)
		log.Printf(mess)
		return errors.Errorf(mess)
	}

	if finalStatus != depStatus {
		kv := w.consulClient.KV()
		err = consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "status"), finalStatus.String())
		if err != nil {
			return errors.Wrapf(err, "Failed to set deployment status to %q for deploymentID:%q", finalStatus.String(), deploymentID)
		}
		events.PublishAndLogDeploymentStatusChange(ctx, kv, deploymentID, strings.ToLower(finalStatus.String()))
	}
	return nil
}

func (w *worker) setDeploymentStatus(ctx context.Context, deploymentID string, finalStatus deployments.DeploymentStatus) error {
	kv := w.consulClient.KV()
	err := consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "status"), fmt.Sprint(finalStatus))
	if err != nil {
		return errors.Wrapf(err, "Failed to set deployment status to %q for deploymentID:%q", finalStatus.String(), deploymentID)
	}
	events.PublishAndLogDeploymentStatusChange(ctx, kv, deploymentID, strings.ToLower(finalStatus.String()))
	return nil
}

func (w *worker) markExecutionAsProcessing(t *taskExecution) {
	log.Debugf("Mark the task execution ID:%q with taskID:%q, step:%q as processing", t.id, t.taskID, t.step)
	err := consulutil.StoreConsulKey(path.Join(consulutil.ExecutionsTaskPrefix, t.id, ".processing"), nil)
	if err != nil {
		log.Printf("Failed to mark task execution as processing with ID:%q, deploymentID:%q, taskID:%q due to error:%+v", t.id, t.targetID, t.taskID, err)
	}
}

func (w *worker) handleExecution(t *taskExecution) {
	log.Debugf("Handle task execution:%+v", t)
	w.markExecutionAsProcessing(t)
	defer t.releaseLock()

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
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	err := checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusRUNNING)
	if err != nil {
		log.Printf("%+v", err)
		return
	}
	w.monitorTaskCancellation(ctx, cancelFunc, t, wfName)
	w.monitorTaskFailure(ctx, cancelFunc, t)
	defer func(t *taskExecution, start time.Time) {
		if taskStatus, err := t.getTaskStatus(); err != nil && taskStatus != tasks.TaskStatusRUNNING {
			metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"task", t.targetID, t.taskType.String(), taskStatus.String()}), 1)
			metrics.MeasureSince(metricsutil.CleanupMetricKey([]string{"task", t.targetID, t.taskType.String()}), start)
		}
	}(t, time.Now())
	switch t.taskType {
	case tasks.TaskTypeDeploy:
		w.runDeploy(ctx, t)
	case tasks.TaskTypeUnDeploy, tasks.TaskTypePurge:
		w.runUndeploy(ctx, t)
	case tasks.TaskTypeCustomCommand:
		err := w.runCustomCommand(ctx, t)
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to get Custom command name: %+v", t.targetID, t.taskID, err)
			checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
			events.PublishAndLogCustomCommandStatusChange(ctx, t.kv, t.targetID, t.taskID, tasks.TaskStatusFAILED.String())
			return
		}
		events.PublishAndLogCustomCommandStatusChange(ctx, t.kv, t.targetID, t.taskID, tasks.TaskStatusDONE.String())
	case tasks.TaskTypeScaleOut:
		w.runScaleOut(ctx, t)
	case tasks.TaskTypeScaleIn:
		w.runScaleIn(ctx, t)
	case tasks.TaskTypeCustomWorkflow:
		w.runCustomWorkflow(ctx, t, wfName)
	case tasks.TaskTypeQuery:
		w.runQuery(ctx, t)
	case tasks.TaskTypeAction:
		w.runAction(ctx, t)
	default:
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, t.targetID).RegisterAsString(fmt.Sprintf("Unknown TaskType %d (%s) for TaskExecution with id %q", t.taskType, t.taskType.String(), t.taskID))
		log.Printf("Unknown TaskType %d (%s) for TaskExecution with id %q and targetId %q", t.taskType, t.taskType.String(), t.taskID, t.targetID)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
	}
}

func (w *worker) runCustomCommand(ctx context.Context, t *taskExecution) error {
	kv := w.consulClient.KV()
	commandName, err := tasks.GetTaskData(kv, t.taskID, "commandName")
	if err != nil {
		return err
	}
	interfaceNameKv, _, err := kv.Get(path.Join(consulutil.TasksPrefix, t.taskID, "interfaceName"), nil)
	if err != nil {
		return err
	}
	nodes, err := tasks.GetTaskRelatedNodes(kv, t.taskID)
	if err != nil {
		return err
	}
	if len(nodes) != 1 {
		return errors.Errorf("Deployment id: %q, Task id: %q, Expecting custom command TaskExecution to be related to \"1\" node while it is actually related to \"%d\" nodes", t.targetID, t.taskID, len(nodes))
	}
	nodeName := nodes[0]
	interfaceName := "custom"
	if interfaceNameKv != nil && len(interfaceNameKv.Value) != 0 {
		interfaceName = string(interfaceNameKv.Value)
	}
	nodeType, err := deployments.GetNodeType(w.consulClient.KV(), t.targetID, nodeName)
	if err != nil {
		return err
	}
	op, err := operations.GetOperation(ctx, kv, t.targetID, nodeName, interfaceName+"."+commandName, "", "")
	if err != nil {
		err = setNodeStatus(ctx, t.kv, t.taskID, t.targetID, nodeName, tosca.NodeStateError.String())
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to set status for node %q: %+v", t.targetID, t.taskID, nodeName, err)
		}
		return errors.Wrapf(err, "Command TaskExecution failed for node %q", nodeName)
	}
	exec, err := getOperationExecutor(kv, t.targetID, op.ImplementationArtifact)
	if err != nil {
		err = setNodeStatus(ctx, t.kv, t.taskID, t.targetID, nodeName, tosca.NodeStateError.String())
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to set status for node %q: %+v", t.targetID, t.taskID, nodeName, err)
		}
		return errors.Wrapf(err, "Command TaskExecution failed for node %q", nodeName)
	}

	ctx = operations.SetOperationLogFields(ctx, op)
	ctx = events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.NodeID: nodeName})

	err = func() error {
		defer metrics.MeasureSince(metricsutil.CleanupMetricKey([]string{"executor", "operation", t.targetID, nodeType, op.Name}), time.Now())
		return exec.ExecOperation(ctx, w.cfg, t.taskID, t.targetID, nodeName, op)
	}()
	if err != nil {
		metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "operation", t.targetID, nodeType, op.Name, "failures"}), 1)
		err = setNodeStatus(ctx, t.kv, t.taskID, t.targetID, nodeName, tosca.NodeStateError.String())
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to set status for node %q: %+v", t.targetID, t.taskID, nodeName, err)
		}
		return errors.Wrapf(err, "Command TaskExecution failed for node %q", nodeName)
	}
	checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusDONE)
	metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "operation", t.targetID, nodeType, op.Name, "successes"}), 1)
	return nil
}

func (w *worker) endAction(ctx context.Context, action *prov.Action, actionErr error) {
	if action.AsyncOperation.TaskID == "" {
		return
	}
	step := &tasks.TaskStep{Name: action.AsyncOperation.StepName, Status: "DONE"}
	endMsg := "operation succeeded"
	if actionErr != nil {
		step.Status = "ERROR"
		endMsg = "operation failed"
	}
	err := tasks.UpdateTaskStepStatus(w.consulClient.KV(), action.AsyncOperation.TaskID, step)
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
		// TODO: replace this with workflow steps events
		events.WithContextOptionalFields(events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.InstanceID: instanceName})).NewLogEntry(events.LogLevelDEBUG, action.AsyncOperation.DeploymentID).RegisterAsString(endMsg)
	}

	if actionErr != nil {
		checkAndSetTaskStatus(ctx, w.consulClient.KV(), action.AsyncOperation.TaskID, action.AsyncOperation.StepName, tasks.TaskStatusFAILED)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, action.AsyncOperation.DeploymentID).Registerf("%v", actionErr)
	} else {
		// we are not stopped or erroed we just have to reschedule next steps
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

		// This is a fake task execution as the original one was deleted
		t, err := buildTaskExecution(w.consulClient.KV(), action.AsyncOperation.ExecutionID)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, action.AsyncOperation.DeploymentID).Registerf("Failed to register steps preceded by %q for execution: %v", action.AsyncOperation.StepName, err)
			log.Debugf("%+v", err)
			return
		}
		s := wrapBuilderStep(bs, w.consulClient.KV(), t)
		_, err = w.registerNextSteps(ctx, s, t, action.AsyncOperation.WorkflowName)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, action.AsyncOperation.DeploymentID).Registerf("Failed to register steps preceded by %q for execution: %v", action.AsyncOperation.StepName, err)
			log.Debugf("%+v", err)
			return
		}
	}
}

func (w *worker) runAction(ctx context.Context, t *taskExecution) {
	// Delete task at the end of it execution
	defer tasks.DeleteTask(w.consulClient.KV(), t.taskID)
	action := &prov.Action{}
	var err error
	action.Data, err = tasks.GetAllTaskData(w.consulClient.KV(), t.taskID)
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q Failed: %+v", t.targetID, t.taskID, err)
		log.Debugf("%+v", err)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}
	action.ID = action.Data["id"]
	action.ActionType = action.Data["actionType"]
	err = json.Unmarshal([]byte(action.Data["asyncOperation"]), &action.AsyncOperation)
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q Failed: %+v", t.targetID, t.taskID, err)
		log.Debugf("%+v", err)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	} // Find an actionOperator which match with this actionType
	var reg = registry.GetRegistry()
	operator, err := reg.GetActionOperator(action.ActionType)
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, t.targetID).
			Registerf("Action Task id: %q Failed to find matching operator: %+v", t.taskID, err)
		log.Debugf("%+v", err)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}

	if action.AsyncOperation.TaskID != "" {
		ctx = operations.SetOperationLogFields(ctx, action.AsyncOperation.Operation)
		ctx = events.AddLogOptionalFields(ctx, events.LogOptionalFields{
			events.ExecutionID: action.AsyncOperation.TaskID,
			events.WorkFlowID:  action.AsyncOperation.WorkflowName,
			events.NodeID:      action.AsyncOperation.NodeName,
		})
	}

	deregister, err := operator.ExecAction(ctx, w.cfg, t.taskID, t.targetID, action)
	if deregister {
		scheduling.UnregisterAction(w.consulClient, action.ID)
		w.endAction(ctx, action, err)
	}
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, t.targetID).Registerf("Action Task id: %q Failed to run action: %v", t.taskID, err)
		log.Debugf("%+v", err)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}
	checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusDONE)
	log.Printf("Action:%+v successfully executed", action)
}

func (w *worker) runQuery(ctx context.Context, t *taskExecution) {
	split := strings.Split(t.targetID, ":")
	if len(split) != 2 {
		log.Printf("Query Task (id: %q): unexpected format for targetID: %q", t.taskID, t.targetID)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}
	query := split[0]
	target := split[1]

	switch query {
	case "infra_usage":
		var reg = registry.GetRegistry()
		collector, err := reg.GetInfraUsageCollector(target)
		if err != nil {
			log.Printf("Query Task id: %q Failed to retrieve target type: %+v", t.taskID, err)
			log.Debugf("%+v", err)
			checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
			return
		}
		res, err := collector.GetUsageInfo(ctx, w.cfg, t.taskID, target)
		if err != nil {
			log.Printf("Query Task id: %q Failed to run query: %+v", t.taskID, err)
			log.Debugf("%+v", err)
			checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
			return
		}

		// store resultSet as a JSON
		resultPrefix := path.Join(consulutil.TasksPrefix, t.taskID, "resultSet")
		if res != nil {
			jsonRes, err := json.Marshal(res)
			if err != nil {
				log.Printf("Failed to marshal infra usage info [%+v]: due to error:%+v", res, err)
				log.Debugf("%+v", err)
				checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
				return
			}
			err = consulutil.StoreConsulKey(resultPrefix, jsonRes)
			if err != nil {
				log.Printf("Query Task id: %q Failed to store result: %+v", t.taskID, errors.Wrap(err, consulutil.ConsulGenericErrMsg))
				log.Debugf("%+v", err)
				checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
				return
			}
		}
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusDONE)
	default:
		mess := fmt.Sprintf("Unknown query: %q for Task with id %q", query, t.taskID)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, t.targetID).RegisterAsString(mess)
		log.Printf(mess)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}
}

func (w *worker) runDeploy(ctx context.Context, t *taskExecution) {
	err := w.checkAndSetDeploymentStatus(ctx, t.targetID, deployments.DEPLOYMENT_IN_PROGRESS)
	if err != nil {
		log.Printf("%+v", err)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}
	wfDone, err := w.runWorkflowStep(ctx, t, "install", false)
	if err != nil {
		log.Printf("%+v", err)
		w.checkAndSetDeploymentStatus(ctx, t.targetID, deployments.DEPLOYMENT_FAILED)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		events.PublishAndLogWorkflowStatusChange(ctx, w.consulClient.KV(), t.targetID, t.taskID, "install", tasks.TaskStatusFAILED.String())
		return
	}
	if wfDone {
		err = w.checkAndSetDeploymentStatus(ctx, t.targetID, deployments.DEPLOYED)
		if err != nil {
			checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
			log.Printf("%+v", err)
			return
		}
		events.PublishAndLogWorkflowStatusChange(ctx, w.consulClient.KV(), t.targetID, t.taskID, "install", tasks.TaskStatusDONE.String())
	}
}

func (w *worker) runUndeploy(ctx context.Context, t *taskExecution) {
	status, err := deployments.GetDeploymentStatus(w.consulClient.KV(), t.targetID)
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q, Failed to get deployment status: %+v", t.targetID, t.taskID, err)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		events.PublishAndLogWorkflowStatusChange(ctx, w.consulClient.KV(), t.targetID, t.taskID, "uninstall", tasks.TaskStatusFAILED.String())
		return
	}
	if status != deployments.UNDEPLOYED {
		if status != deployments.UNDEPLOYMENT_IN_PROGRESS {
			w.setDeploymentStatus(ctx, t.targetID, deployments.UNDEPLOYMENT_IN_PROGRESS)
		}

		wfDone, err := w.runWorkflowStep(ctx, t, "uninstall", true)
		if err != nil {
			log.Printf("%+v", err)
			w.checkAndSetDeploymentStatus(ctx, t.targetID, deployments.UNDEPLOYMENT_FAILED)
			checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
			events.PublishAndLogWorkflowStatusChange(ctx, w.consulClient.KV(), t.targetID, t.taskID, "uninstall", tasks.TaskStatusFAILED.String())
			return
		}
		if wfDone {
			err = w.checkAndSetDeploymentStatus(ctx, t.targetID, deployments.UNDEPLOYED)
			if err != nil {
				checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
				log.Printf("%+v", err)
				return
			}
			if t.taskType == tasks.TaskTypePurge {
				w.runPurge(ctx, t)
			}
			events.PublishAndLogWorkflowStatusChange(ctx, w.consulClient.KV(), t.targetID, t.taskID, "uninstall", tasks.TaskStatusDONE.String())
		}
	} else if t.taskType == tasks.TaskTypePurge {
		w.runPurge(ctx, t)
	}
}

func (w *worker) runPurge(ctx context.Context, t *taskExecution) {
	kv := w.consulClient.KV()
	_, err := kv.DeleteTree(path.Join(consulutil.DeploymentKVPrefix, t.targetID), nil)
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q, Failed to purge deployment definition: %+v", t.targetID, t.taskID, err)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}
	tasksList, err := tasks.GetTasksIdsForTarget(kv, t.targetID)
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.targetID, t.taskID, err)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}
	for _, tid := range tasksList {
		if tid != t.taskID {
			err = tasks.DeleteTask(kv, tid)
			if err != nil {
				log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.targetID, t.taskID, err)
				checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
				return
			}
		}
		_, err = kv.DeleteTree(path.Join(consulutil.WorkflowsPrefix, tid), nil)
		if err != nil {
			log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.targetID, t.taskID, err)
			checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
			return
		}
	}
	// Delete events tree corresponding to the deployment TaskExecution
	_, err = kv.DeleteTree(path.Join(consulutil.EventsPrefix, t.targetID), nil)
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q, Failed to purge events: %+v", t.targetID, t.taskID, err)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}
	// Delete logs tree corresponding to the deployment
	_, err = kv.DeleteTree(path.Join(consulutil.LogsPrefix, t.targetID), nil)
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q, Failed to purge logs: %+v", t.targetID, t.taskID, err)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}
	err = os.RemoveAll(filepath.Join(w.cfg.WorkingDirectory, "deployments", t.targetID))
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.targetID, t.taskID, err)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}
	// Now cleanup: mark it as done so nobody will try to run it, clear the processing lock and finally delete the TaskExecution.
	checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusDONE)
	err = tasks.DeleteTask(kv, t.taskID)
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q, Failed to purge tasks related to deployment: %+v", t.targetID, t.taskID, err)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}
	return
}

func (w *worker) runScaleOut(ctx context.Context, t *taskExecution) {
	err := w.checkAndSetDeploymentStatus(ctx, t.targetID, deployments.SCALING_IN_PROGRESS)
	if err != nil {
		log.Printf("%+v", err)
		return
	}
	wfDone, err := w.runWorkflowStep(ctx, t, "install", false)
	if err != nil {
		log.Printf("%+v", err)
		w.checkAndSetDeploymentStatus(ctx, t.targetID, deployments.DEPLOYMENT_FAILED)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}
	if wfDone {
		err = w.checkAndSetDeploymentStatus(ctx, t.targetID, deployments.DEPLOYED)
		if err != nil {
			log.Printf("%+v", err)
			checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
			return
		}
	}
}

func (w *worker) runScaleIn(ctx context.Context, t *taskExecution) {
	err := w.checkAndSetDeploymentStatus(ctx, t.targetID, deployments.SCALING_IN_PROGRESS)
	if err != nil {
		log.Printf("%+v", err)
		return
	}
	wfDone, err := w.runWorkflowStep(ctx, t, "uninstall", true)
	if err != nil {
		log.Printf("%+v", err)
		w.checkAndSetDeploymentStatus(ctx, t.targetID, deployments.DEPLOYMENT_FAILED)
		return
	}
	if wfDone {
		// Cleanup
		if err = w.cleanupScaledDownNodes(t); err != nil {
			checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
			log.Printf("%+v. Aborting", err)
			w.checkAndSetDeploymentStatus(ctx, t.targetID, deployments.DEPLOYMENT_FAILED)
			return
		}
		err = w.checkAndSetDeploymentStatus(ctx, t.targetID, deployments.DEPLOYED)
		if err != nil {
			log.Printf("%+v", err)
			return
		}
	}
}

func (w *worker) runCustomWorkflow(ctx context.Context, t *taskExecution, wfName string) {
	kv := w.consulClient.KV()
	if wfName == "" {
		log.Printf("Deployment id: %q, Task id: %q Failed: workflow name missing", t.targetID, t.taskID)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}
	continueOnError, err := tasks.GetTaskData(kv, t.taskID, "continueOnError")
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q Failed: %+v", t.targetID, t.taskID, err)
		log.Debugf("%+v", err)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}
	bypassErrors, err := strconv.ParseBool(continueOnError)
	if err != nil {
		log.Printf("Deployment id: %q, Task id: %q Failed to parse continueOnError parameter: %+v", t.targetID, t.taskID, err)
		log.Debugf("%+v", err)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return
	}
	wfDone, err := w.runWorkflowStep(ctx, t, wfName, bypassErrors)
	if err != nil {
		log.Printf("%+v", err)
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		events.PublishAndLogWorkflowStatusChange(ctx, w.consulClient.KV(), t.targetID, t.taskID, wfName, tasks.TaskStatusFAILED.String())
		return
	}
	if wfDone {
		events.PublishAndLogWorkflowStatusChange(ctx, w.consulClient.KV(), t.targetID, t.taskID, wfName, tasks.TaskStatusDONE.String())
	}
}

func (w *worker) runWorkflowStep(ctx context.Context, t *taskExecution, workflowName string, continueOnError bool) (bool, error) {
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, t.targetID).RegisterAsString(fmt.Sprintf("Start processing workflow step %s:%s", workflowName, t.step))
	wfSteps, err := builder.BuildWorkFlow(w.consulClient.KV(), t.targetID, workflowName)
	if err != nil {
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return false, errors.Wrapf(err, "Failed to build step:%q for workflow:%q", t.step, workflowName)
	}
	bs, ok := wfSteps[t.step]
	if !ok {
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		return false, errors.Errorf("Failed to build step: %q for workflow: %q, unknown step", t.step, workflowName)
	}
	s := wrapBuilderStep(bs, w.consulClient.KV(), t)
	err = s.run(ctx, w.cfg, w.consulClient.KV(), t.targetID, continueOnError, workflowName, w)
	if err != nil {
		checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusFAILED)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, t.targetID).RegisterAsString(fmt.Sprintf("Error '%+v' happened in workflow %q.", err, workflowName))
		return false, errors.Wrapf(err, "The workflow %s step %s ended with error:%+v", workflowName, t.step, err)
	}
	if !s.Async {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, t.targetID).RegisterAsString(fmt.Sprintf("DeploymentID:%q, Workflow:%q, step:%q ended without error", t.targetID, workflowName, t.step))
		return w.registerNextSteps(ctx, s, t, workflowName)
	}
	return false, nil
}

func (w *worker) registerNextSteps(ctx context.Context, s *step, t *taskExecution, workflowName string) (bool, error) {
	// If step is terminal, we check if workflow is done
	if s.IsTerminal() {
		log.Debugf("Step:%q is terminal: check if workflow %q is done", s.Name, s.WorkflowName)
		return w.checkIfWorkflowIsDone(ctx, s.t, workflowName)
	}

	// Register workflow step to handle step statuses for next steps
	regSteps := make([]*step, 0)
	for _, bnStep := range s.Next {
		nStep := wrapBuilderStep(bnStep, w.consulClient.KV(), t)
		// In case of join, check each previous status step
		if len(nStep.Previous) > 1 {
			done, err := w.checkIfPreviousOfNextStepAreDone(ctx, nStep, t, workflowName)
			if err != nil {
				return false, err
			}
			if done {
				regSteps = append(regSteps, nStep)
			}
		} else {
			regSteps = append(regSteps, nStep)
		}
	}
	ops := createWorkflowStepsOperations(s.t.taskID, regSteps)
	ok, response, _, err := w.consulClient.KV().Txn(ops, nil)
	if err != nil {
		return false, errors.Wrapf(err, "Failed to register executionTasks with TaskID:%q", s.t.taskID)
	}
	if !ok {
		errs := make([]string, 0)
		for _, e := range response.Errors {
			errs = append(errs, e.What)
		}
		return false, errors.Wrapf(err, "Failed to register executionTasks with TaskID:%q due to:%s", s.t.taskID, strings.Join(errs, ", "))
	}
	return false, nil
}

func (w *worker) checkIfPreviousOfNextStepAreDone(ctx context.Context, s *step, t *taskExecution, workflowName string) (bool, error) {
	cpt := 0
	for _, step := range s.Previous {
		stepStatus, err := tasks.GetTaskStepStatus(w.consulClient.KV(), t.taskID, step.Name)
		if err != nil {
			return false, errors.Wrapf(err, "Failed to retrieve step status with TaskID:%q, step:%q", t.taskID, step.Name)
		}
		if stepStatus == tasks.TaskStepStatusDONE {
			cpt++
		} else if stepStatus == tasks.TaskStepStatusCANCELED || stepStatus == tasks.TaskStepStatusERROR {
			return false, errors.Errorf("An error has been detected on other step:%q for workflow:%q, deploymentID:%q, taskID:%q. No more steps will be executed", step.Name, workflowName, t.targetID, t.taskID)
		}
	}

	// In case of workflow join, the last done of previous steps will register the join step
	if len(s.Previous) == cpt {
		log.Debugf("All previous steps of step:%q are done, so it can be registered to be executed", s.Name)
		return true, nil
	}
	return false, nil
}

func (w *worker) checkIfWorkflowIsDone(ctx context.Context, t *taskExecution, workflowName string) (bool, error) {
	taskSteps, err := tasks.GetTaskRelatedSteps(w.consulClient.KV(), t.taskID)
	if err != nil {
		return false, errors.Wrapf(err, "Failed to retrieve workflow step statuses with TaskID:%q", t.taskID)
	}
	cpt := 0
	for _, step := range taskSteps {
		stepStatus, err := tasks.ParseTaskStepStatus(step.Status)
		if err != nil {
			return false, errors.Wrapf(err, "Failed to retrieve workflow step statuses with TaskID:%q", t.taskID)
		}
		if stepStatus == tasks.TaskStepStatusDONE {
			cpt++
		} else if stepStatus == tasks.TaskStepStatusCANCELED || stepStatus == tasks.TaskStepStatusERROR {
			return false, errors.Errorf("An error has been detected on other step:%q for workflow:%q, deploymentID:%q. No more steps will be executed", t.step, workflowName, t.targetID)
		}
	}
	if len(taskSteps) == cpt {
		log.Debugf("All steps of workflow:%q are done, so workflow is done", workflowName)
		err = checkAndSetTaskStatus(ctx, t.kv, t.taskID, t.step, tasks.TaskStatusDONE)
		if err != nil {
			return false, errors.Wrapf(err, "Failed to update task status to DONE with TaskID:%q due to error:%+v", t.taskID, err)
		}
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, t.targetID).RegisterAsString(fmt.Sprintf("Workflow %q ended without error", workflowName))
		return true, nil
	}
	log.Debugf("Still some workflow steps need to be run for workflow:%q ", workflowName)
	return false, nil
}

func (w *worker) registerInlineWorkflow(ctx context.Context, t *taskExecution, workflowName string) error {
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, t.targetID).RegisterAsString(fmt.Sprintf("Register workflow %q from taskID:%q, deploymentID:%q", workflowName, t.taskID, t.targetID))
	wfOps, err := builder.BuildInitExecutionOperations(t.kv, t.targetID, t.taskID, workflowName, true)
	if err != nil {
		return err
	}
	ok, response, _, err := t.kv.Txn(wfOps, nil)
	if err != nil {
		return errors.Wrapf(err, "Failed to register workflow init operations with workflow:%q, targetID:%q, taskID:%q", workflowName, t.targetID, t.taskID)
	}
	if !ok {
		errs := make([]string, 0)
		for _, e := range response.Errors {
			errs = append(errs, e.What)
		}
		return errors.Wrapf(err, "Failed to register workflow init operations with workflow:%q, targetID:%q, taskID:%q due to error:%q", workflowName, t.targetID, t.taskID, strings.Join(errs, ", "))
	}

	return nil
}
