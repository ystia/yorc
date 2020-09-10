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
	"github.com/ystia/yorc/v4/storage/store"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
)

// StoreWorkflow stores a workflow
func StoreWorkflow(ctx context.Context, deploymentID, workflowName string, workflow *tosca.Workflow) error {
	wfPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "workflows", workflowName)
	return storage.GetStore(types.StoreTypeDeployment).Set(ctx, wfPrefix, *transformedWorkflow(workflow))
}

func transformedWorkflow(workflow *tosca.Workflow) *tosca.Workflow {
	for _, step := range workflow.Steps {
		for _, activity := range step.Activities {
			if activity.CallOperation != nil {
				// Preserve case for requirement and target node name in case of relationship operation
				opSlice := strings.SplitN(activity.CallOperation.Operation, "/", 2)
				opSlice[0] = strings.ToLower(opSlice[0])
				activity.CallOperation.Operation = strings.Join(opSlice, "/")
			}
		}
	}
	return workflow
}

// storeWorkflows stores topology workflows
func storeWorkflows(ctx context.Context, topology tosca.Topology, deploymentID string) error {
	kv := make([]store.KeyValueIn, 0)
	for wfName, workflow := range topology.TopologyTemplate.Workflows {
		// no need to store empty workflow
		if workflow.Steps == nil {
			continue
		}
		wfPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "workflows", wfName)
		kv = append(kv, store.KeyValueIn{
			Key:   wfPrefix,
			Value: *transformedWorkflow(&workflow),
		})
	}
	return storage.GetStore(types.StoreTypeDeployment).SetCollection(ctx, kv)
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
			if activity.Inline != nil {
				if collections.ContainsString(nestedWfs, activity.Inline.Workflow) {
					return errors.Errorf("A cycle has been detected in inline workflows [initial: %q, repeated: %q]", nestedWfs[0], activity.Inline.Workflow)
				}
				if err := checkNestedWorkflow(topology, topology.TopologyTemplate.Workflows[activity.Inline.Workflow], nestedWfs, activity.Inline.Workflow); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
