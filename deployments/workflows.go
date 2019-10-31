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

package deployments

import (
	"context"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/deployments/internal"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/tosca"
)

const (
	workflowsPrefix = "workflows"
)

// GetWorkflows returns the list of workflows names for a given deployment
func GetWorkflows(ctx context.Context, deploymentID string) ([]string, error) {
	workflowsPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, workflowsPrefix)
	keys, err := consulutil.GetKeys(workflowsPath)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	results := make([]string, len(keys))
	for i := range keys {
		results[i] = path.Base(keys[i])
	}
	return results, nil
}

// ReadWorkflow reads a workflow definition from Consul and built its TOSCA representation
func ReadWorkflow(ctx context.Context, deploymentID, workflowName string) (tosca.Workflow, error) {
	workflowPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, workflowsPrefix, workflowName)
	steps, err := consulutil.GetKeys(workflowPath + "/steps")
	wf := tosca.Workflow{}
	if err != nil {
		return wf, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	wf.Steps = make(map[string]*tosca.Step, len(steps))
	for _, stepKey := range steps {
		stepName, err := url.QueryUnescape(path.Base(stepKey))
		if err != nil {
			return wf, errors.Wrapf(err, "Failed to get back step name from Consul")
		}
		step, err := readWfStep(stepKey, stepName, workflowName)
		if err != nil {
			return wf, err
		}
		wf.Steps[stepName] = step
	}
	return wf, nil
}

func readWfStep(stepKey string, stepName string, wfName string) (*tosca.Step, error) {
	step := &tosca.Step{}
	targetIsMandatory := false
	// Get the step operation's host (not mandatory)
	exist, value, err := consulutil.GetStringValue(path.Join(stepKey, "operation_host"))
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if exist && value != "" {
		step.OperationHost = value
	}
	// Get the target relationship (not mandatory)
	exist, value, err = consulutil.GetStringValue(path.Join(stepKey, "target_relationship"))
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if exist && value != "" {
		step.TargetRelationShip = value
	}
	// Get the step's activities
	// Appending a final "/" here is not necessary as there is no other keys starting with "activities" prefix
	activitiesKeys, err := consulutil.List(stepKey + "/activities")
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	step.Activities = make([]tosca.Activity, len(activitiesKeys))
	var i int
	for key, value := range activitiesKeys {
		activity := tosca.Activity{}
		key := strings.TrimPrefix(key, stepKey+"activities/"+strconv.Itoa(i)+"/")
		switch {
		case key == "delegate":
			activity.Delegate = string(value)
			step.Activities[i] = activity
			targetIsMandatory = true
		case key == "set-state":
			activity.SetState = string(value)
			step.Activities[i] = activity
			targetIsMandatory = true
		case key == "call-operation":
			activity.CallOperation = string(value)
			step.Activities[i] = activity
			targetIsMandatory = true
		case key == "inline":
			activity.Inline = string(value)
			step.Activities[i] = activity
		}
		i++
	}
	// Get the step's target (mandatory except if all activities are inline)
	exist, value, err = consulutil.GetStringValue(path.Join(stepKey, "target"))
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if targetIsMandatory && (!exist || value == "") {
		return step, errors.Errorf("Missing mandatory attribute \"target\" for step %q", path.Base(stepKey))
	}
	if exist && value != "" {
		step.Target = value
	}

	// Get the next steps of the current step and use it to set the OnSuccess filed
	nextSteps, err := consulutil.GetKeys(stepKey + "/next")
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if len(nextSteps) > 0 {
		step.OnSuccess = make([]string, len(nextSteps))
		for i, nextKey := range nextSteps {
			nextName, err := url.QueryUnescape(path.Base(nextKey))
			if err != nil {
				return step, errors.Wrapf(err, "Failed to get back step name from Consul")
			}
			step.OnSuccess[i] = nextName
		}
	}
	// Get the on cancel steps of the current step and use it to set the OnSuccess filed
	onCancelSteps, err := consulutil.GetKeys(stepKey + "/on-cancel")
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if len(nextSteps) > 0 {
		step.OnCancel = make([]string, len(onCancelSteps))
		for i, nextKey := range onCancelSteps {
			nextName, err := url.QueryUnescape(path.Base(nextKey))
			if err != nil {
				return step, errors.Wrapf(err, "Failed to get back step name from Consul")
			}
			step.OnCancel[i] = nextName
		}
	}
	// Get the on cancel steps of the current step and use it to set the OnSuccess filed
	onFailureSteps, err := consulutil.GetKeys(stepKey + "/on-failure")
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if len(onFailureSteps) > 0 {
		step.OnFailure = make([]string, len(onFailureSteps))
		for i, nextKey := range onFailureSteps {
			nextName, err := url.QueryUnescape(path.Base(nextKey))
			if err != nil {
				return step, errors.Wrapf(err, "Failed to get back step name from Consul")
			}
			step.OnFailure[i] = nextName
		}
	}
	return step, nil
}

// DeleteWorkflow deletes the given workflow from the Consul store
func DeleteWorkflow(ctx context.Context, deploymentID, workflowName string) error {
	return consulutil.Delete(path.Join(consulutil.DeploymentKVPrefix, deploymentID,
		workflowsPrefix, workflowName)+"/", true)
}

func enhanceWorkflows(ctx context.Context, consulStore consulutil.ConsulStore, deploymentID string) error {
	wf, err := ReadWorkflow(ctx, deploymentID, "run")
	if err != nil {
		return err
	}
	var wasUpdated bool
	for sn, s := range wf.Steps {
		var isCancellable bool
		for _, a := range s.Activities {
			switch strings.ToLower(a.CallOperation) {
			case tosca.RunnableSubmitOperationName, tosca.RunnableRunOperationName:
				isCancellable = true
			}
		}
		if isCancellable && len(s.OnCancel) == 0 {
			// Cancellable and on-cancel not defined
			// Check if there is an cancel op
			hasCancelOp, err := IsOperationImplemented(ctx, deploymentID, s.Target, tosca.RunnableCancelOperationName)
			if err != nil {
				return err
			}
			if hasCancelOp {
				cancelStep := &tosca.Step{
					Target:             s.Target,
					TargetRelationShip: s.TargetRelationShip,
					OperationHost:      s.OperationHost,
					Activities: []tosca.Activity{
						tosca.Activity{
							CallOperation: tosca.RunnableCancelOperationName,
						},
					},
				}
				csName := "yorc_automatic_cancellation_of_" + sn
				wf.Steps[csName] = cancelStep
				s.OnCancel = []string{csName}
				wasUpdated = true
			}
		}
	}
	if wasUpdated {
		internal.StoreWorkflow(consulStore, deploymentID, "run", wf)
	}
	return nil
}
