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

package dispatcher

import (
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/log"
	"path"
	"strconv"
	"strings"
)

// ReadWorkFlow creates a workflow tree from values stored in Consul at the given prefix.
// It returns roots (starting) Steps.
func ReadWorkFlow(kv *api.KV, wfKey string) ([]*Step, error) {
	stepsPrefix := wfKey + "/steps/"
	stepsKeys, _, err := kv.Keys(stepsPrefix, "/", nil)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	steps := make([]*Step, 0)
	visitedMap := make(map[string]*Step, len(stepsKeys))
	for _, stepPrefix := range stepsKeys {
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

func readStep(kv *api.KV, stepsPrefix, stepName string, visitedMap map[string]*Step) (*Step, error) {
	if visitedMap == nil {
		visitedMap = make(map[string]*Step, 0)
	}

	stepPrefix := stepsPrefix + stepName
	s := &Step{Name: stepName, kv: kv, stepPrefix: stepPrefix}
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
			return nil, errors.Errorf("Operation host missing for Step %s with target relationship %s, this is not allowed", stepName, s.TargetRelationship)
		} else if s.OperationHost != "SOURCE" && s.OperationHost != "TARGET" && s.OperationHost != "ORCHESTRATOR" {
			return nil, errors.Errorf("Invalid value %q for operation host with Step %s with target relationship %s : only SOURCE, TARGET and  and ORCHESTRATOR values are accepted", s.OperationHost, stepName, s.TargetRelationship)
		}
	} else if s.OperationHost != "" && s.OperationHost != "SELF" && s.OperationHost != "HOST" && s.OperationHost != "ORCHESTRATOR" {
		return nil, errors.Errorf("Invalid value %q for operation host with Step %s : only SELF, HOST and ORCHESTRATOR values are accepted", s.OperationHost, stepName)
	}

	kvKeys, _, err := kv.List(stepPrefix+"/activities/", nil)
	if err != nil {
		return nil, err
	}
	if len(kvKeys) == 0 {
		return nil, errors.Errorf("Activities missing for Step %s, this is not allowed", stepName)
	}

	s.Activities = make([]Activity, 0)
	targetIsMandatory := false
	for i, actKV := range kvKeys {
		keyString := strings.TrimPrefix(actKV.Key, stepPrefix+"/activities/"+strconv.Itoa(i)+"/")
		key, err := ParseActivityType(keyString)
		if err != nil {
			return nil, errors.Wrapf(err, "unknown activity for Step %q", stepName)
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

	// Get the Step's target (mandatory except if all activities are inline)
	kvPair, _, err = kv.Get(stepPrefix+"/target", nil)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	if targetIsMandatory && (kvPair == nil || len(kvPair.Value) == 0) {
		return nil, errors.Errorf("Missing target attribute for Step %s", stepName)
	}
	if kvPair != nil && len(kvPair.Value) > 0 {
		s.Target = string(kvPair.Value)
	}

	kvPairs, _, err := kv.List(stepPrefix+"/next", nil)
	if err != nil {
		return nil, err
	}
	s.Next = make([]*Step, 0)
	s.Previous = make([]*Step, 0)
	for _, nextKV := range kvPairs {
		var nextStep *Step
		nextStepName := strings.TrimPrefix(nextKV.Key, stepPrefix+"/next/")
		if visitStep, ok := visitedMap[nextStepName]; ok {
			log.Debugf("Found existing Step %s", nextStepName)
			nextStep = visitStep
		} else {
			log.Debugf("Reading new Step %s from Consul", nextStepName)
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

func getCallOperationsFromStep(s *Step) []string {
	ops := make([]string, 0)
	for _, a := range s.Activities {
		if a.Type() == ActivityTypeCallOperation {
			ops = append(ops, a.Value())
		}
	}
	return ops
}
