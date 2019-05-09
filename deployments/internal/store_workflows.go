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
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v3/helper/collections"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/tosca"
)

func storeWorkflowStep(consulStore consulutil.ConsulStore, deploymentID, workflowName, stepName string, step *tosca.Step) {
	stepPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "workflows", url.QueryEscape(workflowName), "steps", url.QueryEscape(stepName))
	if step.Target != "" {
		consulStore.StoreConsulKeyAsString(stepPrefix+"/target", step.Target)
	}
	if step.TargetRelationShip != "" {
		consulStore.StoreConsulKeyAsString(stepPrefix+"/target_relationship", step.TargetRelationShip)
	}
	if step.OperationHost != "" {
		consulStore.StoreConsulKeyAsString(stepPrefix+"/operation_host", strings.ToUpper(step.OperationHost))
	}
	activitiesPrefix := stepPrefix + "/activities"
	for actIndex, activity := range step.Activities {
		activityPrefix := activitiesPrefix + "/" + strconv.Itoa(actIndex)
		if activity.CallOperation != "" {
			// Preserve case for requirement and target node name in case of relationship operation
			opSlice := strings.SplitN(activity.CallOperation, "/", 2)
			opSlice[0] = strings.ToLower(opSlice[0])
			consulStore.StoreConsulKeyAsString(activityPrefix+"/call-operation", strings.Join(opSlice, "/"))
		}
		if activity.Delegate != "" {
			consulStore.StoreConsulKeyAsString(activityPrefix+"/delegate", strings.ToLower(activity.Delegate))
		}
		if activity.SetState != "" {
			consulStore.StoreConsulKeyAsString(activityPrefix+"/set-state", strings.ToLower(activity.SetState))
		}
		if activity.Inline != "" {
			consulStore.StoreConsulKeyAsString(activityPrefix+"/inline", strings.ToLower(activity.Inline))
		}
	}
	for _, next := range step.OnSuccess {
		// store in consul a prefix for the next step to be executed ; this prefix is stepPrefix/next/onSuccess_value
		consulStore.StoreConsulKeyAsString(fmt.Sprintf("%s/next/%s", stepPrefix, url.QueryEscape(next)), "")
	}
	for _, of := range step.OnFailure {
		// store in consul a prefix for the next step to be executed on failure ; this prefix is stepPrefix/on-failure/onFailure_value
		consulStore.StoreConsulKeyAsString(fmt.Sprintf("%s/on-failure/%s", stepPrefix, url.QueryEscape(of)), "")
	}
	for _, oc := range step.OnCancel {
		// store in consul a prefix for the next step to be executed on cancel ; this prefix is stepPrefix/on-cancel/onCancel_value
		consulStore.StoreConsulKeyAsString(fmt.Sprintf("%s/on-cancel/%s", stepPrefix, url.QueryEscape(oc)), "")
	}
}

// StoreWorkflow stores a workflow
func StoreWorkflow(consulStore consulutil.ConsulStore, deploymentID, workflowName string, workflow tosca.Workflow) {
	for stepName, step := range workflow.Steps {
		storeWorkflowStep(consulStore, deploymentID, workflowName, stepName, step)
	}
}

// storeWorkflows stores topology workflows
func storeWorkflows(ctx context.Context, consulStore consulutil.ConsulStore, topology tosca.Topology, deploymentID string) {
	for wfName, workflow := range topology.TopologyTemplate.Workflows {
		StoreWorkflow(consulStore, deploymentID, wfName, workflow)
	}
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
