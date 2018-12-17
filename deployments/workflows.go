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
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/tosca"
)

// GetWorkflows returns the list of workflows names for a given deployment
func GetWorkflows(kv *api.KV, deploymentID string) ([]string, error) {
	workflowsPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "workflows")
	keys, _, err := kv.Keys(workflowsPath+"/", "/", nil)
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
func ReadWorkflow(kv *api.KV, deploymentID, workflowName string) (tosca.Workflow, error) {
	workflowPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "workflows", workflowName)
	steps, _, err := kv.Keys(workflowPath+"/steps/", "/", nil)
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
		step, err := readWfStep(kv, stepKey, stepName, workflowName)
		if err != nil {
			return wf, err
		}
		wf.Steps[stepName] = step
	}
	return wf, nil
}

func readWfStep(kv *api.KV, stepKey string, stepName string, wfName string) (*tosca.Step, error) {
	step := &tosca.Step{}
	targetIsMandatory := false
	// Get the step operation's host (not mandatory)
	kvp, _, err := kv.Get(path.Join(stepKey, "operation_host"), nil)
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil && len(kvp.Value) != 0 {
		step.OperationHost = string(kvp.Value)
	}
	// Get the target relationship (not mandatory)
	kvp, _, err = kv.Get(path.Join(stepKey, "target_relationship"), nil)
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil && len(kvp.Value) != 0 {
		step.TargetRelationShip = string(kvp.Value)
	}
	// Get the step's activities
	activitiesKeys, _, err := kv.List(stepKey+"/activities", nil)
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	step.Activities = make([]tosca.Activity, len(activitiesKeys))
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for i, actKV := range activitiesKeys {
		activity := tosca.Activity{}
		key := strings.TrimPrefix(actKV.Key, stepKey+"activities/"+strconv.Itoa(i)+"/")
		switch {
		case key == "delegate":
			activity.Delegate = string(actKV.Value)
			step.Activities[i] = activity
			targetIsMandatory = true
		case key == "set-state":
			activity.SetState = string(actKV.Value)
			step.Activities[i] = activity
			targetIsMandatory = true
		case key == "call-operation":
			activity.CallOperation = string(actKV.Value)
			step.Activities[i] = activity
			targetIsMandatory = true
		case key == "inline":
			activity.Inline = string(actKV.Value)
			step.Activities[i] = activity
		}
	}
	// Get the step's target (mandatory except if all activities are inline)
	kvp, _, err = kv.Get(path.Join(stepKey, "target"), nil)
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if targetIsMandatory && (kvp == nil || len(kvp.Value) == 0) {
		return step, errors.Errorf("Missing mandatory attribute \"target\" for step %q", path.Base(stepKey))
	}
	if kvp != nil && len(kvp.Value) > 0 {
		step.Target = string(kvp.Value)
	}

	// Get the next steps of the current step and use it to set the OnSuccess filed
	nextSteps, _, err := kv.Keys(stepKey+"/next/", "/", nil)
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
	onCancelSteps, _, err := kv.Keys(stepKey+"/on-cancel/", "/", nil)
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
	onFailureSteps, _, err := kv.Keys(stepKey+"/on-failure/", "/", nil)
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
