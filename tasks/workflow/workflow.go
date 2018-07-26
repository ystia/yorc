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
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
	"path"
	"strconv"
	"strings"
)

// BuildWorkFlow creates a workflow tree from values for a specified workflow name and deploymentID
func BuildWorkFlow(kv *api.KV, deploymentID, wfName string) ([]*Step, error) {
	stepsPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "workflows", wfName, "steps")
	stepsKeys, _, err := kv.Keys(stepsPath+"/", "/", nil)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	steps := make([]*Step, 0)
	visitedMap := make(map[string]*visitStep, len(stepsKeys))
	for _, stepPrefix := range stepsKeys {
		stepName := path.Base(stepPrefix)
		if visitStep, ok := visitedMap[stepName]; !ok {
			step, err := BuildStep(kv, deploymentID, wfName, stepName, visitedMap)
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

// BuildStep returns a representation of the workflow step
func BuildStep(kv *api.KV, deploymentID, wfName, stepName string, visitedMap map[string]*visitStep) (*Step, error) {
	stepPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "workflows", wfName, "steps", stepName)
	if visitedMap == nil {
		visitedMap = make(map[string]*visitStep)
	}

	s := &Step{Name: stepName, kv: kv, workflowName: wfName}
	kvPair, _, err := kv.Get(stepPath+"/target_relationship", nil)
	if err != nil {
		return nil, err
	}
	if kvPair != nil {
		s.TargetRelationship = string(kvPair.Value)
	}
	kvPair, _, err = kv.Get(stepPath+"/operation_host", nil)
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

	kvKeys, _, err := kv.List(stepPath+"/activities/", nil)
	if err != nil {
		return nil, err
	}
	if len(kvKeys) == 0 {
		return nil, errors.Errorf("Activities missing for Step %s, this is not allowed", stepName)
	}

	s.Activities = make([]Activity, 0)
	targetIsMandatory := false
	for i, actKV := range kvKeys {
		keyString := strings.TrimPrefix(actKV.Key, stepPath+"/activities/"+strconv.Itoa(i)+"/")
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
	kvPair, _, err = kv.Get(stepPath+"/target", nil)
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

	kvPairs, _, err := kv.List(stepPath+"/next", nil)
	if err != nil {
		return nil, err
	}
	s.Next = make([]*Step, 0)
	s.Previous = make([]*Step, 0)
	for _, nextKV := range kvPairs {
		var nextStep *Step
		nextStepName := strings.TrimPrefix(nextKV.Key, stepPath+"/next/")
		if visitStep, ok := visitedMap[nextStepName]; ok {
			nextStep = visitStep.s
		} else {
			nextStep, err = BuildStep(kv, deploymentID, wfName, nextStepName, visitedMap)
			if err != nil {
				return nil, err
			}
		}

		s.Next = append(s.Next, nextStep)
		nextStep.Previous = append(nextStep.Previous, s)
		visitedMap[nextStepName].refCount++
	}
	visitedMap[stepName] = &visitStep{refCount: 0, s: s}
	return s, nil

}

// BuildInitExecutionOperations returns Consul transactional KV operations for initiating workflow execution
func BuildInitExecutionOperations(kv *api.KV, deploymentID, taskID, workflowName string, registerWorkflow bool) (api.KVTxnOps, error) {
	ops := make(api.KVTxnOps, 0)
	steps, err := BuildWorkFlow(kv, deploymentID, workflowName)
	if err != nil {
		return nil, err
	}

	for _, step := range steps {
		if registerWorkflow {
			// Register workflow step to handle step statuses for all steps
			ops = append(ops, &api.KVTxnOp{
				Verb:  api.KVSet,
				Key:   path.Join(consulutil.WorkflowsPrefix, taskID, step.Name),
				Value: []byte(tasks.StepStatusINITIAL.String()),
			})
		}

		// Add execution key for initial steps only
		if step.IsInitial() {
			execID := fmt.Sprint(uuid.NewV4())
			log.Debugf("Create initial task execution with ID:%q, taskID:%q and step:%q", execID, taskID, step.Name)
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
					Value: []byte(step.Name),
				},
			}
			ops = append(ops, stepOps...)
		}
	}
	return ops, nil
}

// CreateWorkflowStepsOperations returns Consul transactional KV operations for initiating workflow execution
func CreateWorkflowStepsOperations(taskID string, steps []*Step) api.KVTxnOps {
	ops := make(api.KVTxnOps, 0)
	var stepOps api.KVTxnOps
	for _, step := range steps {
		// Add execution key for initial steps only
		execID := fmt.Sprint(uuid.NewV4())
		log.Debugf("Create task execution with ID:%q, taskID:%q and step:%q", execID, taskID, step.Name)
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
				Value: []byte(step.Name),
			},
		}
		ops = append(ops, stepOps...)
	}
	return ops
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
