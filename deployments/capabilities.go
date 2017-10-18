package deployments

import (
	"context"
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

// HasScalableCapability check if the given nodeName in the specified deployment, has in this capabilities a Key named scalable
func HasScalableCapability(kv *api.KV, deploymentID, nodeName string) (bool, error) {
	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, err
	}

	return TypeHasCapability(kv, deploymentID, nodeType, "tosca.capabilities.Scalable")
}

// TypeHasCapability checks if a given TOSCA type has a capability which type is derived from capabilityTypeName
func TypeHasCapability(kv *api.KV, deploymentID, typeName, capabilityTypeName string) (bool, error) {
	capabilities, err := GetCapabilitiesOfType(kv, deploymentID, typeName, capabilityTypeName)
	return len(capabilities) > 0, err
}

// GetCapabilitiesOfType returns names of all capabilities in a given type hierarchy that derives from a given capability type
func GetCapabilitiesOfType(kv *api.KV, deploymentID, typeName, capabilityTypeName string) ([]string, error) {
	capabilities := make([]string, 0)
	typePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "types", typeName)
	capabilitiesKeys, _, err := kv.Keys(typePath+"/capabilities/", "/", nil)
	if err != nil {
		return capabilities, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	for _, capName := range capabilitiesKeys {
		var kvp *api.KVPair
		kvp, _, err = kv.Get(path.Join(capName, "type"), nil)
		if err != nil {
			return capabilities, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp == nil || len(kvp.Value) == 0 {
			return capabilities, errors.Errorf("Missing \"type\" key for type capability %q", capName)
		}
		var isCorrectType bool
		isCorrectType, err = IsTypeDerivedFrom(kv, deploymentID, string(kvp.Value), capabilityTypeName)
		if err != nil {
			return capabilities, err
		}
		if isCorrectType {
			capabilities = append(capabilities, path.Base(capName))
		}
	}

	parentType, err := GetParentType(kv, deploymentID, typeName)
	if err != nil {
		return capabilities, err
	}

	if parentType != "" {
		var parentCapabilities []string
		parentCapabilities, err = GetCapabilitiesOfType(kv, deploymentID, parentType, capabilityTypeName)
		if err != nil {
			return capabilities, err
		}
		capabilities = append(capabilities, parentCapabilities...)
	}
	return capabilities, nil
}

// GetCapabilityProperty  retrieves the value for a given property in a given node capability
//
// It returns true if a value is found false otherwise as first return parameter.
// If the property is not found in the node then the type hierarchy is explored to find a default value.
func GetCapabilityProperty(kv *api.KV, deploymentID, nodeName, capabilityName, propertyName string, nestedKeys ...string) (bool, string, error) {
	capPropPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "capabilities", capabilityName, "properties", propertyName)
	found, result, err := getValueAssignment(kv, deploymentID, capPropPath, nodeName, "", "", nestedKeys...)
	if err != nil || found {
		// If there is an error or property was found
		return found, result, errors.Wrapf(err, "Failed to get property %q for capability %q on node %q", propertyName, capabilityName, nodeName)
	}

	// Not found let's look at capability type for default
	capabilityType, err := GetNodeCapabilityType(kv, deploymentID, nodeName, capabilityName)
	if err != nil {
		return false, "", err
	}
	found, result, isFunction, err := getTypeDefaultProperty(kv, deploymentID, capabilityType, propertyName, nestedKeys...)
	if err != nil {
		return false, "", err
	}
	if found {
		if !isFunction {
			return true, result, nil
		}
		result, err = resolveValueAssignmentAsString(kv, deploymentID, nodeName, "", "", result, nestedKeys...)
		return true, result, err
	}

	// No default found in type hierarchy
	// then traverse HostedOn relationships to find the value
	host, err := GetHostedOnNode(kv, deploymentID, nodeName)
	if err != nil {
		return false, "", err
	}
	if host != "" {
		return GetCapabilityProperty(kv, deploymentID, host, capabilityName, propertyName, nestedKeys...)
	}
	// Not found anywhere
	return false, "", nil
}

