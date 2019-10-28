// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deployments

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/deployments/internal"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tosca"
)

// HasScalableCapability check if the given nodeName in the specified deployment, has in this capabilities a Key named scalable
func HasScalableCapability(deploymentID, nodeName string) (bool, error) {
	nodeType, err := GetNodeType(deploymentID, nodeName)
	if err != nil {
		return false, err
	}

	return TypeHasCapability(deploymentID, nodeType, "tosca.capabilities.Scalable")
}

// TypeHasCapability checks if a given TOSCA type has a capability which type is derived from capabilityTypeName
func TypeHasCapability(deploymentID, typeName, capabilityTypeName string) (bool, error) {
	capabilities, err := GetCapabilitiesOfType(deploymentID, typeName, capabilityTypeName)
	return len(capabilities) > 0, err
}

// GetCapabilitiesOfType returns names of all capabilities in a given type hierarchy that derives from a given capability type
func GetCapabilitiesOfType(deploymentID, typeName, capabilityTypeName string) ([]string, error) {
	capabilities := make([]string, 0)
	typePath, err := locateTypePath(deploymentID, typeName)
	if err != nil {
		return capabilities, err
	}
	capabilitiesKeys, err := consulutil.GetKeys(path.Join(typePath, "capabilities"))
	if err != nil {
		return capabilities, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	for _, capName := range capabilitiesKeys {
		exist, value, err := consulutil.GetStringValue(path.Join(capName, "type"))
		if err != nil {
			return capabilities, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if !exist || value == "" {
			return capabilities, errors.Errorf("Missing \"type\" key for type capability %q", capName)
		}
		var isCorrectType bool
		isCorrectType, err = IsTypeDerivedFrom(deploymentID, value, capabilityTypeName)
		if err != nil {
			return capabilities, err
		}
		if isCorrectType {
			capabilities = append(capabilities, path.Base(capName))
		}
	}

	parentType, err := GetParentType(deploymentID, typeName)
	if err != nil {
		return capabilities, err
	}

	if parentType != "" {
		var parentCapabilities []string
		parentCapabilities, err = GetCapabilitiesOfType(deploymentID, parentType, capabilityTypeName)
		if err != nil {
			return capabilities, err
		}
		capabilities = append(capabilities, parentCapabilities...)
	}
	return capabilities, nil
}

// GetCapabilityPropertyType retrieves the type for a given property in a given node capability
// It returns false if there is no such property
func GetCapabilityPropertyType(deploymentID, nodeName, capabilityName,
	propertyName string) (bool, string, error) {

	capabilityType, err := GetNodeCapabilityType(deploymentID, nodeName, capabilityName)
	if err != nil {
		return false, "", err
	}
	var propDataType string
	var hasProp bool
	if capabilityType != "" {
		hasProp, err = TypeHasProperty(deploymentID, capabilityType, propertyName, true)
		if err != nil {
			return false, "", err
		}
		if hasProp {
			propDataType, err = GetTypePropertyDataType(deploymentID, capabilityType, propertyName)
			if err != nil {
				return true, "", err
			}
		}
	}

	return hasProp, propDataType, err
}

// GetCapabilityPropertyValue retrieves the value for a given property in a given node capability
//
// It returns true if a value is found false otherwise as first return parameter.
// If the property is not found in the node then the type hierarchy is explored to find a default value.
func GetCapabilityPropertyValue(deploymentID, nodeName, capabilityName, propertyName string, nestedKeys ...string) (*TOSCAValue, error) {
	capabilityType, err := GetNodeCapabilityType(deploymentID, nodeName, capabilityName)
	if err != nil {
		return nil, err
	}

	hasProp, propDataType, err := GetCapabilityPropertyType(deploymentID, nodeName, capabilityName, propertyName)
	if err != nil {
		return nil, err
	}

	capPropPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "capabilities", capabilityName, "properties", propertyName)
	result, err := getValueAssignmentWithDataType(deploymentID, capPropPath, nodeName, "", "", propDataType, nestedKeys...)
	if err != nil || result != nil {
		// If there is an error or property was found
		return result, errors.Wrapf(err, "Failed to get property %q for capability %q on node %q", propertyName, capabilityName, nodeName)
	}

	// Not found: let's look at capability in node type
	nodeType, err := GetNodeType(deploymentID, nodeName)
	if err != nil {
		return nil, err
	}

	result, err = GetNodeTypeCapabilityPropertyValue(deploymentID, nodeType, capabilityName, propertyName, propDataType, nestedKeys...)
	if err != nil || result != nil {
		return result, err
	}

	// Not found let's look at capability type for default
	if capabilityType != "" {
		result, isFunction, err := getTypeDefaultProperty(deploymentID, capabilityType, propertyName, nestedKeys...)
		if err != nil {
			return nil, err
		}
		if result != nil {
			if !isFunction {
				return result, nil
			}
			return resolveValueAssignment(deploymentID, nodeName, "", "", result, nestedKeys...)
		}
	}
	// No default found in type hierarchy
	// then traverse HostedOn relationships to find the value
	host, err := GetHostedOnNode(deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	if host != "" {
		result, err = GetCapabilityPropertyValue(deploymentID, host, capabilityName, propertyName, nestedKeys...)
		if err != nil || result != nil {
			return result, err
		}
	}

	if hasProp && capabilityType != "" {
		// Check if the whole property is optional
		isRequired, err := IsTypePropertyRequired(deploymentID, capabilityType, propertyName)
		if err != nil {
			return nil, err
		}
		if !isRequired {
			// For backward compatibility
			// TODO this doesn't look as a good idea to me
			return &TOSCAValue{Value: ""}, nil
		}

		if len(nestedKeys) > 1 && propDataType != "" {
			// Check if nested type is optional
			nestedKeyType, err := GetNestedDataType(deploymentID, propDataType, nestedKeys[:len(nestedKeys)-1]...)
			if err != nil {
				return nil, err
			}
			isRequired, err = IsTypePropertyRequired(deploymentID, nestedKeyType, nestedKeys[len(nestedKeys)-1])
			if err != nil {
				return nil, err
			}
			if !isRequired {
				// For backward compatibility
				// TODO this doesn't look as a good idea to me
				return &TOSCAValue{Value: ""}, nil
			}
		}
	}

	// Not found anywhere
	return nil, nil
}

// GetInstanceCapabilityAttributeValue retrieves the value for a given attribute in a given node instance capability
//
// It returns true if a value is found false otherwise as first return parameter.
// If the attribute is not found in the node then the type hierarchy is explored to find a default value.
// If still not found check properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
func GetInstanceCapabilityAttributeValue(deploymentID, nodeName, instanceName, capabilityName, attributeName string, nestedKeys ...string) (*TOSCAValue, error) {
	capabilityType, err := GetNodeCapabilityType(deploymentID, nodeName, capabilityName)
	if err != nil {
		return nil, err
	}

	var attrDataType string
	if capabilityType != "" {
		hasProp, err := TypeHasAttribute(deploymentID, capabilityType, attributeName, true)
		if err != nil {
			return nil, err
		}
		if hasProp {
			attrDataType, err = GetTypeAttributeDataType(deploymentID, capabilityType, attributeName)
			if err != nil {
				return nil, err
			}
		}
	}

	// Capability attributes of a Service referencing an application in another
	// deployment are actually available as attributes of the node template
	substitutionInstance, err := isSubstitutionNodeInstance(deploymentID, nodeName, instanceName)
	if err != nil {
		return nil, err
	}
	if substitutionInstance {

		result, err := getSubstitutionInstanceCapabilityAttribute(deploymentID, nodeName, instanceName, capabilityName, attrDataType, attributeName, nestedKeys...)
		if err != nil || result != nil {
			// If there is an error or attribute was found, returning
			// else going back to the generic behavior
			return result, errors.Wrapf(err,
				"Failed to get attribute %q for capability %q on substitutable node %q",
				attributeName, capabilityName, nodeName)
		}
	}

	// First look at instance scoped attributes
	capAttrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "capabilities", capabilityName, "attributes", attributeName)
	result, err := getValueAssignmentWithDataType(deploymentID, capAttrPath, nodeName, instanceName, "", attrDataType, nestedKeys...)
	if err != nil || result != nil {
		// If there is an error or attribute was found
		return result, errors.Wrapf(err, "Failed to get attribute %q for capability %q on node %q (instance %q)", attributeName, capabilityName, nodeName, instanceName)
	}

	// Then look at global node level
	nodeCapPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "capabilities", capabilityName, "attributes", attributeName)
	result, err = getValueAssignmentWithDataType(deploymentID, nodeCapPath, nodeName, instanceName, "", attrDataType, nestedKeys...)
	if err != nil || result != nil {
		// If there is an error or attribute was found
		return result, errors.Wrapf(err, "Failed to get attribute %q for capability %q on node %q (instance %q)", attributeName, capabilityName, nodeName, instanceName)
	}

	// Now look at capability type for default
	if capabilityType != "" {
		result, isFunction, err := getTypeDefaultAttribute(deploymentID, capabilityType, attributeName, nestedKeys...)
		if err != nil {
			return nil, err
		}
		if result != nil {
			if !isFunction {
				return result, nil
			}
			return resolveValueAssignment(deploymentID, nodeName, instanceName, "", result, nestedKeys...)
		}
	}

	// No default found in type hierarchy
	// then traverse HostedOn relationships to find the value
	host, hostInstance, err := GetHostedOnNodeInstance(deploymentID, nodeName, instanceName)
	if err != nil {
		return nil, err
	}
	if host != "" {
		result, err = GetInstanceCapabilityAttributeValue(deploymentID, host, hostInstance, capabilityName, attributeName, nestedKeys...)
		if err != nil || result != nil {
			// If there is an error or attribute was found
			return result, err
		}
	}

	if capabilityType != "" {
		isEndpoint, err := IsTypeDerivedFrom(deploymentID, capabilityType,
			tosca.EndpointCapability)
		if err != nil {
			return nil, err
		}

		// TOSCA specification at :
		// http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_TYPE_CAPABILITIES_ENDPOINT
		// describes that the ip_address attribute of an endpoint is the IP address
		// as propagated up by the associated node’s host (Compute) container.
		if isEndpoint && attributeName == tosca.EndpointCapabilityIPAddressAttribute && host != "" {
			result, err = getIPAddressFromHost(deploymentID, host, hostInstance, nodeName, instanceName, capabilityName)
			if err != nil || result != nil {
				return result, err
			}
		}
	}
	// If still not found check properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
	return GetCapabilityPropertyValue(deploymentID, nodeName, capabilityName, attributeName, nestedKeys...)
}

