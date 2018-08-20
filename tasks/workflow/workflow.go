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
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"

	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
)

// buildWorkFlow creates a workflow tree from values for a specified workflow name and deploymentID
func buildWorkFlow(kv *api.KV, deploymentID, wfName string) ([]*step, error) {
	stepsPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "workflows", wfName, "steps")
	stepsKeys, _, err := kv.Keys(stepsPath+"/", "/", nil)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	steps := make([]*step, 0)
	visitedMap := make(map[string]*visitStep, len(stepsKeys))
	for _, stepPrefix := range stepsKeys {
		stepName := path.Base(stepPrefix)
		if visitStep, ok := visitedMap[stepName]; !ok {
			step, err := buildStep(kv, deploymentID, wfName, stepName, visitedMap)
			if err != nil {
				return nil, err
			}
			steps = append(steps, step)
		} else {
			steps = append(steps, visitStep.s)
		}
	}
	return steps, nil
}

// buildStep returns a representation of the workflow step
func buildStep(kv *api.KV, deploymentID, wfName, stepName string, visitedMap map[string]*visitStep) (*step, error) {
	stepPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "workflows", wfName, "steps", stepName)
	if visitedMap == nil {
		visitedMap = make(map[string]*visitStep)
	}

	s := &step{name: stepName, kv: kv, workflowName: wfName}
	kvPair, _, err := kv.Get(stepPath+"/target_relationship", nil)
	if err != nil {
		return nil, err
	}
	if kvPair != nil {
		s.targetRelationship = string(kvPair.Value)
	}
	kvPair, _, err = kv.Get(stepPath+"/operation_host", nil)
	if err != nil {
		return nil, err
	}
	if kvPair != nil {
		s.operationHost = string(kvPair.Value)
	}
	if s.targetRelationship != "" {
		if s.operationHost == "" {
			return nil, errors.Errorf("Operation host missing for Step %s with target relationship %s, this is not allowed", stepName, s.targetRelationship)
		} else if s.operationHost != "SOURCE" && s.operationHost != "TARGET" && s.operationHost != "ORCHESTRATOR" {
			return nil, errors.Errorf("Invalid value %q for operation host with Step %s with target relationship %s : only SOURCE, TARGET and  and ORCHESTRATOR values are accepted", s.operationHost, stepName, s.targetRelationship)
		}
	} else if s.operationHost != "" && s.operationHost != "SELF" && s.operationHost != "HOST" && s.operationHost != "ORCHESTRATOR" {
		return nil, errors.Errorf("Invalid value %q for operation host with Step %s : only SELF, HOST and ORCHESTRATOR values are accepted", s.operationHost, stepName)
	}

	kvKeys, _, err := kv.List(stepPath+"/activities/", nil)
	if err != nil {
		return nil, err
	}
	if len(kvKeys) == 0 {
		return nil, errors.Errorf("Activities missing for Step %s, this is not allowed", stepName)
	}

	s.activities = make([]Activity, 0)
	targetIsMandatory := false
	for i, actKV := range kvKeys {
		keyString := strings.TrimPrefix(actKV.Key, stepPath+"/activities/"+strconv.Itoa(i)+"/")
		key, err := ParseActivityType(keyString)
		if err != nil {
			return nil, errors.Wrapf(err, "unknown activity for Step %q", stepName)
		}
		switch {
		case key == ActivityTypeDelegate:
			s.activities = append(s.activities, delegateActivity{delegate: string(actKV.Value)})
			targetIsMandatory = true
		case key == ActivityTypeSetState:
			s.activities = append(s.activities, setStateActivity{state: string(actKV.Value)})
			targetIsMandatory = true
		case key == ActivityTypeCallOperation:
			s.activities = append(s.activities, callOperationActivity{operation: string(actKV.Value)})
			targetIsMandatory = true
		case key == ActivityTypeInline:
			s.activities = append(s.activities, inlineActivity{inline: string(actKV.Value)})
		default:
			return nil, errors.Errorf("Unsupported activity type: %s", key)
		}
	}

	// Get the Step's target (mandatory except if all activities are inline)
	kvPair, _, err = kv.Get(stepPath+"/target", nil)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	if targetIsMandatory && (kvPair == nil || len(kvPair.Value) == 0) {
		return nil, errors.Errorf("Missing target attribute for Step %s", stepName)
	}
	if kvPair != nil && len(kvPair.Value) > 0 {
		s.target = string(kvPair.Value)
	}

	kvPairs, _, err := kv.List(stepPath+"/next", nil)
	if err != nil {
		return nil, err
	}
	s.next = make([]*step, 0)
	s.previous = make([]*step, 0)
	for _, nextKV := range kvPairs {
		var nextStep *step
		nextStepName := strings.TrimPrefix(nextKV.Key, stepPath+"/next/")
		if visitStep, ok := visitedMap[nextStepName]; ok {
			nextStep = visitStep.s
		} else {
			nextStep, err = buildStep(kv, deploymentID, wfName, nextStepName, visitedMap)
			if err != nil {
				return nil, err
			}
		}

		s.next = append(s.next, nextStep)
		nextStep.previous = append(nextStep.previous, s)
		visitedMap[nextStepName].refCount++
	}
	visitedMap[stepName] = &visitStep{refCount: 0, s: s}
	return s, nil

}

