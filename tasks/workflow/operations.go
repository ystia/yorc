package workflow

import (
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/registry"
)

func getOperationExecutor(kv *api.KV, deploymentID, artifact string) (prov.OperationExecutor, error) {
	reg := registry.GetRegistry()

	exec, originalErr := reg.GetOperationExecutor(artifact)
	if originalErr == nil {
		return exec, nil
	}
	// Try to get an executor for artifact parent type but return the original error if we do not found any executors
	parentArt, err := deployments.GetParentType(kv, deploymentID, artifact)
	if err != nil {
		return nil, err
	}
	if parentArt != "" {
		exec, err := getOperationExecutor(kv, deploymentID, parentArt)
		if err == nil {
			return exec, nil
		}
	}
	return nil, originalErr
}