// GetInstanceCapabilityAttribute retrieves the value for a given attribute in a given node instance capability
//
// It returns true if a value is found false otherwise as first return parameter.
// If the attribute is not found in the node then the type hierarchy is explored to find a default value.
// If still not found check properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
func GetInstanceCapabilityAttribute(kv *api.KV, deploymentID, nodeName, instanceName, capabilityName, attributeName string, nestedKeys ...string) (bool, string, error) {
	// First look at instance scoped attributes
	capAttrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "capabilities", capabilityName, "attributes", attributeName)
	found, result, err := getValueAssignment(kv, deploymentID, capAttrPath, nodeName, instanceName, "", nestedKeys...)
	if err != nil || found {
		// If there is an error or attribute was found
		return found, result, errors.Wrapf(err, "Failed to get attribute %q for capability %q on node %q (instance %q)", attributeName, capabilityName, nodeName, instanceName)
	}

	// Then look at global node level
	nodeCapPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "capabilities", capabilityName, "attributes", attributeName)
	found, result, err = getValueAssignment(kv, deploymentID, nodeCapPath, nodeName, instanceName, "", nestedKeys...)
	if err != nil || found {
		// If there is an error or attribute was found
		return found, result, errors.Wrapf(err, "Failed to get attribute %q for capability %q on node %q (instance %q)", attributeName, capabilityName, nodeName, instanceName)
	}

	// Now look at capability type for default
	capabilityType, err := GetNodeCapabilityType(kv, deploymentID, nodeName, capabilityName)
	if err != nil {
		return false, "", err
	}
	found, result, isFunction, err := getTypeDefaultAttribute(kv, deploymentID, capabilityType, attributeName, nestedKeys...)
	if err != nil {
		return false, "", err
	}
	if found {
		if !isFunction {
			return true, result, nil
		}
		result, err = resolveValueAssignmentAsString(kv, deploymentID, nodeName, instanceName, "", result, nestedKeys...)
		return true, result, err
	}

	// No default found in type hierarchy
	// then traverse HostedOn relationships to find the value
	host, err := GetHostedOnNode(kv, deploymentID, nodeName)
	if err != nil {
		return false, "", err
	}
	if host != "" {
		found, result, err = GetInstanceCapabilityAttribute(kv, deploymentID, host, instanceName, capabilityName, attributeName, nestedKeys...)
		if err != nil || found {
			// If there is an error or attribute was found
			return found, result, err
		}
	}

	// If still not found check properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
	return GetCapabilityProperty(kv, deploymentID, nodeName, capabilityName, attributeName, nestedKeys...)
}

// SetInstanceCapabilityAttribute sets a capability attribute for a given node instance
func SetInstanceCapabilityAttribute(deploymentID, nodeName, instanceName, capabilityName, attributeName, value string) error {
	return SetInstanceCapabilityAttributeComplex(deploymentID, nodeName, instanceName, capabilityName, attributeName, value)
}

// SetInstanceCapabilityAttributeComplex sets an instance capability attribute that may be a literal or a complex data type
func SetInstanceCapabilityAttributeComplex(deploymentID, nodeName, instanceName, capabilityName, attributeName string, attributeValue interface{}) error {
	attrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "capabilities", capabilityName, "attributes", attributeName)
	_, errGrp, store := consulutil.WithContext(context.Background())
	storeComplexType(store, attrPath, attributeValue)
	return errGrp.Wait()
}

// SetCapabilityAttributeForAllInstances sets the same capability attribute value to all instances of a given node.
//
// It does the same thing than iterating over instances ids and calling SetInstanceCapabilityAttribute but use
// a consulutil.ConsulStore to do it in parallel. We can expect better performances with a large number of instances
func SetCapabilityAttributeForAllInstances(kv *api.KV, deploymentID, nodeName, capabilityName, attributeName, attributeValue string) error {
	return SetCapabilityAttributeComplexForAllInstances(kv, deploymentID, nodeName, capabilityName, attributeName, attributeValue)
}

// SetCapabilityAttributeComplexForAllInstances sets the same capability attribute value  that may be a literal or a complex data type to all instances of a given node.
//
// It does the same thing than iterating over instances ids and calling SetInstanceCapabilityAttributeComplex but use
// a consulutil.ConsulStore to do it in parallel. We can expect better performances with a large number of instances
func SetCapabilityAttributeComplexForAllInstances(kv *api.KV, deploymentID, nodeName, capabilityName, attributeName string, attributeValue interface{}) error {
	ids, err := GetNodeInstancesIds(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	_, errGrp, store := consulutil.WithContext(context.Background())
	for _, instanceName := range ids {
		attrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "capabilities", capabilityName, "attributes", attributeName)
		storeComplexType(store, attrPath, attributeValue)
	}
	return errGrp.Wait()
}

// GetNodeCapabilityType retrieves the type of a node template capability identified by its name
//
// This is a shorthand for GetNodeTypeCapabilityType
func GetNodeCapabilityType(kv *api.KV, deploymentID, nodeName, capabilityName string) (string, error) {
	// Now look at capability type for default
	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return "", err
	}
	return GetNodeTypeCapabilityType(kv, deploymentID, nodeType, capabilityName)
}

// GetNodeTypeCapabilityType retrieves the type of a node type capability identified by its name
//
// It explores the type hierarchy (derived_from) to found the given capability.
// It may return an empty string if the capability is not found in the type hierarchy
func GetNodeTypeCapabilityType(kv *api.KV, deploymentID, nodeType, capabilityName string) (string, error) {

	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeType, "capabilities", capabilityName, "type"), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil && len(kvp.Value) != 0 {
		return string(kvp.Value), nil
	}
	parentType, err := GetParentType(kv, deploymentID, nodeType)
	if err != nil {
		return "", err
	}
	if parentType == "" {
		return "", nil
	}
	return GetNodeTypeCapabilityType(kv, deploymentID, parentType, capabilityName)
}
