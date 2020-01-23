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
	"time"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tosca"
)

// HasScalableCapability check if the given nodeName in the specified deployment, has in this capabilities a Key named scalable
func HasScalableCapability(ctx context.Context, deploymentID, nodeName string) (bool, error) {
	nodeType, err := GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return false, err
	}

	return TypeHasCapability(ctx, deploymentID, nodeType, "tosca.capabilities.Scalable")
}

// TypeHasCapability checks if a given TOSCA type has a capability which type is derived from capabilityTypeName
func TypeHasCapability(ctx context.Context, deploymentID, typeName, capabilityTypeName string) (bool, error) {
	capabilities, err := GetCapabilitiesOfType(ctx, deploymentID, typeName, capabilityTypeName)
	return len(capabilities) > 0, err
}

// GetCapabilitiesOfType returns names of all capabilities for a given node type hierarchy that derives from a given capability type
func GetCapabilitiesOfType(ctx context.Context, deploymentID, typeName, capabilityTypeName string) ([]string, error) {
	capabilities := make([]string, 0)

	node := new(tosca.NodeType)
	err := getExpectedTypeFromName(ctx, deploymentID, typeName, node)
	if err != nil {
		return capabilities, err
	}

	for name, capability := range node.Capabilities {
		if capability.Type == "" {
			return capabilities, errors.Errorf("Missing \"type\" key for type capability %q", name)
		}
		var isCorrectType bool
		isCorrectType, err = IsTypeDerivedFrom(ctx, deploymentID, capability.Type, capabilityTypeName)
		if err != nil {
			return capabilities, err
		}
		if isCorrectType {
			capabilities = append(capabilities, name)
		}
	}

	if node.DerivedFrom != "" {
		var parentCapabilities []string
		parentCapabilities, err = GetCapabilitiesOfType(ctx, deploymentID, node.DerivedFrom, capabilityTypeName)
		if err != nil {
			return capabilities, err
		}
		capabilities = append(capabilities, parentCapabilities...)
	}
	return capabilities, nil
}

// GetCapabilityPropertyType retrieves the type for a given property in a given node capability
// It returns false if there is no such property
func GetCapabilityPropertyType(ctx context.Context, deploymentID, nodeName, capabilityName, propertyName string) (bool, string, error) {
	capabilityType, err := GetNodeCapabilityType(ctx, deploymentID, nodeName, capabilityName)
	if err != nil {
		return false, "", err
	}
	var propDataType string
	var hasProp bool
	if capabilityType != "" {
		hasProp, err = TypeHasProperty(ctx, deploymentID, capabilityType, propertyName, true)
		if err != nil {
			return false, "", err
		}
		if hasProp {
			propDataType, err = GetTypePropertyDataType(ctx, deploymentID, capabilityType, propertyName)
			if err != nil {
				return true, "", err
			}
		}
	}

	return hasProp, propDataType, err
}

func getCapabilityPropertyDefinition(ctx context.Context, deploymentID, capabilityTypeName, propertyName string) (*tosca.PropertyDefinition, error) {
	typ := new(tosca.CapabilityType)
	err := getExpectedTypeFromName(ctx, deploymentID, capabilityTypeName, typ)
	if err != nil {
		return nil, err
	}

	propDef, is := typ.Properties[propertyName]
	if is {
		return &propDef, nil
	}

	// Check parent
	if typ.DerivedFrom != "" {
		return getCapabilityPropertyDefinition(ctx, deploymentID, typ.DerivedFrom, propertyName)
	}

	// Not found
	return nil, nil
}

