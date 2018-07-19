package workflow

import (
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"strings"
	"strconv"
	"path"
	"github.com/ystia/yorc/log"
)
func readStep(kv *api.KV, stepsPrefix, stepName string, visitedMap map[string]*step) (*step, error) {
	if visitedMap == nil {
		visitedMap = make(map[string]*step, 0)
	}

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
			nextStep = visitStep
		} else {
			log.Debugf("Reading new step %s from Consul", nextStepName)
			nextStep, err = readStep(kv, stepsPrefix, nextStepName, visitedMap)
			if err != nil {
				return nil, err
			}
			visitedMap[nextStepName] = nextStep
		}

		s.Next = append(s.Next, nextStep)
		nextStep.Previous = append(nextStep.Previous, s)
	}
	return s, nil

}

// ReadWorkFlowFromConsul creates a workflow tree from values stored in Consul at the given prefix.
// It returns roots (starting) Steps.
func ReadWorkFlowFromConsul(kv *api.KV, wfPrefix string) ([]*step, error) {
	stepsPrefix := wfPrefix + "/steps/"
	stepsPrefixes, _, err := kv.Keys(stepsPrefix, "/", nil)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	steps := make([]*step, 0)
	visitedMap := make(map[string]*step, len(stepsPrefixes))
	for _, stepPrefix := range stepsPrefixes {
		stepName := path.Base(stepPrefix)
		if visitStep, ok := visitedMap[stepName]; !ok {
			step, err := readStep(kv, stepsPrefix, stepName, visitedMap)
			if err != nil {
				return nil, err
			}
			steps = append(steps, step)
		} else {
			steps = append(steps, visitStep)
		}
	}
	return steps, nil
}

func getCallOperationsFromStep(s *step) []string {
	ops := make([]string, 0)
	for _, a := range s.Activities {
		if a.Type() == ActivityTypeCallOperation {
			ops = append(ops, a.Value())
		}
	}
	return ops
}
