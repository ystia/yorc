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

package builder

import (
	"context"
	"path"

	"github.com/gofrs/uuid"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tosca"
)

// BuildWorkFlow creates a workflow tree from values for a specified workflow name and deploymentID
func BuildWorkFlow(ctx context.Context, deploymentID, wfName string) (map[string]*Step, error) {
	wf, err := deployments.GetWorkflow(ctx, deploymentID, wfName)
	if err != nil || wf == nil {
		return nil, err
	}

	if wf == nil {
		return nil, errors.Errorf("Can't build workflow %q in deployment %q, workflow definition not found", wfName, deploymentID)
	}

	if wf.Steps == nil || len(wf.Steps) == 0 {
		return nil, deployments.NewInconsistentDeploymentError(deploymentID)
	}

	steps := make(map[string]*Step, len(wf.Steps))
	visitedMap := make(map[string]*visitStep, len(wf.Steps))
	for stepName := range wf.Steps {
		if visitStep, ok := visitedMap[stepName]; !ok {
			s, err := buildStepFromWFStep(deploymentID, wfName, stepName, wf.Steps, visitedMap)
			if err != nil {
				return nil, err
			}
			steps[stepName] = s
		} else {
			steps[stepName] = visitStep.s
		}
	}

	for _, s := range steps {
		if !s.IsOnFailurePath && !s.IsOnCancelPath {
			// otherwise we are sure we already processed it and its successors in both ways
			for _, n := range s.OnFailure {
				setFailurePath(n)
			}
			for _, n := range s.OnCancel {
				setCancelPath(n)
			}
		}
	}
	return steps, nil
}

func setFailurePath(s *Step) {
	s.IsOnFailurePath = true
	for _, n := range s.Next {
		setFailurePath(n)
	}
	for _, n := range s.OnFailure {
		setFailurePath(n)
	}
	for _, n := range s.OnFailure {
		setFailurePath(n)
		setCancelPath(n)
	}
}

func setCancelPath(s *Step) {
	s.IsOnCancelPath = true
	for _, n := range s.Next {
		setCancelPath(n)
	}
	for _, n := range s.OnFailure {
		setCancelPath(n)
		setFailurePath(n)
	}
	for _, n := range s.OnCancel {
		setCancelPath(n)
	}
}

func buildStepFromWFStep(deploymentID, wfName, stepName string, wfSteps map[string]*tosca.Step, visitedMap map[string]*visitStep) (*Step, error) {
	wfStep, ok := wfSteps[stepName]
	if !ok {
		return nil, errors.Errorf("Referenced step with name:%q doesn't exist in the workflow:%q", stepName, wfName)
	}

	s := &Step{
		Name:               stepName,
		WorkflowName:       wfName,
		OperationHost:      wfStep.OperationHost,
		TargetRelationship: wfStep.TargetRelationShip,
		Target:             wfStep.Target,
		Activities:         make([]Activity, 0, len(wfStep.Activities)),
	}

	targetIsMandatory, err := buildStepActivities(s, wfStep)
	if err != nil {
		return nil, err
	}

	if s.Target == "" && targetIsMandatory {
		return nil, errors.Errorf("Missing target attribute for Step %s", stepName)
	}

	s.Previous = make([]*Step, 0)
	s.Next, err = buildStepsFromList(deploymentID, wfName, stepName, s, wfSteps, wfStep.OnSuccess, visitedMap)
	if err != nil {
		return nil, err
	}
	s.OnFailure, err = buildStepsFromList(deploymentID, wfName, stepName, s, wfSteps, wfStep.OnFailure, visitedMap)
	if err != nil {
		return nil, err
	}
	s.OnCancel, err = buildStepsFromList(deploymentID, wfName, stepName, s, wfSteps, wfStep.OnCancel, visitedMap)
	if err != nil {
		return nil, err
	}
	visitedMap[stepName] = &visitStep{refCount: 0, s: s}
	return s, nil
}

func buildStepActivities(s *Step, wfStep *tosca.Step) (bool, error) {
	var targetIsMandatory bool
	for i, wfActivity := range wfStep.Activities {
		if wfActivity.Delegate != nil {
			targetIsMandatory = true
			s.Activities = append(s.Activities, delegateActivity{wfActivity.Delegate.Workflow, wfActivity.Delegate.Inputs})
		} else if wfActivity.CallOperation != nil {
			targetIsMandatory = true
			s.Activities = append(s.Activities, callOperationActivity{wfActivity.CallOperation.Operation, wfActivity.CallOperation.Inputs})
			if isAsyncOperation(wfActivity.CallOperation.Operation) {
				s.Async = true
			}
		} else if wfActivity.SetState != "" {
			targetIsMandatory = true
			s.Activities = append(s.Activities, setStateActivity{state: wfActivity.SetState})
		} else if wfActivity.Inline != nil {
			s.Activities = append(s.Activities, inlineActivity{wfActivity.Inline.Workflow, wfActivity.Inline.Inputs})
		} else {
			return false, errors.Errorf("Unsupported activity type for step: %q, activity nb: %d, activity: %+v", s.Name, i, wfActivity)
		}
	}

	return targetIsMandatory, nil
}

func buildStepsFromList(deploymentID, wfName, stepName string, currentStep *Step, wfSteps map[string]*tosca.Step, stepsList []string, visitedMap map[string]*visitStep) ([]*Step, error) {
	res := make([]*Step, 0)
	for _, nextStepName := range stepsList {
		var nextStep *Step
		if visitStep, ok := visitedMap[nextStepName]; ok {
			nextStep = visitStep.s
		} else {
			var err error
			nextStep, err = buildStepFromWFStep(deploymentID, wfName, nextStepName, wfSteps, visitedMap)
			if err != nil {
				return nil, err
			}
		}

		res = append(res, nextStep)
		nextStep.Previous = append(nextStep.Previous, currentStep)
		visitedMap[nextStepName].refCount++
	}
	return res, nil
}

// BuildInitExecutionOperations returns Consul transactional KV operations for initiating workflow execution
func BuildInitExecutionOperations(ctx context.Context, deploymentID, taskID, workflowName string, registerWorkflow bool) (api.KVTxnOps, error) {
	ops := make(api.KVTxnOps, 0)
	steps, err := BuildWorkFlow(ctx, deploymentID, workflowName)
	if err != nil {
		return nil, err
	}
	_, errGrp, store := consulutil.WithContext(context.Background())
	for _, step := range steps {
		if registerWorkflow {
			// Register workflow step to handle step statuses for all steps
			// Do not use the transaction here just the async rate-limited behavior
			// This should decrease pressure on the 64 ops limit for transactions
			store.StoreConsulKeyAsString(path.Join(consulutil.WorkflowsPrefix, taskID, step.Name), tasks.TaskStepStatusINITIAL.String())
		}

		// Add execution key for initial steps only
		if step.IsInitial() {
			u, err := uuid.NewV4()
			if err != nil {
				return ops, errors.Wrapf(err, "Failed to generate a UUID")
			}
			execID := u.String()
			log.Debugf("Register initial task execution with ID:%q, taskID:%q and step:%q", execID, taskID, step.Name)
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
				&api.KVTxnOp{
					Verb:  api.KVSet,
					Key:   path.Join(consulutil.TasksPrefix, taskID, ".runningExecutions", execID),
					Value: []byte(""),
				},
			}
			log.Debugf("Will store runningExecutions with id %q in txn for task %q", execID, taskID)
			ops = append(ops, stepOps...)
		}
	}
	return ops, errGrp.Wait()
}
