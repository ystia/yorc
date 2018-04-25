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
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tosca"

	"vbom.ml/util/sortorder"
)

// IsNodeDerivedFrom check if the node's type is derived from another type.
//
// Basically this function is a shorthand for GetNodeType and IsNodeTypeDerivedFrom.
func IsNodeDerivedFrom(kv *api.KV, deploymentID, nodeName, derives string) (bool, error) {
	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, err
	}
	return IsTypeDerivedFrom(kv, deploymentID, nodeType, derives)
}

// GetDefaultNbInstancesForNode retrieves the default number of instances for a given node nodeName in deployment deploymentId.
//
// If the node has a capability that is or is derived from 'tosca.capabilities.Scalable' it will look for a property 'default_instances' in this capability of
// this node. Otherwise it will search for any relationship derived from 'tosca.relationships.HostedOn' in node requirements and reiterate
// the process. If a scalable node is finally found it returns the instances number.
// If there is no node with the Scalable capability at the end of the hosted on chain then assume that there is only one instance
func GetDefaultNbInstancesForNode(kv *api.KV, deploymentID, nodeName string) (uint32, error) {
	return getScalablePropertyForNode(kv, deploymentID, nodeName, "default_instances")
}

// GetMaxNbInstancesForNode retrieves the maximum number of instances for a given node nodeName in deployment deploymentId.
//
// If the node has a capability that is or is derived from 'tosca.capabilities.Scalable' it will look for a property 'max_instances' in this capability of
// this node. Otherwise it will search for any relationship derived from 'tosca.relationships.HostedOn' in node requirements and reiterate
// the process. If a scalable node is finally found it returns the instances number.
// If there is no node with the Scalable capability at the end of the hosted on chain then assume that there is only one instance$
func GetMaxNbInstancesForNode(kv *api.KV, deploymentID, nodeName string) (uint32, error) {
	return getScalablePropertyForNode(kv, deploymentID, nodeName, "max_instances")
}

// GetMinNbInstancesForNode retrieves the minimum number of instances for a given node nodeName in deployment deploymentId.
//
// If the node has a capability that is or is derived from 'tosca.capabilities.Scalable' it will look for a property 'min_instances' in this capability of
// this node. Otherwise it will search for any relationship derived from 'tosca.relationships.HostedOn' in node requirements and reiterate
// the process. If a scalable node is finally found it returns the instances number.œ
// If there is no node with the Scalable capability at the end of the hosted on chain then assume that there is only one instance
func GetMinNbInstancesForNode(kv *api.KV, deploymentID, nodeName string) (uint32, error) {
	return getScalablePropertyForNode(kv, deploymentID, nodeName, "min_instances")
}

// getScalablePropertyForNode retrieves one of the scalable property on number of instances.
//
// If the node has a capability that is or is derived from 'tosca.capabilities.Scalable' it will look for the given property in this capability of
// this node. Otherwise it will search for any relationship derived from 'tosca.relationships.HostedOn' in node requirements and reiterate
// the process. If a scalable node is finally found it returns the instances number.
// If there is no node with the Scalable capability at the end of the hosted on chain then assume that there is only one instance
func getScalablePropertyForNode(kv *api.KV, deploymentID, nodeName, propertyName string) (uint32, error) {

	// TODO: Large part of GetDefaultNbInstancesForNode GetMaxNbInstancesForNode GetMinNbInstancesForNode could be factorized
	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return 0, err
	}
	capabilities, err := GetCapabilitiesOfType(kv, deploymentID, nodeType, "tosca.capabilities.Scalable")
	if err != nil {
		return 0, err
	}
	if len(capabilities) > 0 {
		for _, capability := range capabilities {
			var found bool
			var nbInst string
			found, nbInst, err = GetCapabilityProperty(kv, deploymentID, nodeName, capability, propertyName)
			if err != nil {
				return 0, err
			}
			if found {
				var val uint64
				val, err = strconv.ParseUint(nbInst, 10, 32)
				if err != nil {
					return 0, errors.Errorf("Not a valid integer for property %q of %q capability for node %q. Error: %v", propertyName, capability, nodeName, err)
				}
				return uint32(val), nil
			}
		}
	}
	// So we have to traverse the hosted on relationships...
	// Lets inspect the requirements to found hosted on relationships
	hostNode, err := GetHostedOnNode(kv, deploymentID, nodeName)
	if err != nil {
		return 0, err
	} else if hostNode != "" {
		return getScalablePropertyForNode(kv, deploymentID, hostNode, propertyName)
	}
	// Not hosted on a node having the Scalable capability lets assume one instance
	return 1, nil
}

