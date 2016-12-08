package deployments

import (
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"path"
)

// GetInputValue tries to retrieve the value of the given input name.
//
// GetInputValue first checks if a non-empty field value exists for this input, if it doesn't then it checks for a non-empty field default.
// If none of them exists then it returns an empty string.
func GetInputValue(kv *api.KV, deploymentId, inputName string) (string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/inputs", inputName, "value"), nil)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get value for input named %q in deployment: %q", inputName, deploymentId)
	}
	if kvp != nil && len(kvp.Value) > 0 {
		return string(kvp.Value), nil
	}
	kvp, _, err = kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/inputs", inputName, "default"), nil)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get default value for input named %q in deployment: %q", inputName, deploymentId)
	}
	if kvp != nil && len(kvp.Value) > 0 {
		return string(kvp.Value), nil
	}
	return "", nil
}