// GetCapabilityPropertyValue retrieves the value for a given property in a given node capability
//
// It returns true if a value is found false otherwise as first return parameter.
// If the property is not found in the node then the type hierarchy is explored to find a default value.
func GetCapabilityPropertyValue(ctx context.Context, deploymentID, nodeName, capabilityName, propertyName string, nestedKeys ...string) (*TOSCAValue, error) {
	node, err := getNodeTemplate(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}

	hasProp, propDataType, err := GetCapabilityPropertyType(ctx, deploymentID, nodeName, capabilityName, propertyName)
	if err != nil {
		return nil, err
	}

	// Check if the capability property is set at node template level
	var va *tosca.ValueAssignment
	if node.Capabilities != nil {
		ca, is := node.Capabilities[capabilityName]
		if is && &ca != nil && ca.Properties != nil {
			va = ca.Properties[propertyName]
		}
	}

	value, err := getValueAssignment(ctx, deploymentID, nodeName, "", "", propDataType, va, nestedKeys...)
	if err != nil || value != nil {
		return value, err
	}

	// Retrieve related va from node type
	nodeType, err := GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	va, err = GetNodeTypeCapabilityPropertyValueAssignment(ctx, deploymentID, nodeType, capabilityName, propertyName)
	if err != nil {
		return nil, err
	}
	if va != nil {
		value, err := getValueAssignment(ctx, deploymentID, nodeName, "", "", propDataType, va, nestedKeys...)
		if err != nil || value != nil {
			return value, err
		}
	}

	// Retrieve related propertyDefinition with default property at capability type
	capabilityType, err := GetNodeCapabilityType(ctx, deploymentID, nodeName, capabilityName)
	if err != nil {
		return nil, err
	}
	if capabilityType != "" {
		propDef, err := getCapabilityPropertyDefinition(ctx, deploymentID, capabilityType, propertyName)
		if err != nil {
			return nil, err
		}
		if propDef != nil {
			value, err := getValueAssignment(ctx, deploymentID, nodeName, "", "", propDataType, propDef.Default, nestedKeys...)
			if err != nil || value != nil {
				return value, err
			}
		}
	}

	// No default found in type hierarchy
	// then traverse HostedOn relationships to find the value
	host, err := GetHostedOnNode(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	if host != "" {
		value, err := GetCapabilityPropertyValue(ctx, deploymentID, host, capabilityName, propertyName, nestedKeys...)
		if err != nil || value != nil {
			return value, err
		}
	}

	if hasProp && capabilityType != "" {
		// Check if the whole property is optional
		isRequired, err := IsTypePropertyRequired(ctx, deploymentID, capabilityType, propertyName)
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
			nestedKeyType, err := GetNestedDataType(ctx, deploymentID, propDataType, nestedKeys[:len(nestedKeys)-1]...)
			if err != nil {
				return nil, err
			}
			isRequired, err = IsTypePropertyRequired(ctx, deploymentID, nestedKeyType, nestedKeys[len(nestedKeys)-1])
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
func GetInstanceCapabilityAttributeValue(ctx context.Context, deploymentID, nodeName, instanceName, capabilityName, attributeName string, nestedKeys ...string) (*TOSCAValue, error) {
	capabilityType, err := GetNodeCapabilityType(ctx, deploymentID, nodeName, capabilityName)
	if err != nil {
		return nil, err
	}

	var attrDataType string
	if capabilityType != "" {
		hasProp, err := TypeHasAttribute(ctx, deploymentID, capabilityType, attributeName, true)
		if err != nil {
			return nil, err
		}
		if hasProp {
			attrDataType, err = GetTypeAttributeDataType(ctx, deploymentID, capabilityType, attributeName)
			if err != nil {
				return nil, err
			}
		}
	}

	// Capability attributes of a Service referencing an application in another
	// deployment are actually available as attributes of the node template
	substitutionInstance, err := isSubstitutionNodeInstance(ctx, deploymentID, nodeName, instanceName)
	if err != nil {
		return nil, err
	}
	if substitutionInstance {

		result, err := getSubstitutionInstanceCapabilityAttribute(ctx, deploymentID, nodeName, instanceName, capabilityName, attrDataType, attributeName, nestedKeys...)
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
	result, err := getInstanceValueAssignment(ctx, deploymentID, nodeName, instanceName, "", attrDataType, capAttrPath, nestedKeys...)
	if err != nil || result != nil {
		// If there is an error or attribute was found
		return result, errors.Wrapf(err, "Failed to get attribute %q for capability %q on node %q (instance %q)", attributeName, capabilityName, nodeName, instanceName)
	}

	// Then look at global node level
	node, err := getNodeTemplate(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}

	var va *tosca.ValueAssignment
	if node.Capabilities != nil {
		ca, is := node.Capabilities[capabilityName]
		if is && &ca != nil && ca.Attributes != nil {
			va = ca.Attributes[attributeName]
		}
	}

	result, err = getValueAssignment(ctx, deploymentID, nodeName, instanceName, "", attrDataType, va, nestedKeys...)
	if err != nil || result != nil {
		// If there is an error or attribute was found
		return result, errors.Wrapf(err, "Failed to get attribute %q for capability %q on node %q (instance %q)", attributeName, capabilityName, nodeName, instanceName)
	}

	// Now look at capability type for default
	if capabilityType != "" {
		result, isFunction, err := getTypeDefaultAttribute(ctx, deploymentID, capabilityType, attributeName, nestedKeys...)
		if err != nil {
			return nil, err
		}
		if result != nil {
			if !isFunction {
				return result, nil
			}
			return resolveValueAssignment(ctx, deploymentID, nodeName, instanceName, "", result, nestedKeys...)
		}
	}

	// No default found in type hierarchy
	// then traverse HostedOn relationships to find the value
	host, hostInstance, err := GetHostedOnNodeInstance(ctx, deploymentID, nodeName, instanceName)
	if err != nil {
		return nil, err
	}
	if host != "" {
		result, err = GetInstanceCapabilityAttributeValue(ctx, deploymentID, host, hostInstance, capabilityName, attributeName, nestedKeys...)
		if err != nil || result != nil {
			// If there is an error or attribute was found
			return result, err
		}
	}

	if capabilityType != "" {
		isEndpoint, err := IsTypeDerivedFrom(ctx, deploymentID, capabilityType,
			tosca.EndpointCapability)
		if err != nil {
			return nil, err
		}

		// TOSCA specification at :
		// http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_TYPE_CAPABILITIES_ENDPOINT
		// describes that the ip_address attribute of an endpoint is the IP address
		// as propagated up by the associated node’s host (Compute) container.
		if isEndpoint && attributeName == tosca.EndpointCapabilityIPAddressAttribute && host != "" {
			result, err = getIPAddressFromHost(ctx, deploymentID, host, hostInstance, nodeName, instanceName, capabilityName)
			if err != nil || result != nil {
				return result, err
			}
		}
	}
	// If still not found check properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
	return GetCapabilityPropertyValue(ctx, deploymentID, nodeName, capabilityName, attributeName, nestedKeys...)
}

// LookupInstanceCapabilityAttributeValue executes a lookup to retrieve instance capability attribute value when attribute can be long to retrieve
func LookupInstanceCapabilityAttributeValue(ctx context.Context, deploymentID, nodeName, instanceName, capabilityName, attributeName string, nestedKeys ...string) (string, error) {
	log.Debugf("Capability attribute:%q lookup for deploymentID:%q, node name:%q, instance:%q, capability:%q", attributeName, deploymentID, nodeName, instanceName, capabilityName)
	res := make(chan string, 1)
	go func() {
		for {
			if attr, _ := GetInstanceCapabilityAttributeValue(ctx, deploymentID, nodeName, instanceName, capabilityName, attributeName, nestedKeys...); attr != nil && attr.RawString() != "" {
				if attr != nil && attr.RawString() != "" {
					res <- attr.RawString()
					return
				}
			}

			select {
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				return
			}
		}
	}()

	select {
	case val := <-res:
		return val, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func getEndpointCapabilitityHostIPAttributeNameAndNetName(ctx context.Context, deploymentID, nodeName, capabilityName string) (string, *TOSCAValue, error) {
	// First check the network name in the capability property to find the right
	// IP address attribute (default: private address)
	ipAddressAttrName := "private_address"

	netName, err := GetCapabilityPropertyValue(ctx, deploymentID, nodeName, capabilityName, "network_name")
	if err != nil {
		return "", nil, err
	}

	if netName != nil && strings.ToLower(netName.RawString()) == "public" {
		ipAddressAttrName = "public_address"
	}
	return ipAddressAttrName, netName, nil
}

func getIPAddressFromHost(ctx context.Context, deploymentID, hostName, hostInstance, nodeName, instanceName, capabilityName string) (*TOSCAValue, error) {
	ipAddressAttrName, netName, err := getEndpointCapabilitityHostIPAttributeNameAndNetName(ctx, deploymentID, nodeName, capabilityName)
	if err != nil {
		return nil, err
	}
	result, err := GetInstanceAttributeValue(ctx, deploymentID, hostName, hostInstance,
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
func GetNodeCapabilityAttributeNames(ctx context.Context, deploymentID, nodeName, capabilityName string, exploreParents bool) ([]string, error) {

	capabilityType, err := GetNodeCapabilityType(ctx, deploymentID, nodeName, capabilityName)
	if err != nil {
		return nil, err
	}
	return GetTypeAttributes(ctx, deploymentID, capabilityType, exploreParents)

}

// SetInstanceCapabilityAttribute sets a capability attribute for a given node instance
func SetInstanceCapabilityAttribute(ctx context.Context, deploymentID, nodeName, instanceName, capabilityName, attributeName, value string) error {
	return SetInstanceCapabilityAttributeComplex(ctx, deploymentID, nodeName, instanceName, capabilityName, attributeName, value)
}

// SetInstanceCapabilityAttributeComplex sets an instance capability attribute that may be a literal or a complex data type
func SetInstanceCapabilityAttributeComplex(ctx context.Context, deploymentID, nodeName, instanceName, capabilityName, attributeName string, attributeValue interface{}) error {
	attrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "capabilities", capabilityName, "attributes", attributeName)
	err := consulutil.StoreConsulKeyWithJSONValue(attrPath, attributeValue)
	if err != nil {
		return err
	}
	err = notifyAndPublishCapabilityAttributeValueChange(ctx, deploymentID, nodeName, instanceName, capabilityName, attributeName, attributeValue)
	if err != nil {
		return err
	}
	return nil
}

// SetCapabilityAttributeForAllInstances sets the same capability attribute value to all instances of a given node.
//
// It does the same thing than iterating over instances ids and calling SetInstanceCapabilityAttribute but use
// a consulutil.ConsulStore to do it in parallel. We can expect better performances with a large number of instances
func SetCapabilityAttributeForAllInstances(ctx context.Context, deploymentID, nodeName, capabilityName, attributeName, attributeValue string) error {
	return SetCapabilityAttributeComplexForAllInstances(ctx, deploymentID, nodeName, capabilityName, attributeName, attributeValue)
}

// SetCapabilityAttributeComplexForAllInstances sets the same capability attribute value  that may be a literal or a complex data type to all instances of a given node.
//
// It does the same thing than iterating over instances ids and calling SetInstanceCapabilityAttributeComplex but use
// a consulutil.ConsulStore to do it in parallel. We can expect better performances with a large number of instances
func SetCapabilityAttributeComplexForAllInstances(ctx context.Context, deploymentID, nodeName, capabilityName, attributeName string, attributeValue interface{}) error {
	ids, err := GetNodeInstancesIds(ctx, deploymentID, nodeName)
	if err != nil {
		return err
	}
	for _, instanceName := range ids {
		attrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "capabilities", capabilityName, "attributes", attributeName)
		err := consulutil.StoreConsulKeyWithJSONValue(attrPath, attributeValue)
		if err != nil {
			return err
		}
		err = notifyAndPublishCapabilityAttributeValueChange(ctx, deploymentID, nodeName, instanceName, capabilityName, attributeName, attributeValue)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetNodeCapabilityType retrieves the type of a node template capability identified by its name
//
// This is a shorthand for GetNodeTypeCapabilityType
func GetNodeCapabilityType(ctx context.Context, deploymentID, nodeName, capabilityName string) (string, error) {
	// Now look at capability type for default
	nodeType, err := GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return "", err
	}
	return GetNodeTypeCapabilityType(ctx, deploymentID, nodeType, capabilityName)
}

// GetNodeTypeCapabilityType retrieves the type of a node type capability identified by its name
//
// It explores the type hierarchy (derived_from) to found the given capability.
// It may return an empty string if the capability is not found in the type hierarchy
func GetNodeTypeCapabilityType(ctx context.Context, deploymentID, nodeType, capabilityName string) (string, error) {
	typ := new(tosca.NodeType)
	err := getExpectedTypeFromName(ctx, deploymentID, nodeType, typ)
	if err != nil {
		return "", err
	}

	capDef, is := typ.Capabilities[capabilityName]
	if is {
		return capDef.Type, nil
	}

	if typ.DerivedFrom == "" {
		return "", nil
	}
	return GetNodeTypeCapabilityType(ctx, deploymentID, typ.DerivedFrom, capabilityName)
}

// GetNodeTypeCapabilityPropertyValueAssignment returns a Tosca value assignment
// related to the capability property default value for the defined node type.
// Its descends hierarchy and returns nil if no default value is found
func GetNodeTypeCapabilityPropertyValueAssignment(ctx context.Context, deploymentID, nodeType, capabilityName, propertyName string) (*tosca.ValueAssignment, error) {
	typ := new(tosca.NodeType)
	err := getExpectedTypeFromName(ctx, deploymentID, nodeType, typ)
	if err != nil {
		return nil, err
	}
	capDef, is := typ.Capabilities[capabilityName]
	if is && &capDef != nil {
		va, is := capDef.Properties[propertyName]
		if is && va != nil {
			return va, nil
		}
	}
	if typ.DerivedFrom == "" {
		return nil, nil
	}
	return GetNodeTypeCapabilityPropertyValueAssignment(ctx, deploymentID, typ.DerivedFrom, capabilityName, propertyName)
}

func notifyAndPublishCapabilityAttributeValueChange(ctx context.Context, deploymentID, nodeName, instanceName, capabilityName, attributeName string, attributeValue interface{}) error {
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
	return an.NotifyValueChange(ctx, deploymentID)
}

func isNodeCapabilityOfType(ctx context.Context, deploymentID, nodeName, capabilityName, derives string) (bool, error) {
	capType, err := GetNodeCapabilityType(ctx, deploymentID, nodeName, capabilityName)
	if err != nil {
		return false, err
	}
	if capType == "" {
		return false, nil
	}
	return IsTypeDerivedFrom(ctx, deploymentID, capType, derives)
}
