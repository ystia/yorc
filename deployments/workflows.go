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
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"path"
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
	return storage.GetStore(types.StoreTypeDeployment).Delete(path.Join(consulutil.DeploymentKVPrefix, deploymentID,
		workflowsPrefix, workflowName)+"/", true)
}

func enhanceWorkflows(ctx context.Context, consulStore consulutil.ConsulStore, deploymentID string) error {
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
		internal.StoreWorkflow(ctx, deploymentID, "run", wf)
	}
	return nil
}