// GetNbInstancesForNode retrieves the number of instances for a given node nodeName in deployment deploymentID.
func GetNbInstancesForNode(kv *api.KV, deploymentID, nodeName string) (uint32, error) {
	instancesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName)
	keys, _, err := kv.Keys(instancesPath+"/", "/", nil)
	if err != nil {
		return 0, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return uint32(len(keys)), nil
}

// GetNodeInstancesIds returns the names of the different instances for a given node.
//
// It may be an empty array if the given node is not HostedOn a scalable node.
func GetNodeInstancesIds(kv *api.KV, deploymentID, nodeName string) ([]string, error) {
	names := make([]string, 0)
	instancesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName)
	instances, _, err := kv.Keys(instancesPath+"/", "/", nil)
	if err != nil {
		return names, errors.Wrap(err, "Consul communication error")
	}

	for _, instance := range instances {
		names = append(names, path.Base(instance))
	}

	if len(names) == 0 {
		// Check if this is a node to substitute
		substitutable, err := isSubstitutableNode(kv, deploymentID, nodeName)
		if err != nil {
			return names, err
		}
		if substitutable {
			log.Debugf("Found no instance for %s %s, getting substitutable node instance", deploymentID, nodeName)
			names, err = getSubstitutionNodeInstancesIds(kv, deploymentID, nodeName)
			if err != nil {
				return names, err
			}
		}
	}

	sort.Sort(sortorder.Natural(names))
	log.Debugf("Found node instances %v for %s %s", names, deploymentID, nodeName)
	return names, nil
}

	if len(names) == 0 {
		// Check if this is a node to substitute
		substitutable, err := isSubstitutableNode(kv, deploymentID, nodeName)
		if err != nil {
			return names, err
		}
		if substitutable {
			log.Debugf("Found no instance for %s %s, getting substitutable node instance", deploymentID, nodeName)
			names, err = getSubstitutionNodeInstancesIds(kv, deploymentID, nodeName)
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
func GetHostedOnNode(kv *api.KV, deploymentID, nodeName string) (string, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	// So we have to traverse the hosted on relationships...
	// Lets inspect the requirements to found hosted on relationships
	reqKVPs, _, err := kv.Keys(path.Join(nodePath, "requirements")+"/", "/", nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	log.Debugf("Deployment: %q. Node %q. Requirements %v", deploymentID, nodeName, reqKVPs)
	for _, reqKey := range reqKVPs {
		log.Debugf("Deployment: %q. Node %q. Inspecting requirement %q", deploymentID, nodeName, reqKey)
		// Check requirement relationship
		kvp, _, err := kv.Get(path.Join(reqKey, "relationship"), nil)
		if err != nil {
			return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		// Is this "HostedOn" relationship ?
		if kvp.Value != nil {
			if ok, err := IsTypeDerivedFrom(kv, deploymentID, string(kvp.Value), "tosca.relationships.HostedOn"); err != nil {
				return "", err
			} else if ok {
				// An HostedOn! Great! let inspect the target node.
				kvp, _, err := kv.Get(path.Join(reqKey, "node"), nil)
				if err != nil {
					return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
				}
				if kvp == nil || len(kvp.Value) == 0 {
					return "", errors.Errorf("Missing 'node' attribute for requirement at index %q for node %q in deployment %q", path.Base(reqKey), nodeName, deploymentID)
				}
				return string(kvp.Value), nil
			}
		}
	}
	return "", nil
}

// GetHostedOnNodeInstance returns the node name and instance name of the instance
// defined in the first found relationship derived from "tosca.relationships.HostedOn"
//
// If there is no HostedOn relationship for this node then it returns an empty string
func GetHostedOnNodeInstance(kv *api.KV, deploymentID, nodeName, instanceName string) (string, string, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	// Going through requirements to find hosted on relationships
	reqKVPs, _, err := kv.Keys(path.Join(nodePath, "requirements")+"/", "/", nil)
	if err != nil {
		return "", "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	log.Debugf("Deployment: %q. Node %q. Requirements %v", deploymentID, nodeName, reqKVPs)
	for _, reqKey := range reqKVPs {
		log.Debugf("Deployment: %q. Node %q. Inspecting requirement %q", deploymentID, nodeName, reqKey)
		// Check requirement relationship
		kvp, _, err := kv.Get(path.Join(reqKey, "relationship"), nil)
		if err != nil {
			return "", "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		// Is this "HostedOn" relationship ?
		if kvp.Value != nil {
			if ok, err := IsTypeDerivedFrom(kv, deploymentID, string(kvp.Value), "tosca.relationships.HostedOn"); err != nil {
				return "", "", err
			} else if ok {
				// An HostedOn! Great! let inspect the target node.
				kvp, _, err := kv.Get(path.Join(reqKey, "node"), nil)
				if err != nil {
					return "", "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
				}
				if kvp == nil || len(kvp.Value) == 0 {
					return "", "", errors.Errorf("Missing 'node' attribute for requirement at index %q for node %q in deployment %q", path.Base(reqKey), nodeName, deploymentID)
				}
				// Get the corresponding target instances
				hostNodeName, hostInstances, err := GetTargetInstanceForRequirement(kv, deploymentID, nodeName, path.Base(reqKey), instanceName)
				if err != nil {
					return "", "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
				}
				if hostNodeName != "" && len(hostInstances) > 0 {
					return hostNodeName, hostInstances[0], nil
				}
			}
		}
	}
	return "", "", nil
}

// IsHostedOn checks if a given nodeName is hosted on another given node hostedOn by traversing the hostedOn hierarchy
func IsHostedOn(kv *api.KV, deploymentID, nodeName, hostedOn string) (bool, error) {
	if host, err := GetHostedOnNode(kv, deploymentID, nodeName); err != nil {
		return false, err
	} else if host == "" {
		return false, nil
	} else if host != hostedOn {
		return IsHostedOn(kv, deploymentID, host, hostedOn)
	}
	return true, nil

}

// GetNodesHostedOn returns the list of nodes that are hosted on a given node
func GetNodesHostedOn(kv *api.KV, deploymentID, hostNode string) ([]string, error) {
	// Thinking: maybe we can store at parsing time for each node the list of nodes on which it is hosted on and/or the opposite rather than re-scan the whole node list
	nodesList, err := GetNodes(kv, deploymentID)
	if err != nil {
		return nil, err
	}
	stackNodes := nodesList[:0]
	for _, node := range nodesList {
		var hostedOn bool
		hostedOn, err = IsHostedOn(kv, deploymentID, node, hostNode)
		if err != nil {
			return nil, err
		}

		if hostedOn {
			stackNodes = append(stackNodes, node)
		}
	}
	return stackNodes, nil
}

// GetNodeProperty retrieves the value for a given property in a given node
//
// It returns true if a value is found false otherwise as first return parameter.
// If the property is not found in the node then the type hierarchy is explored to find a default value.
// If the property is still not found then it will explore the HostedOn hierarchy.
func GetNodeProperty(kv *api.KV, deploymentID, nodeName, propertyName string, nestedKeys ...string) (bool, string, error) {
	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, "", err
	}
	var propDataType string
	hasProp, err := TypeHasProperty(kv, deploymentID, nodeType, propertyName, true)
	if err != nil {
		return false, "", err
	}
	if hasProp {
		propDataType, err = GetTypePropertyDataType(kv, deploymentID, nodeType, propertyName)
		if err != nil {
			return false, "", err
		}
	}
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)

	found, result, err := getValueAssignmentWithDataType(kv, deploymentID, path.Join(nodePath, "properties", propertyName), nodeName, "", "", propDataType, nestedKeys...)
	if err != nil {
		return false, "", errors.Wrapf(err, "Failed to get property %q for node %q", propertyName, nodeName)
	}
	if found {
		return true, result, nil
	}
	// Not found look at node type

	ok, value, isFunction, err := getTypeDefaultProperty(kv, deploymentID, nodeType, propertyName, nestedKeys...)
	if err != nil {
		return false, "", err
	}
	if ok {
		if !isFunction {
			return true, value, nil
		}
		value, err = resolveValueAssignmentAsString(kv, deploymentID, nodeName, "", "", value, nestedKeys...)
		return true, value, err
	}
	// No default found in type hierarchy
	// then traverse HostedOn relationships to find the value
	host, err := GetHostedOnNode(kv, deploymentID, nodeName)
	if err != nil {
		return false, "", err
	}
	if host != "" {
		found, result, err = GetNodeProperty(kv, deploymentID, host, propertyName, nestedKeys...)
		if err != nil || found {
			return found, result, err
		}
	}
	if hasProp {
		// Check if the whole property is optional
		isRequired, err := IsTypePropertyRequired(kv, deploymentID, nodeType, propertyName)
		if err != nil {
			return false, "", err
		}
		if !isRequired {
			// For backward compatibility
			// TODO this doesn't look as a good idea to me
			return true, "", nil
		}

		if len(nestedKeys) > 1 && propDataType != "" {
			// Check if nested type is optional
			nestedKeyType, err := GetNestedDataType(kv, deploymentID, propDataType, nestedKeys[:len(nestedKeys)-1]...)
			if err != nil {
				return false, "", err
			}
			isRequired, err = IsTypePropertyRequired(kv, deploymentID, nestedKeyType, nestedKeys[len(nestedKeys)-1])
			if err != nil {
				return false, "", err
			}
			if !isRequired {
				// For backward compatibility
				// TODO this doesn't look as a good idea to me
				return true, "", nil
			}
		}
	}
	// Not found anywhere
	return false, "", nil
}

// SetNodeProperty sets a node property
func SetNodeProperty(kv *api.KV, deploymentID, nodeName, propertyName, propertyValue string) error {
	kvp := &api.KVPair{
		Key:   path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName, "properties", propertyName),
		Value: []byte(propertyValue),
	}
	_, err := kv.Put(kvp, nil)
	return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
}

// GetNodeAttributes retrieves the values for a given attribute in a given node.
//
// As a node may have multiple instances and attributes may be instance-scoped, then returned result is a map with the instance name as key
// and the retrieved attributes as values.
//
// It returns true if a value is found false otherwise as first return parameter.
// If the property is not found in the node then the type hierarchy is explored to find a default value.
// If the property is still not found then it will explore the HostedOn hierarchy.
func GetNodeAttributes(kv *api.KV, deploymentID, nodeName, attributeName string, nestedKeys ...string) (bool, map[string]string, error) {
	instances, err := GetNodeInstancesIds(kv, deploymentID, nodeName)
	if err != nil {
		return false, nil, err
	}

	attributes := make(map[string]string)
	for _, instance := range instances {
		_, result, err := GetInstanceAttribute(kv, deploymentID, nodeName, instance, attributeName, nestedKeys...)
		if err != nil {
			return false, nil, err
		}
		attributes[instance] = result
	}
	return true, attributes, nil
}

// SetNodeInstanceAttribute sets an attribute value to a node instance
//
// Deprecated: Use SetInstanceAttribute
func SetNodeInstanceAttribute(kv *api.KV, deploymentID, nodeName, instanceName, attributeName, attributeValue string) error {
	return SetInstanceAttribute(deploymentID, nodeName, instanceName, attributeName, attributeValue)
}

// GetNodes returns the names of the different nodes for a given deployment.
func GetNodes(kv *api.KV, deploymentID string) ([]string, error) {
	names := make([]string, 0)
	nodesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes")
	nodes, _, err := kv.Keys(nodesPath+"/", "/", nil)
	if err != nil {
		return names, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, node := range nodes {
		names = append(names, path.Base(node))
	}
	return names, nil
}

// GetNodeType returns the type of a given node identified by its name
func GetNodeType(kv *api.KV, deploymentID, nodeName string) (string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "type"), nil)
	if err != nil {
		return "", errors.Wrapf(err, "Can't get type for node %q", nodeName)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", errors.Errorf("Missing mandatory parameter \"type\" for node %q", nodeName)
	}

	nodeType := string(kvp.Value)

	// If the corresponding node is substitutable, get its real node type
	substitutable, err := isSubstitutableNode(kv, deploymentID, nodeName)
	if err != nil {
		return "", err
	}
	if substitutable {
		nodeType, err = getSubstitutableNodeType(kv, deploymentID, nodeName, nodeType)
		if err != nil {
			return "", err
		}
	}
	return nodeType, nil
}

// GetNodeAttributesNames retrieves the list of existing attributes for a given node.
func GetNodeAttributesNames(kv *api.KV, deploymentID, nodeName string) ([]string, error) {
	attributesSet := make(map[string]struct{})

	// Look at node type
	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}

	typeAttrs, err := GetTypeAttributesNames(kv, deploymentID, nodeType)
	if err != nil {
		return nil, err
	}
	for _, attr := range typeAttrs {
		attributesSet[attr] = struct{}{}
	}

	// Look at instances attributes
	instances, err := GetNodeInstancesIds(kv, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	nodeInstancesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName)
	for _, instance := range instances {
		err = storeSubKeysInSet(kv, path.Join(nodeInstancesPath, instance, "attributes"), attributesSet)
		if err != nil {
			return nil, err
		}
	}

	// Look at not instance-scoped attribute
	err = storeSubKeysInSet(kv, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName, "attributes"), attributesSet)
	if err != nil {
		return nil, err
	}

	// Alien4Cloud did not yet implement the management of capability attributes
	// in substitution mappings. It expects for now to read these capability
	// attributes as node attributes
	err = storeSubstitutionMappingAttributeNamesInSet(kv, deploymentID, nodeName, attributesSet)
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
func GetTypeAttributesNames(kv *api.KV, deploymentID, typeName string) ([]string, error) {
	attributesSet := make(map[string]struct{})

	parentType, err := GetParentType(kv, deploymentID, typeName)
	if err != nil {
		return nil, err
	}
	if parentType != "" {
		var parentAttrs []string
		parentAttrs, err = GetTypeAttributesNames(kv, deploymentID, parentType)
		if err != nil {
			return nil, err
		}

		for _, attr := range parentAttrs {
			attributesSet[attr] = struct{}{}
		}
	}
	err = storeSubKeysInSet(kv, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "types", typeName, "attributes"), attributesSet)
	if err != nil {
		return nil, err
	}

	attributesList := make([]string, len(attributesSet))
	i := 0
	for attr := range attributesSet {
		attributesList[i] = attr
		i++
	}
	return attributesList, nil
}

