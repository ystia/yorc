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

	"strconv"

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

type step struct {
	Name               string
	Target             string
	TargetRelationship string
	OperationHost      string
	Activities         []Activity
	Next               []*step
	Previous           []*step
	NotifyChan         chan struct{}
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
		next.NotifyChan <- struct{}{}
	}
}

type visitStep struct {
	refCount int
	s        *step
}

func (s *step) setStatus(status tasks.TaskStepStatus) error {
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

func (s *step) isRelationshipTargetNodeRelated() (bool, error) {
	if s.TargetRelationship != "" && strings.ToUpper(s.OperationHost) == "TARGET" {
		targetNodeName, err := deployments.GetTargetNodeForRequirement(s.kv, s.t.TargetID, s.Target, s.TargetRelationship)
		if err != nil {
			return false, err
		}

		isNodeTargetTask, err := tasks.IsTaskRelatedNode(s.kv, s.t.ID, targetNodeName)
		if err != nil || isNodeTargetTask {
			return isNodeTargetTask, err
		}

	}
	return false, nil
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
		stepStatus, err := tasks.ParseTaskStepStatus(status)
		if err != nil {
			return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}

		if kvp != nil && stepStatus == tasks.TaskStepStatusDONE {
			return false, nil
		}
	}

	if s.t.TaskType == tasks.ScaleOut || s.t.TaskType == tasks.ScaleIn {
		isNodeTargetTask, err := tasks.IsTaskRelatedNode(s.kv, s.t.ID, s.Target)
		if err != nil {
			return false, err
		}
		if !isNodeTargetTask {
			isTargetNodeRelated, err := s.isRelationshipTargetNodeRelated()
			if err != nil || !isTargetNodeRelated {
				return false, err
			}
		}
	}

	return true, nil
}

func setNodeStatus(ctx context.Context, kv *api.KV, taskID, deploymentID, nodeName, status string) error {
	instancesIDs, err := tasks.GetInstances(kv, taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}

	for _, id := range instancesIDs {
		// Publish status change event
		err := deployments.SetInstanceStateString(ctx, kv, deploymentID, nodeName, id, status)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *step) run(ctx context.Context, deploymentID string, kv *api.KV, ignoredErrsChan chan error, shutdownChan chan struct{}, cfg config.Configuration, bypassErrors bool, workflowName string, w worker) error {
	// Fill log optional fields for log registration
	ctx = events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.WorkFlowID: workflowName, events.NodeID: s.Target})
	logOptFields, _ := events.FromContext(ctx)
	s.setStatus(tasks.TaskStepStatusINITIAL)
	haveErr := false
	for i := 0; i < len(s.Previous); i++ {
		// Wait for previous be done
		log.Debugf("Step %q waiting for %d previous steps", s.Name, len(s.Previous)-i)
		for {
			select {
			case <-s.NotifyChan:
				log.Debugf("Step %q caught a notification", s.Name)
				goto BR
			case <-shutdownChan:
				log.Printf("Step %q canceled", s.Name)
				s.setStatus(tasks.TaskStepStatusCANCELED)
				return errors.New("workflow canceled")
			case <-ctx.Done():
				log.Printf("Step %q canceled: %q", s.Name, ctx.Err())
				s.setStatus(tasks.TaskStepStatusCANCELED)
				return ctx.Err()
			case <-time.After(30 * time.Second):
				log.Debugf("Step %q still waiting for %d previous steps", s.Name, len(s.Previous)-i)
			}
		}
	BR:
	}
	// First: we check if step is runnable
	if runnable, err := s.isRunnable(); err != nil {
		return err
	} else if !runnable {
		log.Debugf("Deployment %q: Skipping Step %q", deploymentID, s.Name)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.INFO, deploymentID).RegisterAsString(fmt.Sprintf("Skipping Step %q", s.Name))
		s.setStatus(tasks.TaskStepStatusDONE)
		s.notifyNext()
		return nil
	}

	s.setStatus(tasks.TaskStepStatusRUNNING)

	// Create a new context to handle gracefully current step termination when an error occurred during another step
	wfCtx, cancelWf := context.WithCancel(events.NewContext(context.Background(), logOptFields))
	waitDoneCh := make(chan struct{})
	defer close(waitDoneCh)
	go func() {
		select {
		case <-ctx.Done():
			if s.t.status != tasks.CANCELED {
				// Temporize to allow current step termination before cancelling context and put step in error
				log.Printf("An error occurred on another step while step %q is running: trying to gracefully finish it.", s.Name)
				select {
				case <-time.After(cfg.WfStepGracefulTerminationTimeout):
					cancelWf()
					log.Printf("Step %q not yet finished: we set it on error", s.Name)
					s.setStatus(tasks.TaskStepStatusERROR)
					return
				case <-waitDoneCh:
					return
				}
			} else {
				// We immediately cancel the step
				cancelWf()
				s.setStatus(tasks.TaskStepStatusCANCELED)
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
				events.WithContextOptionalFields(ctx).NewLogEntry(events.DEBUG, deploymentID).Registerf("Step %q: error details: %+v", s.Name, err)
				if !bypassErrors {
					s.setStatus(tasks.TaskStepStatusERROR)
					return err
				}
				events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).Registerf("Step %q: Bypassing error: %v", s.Name, err)
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
		s.setStatus(tasks.TaskStepStatusERROR)
		// Return nil otherwise the rest of the workflow will be canceled
		return nil
	}
	log.Debugf("Step %s done without error.", s.Name)
	s.setStatus(tasks.TaskStepStatusDONE)
	return nil
}

