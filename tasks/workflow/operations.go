package workflow

import (
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

func getOperation(kv *api.KV, deploymentID, nodeName, operationName string) (prov.Operation, error) {
	isRelationshipOp, operationRealName, requirementIndex, targetNodeName, err := deployments.DecodeOperation(kv, deploymentID, nodeName, operationName)
	if err != nil {
		return prov.Operation{}, err
	}
	implArt, err := deployments.GetImplementationArtifactForOperation(kv, deploymentID, nodeName, operationRealName, isRelationshipOp, requirementIndex)
	if err != nil {
		return prov.Operation{}, err
	}
	op := prov.Operation{
		Name: operationRealName,
		ImplementationArtifact: implArt,
		RelOp: prov.RelationshipOperation{
			IsRelationshipOperation: isRelationshipOp,
			RequirementIndex:        requirementIndex,
			TargetNodeName:          targetNodeName,
		},
	}
	return op, nil
}
