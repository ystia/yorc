package deployments

import (
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
func GetCapabilityProperty(kv *api.KV, deploymentID, nodeName, capabilityName, propertyName string) (bool, string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "capabilities", capabilityName, "properties", propertyName), nil)
	if err != nil {
		return false, "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil {
		return true, string(kvp.Value), nil
	}

	// Look at capability type for default
	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, "", err
	}
	ok, value, err := GetTypeDefaultProperty(kv, deploymentID, nodeType, propertyName)
	if err != nil {
		return false, "", nil
	}
	return ok, value, nil
}

// GetCapabilityAttribute  retrieves the value for a given attribute in a given node capability
//
// It returns true if a value is found false otherwise as first return parameter.
// If the attribute is not found in the node then the type hierarchy is explored to find a default value.
// If still not found check properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
func GetCapabilityAttribute(kv *api.KV, deploymentID, nodeName, capabilityName, attributeName string) (bool, string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "capabilities", capabilityName, "attributes", attributeName), nil)
	if err != nil {
		return false, "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil {
		return true, string(kvp.Value), nil
	}

	// Look at capability type for default
	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, "", err
	}
	ok, value, err := GetTypeDefaultAttribute(kv, deploymentID, nodeType, attributeName)
	if err != nil {
		return false, "", nil
	}
	if ok {
		return true, value, nil
	}

	return GetCapabilityProperty(kv, deploymentID, nodeName, capabilityName, attributeName)
}
