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
	"strings"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/helper/metricsutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov/operations"
	"github.com/ystia/yorc/v4/prov/scheduling"
	"github.com/ystia/yorc/v4/registry"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tasks/collector"
	"github.com/ystia/yorc/v4/tasks/workflow/builder"
	"github.com/ystia/yorc/v4/tosca"
)

// step represents the workflow step
type step struct {
	*builder.Step
	cc *api.Client
	t  *taskExecution
}

func wrapBuilderStep(s *builder.Step, cc *api.Client, t *taskExecution) *step {
	return &step{Step: s, cc: cc, t: t}
}

func (s *step) wrapBuilderStep(bs *builder.Step) *step {
	return wrapBuilderStep(bs, s.cc, s.t)
}

func (s *step) setStatus(status tasks.TaskStepStatus) error {
	return tasks.UpdateTaskStepWithStatus(s.t.taskID, s.Name, status)
}

func (s *step) cancelNextSteps() {
	for _, ns := range s.Next {
		log.Debugf("cancel step name:%q", ns.Name)
		// bind next canceled execution to actual one
		sns := s.wrapBuilderStep(ns)
		sns.setStatus(tasks.TaskStepStatusCANCELED)
		sns.cancelNextSteps()
	}
}

func isTargetOperationOnSource(s *step) bool {
	if strings.ToUpper(s.OperationHost) != "SOURCE" {
		return false
	}
	for _, o := range getCallOperationsFromStep(s) {
		if strings.Contains(o, "add_target") || strings.Contains(o, "remove_target") || strings.Contains(o, "target_changed") {
			return true
		}
	}
	return false
}

func isSourceOperationOnTarget(s *step) bool {
	if strings.ToUpper(s.OperationHost) != "TARGET" {
		return false
	}
	for _, o := range getCallOperationsFromStep(s) {
		if strings.Contains(o, "add_source") || strings.Contains(o, "remove_source") || strings.Contains(o, "source_changed") {
			return true
		}
	}
	return false
}

