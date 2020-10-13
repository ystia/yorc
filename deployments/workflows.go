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
	"path"
	"strings"

	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"

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
	keys, err := storage.GetStore(types.StoreTypeDeployment).Keys(workflowsPath)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	results := make([]string, len(keys))
	for i := range keys {
		results[i] = path.Base(keys[i])
	}
	return results, nil
}

// GetWorkflow reads a workflow definition from Consul and built its TOSCA representation
func GetWorkflow(ctx context.Context, deploymentID, workflowName string) (*tosca.Workflow, error) {
	workflowPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, workflowsPrefix, workflowName)
	wf := new(tosca.Workflow)
	exist, err := storage.GetStore(types.StoreTypeDeployment).Get(workflowPath, wf)
	if err != nil {
		return nil, err
	}
	if !exist {
		log.Debugf("No workflow found with name %q for deployment %q", workflowName, deploymentID)
		return nil, nil
	}

	return wf, nil
}

// DeleteWorkflow deletes the given workflow from the Consul store
func DeleteWorkflow(ctx context.Context, deploymentID, workflowName string) error {
	return storage.GetStore(types.StoreTypeDeployment).Delete(ctx, path.Join(consulutil.DeploymentKVPrefix, deploymentID,
		workflowsPrefix, workflowName), false)
}

func enhanceWorkflows(ctx context.Context, deploymentID string) error {
	wf, err := GetWorkflow(ctx, deploymentID, "run")
	if err != nil {
		return err
	}
	if wf == nil {
		return nil
	}
	var wasUpdated bool
	for sn, s := range wf.Steps {
		var isCancellable bool
		for _, a := range s.Activities {
			if a.CallOperation == nil {
				continue
			}
			switch strings.ToLower(a.CallOperation.Operation) {
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
							CallOperation: &tosca.OperationActivity{Operation: tosca.RunnableCancelOperationName},
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
		internal.StoreWorkflow(ctx, deploymentID, "run", wf)
	}
	return nil
}

// ResolveWorkflowOutputs allows to resolve workflow outputs
func ResolveWorkflowOutputs(ctx context.Context, deploymentID, workflowName string) (map[string]*TOSCAValue, error) {
	wf, err := GetWorkflow(ctx, deploymentID, workflowName)
	if err != nil {
		return nil, err
	}

	if wf == nil {
		return nil, errors.Errorf("Can't resolve outputs of workflow %q in deployment %q, workflow definition not found", workflowName, deploymentID)
	}

	outputs := make(map[string]*TOSCAValue)
	for outputName, outputDef := range wf.Outputs {
		dataType := getTypeFromParamDefinition(ctx, &outputDef)

		res, err := getValueAssignment(ctx, deploymentID, "", "0", "", dataType, outputDef.Value)
		if err != nil {
			return nil, err
		}
		// otherwise check the default
		if res == nil {
			res, err = getValueAssignment(ctx, deploymentID, "", "0", "", dataType, outputDef.Default)
			if err != nil {
				return nil, err
			}
		}

		outputs[outputName] = res
	}

	return outputs, nil
}
