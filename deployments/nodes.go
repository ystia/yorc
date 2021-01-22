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
	"sort"
	"strconv"
	"strings"

	"github.com/fvbommel/sortorder"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
)

type nodeNotFoundError struct {
	name         string
	deploymentID string
}

func (e nodeNotFoundError) Error() string {
	return fmt.Sprintf("Looking for a node %q that do not exists in deployment %q.", e.name, e.deploymentID)
}

// IsNodeNotFoundError checks if the given error is a nodeNotFoundError error
func IsNodeNotFoundError(err error) bool {
	cause := errors.Cause(err)
	_, ok := cause.(nodeNotFoundError)
	return ok
}

func getNodeTemplate(ctx context.Context, deploymentID, nodeName string) (*tosca.NodeTemplate, error) {
	node := new(tosca.NodeTemplate)
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	exist, err := storage.GetStore(types.StoreTypeDeployment).Get(nodePath, node)
	if !exist {
		return nil, nodeNotFoundError{deploymentID: deploymentID, name: nodeName}
	}
	return node, err
}

// IsNodeDerivedFrom check if the node's type is derived from another type.
//
// Basically this function is a shorthand for GetNodeType and IsNodeTypeDerivedFrom.
func IsNodeDerivedFrom(ctx context.Context, deploymentID, nodeName, derives string) (bool, error) {
	nodeType, err := GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return false, err
	}
	return IsTypeDerivedFrom(ctx, deploymentID, nodeType, derives)
}

// GetDefaultNbInstancesForNode retrieves the default number of instances for a given node nodeName in deployment deploymentId.
//
// If the node has a capability that is or is derived from 'tosca.capabilities.Scalable' it will look for a property 'default_instances' in this capability of
// this node. Otherwise it will search for any relationship derived from 'tosca.relationships.HostedOn' in node requirements and reiterate
// the process. If a scalable node is finally found it returns the instances number.
// If there is no node with the Scalable capability at the end of the hosted on chain then assume that there is only one instance
func GetDefaultNbInstancesForNode(ctx context.Context, deploymentID, nodeName string) (uint32, error) {
	return getScalablePropertyForNode(ctx, deploymentID, nodeName, "default_instances")
}

// GetMaxNbInstancesForNode retrieves the maximum number of instances for a given node nodeName in deployment deploymentId.
//
// If the node has a capability that is or is derived from 'tosca.capabilities.Scalable' it will look for a property 'max_instances' in this capability of
// this node. Otherwise it will search for any relationship derived from 'tosca.relationships.HostedOn' in node requirements and reiterate
// the process. If a scalable node is finally found it returns the instances number.
// If there is no node with the Scalable capability at the end of the hosted on chain then assume that there is only one instance$
func GetMaxNbInstancesForNode(ctx context.Context, deploymentID, nodeName string) (uint32, error) {
	return getScalablePropertyForNode(ctx, deploymentID, nodeName, "max_instances")
}

// GetMinNbInstancesForNode retrieves the minimum number of instances for a given node nodeName in deployment deploymentId.
//
// If the node has a capability that is or is derived from 'tosca.capabilities.Scalable' it will look for a property 'min_instances' in this capability of
// this node. Otherwise it will search for any relationship derived from 'tosca.relationships.HostedOn' in node requirements and reiterate
// the process. If a scalable node is finally found it returns the instances number.œ
// If there is no node with the Scalable capability at the end of the hosted on chain then assume that there is only one instance
func GetMinNbInstancesForNode(ctx context.Context, deploymentID, nodeName string) (uint32, error) {
	return getScalablePropertyForNode(ctx, deploymentID, nodeName, "min_instances")
}

