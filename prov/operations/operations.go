package operations

import (
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"path"
	"strings"
)

func GetNodeTypeAndPath(kv *api.KV, nodeName, deploymentID string) (string, string, error) {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return "", "", err
	}
	return nodeType, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeType), nil
}

func GetNodePath(nodeName, deploymentID string) string {
	return path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName)
}

func GetRelationshipInfos(isRelationshipOperation bool, kv *api.KV, deploymentID, NodeName, requirementIndex, operation string) (string, bool, bool, bool, error) {
	var err error
	var relationshipType string
	var isRelationshipTargetNode bool
	var isPerInstanceOperation bool
	var isCustomCommand bool

	if isRelationshipOperation {
		relationshipType, err = deployments.GetRelationshipForRequirement(kv, deploymentID, NodeName, requirementIndex)
		if err != nil {
			return "", false, false, false, err
		}

		isRelationshipTargetNode = IsTargetOperation(operation)

		isPerInstanceOperation, err = ResolveIsPerInstanceOperation(operation, deploymentID, relationshipType, kv)
		if err != nil {
			return "", false, false, false, err
		}

	} else if strings.Contains(operation, "custom") {
		isCustomCommand = true
	}

	return relationshipType, isRelationshipTargetNode, isPerInstanceOperation, isCustomCommand, nil
}
