package deployments

import (
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"
)

// GetTopologyOutput returns the value of a given topology output
func GetTopologyOutput(kv *api.KV, deploymentID, outputName string, nestedKeys ...string) (bool, string, error) {
	dataType, err := GetTopologyOutputType(kv, deploymentID, outputName)
	if err != nil {
		return false, "", err
	}
	valuePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/outputs", outputName, "value")
	// TODO this is not clear in the specification but why do we return a single value in this context as in case of attributes and multi-instances
	// we can have different results.
	// We have to improve this.
	found, res, err := getValueAssignmentWithDataType(kv, deploymentID, valuePath, "", "0", "", dataType, nestedKeys...)
	if err != nil || found {
		return found, res, err
	}
	// check the default
	defaultPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/outputs", outputName, "default")
	return getValueAssignmentWithDataType(kv, deploymentID, defaultPath, "", "0", "", dataType, nestedKeys...)
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
