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
	"path"
	"strings"
	"time"
)

// Step represents the workflow step
type Step struct {
	Name               string
	Target             string
	TargetRelationship string
	OperationHost      string
	Activities         []Activity
	Next               []*Step
	Previous           []*Step
	kv                 *api.KV
	stepPrefix         string
	t                  *TaskExecution
}

type visitStep struct {
	refCount int
	s        *Step
}

// IsTerminal returns true is the workflow step has no next step
func (s *Step) IsTerminal() bool {
	return len(s.Next) == 0
}

// IsInitial returns true is the workflow step has no previous step
func (s *Step) IsInitial() bool {
	return len(s.Previous) == 0
}

func (s *Step) setStatus(status tasks.StepStatus) error {
	kvp := &api.KVPair{Key: path.Join(s.stepPrefix, "status"), Value: []byte(status.String())}
	_, err := s.kv.Put(kvp, nil)
	if err != nil {
		return err
	}
	kvp = &api.KVPair{Key: path.Join(consulutil.WorkflowsPrefix, s.t.TaskID, s.Name), Value: []byte(status.String())}
	_, err = s.kv.Put(kvp, nil)
	return err
}

func isTargetOperationOnSource(s *Step) bool {
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

func isSourceOperationOnTarget(s *Step) bool {
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
func (s *Step) isRunnable() (bool, error) {
	kvp, _, err := s.kv.Get(path.Join(consulutil.WorkflowsPrefix, s.t.TaskID, s.Name), nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	// Check if Step is already done
	status := string(kvp.Value)
	if status != "" {
		stepStatus, err := tasks.ParseStepStatus(status)
		if err != nil {
			return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}

		if kvp != nil && stepStatus == tasks.StepStatusDONE {
			return false, nil
		}
	}

	if s.t.TaskType == tasks.TaskTypeScaleOut || s.t.TaskType == tasks.TaskTypeScaleIn {
		// If not a relationship check the actual node
		if s.TargetRelationship == "" {
			return tasks.IsTaskRelatedNode(s.kv, s.t.TaskID, s.Target)
		}

		if isSourceOperationOnTarget(s) {
			// operation on target but Check if Source is implied on scale
			return tasks.IsTaskRelatedNode(s.kv, s.t.TaskID, s.Target)
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
			return tasks.IsTaskRelatedNode(s.kv, s.t.TaskID, targetNodeName)
		}

		// otherwise check the actual node is implied
		return tasks.IsTaskRelatedNode(s.kv, s.t.TaskID, s.Target)

	}

	return true, nil
}

// Run allows to execute a workflow step
func (s *Step) Run(ctx context.Context, cfg config.Configuration, kv *api.KV, deploymentID string, shutdownChan chan struct{}, bypassErrors bool, workflowName string, w worker) error {
	// Fill log optional fields for log registration
	ctx = events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.WorkFlowID: workflowName, events.NodeID: s.Target})
	logOptFields, _ := events.FromContext(ctx)
	s.setStatus(tasks.StepStatusINITIAL)
	// First: we check if Step is runnable
	// TODO for now alien generates a 1 to 1 Step/activity model but we should probably test if an activity is runnable
	if runnable, err := s.isRunnable(); err != nil {
		return err
	} else if !runnable {
		log.Debugf("Deployment %q: Skipping TaskStep %q", deploymentID, s.Name)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(fmt.Sprintf("Skipping TaskStep %q", s.Name))
		s.setStatus(tasks.StepStatusDONE)
		return nil
	}

	s.setStatus(tasks.StepStatusRUNNING)

	// Create a new context to handle gracefully current Step termination when an error occurred during another Step
	wfCtx, cancelWf := context.WithCancel(events.NewContext(context.Background(), logOptFields))
	waitDoneCh := make(chan struct{})
	defer close(waitDoneCh)
	go func() {
		select {
		case <-ctx.Done():
			status, err := s.t.getTaskStatus()
			if err != nil {
				log.Printf("Failed to retrieve task status due to error:%v", err)
			}
			if status != tasks.TaskStatusCANCELED {
				// Temporize to allow current Step termination before cancelling context and put Step in error
				log.Printf("An error occurred on another Step while Step %q is running: trying to gracefully finish it.", s.Name)
				select {
				case <-time.After(cfg.WfStepGracefulTerminationTimeout):
					cancelWf()
					log.Printf("TaskStep %q not yet finished: we set it on error", s.Name)
					s.setStatus(tasks.StepStatusERROR)
					return
				case <-waitDoneCh:
					return
				}
			} else {
				// We immediately cancel the Step
				cancelWf()
				s.setStatus(tasks.StepStatusCANCELED)
				return
			}
		case <-waitDoneCh:
			return
		}
	}()
	defer cancelWf()

	log.Debugf("Processing Step %q", s.Name)
	for _, activity := range s.Activities {
		err := func() error {
			for _, hook := range preActivityHooks {
				hook(wfCtx, cfg, s.t.TaskID, deploymentID, s.Target, activity)
			}
			defer func() {
				for _, hook := range postActivityHooks {
					hook(wfCtx, cfg, s.t.TaskID, deploymentID, s.Target, activity)
				}
			}()
			err := s.runActivity(wfCtx, kv, cfg, deploymentID, bypassErrors, w, activity)
			if err != nil {
				setNodeStatus(wfCtx, kv, s.t.TaskID, deploymentID, s.Target, tosca.NodeStateError.String())
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("TaskStep %q: error details: %+v", s.Name, err)
				if !bypassErrors {
					s.setStatus(tasks.StepStatusERROR)
					return err
				}
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).Registerf("TaskStep %q: Bypassing error: %v but workflow continue", s.Name, err)
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}

	log.Debugf("TaskStep %s done without error.", s.Name)
	s.setStatus(tasks.StepStatusDONE)
	return nil
}

func (s *Step) runActivity(wfCtx context.Context, kv *api.KV, cfg config.Configuration, deploymentID string, bypassErrors bool, w worker, activity Activity) error {
	// Get activity related instances
	instances, err := tasks.GetInstances(kv, s.t.TaskID, deploymentID, s.Target)
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
			return provisioner.ExecDelegate(wfCtx, cfg, s.t.TaskID, deploymentID, s.Target, delegateOp)
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
		setNodeStatus(wfCtx, kv, s.t.TaskID, deploymentID, s.Target, activity.Value())
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

		exec, err := getOperationExecutor(kv, deploymentID, op.ImplementationArtifact)
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
			return exec.ExecOperation(wfCtx, cfg, s.t.TaskID, deploymentID, s.Target, op)
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
