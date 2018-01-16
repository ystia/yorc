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
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/helper/metricsutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/operations"
	"novaforge.bull.com/starlings-janus/janus/registry"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"strconv"
)

const wfDelegateActivity = "delegate"
const wfSetStateActivity = "set-state"
const wfCallOpActivity = "call-operation"
const wfInlineActivity = "inline"

type step struct {
	Name               string
	Target             string
	TargetRelationship string
	OperationHost      string
	Activities         []activity
	Next               []*step
	Previous           []*step
	NotifyChan         chan struct{}
	kv                 *api.KV
	stepPrefix         string
	t                  *task
}

type activity interface {
	ActivityType() string
	ActivityValue() string
}

type delegateActivity struct {
	delegate string
}

func (da delegateActivity) ActivityType() string {
	return wfDelegateActivity
}
func (da delegateActivity) ActivityValue() string {
	return da.delegate
}

type setStateActivity struct {
	state string
}

func (s setStateActivity) ActivityType() string {
	return wfSetStateActivity
}
func (s setStateActivity) ActivityValue() string {
	return s.state
}

type callOperationActivity struct {
	operation string
}

func (c callOperationActivity) ActivityType() string {
	return wfCallOpActivity
}
func (c callOperationActivity) ActivityValue() string {
	return c.operation
}

type inlineActivity struct {
	inline string
}