func getEndpointCapabilitityHostIPAttributeNameAndNetName(deploymentID, nodeName, capabilityName string) (string, *TOSCAValue, error) {
	// First check the network name in the capability property to find the right
	// IP address attribute (default: private address)
	ipAddressAttrName := "private_address"

	netName, err := GetCapabilityPropertyValue(deploymentID, nodeName, capabilityName, "network_name")
	if err != nil {
		return "", nil, err
	}

	if netName != nil && strings.ToLower(netName.RawString()) == "public" {
		ipAddressAttrName = "public_address"
	}
	return ipAddressAttrName, netName, nil
}

func getIPAddressFromHost(deploymentID, hostName, hostInstance, nodeName, instanceName, capabilityName string) (*TOSCAValue, error) {
	ipAddressAttrName, netName, err := getEndpointCapabilitityHostIPAttributeNameAndNetName(deploymentID, nodeName, capabilityName)
	if err != nil {
		return nil, err
	}
	result, err := GetInstanceAttributeValue(deploymentID, hostName, hostInstance,
		ipAddressAttrName)

	if err == nil && result != nil {
		log.Debugf("Found IP address %s for %s %s %s %s on network %s from host %s %s",
			result, deploymentID, nodeName, instanceName, capabilityName, netName, hostName, hostInstance)
	} else {
		log.Debugf("Found no IP address for %s %s %s %s on network %s from host %s %s",
			deploymentID, nodeName, instanceName, capabilityName, netName, hostName, hostInstance)
	}

	return result, err

}

