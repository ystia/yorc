package deployments

import (
	"context"
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
// It returns true if a value is found false otherwise as first return parameter.
// If the attribute is not found in the node then the type hierarchy is explored to find a default value.
// If the attribute is still not found then it will explore the HostedOn hierarchy.
// If still not found then it will check node properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
func GetInstanceAttribute(kv *api.KV, deploymentID, nodeName, instanceName, attributeName string, nestedKeys ...string) (bool, string, error) {
	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, "", err
	}

	var attrDataType string
	hasAttr, err := TypeHasAttribute(kv, deploymentID, nodeType, attributeName)
	if err != nil {
		return false, "", err
	}
	if hasAttr {
		attrDataType, err = GetTypeAttributeDataType(kv, deploymentID, nodeType, attributeName)
		if err != nil {
			return false, "", err
		}
	}

	// First look at instance-scoped attributes
	found, result, err := getValueAssignmentWithDataType(kv, deploymentID, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "attributes", attributeName), nodeName, instanceName, "", attrDataType, nestedKeys...)
	if err != nil {
		return false, "", errors.Wrapf(err, "Failed to get attribute %q for node %q (instance %q)", attributeName, nodeName, instanceName)
	}
	if found {
		return true, result, nil
	}

	// Then look at global node level (not instance-scoped)
	found, result, err = getValueAssignmentWithDataType(kv, deploymentID, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "attributes", attributeName), nodeName, instanceName, "", attrDataType, nestedKeys...)
	if err != nil {
		return false, "", errors.Wrapf(err, "Failed to get attribute %q for node %q", attributeName, nodeName, instanceName)
	}
	if found {
		return true, result, nil
	}

	// Not found look at node type
	found, defaultValue, isFunction, err := getTypeDefaultAttribute(kv, deploymentID, nodeType, attributeName, nestedKeys...)
	if err != nil {
		return false, "", err
	}
	if found {
		if !isFunction {
			return true, defaultValue, nil
		}
		defaultValue, err = resolveValueAssignmentAsString(kv, deploymentID, nodeName, instanceName, "", defaultValue, nestedKeys...)
		return true, defaultValue, err
	}

	// No default found in type hierarchy
	// then traverse HostedOn relationships to find the value
	var host string
	host, err = GetHostedOnNode(kv, deploymentID, nodeName)
	if err != nil {
		return false, "", err
	}
	if host != "" {
		found, hostValue, err := GetInstanceAttribute(kv, deploymentID, host, instanceName, attributeName, nestedKeys...)
		if found || err != nil {
			return found, hostValue, err
		}
	}

	// Now check properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
	return GetNodeProperty(kv, deploymentID, nodeName, attributeName, nestedKeys...)
}

// SetInstanceAttribute sets an instance attribute
func SetInstanceAttribute(deploymentID, nodeName, instanceName, attributeName, attributeValue string) error {
	return SetInstanceAttributeComplex(deploymentID, nodeName, instanceName, attributeName, attributeValue)

}

// SetInstanceAttributeComplex sets an instance attribute that may be a literal or a complex data type
func SetInstanceAttributeComplex(deploymentID, nodeName, instanceName, attributeName string, attributeValue interface{}) error {
	attrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "attributes", attributeName)
	_, errGrp, store := consulutil.WithContext(context.Background())
	storeComplexType(store, attrPath, attributeValue)
	return errGrp.Wait()
}

// SetAttributeForAllInstances sets the same attribute value to all instances of a given node.
//
// It does the same thing than iterating over instances ids and calling SetInstanceAttribute but use
// a consulutil.ConsulStore to do it in parallel. We can expect better performances with a large number of instances
func SetAttributeForAllInstances(kv *api.KV, deploymentID, nodeName, attributeName, attributeValue string) error {
	return SetAttributeComplexForAllInstances(kv, deploymentID, nodeName, attributeName, attributeValue)
}

// SetAttributeComplexForAllInstances sets the same attribute value to all instances of a given node.
//
// It does the same thing than iterating over instances ids and calling SetInstanceAttributeComplex but use
// a consulutil.ConsulStore to do it in parallel. We can expect better performances with a large number of instances
func SetAttributeComplexForAllInstances(kv *api.KV, deploymentID, nodeName, attributeName string, attributeValue interface{}) error {
	ids, err := GetNodeInstancesIds(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	_, errGrp, store := consulutil.WithContext(context.Background())
	for _, instanceName := range ids {
		storeComplexType(store, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "attributes", attributeName), attributeValue)
	}
	return errGrp.Wait()
}