// storeSubKeysInSet store Consul keys directly living under parentPath into the given set.
func storeSubKeysInSet(kv *api.KV, parentPath string, set map[string]struct{}) error {
	parentPath = strings.TrimSpace(parentPath)
	if !strings.HasSuffix(parentPath, "/") {
		parentPath += "/"
	}
	keys, _, err := kv.Keys(parentPath+"/", "/", nil)
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

func getInstancesDependentLinkedNodes(kv *api.KV, deploymentID, nodeName string) ([]string, error) {
	localStorageReqs, err := GetRequirementsKeysByTypeForNode(kv, deploymentID, nodeName, "local_storage")
	if err != nil {
		return nil, err
	}
	nodesList := make([]string, 0)
	for _, req := range localStorageReqs {
		var kvp *api.KVPair
		kvp, _, err = kv.Get(path.Join(req, "node"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp == nil || len(kvp.Value) == 0 {
			return nil, errors.Errorf("Missing attribute \"node\" for requirement %q on node %q", path.Join(path.Base(path.Clean(req+"/..")), path.Base(req)), nodeName)

		}
		nodesList = append(nodesList, string(kvp.Value))
	}
	networkReqs, err := GetRequirementsKeysByTypeForNode(kv, deploymentID, nodeName, "network")
	if err != nil {
		return nil, err
	}
	for _, req := range networkReqs {
		var kvp *api.KVPair

		kvp, _, err = kv.Get(path.Join(req, "capability"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp == nil || len(kvp.Value) == 0 || string(kvp.Value) != "yorc.capabilities.openstack.FIPConnectivity" {
			// Not a floating ip see next
			continue
		}

		kvp, _, err = kv.Get(path.Join(req, "node"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp == nil || len(kvp.Value) == 0 {
			return nil, errors.Errorf("Missing attribute \"node\" for requirement %q on node %q", path.Join(path.Base(path.Clean(req+"/..")), path.Base(req)), nodeName)

		}
		nodesList = append(nodesList, string(kvp.Value))
	}
	return nodesList, nil
}

// SelectNodeStackInstances selects a given number of instances of the given node, all the nodes hosted on this one and all nodes linked to it.
//
// For each node it returns a coma separated list of selected instances
func SelectNodeStackInstances(kv *api.KV, deploymentID, nodeName string, instancesDelta int) (map[string]string, error) {
	nodesStack, err := GetNodesHostedOn(kv, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	nodesStack = append(nodesStack, nodeName)
	linkedNodes, err := getInstancesDependentLinkedNodes(kv, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	nodesStack = append(nodesStack, linkedNodes...)

	// TODO: Improve the way we relate node instances names to dependent (linked nodes) or hosted on instances names
	instances, err := GetNodeInstancesIds(kv, deploymentID, nodeName)
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
func CreateNewNodeStackInstances(kv *api.KV, deploymentID, nodeName string, instances int) (map[string]string, error) {
	nodesMap := make(map[string]string)
	ctx := context.Background()
	_, errGroup, consulStore := consulutil.WithContext(ctx)

	nodes, err := GetNodes(kv, deploymentID)
	if err != nil {
		return nil, err
	}
	stackNodes := nodes[:0]
	for _, node := range nodes {
		if node == nodeName {
			stackNodes = append(stackNodes, node)
		} else {
			var hostedOnNode bool
			if hostedOnNode, err = IsHostedOn(kv, deploymentID, node, nodeName); err != nil {
				return nil, err
			} else if hostedOnNode {
				stackNodes = append(stackNodes, node)
			}
		}
	}

	linkedNodes, err := getInstancesDependentLinkedNodes(kv, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	stackNodes = append(stackNodes, linkedNodes...)

	// Now get existing nodes instances ids to have the
	existingIds, err := GetNodeInstancesIds(kv, deploymentID, nodeName)
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

			createNodeInstance(kv, consulStore, deploymentID, stackNode, id)
			if _, ok := nodesMap[stackNode]; ok {
				nodesMap[stackNode] = nodesMap[stackNode] + "," + id
			} else {
				nodesMap[stackNode] = id
			}
			addOrRemoveInstanceFromTargetRelationship(kv, deploymentID, stackNode, id, true)
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
		createRelationshipInstances(consulStore, kv, deploymentID, node)
	}
	return nodesMap, errors.Wrapf(errGroup.Wait(), "Failed to create instances for node %q", nodeName)

}

// createNodeInstance creates required elements for a new node
func createNodeInstance(kv *api.KV, consulStore consulutil.ConsulStore, deploymentID, nodeName, instanceName string) {

	instancePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName)

	consulStore.StoreConsulKeyAsString(path.Join(instancePath, instanceName, "attributes/state"), tosca.NodeStateInitial.String())
	consulStore.StoreConsulKeyAsString(path.Join(instancePath, instanceName, "attributes/tosca_name"), nodeName)
	consulStore.StoreConsulKeyAsString(path.Join(instancePath, instanceName, "attributes/tosca_id"), nodeName+"-"+instanceName)
	// Publish a status change event
	events.InstanceStatusChange(kv, deploymentID, nodeName, instanceName, tosca.NodeStateInitial.String())
}

// DoesNodeExist checks if a given node exist in a deployment
func DoesNodeExist(kv *api.KV, deploymentID, nodeName string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "name"), nil)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return kvp != nil && len(kvp.Value) > 0, nil
}
