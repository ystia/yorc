package deployments

import (
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

// GetInputValue tries to retrieve the value of the given input name.
//
// GetInputValue first checks if a non-empty field value exists for this input, if it doesn't then it checks for a non-empty field default.
// If none of them exists then it returns an empty string.
func GetInputValue(kv *api.KV, deploymentID, inputName string, nestedKeys ...string) (string, error) {
	found, result, err := getValueAssignment(kv, deploymentID, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/inputs", inputName, "value"), "", "", "", nestedKeys...)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get input %q value", inputName)
	}
	if found {
		return result, nil
	}
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/inputs", inputName, "default"), nil)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get default value for input named %q in deployment: %q", inputName, deploymentID)
	}
	if kvp != nil && len(kvp.Value) > 0 {
		return string(kvp.Value), nil
	}
	return "", nil
}
