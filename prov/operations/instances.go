package operations

import (
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

func ResolveInstances(kv *api.KV, taskID, deploymentID, relationshipTargetName, nodeName string, isRelationshipOperation bool) ([]string, []string, error) {
	var err error
	var sourceNodeInstances []string
	var targetNodeInstances []string

	if isRelationshipOperation {
		targetNodeInstances, err = tasks.GetInstances(kv, taskID, deploymentID, relationshipTargetName)
		if err != nil {
			return sourceNodeInstances, targetNodeInstances, err
		}
	}
	sourceNodeInstances, err = tasks.GetInstances(kv, taskID, deploymentID, nodeName)
	return sourceNodeInstances, targetNodeInstances, err
}
