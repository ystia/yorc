package deployments

import (
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"path"
)

func GetInputValue(kv *api.KV, deploymentId, inputName string) (string, error) {
	kvp, _, err := kv.Get(path.Join(DeploymentKVPrefix, deploymentId, "topology/inputs", inputName, "value"), nil)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get value for input named %q in deployment: %q", inputName, deploymentId)
	}
	if kvp != nil && len(kvp.Value) > 0 {
		return string(kvp.Value), nil
	}
	kvp, _, err = kv.Get(path.Join(DeploymentKVPrefix, deploymentId, "topology/inputs", inputName, "default"), nil)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get default value for input named %q in deployment: %q", inputName, deploymentId)
	}
	if kvp != nil && len(kvp.Value) > 0 {
		return string(kvp.Value), nil
	}
	return "", nil
}
