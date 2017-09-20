package operations

import (
	"github.com/hashicorp/consul/api"

	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

func GetOperation(kv *api.KV, deploymentID, nodeName, operationName string) (prov.Operation, error) {
	isRelationshipOp, operationRealName, requirementIndex, targetNodeName, err := deployments.DecodeOperation(kv, deploymentID, nodeName, operationName)
	if err != nil {
		return prov.Operation{}, err
	}
	var implementingType string
	if isRelationshipOp {
		implementingType, err = deployments.GetRelationshipTypeImplementingAnOperation(kv, deploymentID, nodeName, operationRealName, requirementIndex)
	} else {
		implementingType, err = deployments.GetNodeTypeImplementingAnOperation(kv, deploymentID, nodeName, operationRealName)
	}
	if err != nil {
		return prov.Operation{}, err
	}
	implArt, err := deployments.GetImplementationArtifactForOperation(kv, deploymentID, nodeName, operationRealName, isRelationshipOp, requirementIndex)
	if err != nil {
		return prov.Operation{}, err
	}
	op := prov.Operation{
		Name:                   operationRealName,
		ImplementedInType:      implementingType,
		ImplementationArtifact: implArt,
		RelOp: prov.RelationshipOperation{
			IsRelationshipOperation: isRelationshipOp,
			RequirementIndex:        requirementIndex,
			TargetNodeName:          targetNodeName,
		},
	}
	return op, nil
}