// isRunnable Checks if a Step should be run or bypassed
//
// It first checks if the Step is not already done in this workflow instance
// And for ScaleOut and ScaleDown it checks if the node or the target node in case of an operation running on the target node is part of the operation
func (s *step) isRunnable(ctx context.Context) (bool, error) {
	kv := s.cc.KV()
	kvp, _, err := kv.Get(path.Join(consulutil.WorkflowsPrefix, s.t.taskID, s.Name), nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	// Set default task step status to Initial if not set
	var status string
	if kvp != nil && string(kvp.Value) != "" {
		status = string(kvp.Value)
	} else {
		status = tasks.TaskStepStatusINITIAL.String()
	}

	// Check if Step is already done
	if status != "" {
		stepStatus, err := tasks.ParseTaskStepStatus(status)
		if err != nil {
			return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}

		if kvp != nil && stepStatus == tasks.TaskStepStatusDONE {
			return false, nil
		}
	}

	if s.t.taskType == tasks.TaskTypeScaleOut || s.t.taskType == tasks.TaskTypeScaleIn || s.t.taskType == tasks.TaskTypeAddNodes || s.t.taskType == tasks.TaskTypeRemoveNodes {
		// If not a relationship check the actual node
		if s.TargetRelationship == "" {
			return tasks.IsTaskRelatedNode(s.t.taskID, s.Target)
		}

		if isSourceOperationOnTarget(s) {
			// operation on target but Check if Source is implied on scale
			return tasks.IsTaskRelatedNode(s.t.taskID, s.Target)
		}

		if isTargetOperationOnSource(s) || strings.ToUpper(s.OperationHost) == "TARGET" {
			// Check if Target is implied on scale
			targetReqIndex, err := deployments.GetRequirementIndexByNameForNode(ctx, s.t.targetID, s.Target, s.TargetRelationship)
			if err != nil {
				return false, err
			}
			targetNodeName, err := deployments.GetTargetNodeForRequirement(ctx, s.t.targetID, s.Target, targetReqIndex)
			if err != nil {
				return false, err
			}
			return tasks.IsTaskRelatedNode(s.t.taskID, targetNodeName)
		}

		// otherwise check the actual node is implied
		return tasks.IsTaskRelatedNode(s.t.taskID, s.Target)

	}

	return true, nil
}

// run allows to execute a workflow step
func (s *step) run(ctx context.Context, cfg config.Configuration, deploymentID string, bypassErrors bool, workflowName string, w *worker) error {
	// Fill log optional fields for log registration
	ctx = events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.WorkFlowID: workflowName, events.NodeID: s.Target, events.TaskExecutionID: s.t.id})
	// First: we check if Step is runnable
	if runnable, err := s.isRunnable(ctx); err != nil {
		return err
	} else if !runnable {
		log.Debugf("Deployment %q: Skipping TaskStep %q", deploymentID, s.Name)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(fmt.Sprintf("Skipping TaskStep %q", s.Name))
		s.setStatus(tasks.TaskStepStatusDONE)
		return nil
	}
	s.setStatus(tasks.TaskStepStatusRUNNING)

	ctx, cancelWf := context.WithCancel(ctx)
	defer cancelWf()
	if !s.IsOnCancelPath {
		tasks.MonitorTaskCancellation(ctx, s.t.taskID, func() {
			s.setStatus(tasks.TaskStepStatusCANCELED)
			cancelWf()
			err := s.registerOnCancelOrFailureSteps(ctx, workflowName, s.OnCancel)
			if err != nil {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, deploymentID).Registerf("failed to register cancellation steps: %v", err)
			}
		})
	}

	if !s.IsOnFailurePath {
		tasks.MonitorTaskFailure(ctx, s.t.taskID, func() {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("An error occurred on another step while step %q is running: trying to gracefully finish it.", s.Name)
			select {
			case <-time.After(cfg.WfStepGracefulTerminationTimeout):
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("Step %q not yet finished: we set it on error", s.Name)
				s.setStatus(tasks.TaskStepStatusERROR)
				cancelWf()
			case <-ctx.Done():
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("Step %q finished before the end of the graceful period.", s.Name)
			}
		})
	}

	log.Debugf("Processing Step %q", s.Name)
	for _, activity := range s.Activities {
		err := func() error {
			for _, hook := range preActivityHooks {
				hook(ctx, cfg, s.t.taskID, deploymentID, s.Target, activity)
			}
			defer func() {
				for _, hook := range postActivityHooks {
					hook(ctx, cfg, s.t.taskID, deploymentID, s.Target, activity)
				}
			}()
			err := s.runActivity(ctx, cfg, deploymentID, workflowName, bypassErrors, w, activity)
			if err != nil {
				setNodeStatus(ctx, s.t.taskID, deploymentID, s.Target, tosca.NodeStateError.String())
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("TaskStep %q: error details: %+v", s.Name, err)
				// Set step in error but continue if needed
				s.setStatus(tasks.TaskStepStatusERROR)
				if !bypassErrors {
					if len(s.OnFailure) > 0 {
						// Do not end the workflow as there is a on failure branch
						events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).Registerf(
							"TaskStep %q had error: %+v, workflow continues with on failure step", s.Name, err)
					} else {
						tasks.NotifyErrorOnTask(s.t.taskID)
					}
					// only set generic error message here.
					// Task status is handled in task execution final function
					tasks.CheckAndSetTaskErrorMessage(s.t.taskID, fmt.Sprintf("Workflow %q step %q failed.", workflowName, s.Name), false)

					err2 := s.registerOnCancelOrFailureSteps(ctx, workflowName, s.OnFailure)
					if err2 != nil {
						events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, deploymentID).Registerf("failed to register on failure steps: %v", err2)
					}
					return err
				}
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).Registerf("TaskStep %q: Bypassing error: %+v but workflow continue", s.Name, err)
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	if !s.Async {
		log.Debugf("Task execution: %q for step: %q, workflow: %q, taskID: %q done successfully.", s.t.id, s.Name, s.WorkflowName, s.t.taskID)
		s.setStatus(tasks.TaskStepStatusDONE)
	}
	return nil
}

