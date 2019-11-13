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

package internal

import (
	"context"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
	"net/url"
	"path"
)

func storeWorkflowStep(ctx context.Context, deploymentID, workflowName, stepName string, step *tosca.Step) error {
	stepPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "workflows", url.QueryEscape(workflowName), "steps", url.QueryEscape(stepName))
	return storage.GetStore(types.StoreTypeDeployment).Set(stepPrefix, step)
}

// StoreWorkflow stores a workflow
func StoreWorkflow(ctx context.Context, deploymentID, workflowName string, workflow tosca.Workflow) error {
	for stepName, step := range workflow.Steps {
		err := storeWorkflowStep(ctx, deploymentID, workflowName, stepName, step)
		if err != nil {
			return err
		}
	}
	return nil
}

// storeWorkflows stores topology workflows
func storeWorkflows(ctx context.Context, topology tosca.Topology, deploymentID string) error {
	for wfName, workflow := range topology.TopologyTemplate.Workflows {
		err := StoreWorkflow(ctx, deploymentID, wfName, workflow)
		if err != nil {
			return err
		}
	}
	return nil
}

// checkNestedWorkflows detect potential cycle in all nested workflows
func checkNestedWorkflows(topology tosca.Topology) error {
	for wfName, workflow := range topology.TopologyTemplate.Workflows {
		nestedWfs := make([]string, 0)
		if err := checkNestedWorkflow(topology, workflow, nestedWfs, wfName); err != nil {
			return err
		}
	}
	return nil
}

// checkNestedWorkflows detect potential cycle in a nested workflow
func checkNestedWorkflow(topology tosca.Topology, workflow tosca.Workflow, nestedWfs []string, wfName string) error {
	nestedWfs = append(nestedWfs, wfName)
	for _, step := range workflow.Steps {
		for _, activity := range step.Activities {
			if activity.Inline != "" {
				if collections.ContainsString(nestedWfs, activity.Inline) {
					return errors.Errorf("A cycle has been detected in inline workflows [initial: %q, repeated: %q]", nestedWfs[0], activity.Inline)
				}
				if err := checkNestedWorkflow(topology, topology.TopologyTemplate.Workflows[activity.Inline], nestedWfs, activity.Inline); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