// GetNodeCapabilityAttributeNames retrieves the names for all capability attributes
// of a capability on a given node name
func GetNodeCapabilityAttributeNames(deploymentID, nodeName, capabilityName string, exploreParents bool) ([]string, error) {

	capabilityType, err := GetNodeCapabilityType(deploymentID, nodeName, capabilityName)
	if err != nil {
		return nil, err
	}
	return GetTypeAttributes(deploymentID, capabilityType, exploreParents)

}

// SetInstanceCapabilityAttribute sets a capability attribute for a given node instance
func SetInstanceCapabilityAttribute(deploymentID, nodeName, instanceName, capabilityName, attributeName, value string) error {
	return SetInstanceCapabilityAttributeComplex(deploymentID, nodeName, instanceName, capabilityName, attributeName, value)
}

// SetInstanceCapabilityAttributeComplex sets an instance capability attribute that may be a literal or a complex data type
func SetInstanceCapabilityAttributeComplex(deploymentID, nodeName, instanceName, capabilityName, attributeName string, attributeValue interface{}) error {
	attrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "capabilities", capabilityName, "attributes", attributeName)
	_, errGrp, store := consulutil.WithContext(context.Background())
	internal.StoreComplexType(store, attrPath, attributeValue)
	err := notifyAndPublishCapabilityAttributeValueChange(deploymentID, nodeName, instanceName, capabilityName, attributeName, attributeValue)
	if err != nil {
		return err
	}
	return errGrp.Wait()
}

// SetCapabilityAttributeForAllInstances sets the same capability attribute value to all instances of a given node.
//
// It does the same thing than iterating over instances ids and calling SetInstanceCapabilityAttribute but use
// a consulutil.ConsulStore to do it in parallel. We can expect better performances with a large number of instances
func SetCapabilityAttributeForAllInstances(deploymentID, nodeName, capabilityName, attributeName, attributeValue string) error {
	return SetCapabilityAttributeComplexForAllInstances(deploymentID, nodeName, capabilityName, attributeName, attributeValue)
}

