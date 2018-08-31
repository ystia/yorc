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

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/helper/metricsutil"
	"github.com/ystia/yorc/helper/stringutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov/operations"
	"github.com/ystia/yorc/registry"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tosca"
)

// step represents the workflow step
type step struct {
	name               string
	target             string
	targetRelationship string
	operationHost      string
	activities         []Activity
	next               []*step
	previous           []*step
	kv                 *api.KV
	workflowName       string
	t                  *taskExecution
}

type visitStep struct {
	refCount int
	s        *step
}

// isTerminal returns true is the workflow step has no next step
func (s *step) isTerminal() bool {
	return len(s.next) == 0
}

// isInitial returns true is the workflow step has no previous step
func (s *step) isInitial() bool {
	return len(s.previous) == 0
}

func (s *step) setStatus(status tasks.TaskStepStatus) error {
	kvp := &api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, s.t.targetID, "workflows", s.workflowName, "steps", s.name, "status"), Value: []byte(status.String())}
	_, err := s.kv.Put(kvp, nil)
	if err != nil {
		return err
	}
	kvp = &api.KVPair{Key: path.Join(consulutil.WorkflowsPrefix, s.t.taskID, s.name), Value: []byte(status.String())}
	_, err = s.kv.Put(kvp, nil)
	return err
}

func (s *step) cancelNextSteps() {
	for _, ns := range s.next {
		log.Debugf("cancel step name:%q", ns.name)
		// bind next canceled execution to actual one
		ns.t = s.t
		ns.setStatus(tasks.TaskStepStatusCANCELED)
		ns.cancelNextSteps()
	}
}