func (i inlineActivity) ActivityType() string {
	return wfInlineActivity
}
func (i inlineActivity) ActivityValue() string {
	return i.inline
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
// And for ScaleUp and ScaleDown it checks if the node or the target node in case of an operation running on the target node is part of the operation
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

	if s.t.TaskType == tasks.ScaleUp || s.t.TaskType == tasks.ScaleDown {
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

func setNodeStatus(kv *api.KV, taskID, deploymentID, nodeName, status string) error {
	instancesIDs, err := tasks.GetInstances(kv, taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}

	for _, id := range instancesIDs {
		// Publish status change event
		err := deployments.SetInstanceStateString(kv, deploymentID, nodeName, id, status)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *step) run(ctx context.Context, deploymentID string, kv *api.KV, ignoredErrsChan chan error, shutdownChan chan struct{}, cfg config.Configuration, bypassErrors bool, workflowName string, w worker) error {
	// Fill log optional fields for log registration
	var logOptFields events.LogOptionalFields
	if s.Target != "" {
		logOptFields = events.LogOptionalFields{
			events.WorkFlowID: workflowName,
			events.NodeID:     s.Target,
		}
	} else {
		logOptFields = events.LogOptionalFields{
			events.WorkFlowID: workflowName,
		}
	}

	// First: we check if step is runnable
	if runnable, err := s.isRunnable(); err != nil {
		return err
	} else if !runnable {
		log.Debugf("Deployment %q: Skipping Step %q", deploymentID, s.Name)
		events.WithOptionalFields(logOptFields).NewLogEntry(events.INFO, deploymentID).RegisterAsString(fmt.Sprintf("Skipping Step %q", s.Name))
		s.setStatus(tasks.TaskStepStatusDONE)
		s.notifyNext()
		return nil
	}

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
	s.setStatus(tasks.TaskStepStatusRUNNING)

	// Create a new context to handle gracefully current step termination when an error occurred during another step
	wfCtx, cancelWf := context.WithCancel(context.Background())
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
		actType := activity.ActivityType()
		switch actType {
		case wfDelegateActivity:
			nodeType, err := deployments.GetNodeType(kv, deploymentID, s.Target)
			if err != nil {
				setNodeStatus(kv, s.t.ID, deploymentID, s.Target, tosca.NodeStateError.String())
				log.Printf("Deployment %q, Step %q: Sending error %v to error channel", deploymentID, s.Name, err)
				log.Debugf("Deployment %q, Step %q: error details: %+v", deploymentID, s.Name, err)
				s.setStatus(tasks.TaskStepStatusERROR)
				if bypassErrors {
					ignoredErrsChan <- err
					haveErr = true
				} else {
					return err
				}
			}
			provisioner, err := registry.GetRegistry().GetDelegateExecutor(nodeType)
			if err != nil {
				setNodeStatus(kv, s.t.ID, deploymentID, s.Target, tosca.NodeStateError.String())
				log.Printf("Deployment %q, Step %q: Sending error %v to error channel", deploymentID, s.Name, err)
				log.Debugf("Deployment %q, Step %q: error details: %+v", deploymentID, s.Name, err)
				s.setStatus(tasks.TaskStepStatusERROR)
				if bypassErrors {
					ignoredErrsChan <- err
					haveErr = true
				} else {
					return err
				}
			} else {
				delegateOp := activity.ActivityValue()
				err := func() error {
					defer metrics.MeasureSince(metricsutil.CleanupMetricKey([]string{"executor", "delegate", deploymentID, nodeType, delegateOp}), time.Now())
					return provisioner.ExecDelegate(wfCtx, cfg, s.t.ID, deploymentID, s.Target, delegateOp)
				}()

				if err != nil {
					metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "delegate", deploymentID, nodeType, delegateOp, "failures"}), 1)
					setNodeStatus(kv, s.t.ID, deploymentID, s.Target, tosca.NodeStateError.String())
					log.Printf("Deployment %q, Step %q: Sending error %v to error channel", deploymentID, s.Name, err)
					log.Debugf("Deployment %q, Step %q: error details: %+v", deploymentID, s.Name, err)
					s.setStatus(tasks.TaskStepStatusERROR)
					if bypassErrors {
						ignoredErrsChan <- err
						haveErr = true
					} else {
						return err
					}
				} else {
					metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "delegate", deploymentID, nodeType, delegateOp, "successes"}), 1)
				}
			}
		case wfSetStateActivity:
			setNodeStatus(kv, s.t.ID, deploymentID, s.Target, activity.ActivityValue())
		case wfCallOpActivity:
			op, err := operations.GetOperation(kv, s.t.TargetID, s.Target, activity.ActivityValue(), s.TargetRelationship, s.OperationHost)
			if err != nil {
				if deployments.IsOperationNotImplemented(err) {
					// Operation not implemented just skip it
					log.Debugf("Voluntary bypassing error: %s.", err.Error())
					continue
				}

				setNodeStatus(kv, s.t.ID, deploymentID, s.Target, tosca.NodeStateError.String())
				log.Printf("Deployment %q, Step %q: Sending error %v to error channel", deploymentID, s.Name, err)
				log.Debugf("Deployment %q, Step %q: error details: %+v", deploymentID, s.Name, err)
				if bypassErrors {
					ignoredErrsChan <- err
					haveErr = true
				} else {
					s.setStatus(tasks.TaskStepStatusERROR)
					return err
				}
			}

			exec, err := getOperationExecutor(kv, deploymentID, op.ImplementationArtifact)
			if err != nil {
				setNodeStatus(kv, s.t.ID, deploymentID, s.Target, tosca.NodeStateError.String())
				log.Debugf("Deployment %q, Step %q: error details: %+v", deploymentID, s.Name, err)
				log.Printf("Deployment %q, Step %q: Sending error %v to error channel", deploymentID, s.Name, err)
				if bypassErrors {
					ignoredErrsChan <- err
					haveErr = true
				} else {
					s.setStatus(tasks.TaskStepStatusERROR)
					return err
				}
			} else {
				nodeType, err := deployments.GetNodeType(kv, deploymentID, s.Target)
				if err != nil {
					setNodeStatus(kv, s.t.ID, deploymentID, s.Target, tosca.NodeStateError.String())
					log.Printf("Deployment %q, Step %q: Sending error %v to error channel", deploymentID, s.Name, err)
					log.Debugf("Deployment %q, Step %q: error details: %+v", deploymentID, s.Name, err)
					s.setStatus(tasks.TaskStepStatusERROR)
					if bypassErrors {
						ignoredErrsChan <- err
						haveErr = true
					} else {
						return err
					}
				}
				err = func() error {
					defer metrics.MeasureSince(metricsutil.CleanupMetricKey([]string{"executor", "operation", deploymentID, nodeType, op.Name}), time.Now())
					return exec.ExecOperation(wfCtx, cfg, s.t.ID, deploymentID, s.Target, op)
				}()
				if err != nil {
					metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "operation", deploymentID, nodeType, op.Name, "failures"}), 1)
					setNodeStatus(kv, s.t.ID, deploymentID, s.Target, tosca.NodeStateError.String())
					log.Debugf("Deployment %q, Step %q: error details: %+v", deploymentID, s.Name, err)
					log.Printf("Deployment %q, Step %q: Sending error %v to error channel", deploymentID, s.Name, err)
					if bypassErrors {
						ignoredErrsChan <- err
						haveErr = true
					} else {
						s.setStatus(tasks.TaskStepStatusERROR)
						return err
					}
				} else {
					metrics.IncrCounter(metricsutil.CleanupMetricKey([]string{"executor", "operation", deploymentID, nodeType, op.Name, "successes"}), 1)
				}
			}
		case wfInlineActivity:
			if err := w.runWorkflows(ctx, s.t, []string{activity.ActivityValue()}, bypassErrors); err != nil {
				return err
			}
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
		} else if s.OperationHost != "SOURCE" && s.OperationHost != "TARGET" {
			return nil, errors.Errorf("Invalid value %q for operation host with step %s with target relationship %s : only SOURCE or TARGET values are accepted", s.OperationHost, stepName, s.TargetRelationship)
		}
	} else if s.OperationHost != "" && s.OperationHost != "SELF" && s.OperationHost != "HOST" {
		return nil, errors.Errorf("Invalid value %q for operation host with step %s : only SELF or HOST values are accepted", s.OperationHost, stepName)
	}

	kvKeys, _, err := kv.List(stepPrefix+"/activities/", nil)
	if err != nil {
		return nil, err
	}
	if len(kvKeys) == 0 {
		return nil, errors.Errorf("Activities missing for step %s, this is not allowed", stepName)
	}

	s.Activities = make([]activity, 0)
	targetIsMandatory := false
	for i, actKV := range kvKeys {
		key := strings.TrimPrefix(actKV.Key, stepPrefix+"/activities/"+strconv.Itoa(i)+"/")
		switch {
		case key == wfDelegateActivity:
			s.Activities = append(s.Activities, delegateActivity{delegate: string(actKV.Value)})
			targetIsMandatory = true
		case key == wfSetStateActivity:
			s.Activities = append(s.Activities, setStateActivity{state: string(actKV.Value)})
			targetIsMandatory = true
		case key == wfCallOpActivity:
			s.Activities = append(s.Activities, callOperationActivity{operation: string(actKV.Value)})
			targetIsMandatory = true
		case key == wfInlineActivity:
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