// getScalablePropertyForNode retrieves one of the scalable property on number of instances.
//
// If the node has a capability that is or is derived from 'tosca.capabilities.Scalable' it will look for the given property in this capability of
// this node. Otherwise it will search for any relationship derived from 'tosca.relationships.HostedOn' in node requirements and reiterate
// the process. If a scalable node is finally found it returns the instances number.
// If there is no node with the Scalable capability at the end of the hosted on chain then assume that there is only one instance
func getScalablePropertyForNode(ctx context.Context, deploymentID, nodeName, propertyName string) (uint32, error) {

	// TODO: Large part of GetDefaultNbInstancesForNode GetMaxNbInstancesForNode GetMinNbInstancesForNode could be factorized
	nodeType, err := GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return 0, err
	}
	capabilities, err := GetCapabilitiesOfType(ctx, deploymentID, nodeType, "tosca.capabilities.Scalable")
	if err != nil {
		return 0, err
	}
	if len(capabilities) > 0 {
		for _, capability := range capabilities {
			nbInst, err := GetCapabilityPropertyValue(ctx, deploymentID, nodeName, capability, propertyName)
			if err != nil {
				return 0, err
			}
			if nbInst != nil {
				var val uint64
				val, err = strconv.ParseUint(nbInst.RawString(), 10, 32)
				if err != nil {
					return 0, errors.Errorf("Not a valid integer for property %q of %q capability for node %q. Error: %v", propertyName, capability, nodeName, err)
				}
				return uint32(val), nil
			}
		}
	}
	// So we have to traverse the hosted on relationships...
	// Lets inspect the requirements to found hosted on relationships
	hostNode, err := GetHostedOnNode(ctx, deploymentID, nodeName)
	if err != nil {
		return 0, err
	} else if hostNode != "" {
		return getScalablePropertyForNode(ctx, deploymentID, hostNode, propertyName)
	}
	// Not hosted on a node having the Scalable capability lets assume one instance
	return 1, nil
}

// GetNbInstancesForNode retrieves the number of instances for a given node nodeName in deployment deploymentID.
func GetNbInstancesForNode(ctx context.Context, deploymentID, nodeName string) (uint32, error) {
	instancesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName)
	keys, err := consulutil.GetKeys(instancesPath)
	if err != nil {
		return 0, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return uint32(len(keys)), nil
}

// GetNodeInstancesIds returns the names of the different instances for a given node.
//
// It may be an empty array if the given node is not HostedOn a scalable node.
func GetNodeInstancesIds(ctx context.Context, deploymentID, nodeName string) ([]string, error) {
	names := make([]string, 0)
	instancesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName)
	instances, err := consulutil.GetKeys(instancesPath)
	if err != nil {
		return names, errors.Wrap(err, "Consul communication error")
	}

	for _, instance := range instances {
		names = append(names, path.Base(instance))
	}

	if len(names) == 0 {
		// Check if this is a node to substitute
		substitutable, err := isSubstitutableNode(ctx, deploymentID, nodeName)
		if err != nil {
			return names, err
		}
		if substitutable {
			log.Debugf("Found no instance for %s %s, getting substitutable node instance", deploymentID, nodeName)
			names, err = getSubstitutionNodeInstancesIds(ctx, deploymentID, nodeName)
			if err != nil {
				return names, err
			}
		}
	}

	sort.Sort(sortorder.Natural(names))
	log.Debugf("Found node instances %v for %s %s", names, deploymentID, nodeName)
	return names, nil
}

// GetHostedOnNode returns the node name of the node defined in the first found relationship derived from "tosca.relationships.HostedOn"
//
// If there is no HostedOn relationship for this node then it returns an empty string
func GetHostedOnNode(ctx context.Context, deploymentID, nodeName string) (string, error) {
	node, _, err := getHostedOnNodeAndInstance(ctx, deploymentID, nodeName, "")
	return node, err
}

func getHostedOnNodeAndInstance(ctx context.Context, deploymentID, nodeName, instanceName string) (string, string, error) {
	node, err := getNodeTemplate(ctx, deploymentID, nodeName)
	if err != nil {
		return "", "", err
	}

	// So we have to traverse the hosted on relationships...
	// Lets inspect the requirements to found hosted on relationships
	for index, reqList := range node.Requirements {
		for _, req := range reqList {
			if req.Node == "" || req.Relationship == "" {
				continue
			}
			ok, err := IsTypeDerivedFrom(ctx, deploymentID, req.Relationship, "tosca.relationships.HostedOn")
			if err != nil {
				return "", "", err
			}
			if ok {
				if instanceName == "" {
					return req.Node, "", err
				}
				// Get the corresponding target instances
				hostNodeName, hostInstances, err := GetTargetInstanceForRequirement(ctx, deploymentID, nodeName, strconv.Itoa(index), instanceName)
				if err != nil {
					return "", "", err
				}
				if hostNodeName != "" && len(hostInstances) > 0 {
					return hostNodeName, hostInstances[0], nil
				}
			}
		}
	}
	return "", "", nil
}

