package tasks

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/ansible"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform"
)

var wfCanceled = errors.New("workflow canceled")

type Step struct {
	Name       string
	Node       string
	Activities []Activity
	Next       []*Step
	Previous   []*Step
	NotifyChan chan struct{}
	kv         *api.KV
	stepPrefix string
}

type Activity interface {
	ActivityType() string
	ActivityValue() string
}

type DelegateActivity struct {
	delegate string
}

func (da DelegateActivity) ActivityType() string {
	return "delegate"
}
func (da DelegateActivity) ActivityValue() string {
	return da.delegate
}

type SetStateActivity struct {
	state string
}

func (s SetStateActivity) ActivityType() string {
	return "set-state"
}
func (s SetStateActivity) ActivityValue() string {
	return s.state
}

type CallOperationActivity struct {
	operation string
}

func (c CallOperationActivity) ActivityType() string {
	return "call-operation"
}
func (c CallOperationActivity) ActivityValue() string {
	return c.operation
}

func (s *Step) IsTerminal() bool {
	return len(s.Next) == 0
}

func (s *Step) notifyNext() {
	for _, next := range s.Next {
		log.Debugf("Step %q, notifying step %q", s.Name, next.Name)
		next.NotifyChan <- struct{}{}
	}
}

type visitStep struct {
	refCount int
	step     *Step
}

func (s *Step) setStatus(status string) error {
	kvp := &api.KVPair{Key: path.Join(s.stepPrefix, "status"), Value: []byte(status)}
	_, err := s.kv.Put(kvp, nil)
	return err
}

func (s *Step) isRunnable() (bool, error) {
	kvp, _, err := s.kv.Get(path.Join(s.stepPrefix, "status"), nil)
	if err != nil {
		return false, err
	}

	if kvp == nil || len(kvp.Value) == 0 || string(kvp.Value) != "done" {
		return true, nil
	}
	return false, nil
}

func setNodeStatus(kv *api.KV, eventPub events.Publisher, deploymentId, nodeName, status string) {
	kv.Put(&api.KVPair{Key: path.Join(deployments.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeName, "status"), Value: []byte(status)}, nil)
	ids, _ := deployments.GetNodeInstancesIds(kv, deploymentId, nodeName)
	for _, id := range ids {
		kv.Put(&api.KVPair{Key: path.Join(deployments.DeploymentKVPrefix, deploymentId, "topology/instances", nodeName, id, "status"), Value: []byte(status)}, nil)
	}
	// Publish status change event
	eventPub.StatusChange(nodeName, status)
}

func (s *Step) run(ctx context.Context, deploymentId string, kv *api.KV, uninstallerrc chan error, shutdownChan chan struct{}, cfg config.Configuration, isUndeploy bool) error {
	haveErr := false
	eventPub := events.NewPublisher(kv, deploymentId)
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
				s.setStatus("canceled")
				return wfCanceled
			case <-ctx.Done():
				log.Printf("Step %q canceled: %q", s.Name, ctx.Err())
				s.setStatus("canceled")
				return ctx.Err()
			case <-time.After(30 * time.Second):
				log.Debugf("Step %q still waiting for %d previous steps", s.Name, len(s.Previous)-i)
			}
		}
	BR:
	}

	if runnable, err := s.isRunnable(); err != nil {
		return err
	} else if !runnable {
		log.Printf("Deployment %q: Skipping Step %q", deploymentId, s.Name)
		deployments.LogInConsul(kv, deploymentId, fmt.Sprintf("Skipping Step %q", s.Name))
		s.setStatus("done")
		s.notifyNext()
		return nil
	}

	log.Debugf("Processing step %q", s.Name)
	for _, activity := range s.Activities {
		actType := activity.ActivityType()
		switch {
		case actType == "delegate":
			provisioner := terraform.NewExecutor(kv, cfg)
			delegateOp := activity.ActivityValue()
			switch delegateOp {
			case "install":
				if err := provisioner.ProvisionNode(ctx, deploymentId, s.Node); err != nil {
					setNodeStatus(kv, eventPub, deploymentId, s.Node, "error")
					log.Printf("Deployment %q, Step %q: Sending error %v to error channel", deploymentId, s.Name, err)
					s.setStatus("error")
					return err
				}
				setNodeStatus(kv, eventPub, deploymentId, s.Node, "started")
			case "uninstall":
				if err := provisioner.DestroyNode(ctx, deploymentId, s.Node); err != nil {
					setNodeStatus(kv, eventPub, deploymentId, s.Node, "error")
					uninstallerrc <- err
					haveErr = true
				} else {
					setNodeStatus(kv, eventPub, deploymentId, s.Node, "deleted")
				}
			default:
				setNodeStatus(kv, eventPub, deploymentId, s.Node, "error")
				s.setStatus("error")
				return fmt.Errorf("Unsupported delegate operation '%s' for step '%s'", delegateOp, s.Name)
			}
		case actType == "set-state":
			setNodeStatus(kv, eventPub, deploymentId, s.Node, activity.ActivityValue())
		case actType == "call-operation":
			exec := ansible.NewExecutor(kv)
			if err := exec.ExecOperation(ctx, deploymentId, s.Node, activity.ActivityValue()); err != nil {
				setNodeStatus(kv, eventPub, deploymentId, s.Node, "error")
				log.Printf("Deployment %q, Step %q: Sending error %v to error channel", deploymentId, s.Name, err)
				if isUndeploy {
					uninstallerrc <- err
				} else {
					s.setStatus("error")
					return err
				}
			}
		}
	}

	s.notifyNext()

	if haveErr {
		log.Debug("Step %s generate an error but workflow continue", s.Name)
		s.setStatus("error")
		// Return nil otherwise the rest of the workflow will be canceled
		return nil
	}
	log.Debugf("Step %s done without error.", s.Name)
	s.setStatus("done")
	return nil
}