func (s *step) runActivity(wfCtx context.Context, cfg config.Configuration, deploymentID, workflowName string, bypassErrors bool, w *worker, activity builder.Activity) error {
	// Get activity related instances
	instances, err := tasks.GetInstances(wfCtx, s.t.taskID, deploymentID, s.Target)
	if err != nil {
		return err
	}

	eventInfo := &events.WorkflowStepInfo{WorkflowName: workflowName, NodeName: s.Target, StepName: s.Name}
	switch activity.Type() {
	case builder.ActivityTypeDelegate:
		nodeType, err := deployments.GetNodeType(wfCtx, deploymentID, s.Target)
		if err != nil {
			return err
		}
		provisioner, err := registry.GetRegistry().GetDelegateExecutor(nodeType)
		if err != nil {
			return err
		}
		delegateOp := activity.Value()
		wfCtx = events.AddLogOptionalFields(wfCtx, events.LogOptionalFields{events.InterfaceName: "delegate", events.OperationName: delegateOp})
		for _, instanceName := range instances {
			eventInfo.OperationName = fmt.Sprintf("delegate.%s", delegateOp)
			// Need to publish INITIAL status before RUNNING one with operationName set
			s.publishInstanceRelatedEvents(wfCtx, deploymentID, instanceName, eventInfo, tasks.TaskStepStatusINITIAL, tasks.TaskStepStatusRUNNING)
		}

		executorDelegateLabels := []metrics.Label{
			metrics.Label{Name: "Deployment", Value: deploymentID},
			metrics.Label{Name: "Name", Value: delegateOp},
			metrics.Label{Name: "Node", Value: nodeType},
		}

		err = func() error {
			defer metrics.MeasureSinceWithLabels(metricsutil.CleanupMetricKey([]string{"executor", "delegate", "duration"}), time.Now(), executorDelegateLabels)
			return provisioner.ExecDelegate(wfCtx, cfg, s.t.taskID, deploymentID, s.Target, delegateOp)
		}()

		if err != nil {
			metrics.IncrCounterWithLabels(metricsutil.CleanupMetricKey([]string{"executor", "delegate", "failures"}), 1, executorDelegateLabels)
			for _, instanceName := range instances {
				s.publishInstanceRelatedEvents(wfCtx, deploymentID, instanceName, eventInfo, tasks.TaskStepStatusERROR)
			}
			return err
		}
		metrics.IncrCounterWithLabels(metricsutil.CleanupMetricKey([]string{"executor", "delegate", "successes"}), 1, executorDelegateLabels)
		for _, instanceName := range instances {
			s.publishInstanceRelatedEvents(wfCtx, deploymentID, instanceName, eventInfo, tasks.TaskStepStatusDONE)
		}
	case builder.ActivityTypeSetState:
		setNodeStatus(wfCtx, s.t.taskID, deploymentID, s.Target, activity.Value())
	case builder.ActivityTypeCallOperation:
		inputParameters, err := s.getActivityInputParameters(wfCtx, activity, deploymentID, workflowName)
		if err != nil {
			return err
		}
		op, err := operations.GetOperation(wfCtx, s.t.targetID, s.Target, activity.Value(), s.TargetRelationship, s.OperationHost, inputParameters)
		if err != nil {
			if deployments.IsOperationNotImplemented(err) {
				// Operation not implemented just skip it
				log.Debugf("Voluntary bypassing error: %s.", err.Error())
				return nil
			}
			return err
		}

		exec, err := getOperationExecutor(wfCtx, deploymentID, op.ImplementationArtifact)
		if err != nil {
			return err
		}
		nodeType, err := deployments.GetNodeType(wfCtx, deploymentID, s.Target)
		if err != nil {
			return err
		}
		wfCtx = operations.SetOperationLogFields(wfCtx, op)
		for _, instanceName := range instances {
			// Check for specific info about relationships
			eventInfo.OperationName = op.Name
			if op.RelOp.IsRelationshipOperation {
				eventInfo.TargetNodeID = op.RelOp.TargetNodeName
			}
			// Need to publish INITIAL status before RUNNING one with operationName set
			s.publishInstanceRelatedEvents(wfCtx, deploymentID, instanceName, eventInfo, tasks.TaskStepStatusINITIAL, tasks.TaskStepStatusRUNNING)
		}

		executorOperationLabels := []metrics.Label{
			metrics.Label{Name: "Deployment", Value: deploymentID},
			metrics.Label{Name: "Name", Value: op.Name},
			metrics.Label{Name: "Node", Value: nodeType},
		}
		// In function of the operation, the execution is sync or async
		if s.Async {
			err = func() error {
				defer metrics.MeasureSinceWithLabels(metricsutil.CleanupMetricKey([]string{"executor", "operation", "duration"}), time.Now(), executorOperationLabels)
				action, timeInterval, err := exec.ExecAsyncOperation(wfCtx, cfg, s.t.taskID, deploymentID, s.Target, op, s.Name)
				if err != nil {
					return err
				}
				action.AsyncOperation.DeploymentID = deploymentID
				action.AsyncOperation.TaskID = s.t.taskID
				action.AsyncOperation.ExecutionID = s.t.id
				action.AsyncOperation.WorkflowName = workflowName
				action.AsyncOperation.StepName = s.Name
				action.AsyncOperation.NodeName = s.Target
				action.AsyncOperation.Operation = op
				action.AsyncOperation.WorkflowStepInfo = eventInfo
				// Register scheduled action for asynchronous execution
				id, err := scheduling.RegisterAction(w.consulClient, deploymentID, timeInterval, action)
				action.ID = id
				log.Debugf("Scheduled action with ID: %q has been registered with timeInterval: %s", action.ID, timeInterval.String())
				if err != nil {
					return err
				}
				l, err := acquireRunningExecLock(s.cc, s.t.taskID)
				if err != nil {
					return err
				}
				defer l.Unlock()
				log.Debugf("Storing runningExecutions with id %q for task %q", id, s.t.taskID)
				return consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, s.t.taskID, ".runningExecutions", id), "recurrent action")
			}()
		} else {
			err = func() error {
				defer metrics.MeasureSinceWithLabels(metricsutil.CleanupMetricKey([]string{"executor", "operation", "duration"}), time.Now(), executorOperationLabels)
				return exec.ExecOperation(wfCtx, cfg, s.t.taskID, deploymentID, s.Target, op)
			}()
		}
		if err != nil {
			metrics.IncrCounterWithLabels(metricsutil.CleanupMetricKey([]string{"executor", "operation", "failures"}), 1, executorOperationLabels)
			for _, instanceName := range instances {
				s.publishInstanceRelatedEvents(wfCtx, deploymentID, instanceName, eventInfo, tasks.TaskStepStatusERROR)
			}
			return err
		}
		metrics.IncrCounterWithLabels(metricsutil.CleanupMetricKey([]string{"executor", "operation", "successes"}), 1, executorOperationLabels)
		if !s.Async {
			for _, instanceName := range instances {
				s.publishInstanceRelatedEvents(wfCtx, deploymentID, instanceName, eventInfo, tasks.TaskStepStatusDONE)
			}
		}
	case builder.ActivityTypeInline:
		// Register inline workflow associated to the original task
		return s.registerInlineWorkflow(wfCtx, deploymentID, activity)
	}
	return nil
}

