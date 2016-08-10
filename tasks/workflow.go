package tasks

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"golang.org/x/net/context"
	"novaforge.bull.com/starlings-janus/janus/commands/jconfig"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/ansible"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform"
	"path"
	"strings"
	"sync"
)

type Step struct {
	Name       string
	Node       string
	Activities []Activity
	Next       []*Step
	Previous   []*Step
	NotifyChan chan struct{}
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

type visitStep struct {
	refCount int
	step     *Step
}

func (s *Step) run(ctx context.Context, deploymentId string, wg *sync.WaitGroup, kv *api.KV, errc chan error, shutdownChan chan struct{}, cfg jconfig.Configuration) {

	defer wg.Done()

	for i := 0; i < len(s.Previous); i++ {
		// Wait for previous be done
		select {
		case <-s.NotifyChan:
			log.Debugf("Step %q caught a notification", s.Name)
		case <-shutdownChan:
			log.Printf("Step %q cancelled", s.Name)
			return
		case <-ctx.Done():
			log.Printf("Step %q cancelled: %q", s.Name, ctx.Err())
			return
		}
	}

	log.Debugf("Processing step %s", s.Name)
	for _, activity := range s.Activities {
		actType := activity.ActivityType()
		switch {
		case actType == "delegate":
			provisioner := terraform.NewExecutor(kv, cfg)
			delegateOp := activity.ActivityValue()
			switch delegateOp {
			case "install":
				if err := provisioner.ProvisionNode(deploymentId, s.Node); err != nil {
					log.Printf("Sending error %v to error channel", err)
					errc <- err
					return
				}
			case "uninstall":
				if err := provisioner.DestroyNode(deploymentId, s.Node); err != nil {
					errc <- err
					return
				}
			default:
				errc <- fmt.Errorf("Unsupported delegate operation '%s' for step '%s'", delegateOp, s.Name)
				return
			}
		case actType == "set-state":
			kv.Put(&api.KVPair{Key: path.Join(deployments.DeploymentKVPrefix, deploymentId, "topology/nodes", s.Node, "status"), Value: []byte(activity.ActivityValue())}, nil)
		case actType == "call-operation":
			exec := ansible.NewExecutor(kv)
			if err := exec.ExecOperation(deploymentId, s.Node, activity.ActivityValue()); err != nil {
				errc <- err
				return
			}
		}
	}
	for _, next := range s.Next {
		next.NotifyChan <- struct{}{}
	}
	log.Debugf("Step %s done without error.", s.Name)
}

func readStep(kv *api.KV, stepsPrefix, stepName string, visitedMap map[string]*visitStep) (*Step, error) {
	stepPrefix := stepsPrefix + stepName
	step := &Step{Name: stepName}
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
	potentialRoots := make([]*Step, 0)
	visitedMap := make(map[string]*visitStep, len(stepsPrefixes))
	for _, stepPrefix := range stepsPrefixes {
		stepFields := strings.FieldsFunc(stepPrefix, func(r rune) bool {
			return r == rune('/')
		})
		stepName := stepFields[len(stepFields)-1]
		if _, ok := visitedMap[stepName]; !ok {
			step, err := readStep(kv, stepsPrefix, stepName, visitedMap)
			if err != nil {
				return nil, err
			}
			potentialRoots = append(potentialRoots, step)
		}
	}

	roots := potentialRoots[:0]
	for _, step := range potentialRoots {
		if visitedMap[step.Name].refCount == 0 {
			roots = append(roots, step)
		}
	}

	return roots, nil
}