func (s *step) runActivity(wfCtx context.Context, kv *api.KV, cfg config.Configuration, deploymentID string, bypassErrors bool, w worker, activity Activity) error {
	// Get activity related instances
	instances, err := tasks.GetInstances(kv, s.t.ID, deploymentID, s.Target)
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
			events.WithContextOptionalFields(events.AddLogOptionalFields(wfCtx, events.LogOptionalFields{events.InstanceID: instanceName})).NewLogEntry(events.DEBUG, deploymentID).RegisterAsString("executing delegate operation")
		}
		err = func() error {
			defer metrics.MeasureSince(metricsutil.CleanupMetricKey([]string{"executor", "delegate", deploymentID, nodeType, delegateOp}), time.Now())
			return provisioner.ExecDelegate(wfCtx, cfg, s.t.ID, deploymentID, s.Target, delegateOp)
		}()

		if err != nil {
			metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "delegate", deploymentID, nodeType, delegateOp, "failures"}), 1)
			for _, instanceName := range instances {
				// TODO: replace this with workflow steps events
				events.WithContextOptionalFields(events.AddLogOptionalFields(wfCtx, events.LogOptionalFields{events.InstanceID: instanceName})).NewLogEntry(events.DEBUG, deploymentID).RegisterAsString("delegate operation failed")
			}
			return err
		}
		metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "delegate", deploymentID, nodeType, delegateOp, "successes"}), 1)
		for _, instanceName := range instances {
			// TODO: replace this with workflow steps events
			events.WithContextOptionalFields(events.AddLogOptionalFields(wfCtx, events.LogOptionalFields{events.InstanceID: instanceName})).NewLogEntry(events.DEBUG, deploymentID).RegisterAsString("delegate operation succeeded")
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
			events.WithContextOptionalFields(events.AddLogOptionalFields(wfCtx, events.LogOptionalFields{events.InstanceID: instanceName})).NewLogEntry(events.DEBUG, deploymentID).RegisterAsString("executing operation")
		}
		err = func() error {
			defer metrics.MeasureSince(metricsutil.CleanupMetricKey([]string{"executor", "operation", deploymentID, nodeType, op.Name}), time.Now())
			return exec.ExecOperation(wfCtx, cfg, s.t.ID, deploymentID, s.Target, op)
		}()
		if err != nil {
			metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "operation", deploymentID, nodeType, op.Name, "failures"}), 1)
			for _, instanceName := range instances {
				// TODO: replace this with workflow steps events
				events.WithContextOptionalFields(events.AddLogOptionalFields(wfCtx, events.LogOptionalFields{events.InstanceID: instanceName})).NewLogEntry(events.DEBUG, deploymentID).RegisterAsString("operation failed")
			}
			return err
		}
		metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "operation", deploymentID, nodeType, op.Name, "successes"}), 1)
		for _, instanceName := range instances {
			// TODO: replace this with workflow steps events
			events.WithContextOptionalFields(events.AddLogOptionalFields(wfCtx, events.LogOptionalFields{events.InstanceID: instanceName})).NewLogEntry(events.DEBUG, deploymentID).RegisterAsString("operation succeeded")
		}
	case ActivityTypeInline:
		if err := w.runWorkflows(wfCtx, s.t, []string{activity.Value()}, bypassErrors); err != nil {
			return err
		}
	}
	return nil
}