func (s *step) getActivityInputParameters(ctx context.Context, activity builder.Activity,
	deploymentID, workflowName string) (map[string]tosca.ParameterDefinition, error) {

	// Getting activity input parameters first
	result := make(map[string]tosca.ParameterDefinition)
	for inputName, paramDef := range activity.Inputs() {

		if paramDef.Value != nil || paramDef.Default != nil {
			result[inputName] = paramDef
		}
	}

	// Getting workflow inputs
	wf, err := deployments.GetWorkflow(ctx, deploymentID, workflowName)
	if err != nil {
		return nil, err
	}

	if wf == nil {
		return nil, errors.Errorf("Can't retrieve inputs for an activity of workflow %q in deployment %q, workflow definition not found", workflowName, deploymentID)
	}

	for inputName, propDef := range wf.Inputs {

		if _, ok := result[inputName]; ok {
			// Already defined in activity
			continue
		}

		valueAssign, err := s.getWorkflowInputValue(ctx, deploymentID, workflowName, inputName, propDef)
		if err != nil {
			return result, err
		}

		result[inputName] = tosca.ParameterDefinition{
			Type:        propDef.Type,
			Description: propDef.Description,
			Required:    propDef.Required,
			Default:     propDef.Default,
			Status:      propDef.Status,
			EntrySchema: propDef.EntrySchema,
			Value:       valueAssign,
		}
	}

	// Getting inputs at the topology level that can be used
	// if inputs aren't defined at lower levels (workflow or activity)
	err = addTopologyInputs(ctx, deploymentID, result)

	return result, err
}

