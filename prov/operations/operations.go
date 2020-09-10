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

package operations

import (
	"context"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/stringutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/tosca"
)

// GetOperation returns a Prov.Operation structure describing precisely operation in order to execute it
func GetOperation(ctx context.Context, deploymentID, nodeName, operationName, requirementName, operationHost string,
	inputs map[string]tosca.ParameterDefinition) (prov.Operation, error) {

	var (
		implementingType, implementingNode, requirementIndex, targetNodeName string
		err                                                                  error
		isRelationshipOp, isNodeImplOpe                                      bool
	)
	// if requirementName is filled, operation is associated to a relationship
	isRelationshipOp = requirementName != ""
	if isRelationshipOp {
		requirementIndex, implementingType, targetNodeName, err =
			getRelationshipOperationInfo(ctx, deploymentID, nodeName, operationName, requirementName)
		if err != nil {
			return prov.Operation{}, err
		}
	} else {
		isNodeImplOpe, implementingNode, implementingType, err =
			getNodeOperationInfo(ctx, deploymentID, nodeName, operationName)
		if err != nil {
			return prov.Operation{}, err
		}
	}
	implArt, err := deployments.GetImplementationArtifactForOperation(ctx, deploymentID, nodeName, operationName, isNodeImplOpe, isRelationshipOp, requirementIndex)
	if err != nil {
		return prov.Operation{}, err
	}
	implemOperationHost, err := deployments.GetOperationHostFromTypeOperationByName(ctx, deploymentID, implementingType, operationName)
	if err != nil {
		return prov.Operation{}, err
	}
	if operationHost != "" && implemOperationHost != "" && operationHost != implemOperationHost {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).Registerf("operation host defined in the implementation of operation %q (%q) is different from the one defined in the workflow step (%q). We will use the one from the workflow.", implemOperationHost, operationName, operationHost)
	} else if operationHost == "" && implemOperationHost != "" {
		operationHost = implemOperationHost
	}
	op := prov.Operation{
		Name:                      operationName,
		ImplementedInType:         implementingType,
		ImplementedInNodeTemplate: implementingNode,
		ImplementationArtifact:    implArt,
		RelOp: prov.RelationshipOperation{
			IsRelationshipOperation: isRelationshipOp,
			RequirementIndex:        requirementIndex,
			TargetNodeName:          targetNodeName,
			TargetRelationship:      requirementName,
		},
		OperationHost: operationHost,
		Inputs:        inputs,
	}
	log.Debugf("operation:%+v", op)
	return op, nil
}

func getRelationshipOperationInfo(ctx context.Context, deploymentID, nodeName,
	operationName, requirementName string) (requirementIndex, implementingType, targetNodeName string, err error) {

	requirementIndex, err = deployments.GetRequirementIndexByNameForNode(ctx, deploymentID, nodeName, requirementName)
	if err != nil || requirementIndex == "" {
		err = errors.Wrapf(err, "Unable to find requirement key for requirement name:%q", requirementName)
		return
	}
	implementingType, err = deployments.GetRelationshipTypeImplementingAnOperation(ctx, deploymentID, nodeName, operationName, requirementIndex)
	if err != nil {
		return
	}
	targetNodeName, err = deployments.GetTargetNodeForRequirement(ctx, deploymentID, nodeName, requirementIndex)
	return
}

func getNodeOperationInfo(ctx context.Context, deploymentID, nodeName,
	operationName string) (isNodeImplOpe bool, implementingNode string, implementingType string, err error) {

	isNodeImplOpe, err = deployments.IsNodeTemplateImplementingOperation(ctx, deploymentID, nodeName, operationName)
	if err != nil {
		return
	}
	if isNodeImplOpe {
		implementingNode = nodeName
	} else {
		implementingType, err = deployments.GetNodeTypeImplementingAnOperation(ctx, deploymentID, nodeName, operationName)
		if err != nil {
			return
		}
	}
	return
}

// IsRelationshipOperation checks if an operation is part of a relationship
func IsRelationshipOperation(op prov.Operation) bool {
	return op.RelOp.IsRelationshipOperation
}

// IsRelationshipTargetNodeOperation checks if an operation is part of a relationship and should be
// executed on the target node
func IsRelationshipTargetNodeOperation(op prov.Operation) bool {
	return IsRelationshipOperation(op) && op.OperationHost == "TARGET"
}

// IsOrchestratorHostOperation checks if the operation should be executed on orchestrator host
func IsOrchestratorHostOperation(op prov.Operation) bool {
	return op.OperationHost == "ORCHESTRATOR"
}

// SetOperationLogFields set logs optionals fields related to the given action
func SetOperationLogFields(ctx context.Context, op prov.Operation) context.Context {
	logOptFields, ok := events.FromContext(ctx)
	if !ok {
		logOptFields = make(events.LogOptionalFields)
	}
	logOptFields[events.OperationName] = stringutil.GetLastElement(op.Name, ".")
	logOptFields[events.InterfaceName] = stringutil.GetAllExceptLastElement(op.Name, ".")

	return events.NewContext(ctx, logOptFields)
}