// GetHostedOnNodeInstance returns the node name and instance name of the instance
// defined in the first found relationship derived from "tosca.relationships.HostedOn"
//
// If there is no HostedOn relationship for this node then it returns an empty string
func GetHostedOnNodeInstance(ctx context.Context, deploymentID, nodeName, instanceName string) (string, string, error) {
	return getHostedOnNodeAndInstance(ctx, deploymentID, nodeName, instanceName)
}

// IsHostedOn checks if a given nodeName is hosted on another given node hostedOn by traversing the hostedOn hierarchy
func IsHostedOn(ctx context.Context, deploymentID, nodeName, hostedOn string) (bool, error) {
	if host, err := GetHostedOnNode(ctx, deploymentID, nodeName); err != nil {
		return false, err
	} else if host == "" {
		return false, nil
	} else if host != hostedOn {
		return IsHostedOn(ctx, deploymentID, host, hostedOn)
	}
	return true, nil

}

// GetNodesHostedOn returns the list of nodes that are hosted on a given node
func GetNodesHostedOn(ctx context.Context, deploymentID, hostNode string) ([]string, error) {
	// Thinking: maybe we can store at parsing time for each node the list of nodes on which it is hosted on and/or the opposite rather than re-scan the whole node list
	nodesList, err := GetNodes(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	stackNodes := nodesList[:0]
	for _, node := range nodesList {
		var hostedOn bool
		hostedOn, err = IsHostedOn(ctx, deploymentID, node, hostNode)
		if err != nil {
			return nil, err
		}

		if hostedOn {
			stackNodes = append(stackNodes, node)
		}
	}
	return stackNodes, nil
}

func getNodeAttributeValue(ctx context.Context, deploymentID, nodeName, instanceName, attributeName, attrDataType string, nestedKeys ...string) (*TOSCAValue, error) {
	node, err := getNodeTemplate(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}

	// Check type if node is substitutable
	nodeType, err := checkTypeForSubstitutableNode(ctx, deploymentID, nodeName, node.Type)
	if err != nil {
		return nil, err
	}

	// Check if the node attribute value can be retrieved at node template level
	var va *tosca.ValueAssignment
	if node.Attributes != nil {
		va = node.Attributes[attributeName]
	}

	value, err := getValueAssignment(ctx, deploymentID, nodeName, instanceName, "", attrDataType, va, nestedKeys...)
	if err != nil || value != nil {
		return value, err
	}

	// Not found look at node type
	value, isFunction, err := getTypeDefaultAttribute(ctx, deploymentID, nodeType, attributeName, nestedKeys...)
	if err != nil {
		return nil, err
	}
	if value != nil {
		if !isFunction {
			return value, nil
		}
		return resolveValueAssignment(ctx, deploymentID, nodeName, instanceName, "", value, nestedKeys...)
	}

	// No default found in type hierarchy
	// then traverse HostedOn relationships to find the value
	host, err := GetHostedOnNode(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	if host != "" {
		value, err := getNodeAttributeValue(ctx, deploymentID, host, instanceName, attributeName, attrDataType, nestedKeys...)
		if err != nil || value != nil {
			return value, err
		}
	}
	// Not found anywhere
	return nil, nil
}

// GetNodePropertyValue retrieves the value for a given property in a given node
//
// It returns true if a value is found false otherwise as first return parameter.
// If the property is not found in the node then the type hierarchy is explored to find a default value.
// If the property is still not found then it will explore the HostedOn hierarchy
func GetNodePropertyValue(ctx context.Context, deploymentID, nodeName, propertyName string, nestedKeys ...string) (*TOSCAValue, error) {
	node, err := getNodeTemplate(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}

	// Check type if node is substitutable
	nodeType, err := checkTypeForSubstitutableNode(ctx, deploymentID, nodeName, node.Type)
	if err != nil {
		return nil, err
	}

	var propDataType string
	hasProp, err := TypeHasProperty(ctx, deploymentID, nodeType, propertyName, true)
	if err != nil {
		return nil, err
	}
	if hasProp {
		propDataType, err = GetTypePropertyDataType(ctx, deploymentID, nodeType, propertyName)
		if err != nil {
			return nil, err
		}
	}

	// Check if the node property value can be retrieved at node template level
	var va *tosca.ValueAssignment
	if node.Properties != nil {
		va = node.Properties[propertyName]
	}

	value, err := getValueAssignment(ctx, deploymentID, nodeName, "", "", propDataType, va, nestedKeys...)
	if err != nil || value != nil {
		return value, err
	}

	// Not found look at node type
	value, isFunction, err := getTypeDefaultProperty(ctx, deploymentID, nodeType, propertyName, nestedKeys...)
	if err != nil {
		return nil, err
	}
	if value != nil {
		if !isFunction {
			return value, nil
		}
		return resolveValueAssignment(ctx, deploymentID, nodeName, "", "", value, nestedKeys...)
	}

	// No default found in type hierarchy
	// then traverse HostedOn relationships to find the value
	host, err := GetHostedOnNode(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	if host != "" {
		value, err := GetNodePropertyValue(ctx, deploymentID, host, propertyName, nestedKeys...)
		if err != nil || value != nil {
			return value, err
		}
	}

	if hasProp {
		propDataType, err = GetTypePropertyDataType(ctx, deploymentID, nodeType, propertyName)
		if err != nil {
			return nil, err
		}
		// Check if the whole property is optional
		isRequired, err := IsTypePropertyRequired(ctx, deploymentID, nodeType, propertyName)
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

// GetStringNodeProperty returns the string value of a property.
// If this value is empty and the argument mandatory is true, an error is returned
func GetStringNodeProperty(ctx context.Context, deploymentID, nodeName, propertyName string, mandatory bool) (string, error) {

	result, err := GetNodePropertyValue(ctx, deploymentID, nodeName, propertyName)
	if err != nil {
		return "", err
	}

	if result == nil && mandatory {
		return "", errors.Errorf("Missing value for mandatory parameter %s of %s", propertyName, nodeName)
	}
	r := ""
	if result != nil {
		r = result.RawString()
	}
	return r, nil
}

// GetBooleanNodeProperty returns the boolean value of a property (default: false)
func GetBooleanNodeProperty(ctx context.Context, deploymentID, nodeName, propertyName string) (bool, error) {
	var result bool
	va, err := GetNodePropertyValue(ctx, deploymentID, nodeName, propertyName)
	if err != nil {
		return result, err
	}

	if va != nil && va.RawString() != "" {
		result, err = strconv.ParseBool(va.RawString())
		if err != nil {
			// TODO: who reads logs in production? Should be in app logs
			log.Printf("Unexpected value for %s %s: '%+v', considering it is set to 'false'", nodeName, propertyName, va)
		}
	}
	return result, nil
}

// GetStringArrayNodeProperty returns the string Array value of a node property (default: false)
// This function returns a nil array for an empty string property value
func GetStringArrayNodeProperty(ctx context.Context, deploymentID, nodeName, propertyName string) ([]string, error) {
	var result []string
	strValue, err := GetNodePropertyValue(ctx, deploymentID, nodeName, propertyName)
	if err != nil {
		return nil, err
	}

	if strValue != nil && strValue.RawString() != "" {
		values := strings.Split(strValue.RawString(), ",")
		for _, val := range values {
			result = append(result, strings.TrimSpace(val))
		}
	}

	return result, nil
}

// GetKeyValuePairsNodeProperty returns a key/value string map value of a node property (default: false)
func GetKeyValuePairsNodeProperty(ctx context.Context, deploymentID, nodeName, propertyName string) (map[string]string, error) {
	var result map[string]string
	strValue, err := GetNodePropertyValue(ctx, deploymentID, nodeName, propertyName)
	if err != nil {
		return nil, err
	}

	if strValue != nil && strValue.RawString() != "" {
		values := strings.Split(strValue.RawString(), ",")
		result = make(map[string]string, len(values))
		for _, val := range values {
			keyValuePair := strings.SplitN(val, "=", 2)
			if len(keyValuePair) != 2 {
				return result, errors.Errorf("Expected KEY=VALUE format, got %s for property %s on %s",
					val, propertyName, nodeName)
			}
			result[strings.TrimSpace(keyValuePair[0])] =
				strings.TrimSpace(keyValuePair[1])
		}
	}
	return result, nil
}

// GetNodeAttributesValues retrieves the values for a given attribute in a given node.
//
// As a node may have multiple instances and attributes may be instance-scoped, then returned result is a map with the instance name as key
// and the retrieved attributes as values.
//
// If the property is not found in the node then the type hierarchy is explored to find a default value.
// If the property is still not found then it will explore the HostedOn hierarchy.
func GetNodeAttributesValues(ctx context.Context, deploymentID, nodeName, attributeName string, nestedKeys ...string) (map[string]*TOSCAValue, error) {
	instances, err := GetNodeInstancesIds(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}

	attributes := make(map[string]*TOSCAValue)
	for _, instance := range instances {
		result, err := GetInstanceAttributeValue(ctx, deploymentID, nodeName, instance, attributeName, nestedKeys...)
		if err != nil {
			return nil, err
		}
		attributes[instance] = result
	}
	return attributes, nil
}

// GetStringNodePropertyValue returns the string value of a property.
// If there is no such property defined, an empty string is returned
func GetStringNodePropertyValue(ctx context.Context, deploymentID, nodeName, propertyName string, nestedKeys ...string) (string, error) {

	var result string
	propVal, err := GetNodePropertyValue(ctx, deploymentID, nodeName, propertyName, nestedKeys...)
	if err != nil {
		return "", err
	}

	if propVal != nil {
		result = propVal.RawString()
	}
	return result, err
}

// GetNodes returns the names of the different nodes for a given deployment.
func GetNodes(ctx context.Context, deploymentID string) ([]string, error) {
	names := make([]string, 0)
	nodesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes")
	nodes, err := storage.GetStore(types.StoreTypeDeployment).Keys(nodesPath)
	if err != nil {
		return names, err
	}
	for _, node := range nodes {
		names = append(names, path.Base(node))
	}
	return names, nil
}

// GetNodeType returns the type of a given node identified by its name
func GetNodeType(ctx context.Context, deploymentID, nodeName string) (string, error) {
	var nodeType string
	node, err := getNodeTemplate(ctx, deploymentID, nodeName)
	if err != nil {
		return "", errors.Wrapf(err, "Can't get type for node %q", nodeName)
	}
	nodeType = node.Type
	if node.Type == "" {
		return "", errors.Errorf("Missing mandatory parameter \"type\" for node %q", nodeName)
	}

	// If the corresponding node is substitutable, get its real node type
	return checkTypeForSubstitutableNode(ctx, deploymentID, nodeName, nodeType)
}

func checkTypeForSubstitutableNode(ctx context.Context, deploymentID, nodeName, nodeType string) (string, error) {
	// If the corresponding node is substitutable, get its real node type
	substitutable, err := isSubstitutableNode(ctx, deploymentID, nodeName)
	if err != nil {
		return "", err
	}
	if substitutable {
		nodeType, err = getSubstitutableNodeType(ctx, deploymentID, nodeName, nodeType)
		if err != nil {
			return "", err
		}
	}
	return nodeType, nil
}

// GetNodeAttributesNames retrieves the list of existing attributes for a given node.
func GetNodeAttributesNames(ctx context.Context, deploymentID, nodeName string) ([]string, error) {
	attributesSet := make(map[string]struct{})

	// Look at node type
	node, err := getNodeTemplate(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	for k := range node.Attributes {
		attributesSet[k] = struct{}{}
	}

	// Check type if node is substitutable
	nodeType, err := checkTypeForSubstitutableNode(ctx, deploymentID, nodeName, node.Type)
	if err != nil {
		return nil, err
	}
	typeAttrs, err := GetTypeAttributes(ctx, deploymentID, nodeType, true)
	if err != nil {
		return nil, err
	}
	for _, attr := range typeAttrs {
		attributesSet[attr] = struct{}{}
	}

	// Look at instances attributes
	instances, err := GetNodeInstancesIds(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	nodeInstancesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName)
	for _, instance := range instances {
		err = storeSubKeysInSet(path.Join(nodeInstancesPath, instance, "attributes"), attributesSet)
		if err != nil {
			return nil, err
		}
	}

	// Alien4Cloud did not yet implement the management of capability attributes
	// in substitution mappings. It expects for now to read these capability
	// attributes as node attributes
	err = storeSubstitutionMappingAttributeNamesInSet(ctx, deploymentID, nodeName, attributesSet)
	if err != nil {
		return nil, err
	}

	attributesList := make([]string, len(attributesSet))
	i := 0
	for attr := range attributesSet {
		attributesList[i] = attr
		i++
	}

	log.Debugf("Found attributes %v for %s, %s", attributesList, deploymentID, nodeName)

	return attributesList, nil
}

// GetTypeAttributesNames returns the list of attributes names found in the type hierarchy
func GetTypeAttributesNames(ctx context.Context, deploymentID, typeName string) ([]string, error) {
	return GetTypeAttributes(ctx, deploymentID, typeName, true)
}

// storeSubKeysInSet store Consul keys directly living under parentPath into the given set.
func storeSubKeysInSet(parentPath string, set map[string]struct{}) error {
	parentPath = strings.TrimSpace(parentPath)
	if !strings.HasSuffix(parentPath, "/") {
		parentPath += "/"
	}
	keys, err := consulutil.GetKeys(parentPath)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, key := range keys {
		attr := path.Base(key)
		if _, found := set[attr]; !found {
			set[attr] = struct{}{}
		}
	}
	return nil
}

func getInstancesDependentLinkedNodes(ctx context.Context, deploymentID, nodeName string) ([]string, error) {
	localStorageReqs, err := GetRequirementsByTypeForNode(ctx, deploymentID, nodeName, "local_storage")
	if err != nil {
		return nil, err
	}
	nodesList := make([]string, 0)
	for _, req := range localStorageReqs {
		nodesList = append(nodesList, req.Node)
	}
	networkReqs, err := GetRequirementsByTypeForNode(ctx, deploymentID, nodeName, "network")
	if err != nil {
		return nil, err
	}
	assignmentReqs, err := GetRequirementsByTypeForNode(ctx, deploymentID, nodeName, "assignment")
	if err != nil {
		return nil, err
	}
	allReqs := append(networkReqs, assignmentReqs...)
	for _, req := range allReqs {
		if req.Capability != "yorc.capabilities.openstack.FIPConnectivity" && req.Capability != "yorc.capabilities.Assignable" {
			// Neither a FIP connectivity nor an assignable cap: see next
			continue
		}
		if req.Node == "" {
			return nil, errors.Errorf("Missing attribute \"node\" for requirement %+v on node %q", req, nodeName)

		}
		nodesList = append(nodesList, req.Node)
	}
	return nodesList, nil
}

// SelectNodeStackInstances selects a given number of instances of the given node, all the nodes hosted on this one and all nodes linked to it.
//
// For each node it returns a coma separated list of selected instances
func SelectNodeStackInstances(ctx context.Context, deploymentID, nodeName string, instancesDelta int) (map[string]string, error) {
	nodesStack, err := GetNodesHostedOn(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	nodesStack = append(nodesStack, nodeName)
	linkedNodes, err := getInstancesDependentLinkedNodes(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	nodesStack = append(nodesStack, linkedNodes...)

	// TODO: Improve the way we relate node instances names to dependent (linked nodes) or hosted on instances names
	instances, err := GetNodeInstancesIds(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	instancesList := strings.Join(instances[len(instances)-int(instancesDelta):], ",")
	nodesMap := make(map[string]string)
	for _, node := range nodesStack {
		nodesMap[node] = instancesList
	}
	return nodesMap, nil
}

// CreateNewNodeStackInstances create the given number of new instances of the given node and all other nodes hosted on this one and all linked nodes
//
// CreateNewNodeStackInstances returns a map of newly created instances IDs indexed by node name
func CreateNewNodeStackInstances(ctx context.Context, deploymentID, nodeName string, instances int) (map[string]string, error) {
	nodesMap := make(map[string]string)
	_, errGroup, consulStore := consulutil.WithContext(ctx)

	nodes, err := GetNodes(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	stackNodes := nodes[:0]
	for _, node := range nodes {
		if node == nodeName {
			stackNodes = append(stackNodes, node)
		} else {
			var hostedOnNode bool
			if hostedOnNode, err = IsHostedOn(ctx, deploymentID, node, nodeName); err != nil {
				return nil, err
			} else if hostedOnNode {
				stackNodes = append(stackNodes, node)
			}
		}
	}

	linkedNodes, err := getInstancesDependentLinkedNodes(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	stackNodes = append(stackNodes, linkedNodes...)

	// Now get existing nodes instances ids to have the
	existingIds, err := GetNodeInstancesIds(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	instancesIDs := make([]string, 0)
	initialID := len(existingIds)
	// TODO: rethink the instances ids
	for i := initialID; i < initialID+instances; i++ {
		id := strconv.FormatUint(uint64(i), 10)
		instancesIDs = append(instancesIDs, id)
		for _, stackNode := range stackNodes {
			createNodeInstance(consulStore, deploymentID, stackNode, id)
			if _, ok := nodesMap[stackNode]; ok {
				nodesMap[stackNode] = nodesMap[stackNode] + "," + id
			} else {
				nodesMap[stackNode] = id
			}
			addOrRemoveInstanceFromTargetRelationship(ctx, deploymentID, stackNode, id, true)
		}
	}
	// Wait for instances to be created
	err = errGroup.Wait()
	if err != nil {
		return nil, err
	}
	// Then create relationship instances
	_, errGroup, consulStore = consulutil.WithContext(ctx)
	for node := range nodesMap {
		createRelationshipInstances(ctx, consulStore, deploymentID, node)
	}

	// Wait for relationship instances to be created
	err = errGroup.Wait()
	if err != nil {
		return nil, err
	}
	// Add possible attribute notifications
	err = enhanceAttributes(ctx, deploymentID, stackNodes)
	return nodesMap, errors.Wrapf(err, "Failed to create instances for node %q", nodeName)

}

// createNodeInstance creates required elements for a new node
func createNodeInstance(consulStore consulutil.ConsulStore, deploymentID, nodeName, instanceName string) {
	ctx := context.Background()
	instancePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName)
	toscaID := nodeName + "-" + instanceName
	consulStore.StoreConsulKeyAsString(path.Join(instancePath, instanceName, "attributes/state"), tosca.NodeStateInitial.String())
	consulStore.StoreConsulKeyAsString(path.Join(instancePath, instanceName, "attributes/tosca_name"), nodeName)
	consulStore.StoreConsulKeyAsString(path.Join(instancePath, instanceName, "attributes/tosca_id"), toscaID)
	// Publish a status change event and attribute update
	_, err := events.PublishAndLogInstanceStatusChange(ctx, deploymentID, nodeName, instanceName, tosca.NodeStateInitial.String())
	if err != nil {
		log.Printf("%+v", err)
	}

	// Publish a status change event
	attrs := make(map[string]string)
	attrs["state"] = tosca.NodeStateInitial.String()
	attrs["tosca_name"] = nodeName
	attrs["tosca_id"] = toscaID
	err = events.PublishAndLogMapAttributeValueChange(ctx, deploymentID, nodeName, instanceName, attrs, "updated")
	if err != nil {
		log.Printf("%+v", err)
	}
}

// DoesNodeExist checks if a given node exist in a deployment
func DoesNodeExist(ctx context.Context, deploymentID, nodeName string) (bool, error) {
	exist, err := storage.GetStore(types.StoreTypeDeployment).Exist(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName))
	if err != nil {
		return false, err
	}
	return exist, nil
}

// GetNodeMetadata retrieves the related node metadata key if exists
func GetNodeMetadata(ctx context.Context, deploymentID, nodeName, key string) (bool, string, error) {
	node, err := getNodeTemplate(ctx, deploymentID, nodeName)
	if err != nil {
		return false, "", err
	}
	value, exist := node.Metadata[key]
	if !exist || value == "" {
		return false, "", nil
	}
	return true, value, nil
}

// NodeHasAttribute returns true if the node type has an attribute named attributeName defined
//
// exploreParents switch enable attribute check on parent types
//
// This is basically a shorthand for GetNodeType then TypeHasAttribute
func NodeHasAttribute(ctx context.Context, deploymentID, nodeName, attributeName string, exploreParents bool) (bool, error) {
	typeName, err := GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return false, err
	}
	return TypeHasAttribute(ctx, deploymentID, typeName, attributeName, exploreParents)
}

// NodeHasProperty returns true if the node type has a property named propertyName defined
//
// exploreParents switch enable property check on parent types
//
// This is basically a shorthand for GetNodeType then TypeHasProperty
func NodeHasProperty(ctx context.Context, deploymentID, nodeName, propertyName string, exploreParents bool) (bool, error) {
	typeName, err := GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return false, err
	}
	return TypeHasProperty(ctx, deploymentID, typeName, propertyName, exploreParents)
}

// DeleteNode deletes the given node from the Consul store
func DeleteNode(ctx context.Context, deploymentID, nodeName string) error {
	return storage.GetStore(types.StoreTypeDeployment).Delete(ctx, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName)+"/", true)
}