func (s *step) getWorkflowInputValue(ctx context.Context, deploymentID, workflowName, inputName string,
	propDef tosca.PropertyDefinition) (*tosca.ValueAssignment, error) {

	var valueAssign *tosca.ValueAssignment
	inputValue, err := tasks.GetTaskInput(s.t.taskID, inputName)
	if err != nil {
		if !tasks.IsTaskDataNotFoundError(err) {
			return valueAssign, err
		}

		// No input value in task, defining an input parameter if this property
		// has a default value or is defined in the topology
		if propDef.Default == nil {
			// No default value, and no input in this execution context
			// => no parameter is defined in this execution context
			// It can still be defined in the topology
			valueAssign, err = getValueFromTopology(ctx, deploymentID, inputName)
			if err != nil {
				return valueAssign, err
			}
			if propDef.Required != nil && *propDef.Required && valueAssign == nil {
				return valueAssign, errors.Errorf("Missing required value for input %q in step:%q workflow:%q, deploymentID:%q, taskID:%q",
					inputName, s.Name, workflowName, deploymentID, s.t.taskID)
			}
		}
	} else {
		valueAssign, err = tosca.ToValueAssignment(inputValue)
	}
	return valueAssign, err

}

func getValueFromTopology(ctx context.Context, deploymentID, inputName string) (*tosca.ValueAssignment, error) {
	var valueAssign *tosca.ValueAssignment

	found, paramDef, err := deployments.GetTopologyInputParameter(ctx, deploymentID, inputName)
	if err != nil {
		return valueAssign, err
	}
	if found {
		valueAssign = paramDef.Value
		if valueAssign == nil {
			valueAssign = paramDef.Default
		}
	}
	return valueAssign, err
}

func addTopologyInputs(ctx context.Context, deploymentID string, result map[string]tosca.ParameterDefinition) error {

	topologyInputNames, err := deployments.GetTopologyInputsNames(ctx, deploymentID)
	if err != nil {
		return err
	}

	for _, inputName := range topologyInputNames {
		if _, ok := result[inputName]; ok {
			// Already defined
			continue
		}

		found, paramDef, err := deployments.GetTopologyInputParameter(ctx, deploymentID, inputName)
		if err != nil {
			return err
		}

		if found {
			result[inputName] = *paramDef
		}

	}
	return err
}

func (s *step) registerInlineWorkflow(ctx context.Context, deploymentID string, activity builder.Activity) error {
	workflowName := activity.Value()
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, s.t.targetID).RegisterAsString(
		fmt.Sprintf("Register workflow %q from taskID:%q, deploymentID:%q", workflowName, s.t.taskID, s.t.targetID))

	// Preparing a new task with its own data referencing the parent workflow step
	parentData, err := tasks.GetAllTaskData(s.t.taskID)
	if err != nil {
		return err
	}
	data := make(map[string]string)
	for k, v := range parentData {
		data[k] = v
	}
	data[taskDataParentWorkflowName] = s.WorkflowName
	data[taskDataParentStepName] = s.Name
	data[taskDataParentTaskID] = s.t.taskID
	data[taskDataDeploymentID] = deploymentID
	data[taskDataWorkflowName] = workflowName

	// Add workflow input parameters if any
	inputParameters, err := s.getActivityInputParameters(ctx, activity, deploymentID, workflowName)
	if err != nil {
		return err
	}
	for inputName, param := range inputParameters {
		inputValue := param.Value
		if inputValue == nil {
			inputValue = param.Default
		}
		if inputValue != nil {
			log.Debugf("Adding to inline workflow %s input %s value %s", workflowName, inputName, inputValue.String())
			data[path.Join("inputs", inputName)] = fmt.Sprintf("%v", inputValue)
		}
	}

	taskID, err := collector.NewCollector(s.cc).RegisterTaskWithData(deploymentID, tasks.TaskTypeCustomWorkflow, data)
	if err != nil {
		err = errors.Wrapf(err, "Failed to register inline workflow %s in parent workflow %s step %s, targetID %s, taskID %s",
			workflowName, s.WorkflowName, s.Name, s.t.targetID, s.t.taskID)
		_ = s.setStatus(tasks.TaskStepStatusERROR)
		return err
	}

	log.Debugf("Registered task %s for inline workflow %s in parent workflow %s step %s",
		taskID, workflowName, s.WorkflowName, s.Name)
	_ = s.setStatus(tasks.TaskStepStatusRUNNING)

	// Marking this step as asynchronous as it should not be considered as
	// done by the caller
	s.Async = true
	return err
}

