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
	"github.com/hashicorp/consul/api"
	"github.com/ystia/yorc/tosca"
)

// This file contains deprecated functions that will be removed by Yorc 4.0
// TODO: should we setup a build tag?

// GetCapabilityProperty  retrieves the value for a given property in a given node capability
//
// It returns true if a value is found false otherwise as first return parameter.
// If the property is not found in the node then the type hierarchy is explored to find a default value.
//
// Deprecated: use GetCapabilityPropertyValue instead
//             will be removed in Yorc 4.0
func GetCapabilityProperty(kv *api.KV, deploymentID, nodeName, capabilityName, propertyName string, nestedKeys ...string) (bool, string, error) {
	return convertTOSCAValueResultToDeprecated(GetCapabilityPropertyValue(kv, deploymentID, nodeName, capabilityName, propertyName, nestedKeys...))
}

// GetInstanceCapabilityAttribute retrieves the value for a given attribute in a given node instance capability
//
// It returns true if a value is found false otherwise as first return parameter.
// If the attribute is not found in the node then the type hierarchy is explored to find a default value.
// If still not found check properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
//
// Deprecated: use GetInstanceCapabilityAttributeValue instead
//             will be removed in Yorc 4.0
func GetInstanceCapabilityAttribute(kv *api.KV, deploymentID, nodeName, instanceName, capabilityName, attributeName string, nestedKeys ...string) (bool, string, error) {
	return convertTOSCAValueResultToDeprecated(GetInstanceCapabilityAttributeValue(kv, deploymentID, nodeName, instanceName, capabilityName, attributeName, nestedKeys...))
}

// GetNodeTypeCapabilityProperty retrieves the property value of a node type capability identified by its name
//
// It explores the type hierarchy (derived_from) to found the given capability.
//
// Deprecated: use GetNodeTypeCapabilityPropertyValue instead
//             will be removed in Yorc 4.0
func GetNodeTypeCapabilityProperty(kv *api.KV, deploymentID, nodeType, capabilityName, propertyName, propDataType string, nestedKeys ...string) (bool, string, error) {
	return convertTOSCAValueResultToDeprecated(GetNodeTypeCapabilityPropertyValue(kv, deploymentID, nodeType, capabilityName, propertyName, propDataType, nestedKeys...))
}

// SetInstanceStateString stores the state of a given node instance and publishes a status change event
//
// Deprecated: use SetInstanceStateStringWithContextualLogs instead
//             will be removed in Yorc 4.0
func SetInstanceStateString(kv *api.KV, deploymentID, nodeName, instanceName, state string) error {
	return SetInstanceStateStringWithContextualLogs(nil, kv, deploymentID, nodeName, instanceName, state)
}

// SetInstanceState stores the state of a given node instance and publishes a status change event
//
// Deprecated: use SetInstanceStateWithContextualLogs instead
//             will be removed in Yorc 4.0
func SetInstanceState(kv *api.KV, deploymentID, nodeName, instanceName string, state tosca.NodeState) error {
	return SetInstanceStateStringWithContextualLogs(nil, kv, deploymentID, nodeName, instanceName, state.String())
}

// GetInstanceAttribute retrieves the given attribute for a node instance
//
// It returns true if a value is found false otherwise as first return parameter.
// If the attribute is not found in the node then the type hierarchy is explored to find a default value.
// If the attribute is still not found then it will explore the HostedOn hierarchy.
// If still not found then it will check node properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
//
// Deprecated: use GetInstanceAttributeValue instead
//             will be removed in Yorc 4.0
func GetInstanceAttribute(kv *api.KV, deploymentID, nodeName, instanceName, attributeName string, nestedKeys ...string) (bool, string, error) {
	return convertTOSCAValueResultToDeprecated(GetInstanceAttributeValue(kv, deploymentID, nodeName, instanceName, attributeName, nestedKeys...))
}

// SetNodeInstanceAttribute sets an attribute value to a node instance
//
// Deprecated: Use SetInstanceAttribute
//             will be removed in Yorc 4.0
func SetNodeInstanceAttribute(kv *api.KV, deploymentID, nodeName, instanceName, attributeName, attributeValue string) error {
	return SetInstanceAttribute(deploymentID, nodeName, instanceName, attributeName, attributeValue)
}

// GetRelationshipPropertyFromRequirement returns the value of a relationship's property identified by a requirement index on a node
//
// Deprecated: use GetRelationshipPropertyValueFromRequirement instead
//             will be removed in Yorc 4.0
func GetRelationshipPropertyFromRequirement(kv *api.KV, deploymentID, nodeName, requirementIndex, propertyName string, nestedKeys ...string) (bool, string, error) {
	return convertTOSCAValueResultToDeprecated(GetRelationshipPropertyValueFromRequirement(kv, deploymentID, nodeName, requirementIndex, propertyName, nestedKeys...))
}

// GetTopologyOutput returns the value of a given topology output
//
// Deprecated: use GetTopologyOutputValue instead
//             will be removed in Yorc 4.0
func GetTopologyOutput(kv *api.KV, deploymentID, outputName string, nestedKeys ...string) (bool, string, error) {
	return convertTOSCAValueResultToDeprecated(GetTopologyOutputValue(kv, deploymentID, outputName, nestedKeys...))
}

// GetNodeProperty retrieves the value for a given property in a given node
//
// It returns true if a value is found false otherwise as first return parameter.
// If the property is not found in the node then the type hierarchy is explored to find a default value.
// If the property is still not found then it will explore the HostedOn hierarchy.
//
// Deprecated: use GetNodePropertyValue instead
func GetNodeProperty(kv *api.KV, deploymentID, nodeName, propertyName string, nestedKeys ...string) (bool, string, error) {
	return convertTOSCAValueResultToDeprecated(GetNodePropertyValue(kv, deploymentID, nodeName, propertyName, nestedKeys...))
}

// GetRelationshipAttributeFromRequirement retrieves the value for a given attribute in a given node instance requirement
//
// It returns true if a value is found false otherwise as first return parameter.
// If the attribute is not found in the node then the type hierarchy is explored to find a default value.
// If still not found check properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
//
// Deprecated: use GetRelationshipAttributeValueFromRequirement instead
func GetRelationshipAttributeFromRequirement(kv *api.KV, deploymentID, nodeName, instanceName, requirementIndex, attributeName string, nestedKeys ...string) (bool, string, error) {
	return convertTOSCAValueResultToDeprecated(GetRelationshipAttributeValueFromRequirement(kv, deploymentID, nodeName, instanceName, requirementIndex, attributeName, nestedKeys...))
}

// used to reduce duplication
func convertTOSCAValueResultToDeprecated(v *TOSCAValue, err error) (bool, string, error) {
	if err != nil || v == nil {
		return false, "", err
	}
	// We return RawString for backward compatibility but we loose the IsSecret context
	return true, v.RawString(), nil
}
