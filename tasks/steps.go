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
	"context"
	"fmt"
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
	"github.com/ystia/yorc/tosca"
	"path"
	"strings"
	"time"
)

//go:generate go-enum -f=steps.go --lower

// StepStatus x ENUM(
// INITIAL,
// RUNNING,
// DONE,
// ERROR,
// CANCELED
// )
type StepStatus int

// Step represents a step related to a workflow
type Step struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type step struct {
	Name               string
	Target             string
	TargetRelationship string
	OperationHost      string
	Activities         []Activity
	Next               []*step
	Previous           []*step
	kv                 *api.KV
	stepPrefix         string
	t                  *task
}

func (s *step) IsTerminal() bool {
	return len(s.Next) == 0
}

func (s *step) SetTaskID(taskID *task) {
	s.t = taskID
}

func (s *step) notifyNext() {
	for _, next := range s.Next {
		log.Debugf("Step %q, notifying step %q", s.Name, next.Name)
		// Create execution key for next step
	}
}

func (s *step) setStatus(status StepStatus) error {
	statusStr := strings.ToLower(status.String())
	kvp := &api.KVPair{Key: path.Join(s.stepPrefix, "status"), Value: []byte(statusStr)}
	_, err := s.kv.Put(kvp, nil)
	if err != nil {
		return err
	}
	kvp = &api.KVPair{Key: path.Join(consulutil.WorkflowsPrefix, s.t.ID, s.Name), Value: []byte(statusStr)}
	_, err = s.kv.Put(kvp, nil)
	return err
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

// isRunnable Checks if a step should be run or bypassed
//
// It first checks if the step is not already done in this workflow instance
// And for ScaleOut and ScaleDown it checks if the node or the target node in case of an operation running on the target node is part of the operation
func (s *step) isRunnable() (bool, error) {
	kvp, _, err := s.kv.Get(path.Join(consulutil.WorkflowsPrefix, s.t.ID, s.Name), nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	// Check if step is already done
	status := string(kvp.Value)
	if status != "" {
		stepStatus, err := ParseStepStatus(status)
		if err != nil {
			return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}

		if kvp != nil && stepStatus == StepStatusDONE {
			return false, nil
		}
	}

	if s.t.TaskType == TaskTypeScaleOut || s.t.TaskType == TaskTypeScaleIn {
		// If not a relationship check the actual node
		if s.TargetRelationship == "" {
			return IsTaskRelatedNode(s.kv, s.t.ID, s.Target)
		}

		if isSourceOperationOnTarget(s) {
			// operation on target but Check if Source is implied on scale
			return IsTaskRelatedNode(s.kv, s.t.ID, s.Target)
		}

		if isTargetOperationOnSource(s) || strings.ToUpper(s.OperationHost) == "TARGET" {
			// Check if Target is implied on scale
			targetReqIndex, err := deployments.GetRequirementIndexByNameForNode(s.kv, s.t.TargetID, s.Target, s.TargetRelationship)
			if err != nil {
				return false, err
			}
			targetNodeName, err := deployments.GetTargetNodeForRequirement(s.kv, s.t.TargetID, s.Target, targetReqIndex)
			if err != nil {
				return false, err
			}
			return IsTaskRelatedNode(s.kv, s.t.ID, targetNodeName)
		}

		// otherwise check the actual node is implied
		return IsTaskRelatedNode(s.kv, s.t.ID, s.Target)

	}

	return true, nil
}

func (s *step) run(ctx context.Context, deploymentID string, kv *api.KV, ignoredErrsChan chan error, shutdownChan chan struct{}, cfg config.Configuration, bypassErrors bool, workflowName string, w worker) error {
	// Fill log optional fields for log registration
	ctx = events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.WorkFlowID: workflowName, events.NodeID: s.Target})
	logOptFields, _ := events.FromContext(ctx)
	s.setStatus(StepStatusINITIAL)
	haveErr := false
	// First: we check if step is runnable
	// TODO for now alien generates a 1 to 1 step/activity model but we should probably test if an activity is runnable
	if runnable, err := s.isRunnable(); err != nil {
		return err
	} else if !runnable {
		log.Debugf("Deployment %q: Skipping Step %q", deploymentID, s.Name)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(fmt.Sprintf("Skipping Step %q", s.Name))
		s.setStatus(StepStatusDONE)
		s.notifyNext()
		return nil
	}

	s.setStatus(StepStatusRUNNING)

	// Create a new context to handle gracefully current step termination when an error occurred during another step
	wfCtx, cancelWf := context.WithCancel(events.NewContext(context.Background(), logOptFields))
	waitDoneCh := make(chan struct{})
	defer close(waitDoneCh)
	go func() {
		select {
		case <-ctx.Done():
			if s.t.status != TaskStatusCANCELED {
				// Temporize to allow current step termination before cancelling context and put step in error
				log.Printf("An error occurred on another step while step %q is running: trying to gracefully finish it.", s.Name)
				select {
				case <-time.After(cfg.WfStepGracefulTerminationTimeout):
					cancelWf()
					log.Printf("Step %q not yet finished: we set it on error", s.Name)
					s.setStatus(StepStatusERROR)
					return
				case <-waitDoneCh:
					return
				}
			} else {
				// We immediately cancel the step
				cancelWf()
				s.setStatus(StepStatusCANCELED)
				return
			}
		case <-waitDoneCh:
			return
		}
	}()
	defer cancelWf()

	log.Debugf("Processing step %q", s.Name)
	for _, activity := range s.Activities {
		err := func() error {
			for _, hook := range preActivityHooks {
				hook(wfCtx, cfg, s.t.ID, deploymentID, s.Target, activity)
			}
			defer func() {
				for _, hook := range postActivityHooks {
					hook(wfCtx, cfg, s.t.ID, deploymentID, s.Target, activity)
				}
			}()
			err := s.runActivity(wfCtx, kv, cfg, deploymentID, bypassErrors, w, activity)
			if err != nil {
				setNodeStatus(wfCtx, kv, s.t.ID, deploymentID, s.Target, tosca.NodeStateError.String())
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("Step %q: error details: %+v", s.Name, err)
				if !bypassErrors {
					s.setStatus(StepStatusERROR)
					return err
				}
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).Registerf("Step %q: Bypassing error: %v", s.Name, err)
				ignoredErrsChan <- err
				haveErr = true
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}

	s.notifyNext()

	if haveErr {
		log.Debug("Step %s generate an error but workflow continue", s.Name)
		s.setStatus(StepStatusERROR)
		// Return nil otherwise the rest of the workflow will be canceled
		return nil
	}
	log.Debugf("Step %s done without error.", s.Name)
	s.setStatus(StepStatusDONE)
	return nil
}

func (s *step) runActivity(wfCtx context.Context, kv *api.KV, cfg config.Configuration, deploymentID string, bypassErrors bool, w worker, activity Activity) error {
	// Get activity related instances
	instances, err := GetInstances(kv, s.t.ID, deploymentID, s.Target)
	if err != nil {
		return err
	}

	switch activity.Type() {
	case ActivityTypeDelegate:
		nodeType, err := deployments.GetNodeType(kv, deploymentID, s.Target)
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
			return provisioner.ExecDelegate(wfCtx, cfg, s.t.ID, deploymentID, s.Target, delegateOp)
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
		setNodeStatus(wfCtx, kv, s.t.ID, deploymentID, s.Target, activity.Value())
	case ActivityTypeCallOperation:
		op, err := operations.GetOperation(wfCtx, kv, s.t.TargetID, s.Target, activity.Value(), s.TargetRelationship, s.OperationHost)
		if err != nil {
			if deployments.IsOperationNotImplemented(err) {
				// Operation not implemented just skip it
				log.Debugf("Voluntary bypassing error: %s.", err.Error())
				return nil
			}
			return err
		}

		exec, err := GetOperationExecutor(kv, deploymentID, op.ImplementationArtifact)
		if err != nil {
			return err
		}
		nodeType, err := deployments.GetNodeType(kv, deploymentID, s.Target)
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
			return exec.ExecOperation(wfCtx, cfg, s.t.ID, deploymentID, s.Target, op)
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
		//TODO Handle inline workflow
		//if err := w.runWorkflows(wfCtx, s.t, []string{activity.Value()}, bypassErrors); err != nil {
		//	return err
		//}
	}
	return nil
}