// BuildInitExecutionOperations returns Consul transactional KV operations for initiating workflow execution
func BuildInitExecutionOperations(kv *api.KV, deploymentID, taskID, workflowName string, registerWorkflow bool) (api.KVTxnOps, error) {
	//FIXME number of operations in a Consul transaction is limited to 64: it can be an issue for large workflow. Need to increase this param if configurable or split transactions
	ops := make(api.KVTxnOps, 0)
	steps, err := buildWorkFlow(kv, deploymentID, workflowName)
	if err != nil {
		return nil, err
	}

	for _, step := range steps {
		if registerWorkflow {
			// Register workflow step to handle step statuses for all steps
			ops = append(ops, &api.KVTxnOp{
				Verb:  api.KVSet,
				Key:   path.Join(consulutil.WorkflowsPrefix, taskID, step.name),
				Value: []byte(tasks.StepStatusINITIAL.String()),
			})
		}

		// Add execution key for initial steps only
		if step.isInitial() {
			execID := fmt.Sprint(uuid.NewV4())
			log.Debugf("Create initial task execution with ID:%q, taskID:%q and step:%q", execID, taskID, step.name)
			stepExecPath := path.Join(consulutil.ExecutionsTaskPrefix, execID)
			stepOps := api.KVTxnOps{
				&api.KVTxnOp{
					Verb:  api.KVSet,
					Key:   path.Join(stepExecPath, "taskID"),
					Value: []byte(taskID),
				},
				&api.KVTxnOp{
					Verb:  api.KVSet,
					Key:   path.Join(stepExecPath, "step"),
					Value: []byte(step.name),
				},
			}
			ops = append(ops, stepOps...)
		}
	}
	return ops, nil
}

// CreateWorkflowStepsOperations returns Consul transactional KV operations for initiating workflow execution
func CreateWorkflowStepsOperations(taskID string, steps []*step) api.KVTxnOps {
	ops := make(api.KVTxnOps, 0)
	var stepOps api.KVTxnOps
	for _, step := range steps {
		// Add execution key for initial steps only
		execID := fmt.Sprint(uuid.NewV4())
		log.Debugf("Create task execution with ID:%q, taskID:%q and step:%q", execID, taskID, step.name)
		stepExecPath := path.Join(consulutil.ExecutionsTaskPrefix, execID)
		stepOps = api.KVTxnOps{
			&api.KVTxnOp{
				Verb:  api.KVSet,
				Key:   path.Join(stepExecPath, "taskID"),
				Value: []byte(taskID),
			},
			&api.KVTxnOp{
				Verb:  api.KVSet,
				Key:   path.Join(stepExecPath, "step"),
				Value: []byte(step.name),
			},
		}
		ops = append(ops, stepOps...)
	}
	return ops
}

func getCallOperationsFromStep(s *step) []string {
	ops := make([]string, 0)
	for _, a := range s.activities {
		if a.Type() == ActivityTypeCallOperation {
			ops = append(ops, a.Value())
		}
	}
	return ops
}