func readStep(kv *api.KV, stepsPrefix, stepName string, visitedMap map[string]*visitStep) (*Step, error) {
	stepPrefix := stepsPrefix + stepName
	step := &Step{Name: stepName, kv: kv, stepPrefix: stepPrefix}
	kvPair, _, err := kv.Get(stepPrefix+"/node", nil)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	if kvPair == nil {
		return nil, fmt.Errorf("Missing node attribute for step %s", stepName)
	}
	step.Node = string(kvPair.Value)

	kvPairs, _, err := kv.List(stepPrefix+"/activity", nil)
	if err != nil {
		return nil, err
	}
	if len(kvPairs) == 0 {
		return nil, fmt.Errorf("Activity missing for step %s, this is not allowed.", stepName)
	}
	step.Activities = make([]Activity, 0)
	for _, actKV := range kvPairs {
		key := strings.TrimPrefix(actKV.Key, stepPrefix+"/activity/")
		switch {
		case key == "delegate":
			step.Activities = append(step.Activities, DelegateActivity{delegate: string(actKV.Value)})
		case key == "set-state":
			step.Activities = append(step.Activities, SetStateActivity{state: string(actKV.Value)})
		case key == "operation":
			step.Activities = append(step.Activities, CallOperationActivity{operation: string(actKV.Value)})
		default:
			return nil, fmt.Errorf("Unsupported activity type: %s", key)
		}
	}
	step.NotifyChan = make(chan struct{})

	kvPairs, _, err = kv.List(stepPrefix+"/next", nil)
	if err != nil {
		return nil, err
	}
	step.Next = make([]*Step, 0)
	step.Previous = make([]*Step, 0)
	for _, nextKV := range kvPairs {
		var nextStep *Step
		nextStepName := strings.TrimPrefix(nextKV.Key, stepPrefix+"/next/")
		if visitStep, ok := visitedMap[nextStepName]; ok {
			log.Debugf("Found existing step %s", nextStepName)
			nextStep = visitStep.step
		} else {
			log.Debugf("Reading new step %s from Consul", nextStepName)
			nextStep, err = readStep(kv, stepsPrefix, nextStepName, visitedMap)
			if err != nil {
				return nil, err
			}
		}

		step.Next = append(step.Next, nextStep)
		nextStep.Previous = append(nextStep.Previous, step)
		visitedMap[nextStepName].refCount++
		log.Debugf("RefCount for step %s set to %d", nextStepName, visitedMap[nextStepName].refCount)
	}
	visitedMap[stepName] = &visitStep{refCount: 0, step: step}
	return step, nil

}

// Creates a workflow tree from values stored in Consul at the given prefix.
// It returns roots (starting) Steps.
func readWorkFlowFromConsul(kv *api.KV, wfPrefix string) ([]*Step, error) {
	stepsPrefix := wfPrefix + "/steps/"
	stepsPrefixes, _, err := kv.Keys(stepsPrefix, "/", nil)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	steps := make([]*Step, 0)
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
			steps = append(steps, visitStep.step)
		}
	}

	return steps, nil
}