// SetCapabilityAttributeComplexForAllInstances sets the same capability attribute value  that may be a literal or a complex data type to all instances of a given node.
//
// It does the same thing than iterating over instances ids and calling SetInstanceCapabilityAttributeComplex but use
// a consulutil.ConsulStore to do it in parallel. We can expect better performances with a large number of instances
func SetCapabilityAttributeComplexForAllInstances(deploymentID, nodeName, capabilityName, attributeName string, attributeValue interface{}) error {
	ids, err := GetNodeInstancesIds(deploymentID, nodeName)
	if err != nil {
		return err
	}
	_, errGrp, store := consulutil.WithContext(context.Background())
	for _, instanceName := range ids {
		attrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "capabilities", capabilityName, "attributes", attributeName)
		internal.StoreComplexType(store, attrPath, attributeValue)
		err = notifyAndPublishCapabilityAttributeValueChange(deploymentID, nodeName, instanceName, capabilityName, attributeName, attributeValue)
		if err != nil {
			return err
		}
	}
	return errGrp.Wait()
}

// GetNodeCapabilityType retrieves the type of a node template capability identified by its name
//
// This is a shorthand for GetNodeTypeCapabilityType
func GetNodeCapabilityType(deploymentID, nodeName, capabilityName string) (string, error) {
	// Now look at capability type for default
	nodeType, err := GetNodeType(deploymentID, nodeName)
	if err != nil {
		return "", err
	}
	return GetNodeTypeCapabilityType(deploymentID, nodeType, capabilityName)
}

// GetNodeTypeCapabilityType retrieves the type of a node type capability identified by its name
//
// It explores the type hierarchy (derived_from) to found the given capability.
// It may return an empty string if the capability is not found in the type hierarchy
func GetNodeTypeCapabilityType(deploymentID, nodeType, capabilityName string) (string, error) {
	typePath, err := locateTypePath(deploymentID, nodeType)
	if err != nil {
		return "", err
	}
	exist, value, err := consulutil.GetStringValue(path.Join(typePath, "capabilities", capabilityName, "type"))
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if exist || value != "" {
		return value, nil
	}
	parentType, err := GetParentType(deploymentID, nodeType)
	if err != nil {
		return "", err
	}
	if parentType == "" {
		return "", nil
	}
	return GetNodeTypeCapabilityType(deploymentID, parentType, capabilityName)
}

// GetNodeTypeCapabilityPropertyValue retrieves the property value of a node type capability identified by its name
//
// It explores the type hierarchy (derived_from) to found the given capability.
func GetNodeTypeCapabilityPropertyValue(deploymentID, nodeType, capabilityName, propertyName, propDataType string, nestedKeys ...string) (*TOSCAValue, error) {
	typePath, err := locateTypePath(deploymentID, nodeType)
	if err != nil {
		return nil, err
	}
	capPropPath := path.Join(typePath, "capabilities", capabilityName, "properties", propertyName)
	result, err := getValueAssignmentWithDataType(deploymentID, capPropPath, "", "", "", propDataType, nestedKeys...)
	if err != nil || result != nil {
		return result, errors.Wrapf(err, "Failed to get property %q for capability %q on node type %q", propertyName, capabilityName, nodeType)
	}

	parentType, err := GetParentType(deploymentID, nodeType)
	if err != nil {
		return nil, err
	}
	if parentType == "" {
		return nil, nil
	}
	return GetNodeTypeCapabilityPropertyValue(deploymentID, parentType, capabilityName, propertyName, propDataType, nestedKeys...)
}

func notifyAndPublishCapabilityAttributeValueChange(deploymentID, nodeName, instanceName, capabilityName, attributeName string, attributeValue interface{}) error {
	sValue, ok := attributeValue.(string)
	if ok {
		// First, Publish event
		capabilityAttribute := fmt.Sprintf("capabilities.%s.%s", capabilityName, attributeName)
		_, err := events.PublishAndLogAttributeValueChange(context.Background(), deploymentID, nodeName, instanceName, capabilityAttribute, sValue, "updated")
		if err != nil {
			return err
		}
	}
	// Next, notify dependent attributes if existing
	an := &AttributeNotifier{
		NodeName:       nodeName,
		InstanceName:   instanceName,
		AttributeName:  attributeName,
		CapabilityName: capabilityName,
	}
	return an.NotifyValueChange(deploymentID)
}

func isNodeCapabilityOfType(deploymentID, nodeName, capabilityName, derives string) (bool, error) {
	capType, err := GetNodeCapabilityType(deploymentID, nodeName, capabilityName)
	if err != nil {
		return false, err
	}
	if capType == "" {
		return false, nil
	}
	return IsTypeDerivedFrom(deploymentID, capType, derives)
}
