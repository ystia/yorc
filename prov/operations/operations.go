package operations

import (
	"github.com/hashicorp/consul/api"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/prov"
)

// GetOperation returns a Prov.Operation structure describing precisely operation in order to execute it
func GetOperation(kv *api.KV, deploymentID, nodeName, operationName, requirementName, operationHost string) (prov.Operation, error) {
	var (
		implementingType, requirementIndex string
		err                                error
		isRelationshipOp                   bool
	)
	// if requirementName is filled, operation is associated to a relationship
	isRelationshipOp = requirementName != ""
	if requirementName != "" {
		key, err := deployments.GetRequirementKeyByNameForNode(kv, deploymentID, nodeName, requirementName)
		if err != nil {
			return prov.Operation{}, err
		}
		if key == "" {
			return prov.Operation{}, errors.Errorf("Unable to found requirement key for requirement name:%q", requirementName)
		}
		requirementIndex = deployments.GetRequirementIndexFromRequirementKey(key)
	}
	if isRelationshipOp {
		implementingType, err = deployments.GetRelationshipTypeImplementingAnOperation(kv, deploymentID, nodeName, operationName, requirementIndex)
	} else {
		implementingType, err = deployments.GetNodeTypeImplementingAnOperation(kv, deploymentID, nodeName, operationName)
	}
	if err != nil {
		return prov.Operation{}, err
	}
	implArt, err := deployments.GetImplementationArtifactForOperation(kv, deploymentID, nodeName, operationName, isRelationshipOp, requirementIndex)
	if err != nil {
		return prov.Operation{}, err
	}
	targetNodeName, err := deployments.GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)
	if err != nil {
		return prov.Operation{}, err
	}

	op := prov.Operation{
		Name:                   operationName,
		ImplementedInType:      implementingType,
		ImplementationArtifact: implArt,
		RelOp: prov.RelationshipOperation{
			IsRelationshipOperation: isRelationshipOp,
			RequirementIndex:        requirementIndex,
			TargetNodeName:          targetNodeName,
		},
		OperationHost:      operationHost,
		TargetRelationship: requirementName,
	}
	return op, nil
}
