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

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"

	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tosca"
)

// buildWorkFlow creates a workflow tree from values for a specified workflow name and deploymentID
func buildWorkFlow(kv *api.KV, deploymentID, wfName string) (map[string]*step, error) {
	wf, err := deployments.ReadWorkflow(kv, deploymentID, wfName)
	if err != nil {
		log.Print(err)
		return nil, err
	}

	steps := make(map[string]*step, len(wf.Steps))
	visitedMap := make(map[string]*visitStep, len(wf.Steps))
	for stepName := range wf.Steps {
		if visitStep, ok := visitedMap[stepName]; !ok {
			s, err := buildStepFromWFStep(kv, deploymentID, wfName, stepName, wf.Steps, visitedMap)
			if err != nil {
				return nil, err
			}
			steps[stepName] = s
		} else {
			steps[stepName] = visitStep.s
		}
	}
	return steps, nil
}

func buildStepFromWFStep(kv *api.KV, deploymentID, wfName, stepName string, wfSteps map[string]tosca.Step, visitedMap map[string]*visitStep) (*step, error) {
	wfStep := wfSteps[stepName]
	s := &step{
		name:               stepName,
		kv:                 kv,
		workflowName:       wfName,
		operationHost:      wfStep.OperationHost,
		targetRelationship: wfStep.TargetRelationShip,
		target:             wfStep.Target,
		activities:         make([]Activity, 0, len(wfStep.Activities)),
	}
	var targetIsMandatory bool
	for _, wfActivity := range wfStep.Activities {
		if wfActivity.Delegate != "" {
			targetIsMandatory = true
			s.activities = append(s.activities, delegateActivity{delegate: wfActivity.Delegate})
		} else if wfActivity.CallOperation != "" {
			targetIsMandatory = true
			s.activities = append(s.activities, callOperationActivity{operation: wfActivity.CallOperation})
		} else if wfActivity.SetState != "" {
			targetIsMandatory = true
			s.activities = append(s.activities, setStateActivity{state: wfActivity.SetState})
		} else if wfActivity.Inline != "" {
			s.activities = append(s.activities, inlineActivity{inline: wfActivity.Inline})
		} else {
			return nil, errors.Errorf("Unsupported activity type for step: %q", stepName)
		}
	}

	if s.target == "" && targetIsMandatory {
		return nil, errors.Errorf("Missing target attribute for Step %s", stepName)
	}

	s.next = make([]*step, 0)
	s.previous = make([]*step, 0)
	for _, nextStepName := range wfStep.OnSuccess {
		var nextStep *step
		if visitStep, ok := visitedMap[nextStepName]; ok {
			nextStep = visitStep.s
		} else {
			var err error
			nextStep, err = buildStepFromWFStep(kv, deploymentID, wfName, nextStepName, wfSteps, visitedMap)
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
	ops := make(api.KVTxnOps, 0)
	steps, err := buildWorkFlow(kv, deploymentID, workflowName)
	if err != nil {
		return nil, err
	}
	_, errGrp, store := consulutil.WithContext(context.Background())
	for _, step := range steps {
		if registerWorkflow {
			// Register workflow step to handle step statuses for all steps
			// Do not use the transaction here just the async rate-limited behavior
			// This should decrease pressure on the 64 ops limit for transactions
			store.StoreConsulKeyAsString(path.Join(consulutil.WorkflowsPrefix, taskID, step.name), tasks.TaskStepStatusINITIAL.String())
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
	return ops, errGrp.Wait()
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