func readStep(kv *api.KV, stepsPrefix, stepName string, visitedMap map[string]*visitStep) (*step, error) {
	stepPrefix := stepsPrefix + stepName
	s := &step{Name: stepName, kv: kv, stepPrefix: stepPrefix}
	kvPair, _, err := kv.Get(stepPrefix+"/target_relationship", nil)
	if err != nil {
		return nil, err
	}
	if kvPair != nil {
		s.TargetRelationship = string(kvPair.Value)
	}
	kvPair, _, err = kv.Get(stepPrefix+"/operation_host", nil)
	if err != nil {
		return nil, err
	}
	if kvPair != nil {
		s.OperationHost = string(kvPair.Value)
	}
	if s.TargetRelationship != "" {
		if s.OperationHost == "" {
			return nil, errors.Errorf("Operation host missing for step %s with target relationship %s, this is not allowed", stepName, s.TargetRelationship)
		} else if s.OperationHost != "SOURCE" && s.OperationHost != "TARGET" && s.OperationHost != "ORCHESTRATOR" {
			return nil, errors.Errorf("Invalid value %q for operation host with step %s with target relationship %s : only SOURCE, TARGET and  and ORCHESTRATOR values are accepted", s.OperationHost, stepName, s.TargetRelationship)
		}
	} else if s.OperationHost != "" && s.OperationHost != "SELF" && s.OperationHost != "HOST" && s.OperationHost != "ORCHESTRATOR" {
		return nil, errors.Errorf("Invalid value %q for operation host with step %s : only SELF, HOST and ORCHESTRATOR values are accepted", s.OperationHost, stepName)
	}

	kvKeys, _, err := kv.List(stepPrefix+"/activities/", nil)
	if err != nil {
		return nil, err
	}
	if len(kvKeys) == 0 {
		return nil, errors.Errorf("Activities missing for step %s, this is not allowed", stepName)
	}

	s.Activities = make([]Activity, 0)
	targetIsMandatory := false
	for i, actKV := range kvKeys {
		keyString := strings.TrimPrefix(actKV.Key, stepPrefix+"/activities/"+strconv.Itoa(i)+"/")
		key, err := ParseActivityType(keyString)
		if err != nil {
			return nil, errors.Wrapf(err, "unknown activity for step %q", stepName)
		}
		switch {
		case key == ActivityTypeDelegate:
			s.Activities = append(s.Activities, delegateActivity{delegate: string(actKV.Value)})
			targetIsMandatory = true
		case key == ActivityTypeSetState:
			s.Activities = append(s.Activities, setStateActivity{state: string(actKV.Value)})
			targetIsMandatory = true
		case key == ActivityTypeCallOperation:
			s.Activities = append(s.Activities, callOperationActivity{operation: string(actKV.Value)})
			targetIsMandatory = true
		case key == ActivityTypeInline:
			s.Activities = append(s.Activities, inlineActivity{inline: string(actKV.Value)})
		default:
			return nil, errors.Errorf("Unsupported activity type: %s", key)
		}
	}

	// Get the step's target (mandatory except if all activities are inline)
	kvPair, _, err = kv.Get(stepPrefix+"/target", nil)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	if targetIsMandatory && (kvPair == nil || len(kvPair.Value) == 0) {
		return nil, errors.Errorf("Missing target attribute for step %s", stepName)
	}
	if kvPair != nil && len(kvPair.Value) > 0 {
		s.Target = string(kvPair.Value)
	}

	kvPairs, _, err := kv.List(stepPrefix+"/next", nil)
	if err != nil {
		return nil, err
	}
	s.Next = make([]*step, 0)
	s.Previous = make([]*step, 0)
	for _, nextKV := range kvPairs {
		var nextStep *step
		nextStepName := strings.TrimPrefix(nextKV.Key, stepPrefix+"/next/")
		if visitStep, ok := visitedMap[nextStepName]; ok {
			log.Debugf("Found existing step %s", nextStepName)
			nextStep = visitStep.s
		} else {
			log.Debugf("Reading new step %s from Consul", nextStepName)
			nextStep, err = readStep(kv, stepsPrefix, nextStepName, visitedMap)
			if err != nil {
				return nil, err
			}
		}

		s.Next = append(s.Next, nextStep)
		nextStep.Previous = append(nextStep.Previous, s)
		visitedMap[nextStepName].refCount++
		log.Debugf("RefCount for step %s set to %d", nextStepName, visitedMap[nextStepName].refCount)
	}
	visitedMap[stepName] = &visitStep{refCount: 0, s: s}
	return s, nil

}

// Creates a workflow tree from values stored in Consul at the given prefix.
// It returns roots (starting) Steps.
func readWorkFlowFromConsul(kv *api.KV, wfPrefix string) ([]*step, error) {
	stepsPrefix := wfPrefix + "/steps/"
	stepsPrefixes, _, err := kv.Keys(stepsPrefix, "/", nil)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	steps := make([]*step, 0)
	visitedMap := make(map[string]*visitStep, len(stepsPrefixes))
	for _, stepPrefix := range stepsPrefixes {
		stepName := path.Base(stepPrefix)
		if visitStep, ok := visitedMap[stepName]; !ok {
			step, err := readStep(kv, stepsPrefix, stepName, visitedMap)
			if err != nil {
				return nil, err
			}
			steps = append(steps, step)
		} else {
			steps = append(steps, visitStep.s)
		}
	}

	// build buffered NotifyChan with a size related to the previous steps nb
	for _, s := range steps {
		s.NotifyChan = make(chan struct{}, len(s.Previous))
	}
	return steps, nil
}