func isTargetOperationOnSource(s *step) bool {
	if strings.ToUpper(s.operationHost) != "SOURCE" {
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
	if strings.ToUpper(s.operationHost) != "TARGET" {
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
func (s *step) isRunnable() (bool, error) {
	kvp, _, err := s.kv.Get(path.Join(consulutil.WorkflowsPrefix, s.t.taskID, s.name), nil)
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

	if s.t.taskType == tasks.TaskTypeScaleOut || s.t.taskType == tasks.TaskTypeScaleIn {
		// If not a relationship check the actual node
		if s.targetRelationship == "" {
			return tasks.IsTaskRelatedNode(s.kv, s.t.taskID, s.target)
		}

		if isSourceOperationOnTarget(s) {
			// operation on target but Check if Source is implied on scale
			return tasks.IsTaskRelatedNode(s.kv, s.t.taskID, s.target)
		}

		if isTargetOperationOnSource(s) || strings.ToUpper(s.operationHost) == "TARGET" {
			// Check if Target is implied on scale
			targetReqIndex, err := deployments.GetRequirementIndexByNameForNode(s.kv, s.t.targetID, s.target, s.targetRelationship)
			if err != nil {
				return false, err
			}
			targetNodeName, err := deployments.GetTargetNodeForRequirement(s.kv, s.t.targetID, s.target, targetReqIndex)
			if err != nil {
				return false, err
			}
			return tasks.IsTaskRelatedNode(s.kv, s.t.taskID, targetNodeName)
		}

		// otherwise check the actual node is implied
		return tasks.IsTaskRelatedNode(s.kv, s.t.taskID, s.target)

	}

	return true, nil
}

// run allows to execute a workflow step
func (s *step) run(ctx context.Context, cfg config.Configuration, kv *api.KV, deploymentID string, bypassErrors bool, workflowName string, w *worker) error {
	// Fill log optional fields for log registration
	ctx = events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.WorkFlowID: workflowName, events.NodeID: s.target})
	// First: we check if Step is runnable
	if runnable, err := s.isRunnable(); err != nil {
		return err
	} else if !runnable {
		log.Debugf("Deployment %q: Skipping TaskStep %q", deploymentID, s.name)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(fmt.Sprintf("Skipping TaskStep %q", s.name))
		s.setStatus(tasks.TaskStepStatusDONE)
		return nil
	}
	s.setStatus(tasks.TaskStepStatusRUNNING)

	logOptFields, _ := events.FromContext(ctx)
	// Create a new context to handle gracefully current step termination when an error occurred during another step
	wfCtx, cancelWf := context.WithCancel(events.NewContext(context.Background(), logOptFields))
	waitDoneCh := make(chan struct{})
	defer close(waitDoneCh)
	go func() {
		select {
		case <-w.shutdownCh:
			log.Printf("Shutdown signal has been sent.Step %q will be canceled", s.name)
			s.setStatus(tasks.TaskStepStatusCANCELED)
			cancelWf()
			return
		case <-ctx.Done():
			status, err := s.t.getTaskStatus()
			if err != nil {
				log.Debugf("Failed to get task status with taskID:%q due to error:%+v", s.t.taskID, err)
			}
			if status == tasks.TaskStatusCANCELED {
				// We immediately cancel the step
				cancelWf()
				log.Printf("Cancel event has been sent.This step will fail and next ones %q will be canceled", s.name)
				s.cancelNextSteps()
				return
			} else if status == tasks.TaskStatusFAILED {
				log.Printf("An error occurred on another step while step %q is running: trying to gracefully finish it.", s.name)
				select {
				case <-time.After(cfg.WfStepGracefulTerminationTimeout):
					cancelWf()
					log.Printf("Step %q not yet finished: we set it on error", s.name)
					s.setStatus(tasks.TaskStepStatusERROR)
					return
				case <-waitDoneCh:
					return
				}
			}
		case <-waitDoneCh:
			return
		}
	}()
	defer cancelWf()

	log.Debugf("Processing Step %q", s.name)
	for _, activity := range s.activities {
		err := func() error {
			for _, hook := range preActivityHooks {
				hook(ctx, cfg, s.t.taskID, deploymentID, s.target, activity)
			}
			defer func() {
				for _, hook := range postActivityHooks {
					hook(ctx, cfg, s.t.taskID, deploymentID, s.target, activity)
				}
			}()
			err := s.runActivity(wfCtx, kv, cfg, deploymentID, bypassErrors, w, activity)
			if err != nil {
				setNodeStatus(wfCtx, kv, s.t.taskID, deploymentID, s.target, tosca.NodeStateError.String())
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("TaskStep %q: error details: %+v", s.name, err)
				if !bypassErrors {
					s.setStatus(tasks.TaskStepStatusERROR)
					return err
				}
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).Registerf("TaskStep %q: Bypassing error: %+v but workflow continue", s.name, err)
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}

	log.Debugf("Task execution:%q for step:%q, workflow:%q, taskID:%q done without error.", s.t.id, s.name, s.workflowName, s.t.taskID)
	s.setStatus(tasks.TaskStepStatusDONE)
	return nil
}

func (s *step) runActivity(wfCtx context.Context, kv *api.KV, cfg config.Configuration, deploymentID string, bypassErrors bool, w *worker, activity Activity) error {
	// Get activity related instances
	instances, err := tasks.GetInstances(kv, s.t.taskID, deploymentID, s.target)
	if err != nil {
		return err
	}

	switch activity.Type() {
	case ActivityTypeDelegate:
		nodeType, err := deployments.GetNodeType(kv, deploymentID, s.target)
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
			// TODO: replace this with workflow steps events
			events.WithContextOptionalFields(events.AddLogOptionalFields(wfCtx, events.LogOptionalFields{events.InstanceID: instanceName})).NewLogEntry(events.LogLevelDEBUG, deploymentID).RegisterAsString("executing delegate operation")
		}
		err = func() error {
			defer metrics.MeasureSince(metricsutil.CleanupMetricKey([]string{"executor", "delegate", deploymentID, nodeType, delegateOp}), time.Now())
			return provisioner.ExecDelegate(wfCtx, cfg, s.t.taskID, deploymentID, s.target, delegateOp)
		}()

		if err != nil {
			metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "delegate", deploymentID, nodeType, delegateOp, "failures"}), 1)
			for _, instanceName := range instances {
				// TODO: replace this with workflow steps events
				events.WithContextOptionalFields(events.AddLogOptionalFields(wfCtx, events.LogOptionalFields{events.InstanceID: instanceName})).NewLogEntry(events.LogLevelDEBUG, deploymentID).RegisterAsString("delegate operation failed")
			}
			return err
		}
		metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "delegate", deploymentID, nodeType, delegateOp, "successes"}), 1)
		for _, instanceName := range instances {
			// TODO: replace this with workflow steps events
			events.WithContextOptionalFields(events.AddLogOptionalFields(wfCtx, events.LogOptionalFields{events.InstanceID: instanceName})).NewLogEntry(events.LogLevelDEBUG, deploymentID).RegisterAsString("delegate operation succeeded")
		}
	case ActivityTypeSetState:
		setNodeStatus(wfCtx, kv, s.t.taskID, deploymentID, s.target, activity.Value())
	case ActivityTypeCallOperation:
		op, err := operations.GetOperation(wfCtx, kv, s.t.targetID, s.target, activity.Value(), s.targetRelationship, s.operationHost)
		if err != nil {
			if deployments.IsOperationNotImplemented(err) {
				// Operation not implemented just skip it
				log.Debugf("Voluntary bypassing error: %s.", err.Error())
				return nil
			}
			return err
		}

		exec, err := getOperationExecutor(kv, deploymentID, op.ImplementationArtifact)
		if err != nil {
			return err
		}
		nodeType, err := deployments.GetNodeType(kv, deploymentID, s.target)
		if err != nil {
			return err
		}
		wfCtx = events.AddLogOptionalFields(wfCtx, events.LogOptionalFields{events.InterfaceName: stringutil.GetAllExceptLastElement(op.Name, "."), events.OperationName: stringutil.GetLastElement(op.Name, ".")})
		for _, instanceName := range instances {
			// TODO: replace this with workflow steps events
			events.WithContextOptionalFields(events.AddLogOptionalFields(wfCtx, events.LogOptionalFields{events.InstanceID: instanceName})).NewLogEntry(events.LogLevelDEBUG, deploymentID).RegisterAsString("executing operation")
		}
		err = func() error {
			defer metrics.MeasureSince(metricsutil.CleanupMetricKey([]string{"executor", "operation", deploymentID, nodeType, op.Name}), time.Now())
			return exec.ExecOperation(wfCtx, cfg, s.t.taskID, deploymentID, s.target, op)
		}()
		if err != nil {
			metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "operation", deploymentID, nodeType, op.Name, "failures"}), 1)
			for _, instanceName := range instances {
				// TODO: replace this with workflow steps events
				events.WithContextOptionalFields(events.AddLogOptionalFields(wfCtx, events.LogOptionalFields{events.InstanceID: instanceName})).NewLogEntry(events.LogLevelDEBUG, deploymentID).RegisterAsString("operation failed")
			}
			return err
		}
		metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "operation", deploymentID, nodeType, op.Name, "successes"}), 1)
		for _, instanceName := range instances {
			// TODO: replace this with workflow steps events
			events.WithContextOptionalFields(events.AddLogOptionalFields(wfCtx, events.LogOptionalFields{events.InstanceID: instanceName})).NewLogEntry(events.LogLevelDEBUG, deploymentID).RegisterAsString("operation succeeded")
		}
	case ActivityTypeInline:
		// Register inline workflow associated to the original task
		return w.registerInlineWorkflow(wfCtx, s.t, activity.Value())
	}
	return nil
}
