package deployments

import (
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

// SetInstanceState stores the state of a given node instance
func SetInstanceState(kv *api.KV, deploymentID, nodeName, instanceName string, state tosca.NodeState) error {
	_, err := kv.Put(&api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "attributes/state"), Value: []byte(state.String())}, nil)
	return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
}

// GetInstanceState retrieves the state of a given node instance
func GetInstanceState(kv *api.KV, deploymentID, nodeName, instanceName string) (tosca.NodeState, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "attributes/state"), nil)
	if err != nil {
		return tosca.NodeStateError, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return tosca.NodeStateError, errors.Errorf("Missing mandatory attribute \"state\" on instance %q for node %q", instanceName, nodeName)
	}
	state, err := tosca.NodeStateString(string(kvp.Value))
	if err != nil {
		return tosca.NodeStateError, err
	}
	return state, nil
}
