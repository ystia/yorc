package deployments

import (
	"path"

	"net/url"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/tosca"
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
	wf.Steps = make(map[string]tosca.Step, len(steps))
	for _, stepKey := range steps {
		step, err := readWfStep(kv, stepKey)
		if err != nil {
			return wf, err
		}
		stepName, err := url.QueryUnescape(path.Base(stepKey))
		if err != nil {
			return wf, errors.Wrapf(err, "Failed to get back step name from Consul")
		}
		wf.Steps[stepName] = step
	}
	return wf, nil
}

func readWfStep(kv *api.KV, stepKey string) (tosca.Step, error) {
	kvp, _, err := kv.Get(path.Join(stepKey, "target"), nil)
	step := tosca.Step{}
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return step, errors.Errorf("Missing mandatory attribute \"target\" for step %q", path.Base(stepKey))
	}
	step.Target = string(kvp.Value)

	kvp, _, err = kv.Get(path.Join(stepKey, "operation_host"), nil)
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil && len(kvp.Value) != 0 {
		step.OperationHost = string(kvp.Value)
	}
	activity := tosca.Activity{}
	kvp, _, err = kv.Get(path.Join(stepKey, "activity", "call-operation"), nil)
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil && len(kvp.Value) != 0 {
		activity.CallOperation = string(kvp.Value)
	}
	kvp, _, err = kv.Get(path.Join(stepKey, "activity", "set-state"), nil)
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil && len(kvp.Value) != 0 {
		activity.SetState = string(kvp.Value)
	}
	kvp, _, err = kv.Get(path.Join(stepKey, "activity", "delegate"), nil)
	if err != nil {
		return step, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil && len(kvp.Value) != 0 {
		activity.Delegate = string(kvp.Value)
	}
	step.Activity = activity

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
	return step, nil
}
