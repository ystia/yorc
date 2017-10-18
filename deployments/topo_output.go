package deployments

import (
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

func GetTopologyOutput(kv *api.KV, deploymentID, outputName string) (bool, string, error) {
	valuePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/outputs", outputName, "value")
	found, res, err := getValueAssignment(kv, deploymentID, valuePath, "", "0", "")
	if err != nil || found {
		return found, res, err
	}
	// check the default
	defaultPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/outputs", outputName, "default")
	return getValueAssignment(kv, deploymentID, defaultPath, "", "0", "")
}

// GetTopologyOutputsNames returns the list of outputs for the deployment
func GetTopologyOutputsNames(kv *api.KV, deploymentID string) ([]string, error) {
	optPaths, _, err := kv.Keys(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "/topology/outputs")+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for i := range optPaths {
		optPaths[i] = path.Base(optPaths[i])
	}
	return optPaths, nil
}
