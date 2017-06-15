package deployments

import (
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

// SetInstanceStateString stores the state of a given node instance and publishes a status change event
func SetInstanceStateString(kv *api.KV, deploymentID, nodeName, instanceName, state string) error {
	_, err := kv.Put(&api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "attributes/state"), Value: []byte(state)}, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	_, err = events.InstanceStatusChange(kv, deploymentID, nodeName, instanceName, state)
	return err
}

// SetInstanceState stores the state of a given node instance and publishes a status change event
func SetInstanceState(kv *api.KV, deploymentID, nodeName, instanceName string, state tosca.NodeState) error {
	return SetInstanceStateString(kv, deploymentID, nodeName, instanceName, state.String())
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

// DeleteInstance deletes the given instance of the given node from the Consul store
func DeleteInstance(kv *api.KV, deploymentID, nodeName, instanceName string) error {
	_, err := kv.DeleteTree(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName), nil)
	return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
}

// GetInstanceAttribute retrieves the given attribute for a node instance
//
// This function only look at the given node level. It doesn't inspect the node type hierarchy to find a default value.
func GetInstanceAttribute(kv *api.KV, deploymentID, nodeName, instanceName, attributeName string) (string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "attributes", attributeName), nil)
	if err != nil || kvp == nil || len(kvp.Value) == 0 {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return string(kvp.Value), err
}