func (s *step) checkIfPreviousOfNextStepAreDone(ctx context.Context, nextStep *step, workflowName string) (bool, error) {
	cpt := 0
	for _, step := range nextStep.Previous {
		stepStatus, err := tasks.GetTaskStepStatus(s.t.taskID, step.Name)
		if err != nil {
			return false, errors.Wrapf(err, "Failed to retrieve step status with TaskID:%q, step:%q", s.t.taskID, step.Name)
		}
		if stepStatus == tasks.TaskStepStatusDONE {
			cpt++
		} else if stepStatus == tasks.TaskStepStatusCANCELED || stepStatus == tasks.TaskStepStatusERROR {
			return false, errors.Errorf("An error has been detected on other step:%q for workflow:%q, deploymentID:%q, taskID:%q. No more steps will be executed", step.Name, workflowName, s.t.targetID, s.t.taskID)
		}
	}

	// In case of workflow join, the last done of previous steps will register the join step
	if len(nextStep.Previous) == cpt {
		log.Debugf("All previous steps of step:%q are done, so it can be registered to be executed", nextStep.Name)
		return true, nil
	}
	return false, nil
}

func (s *step) registerNextSteps(ctx context.Context, workflowName string) error {
	// If step is terminal, we check if workflow is done
	if s.IsTerminal() {
		return nil
	}

	// Register workflow step to handle step statuses for next steps
	regSteps := make([]*step, 0)
	for _, bnStep := range s.Next {
		nStep := wrapBuilderStep(bnStep, s.cc, s.t)
		// In case of join, check each previous status step
		if len(nStep.Previous) > 1 {
			done, err := s.checkIfPreviousOfNextStepAreDone(ctx, nStep, workflowName)
			if err != nil {
				return err
			}
			if done {
				regSteps = append(regSteps, nStep)
			}
		} else {
			regSteps = append(regSteps, nStep)
		}
	}
	l, err := acquireRunningExecLock(s.cc, s.t.taskID)
	if err != nil {
		return err
	}
	defer l.Unlock()
	ops := createWorkflowStepsOperations(s.t.taskID, regSteps)
	err = tasks.StoreOperations(s.t.taskID, ops)
	if err != nil {
		err = errors.Wrapf(err, "Failed to register executionTasks with TaskID:%q", s.t.taskID)
	}
	return err
}

func (s *step) registerOnCancelOrFailureSteps(ctx context.Context, workflowName string, steps []*builder.Step) error {
	if len(steps) == 0 {
		return nil
	}
	// Register workflow step to handle cancellation
	regSteps := make([]*step, 0)
	for _, bnStep := range steps {
		nStep := wrapBuilderStep(bnStep, s.cc, s.t)
		regSteps = append(regSteps, nStep)
	}

	l, err := acquireRunningExecLock(s.cc, s.t.taskID)
	if err != nil {
		return err
	}
	defer l.Unlock()
	ops := createWorkflowStepsOperations(s.t.taskID, regSteps)
	err = tasks.StoreOperations(s.t.taskID, ops)
	if err != nil {
		err = errors.Wrapf(err, "Failed to register executionTasks with TaskID:%q", s.t.taskID)
	}
	return err
}

func (s *step) publishInstanceRelatedEvents(ctx context.Context, deploymentID, instanceName string, eventInfo *events.WorkflowStepInfo, statuses ...tasks.TaskStepStatus) {
	// taskExecutionID has to be unique for each instance, so we concat it to instanceName
	eventInfo.InstanceName = instanceName
	instanceTaskExecutionID := fmt.Sprintf("%s-%s", s.t.id, instanceName)
	for _, status := range statuses {
		events.PublishAndLogWorkflowStepStatusChange(ctx, deploymentID, s.t.taskID, eventInfo, status.String())
		events.PublishAndLogAlienTaskStatusChange(ctx, deploymentID, s.t.taskID, instanceTaskExecutionID, eventInfo, status.String())
	}
}
