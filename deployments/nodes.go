package deployments

import (
	"fmt"
	"path"
	"strconv"

	"strings"

	"context"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

// IsNodeDerivedFrom check if the node's type is derived from another type.
//
// Basically this function is a shorthand for GetNodeType and IsNodeTypeDerivedFrom.
func IsNodeDerivedFrom(kv *api.KV, deploymentID, nodeName, derives string) (bool, error) {
	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, err
	}
	return IsNodeTypeDerivedFrom(kv, deploymentID, nodeType, derives)
}

// GetDefaultNbInstancesForNode retrieves the number of instances for a given node nodeName in deployment deploymentId.
//
// If the node is or is derived from 'tosca.nodes.Compute' it will look for a property 'default_instances' in the 'scalable' capability of
// this node. Otherwise it will search for any relationship derived from 'tosca.relationships.HostedOn' in node requirements and reiterate
// the process. If a Compute is finally found it returns 'true' and the instances number.
// If there is no 'tosca.nodes.Compute' at the end of the hosted on chain then assume that there is only one instance and return 'false'
func GetDefaultNbInstancesForNode(kv *api.KV, deploymentId, nodeName string) (bool, uint32, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology", "nodes", nodeName)
	kvp, _, err := kv.Get(nodePath+"/type", nil)
	if err != nil {
		return false, 0, err
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return false, 0, fmt.Errorf("Missing type for node %q, in deployment %q", nodeName, deploymentId)
	}
	nodeType := string(kvp.Value)
	// It would be a better solution to check if the type or its parent have a scalable capability
	if ok, err := IsNodeTypeDerivedFrom(kv, deploymentId, nodeType, "tosca.nodes.Compute"); err != nil {
		return false, 0, err
	} else if ok {
		//For now we look into default instances in scalable capability but it will be dynamic at runtime we will have to store the
		//current number of instances somewhere else
		kvp, _, err = kv.Get(nodePath+"/capabilities/scalable/properties/default_instances", nil)
		if err != nil {
			return false, 0, err
		}
		if kvp == nil || len(kvp.Value) == 0 {
			log.Debugf("Missing property 'default_instances' of 'scalable' capability for node %q derived from 'tosca.nodes.Compute', in deployment %q. Lets assume that it is 1.", nodeName, deploymentId)
			return true, 1, nil
		}
		if val, err := strconv.ParseUint(string(kvp.Value), 10, 32); err != nil {
			return false, 0, fmt.Errorf("Not a valid integer for property 'default_instances' of 'scalable' capability for node %q derived from 'tosca.nodes.Compute', in deployment %q. Error: %v", nodeName, deploymentId, err)
		} else {
			return true, uint32(val), nil
		}
	}
	// So we have to traverse the hosted on relationships...
	// Lets inspect the requirements to found hosted on relationships
	hostNode, err := GetHostedOnNode(kv, deploymentId, nodeName)
	if err != nil {
		return false, 0, err
	} else if hostNode != "" {
		return GetDefaultNbInstancesForNode(kv, deploymentId, hostNode)
	}
	// Not hosted on a tosca.nodes.Compute assume one instance
	return false, 1, nil
}

// GetMaxNbInstancesForNode retrieves the maximum number of instances for a given node nodeName in deployment deploymentId.
//
// If the node is or is derived from 'tosca.nodes.Compute' it will look for a property 'max_instances' in the 'scalable' capability of
// this node. Otherwise it will search for any relationship derived from 'tosca.relationships.HostedOn' in node requirements and reiterate
// the process. If a Compute is finally found it returns 'true' and the instances number.
// If there is no 'tosca.nodes.Compute' at the end of the hosted on chain then assume that there is only one instance and return 'false'
func GetMaxNbInstancesForNode(kv *api.KV, deploymentId, nodeName string) (bool, uint32, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology", "nodes", nodeName)
	kvp, _, err := kv.Get(nodePath+"/type", nil)
	if err != nil {
		return false, 0, err
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return false, 0, fmt.Errorf("Missing type for node %q, in deployment %q", nodeName, deploymentId)
	}
	nodeType := string(kvp.Value)
	if ok, err := IsNodeTypeDerivedFrom(kv, deploymentId, nodeType, "tosca.nodes.Compute"); err != nil {
		return false, 0, err
	} else if ok {
		//For now we look into default instances in scalable capability but it will be dynamic at runtime we will have to store the
		//current number of instances somewhere else
		kvp, _, err = kv.Get(nodePath+"/capabilities/scalable/properties/max_instances", nil)
		if err != nil {
			return false, 0, err
		}
		if kvp == nil || len(kvp.Value) == 0 {
			log.Debugf("Missing property 'max_instances'")
			return true, 1, nil
		}
		if val, err := strconv.ParseUint(string(kvp.Value), 10, 32); err != nil {
			return false, 0, fmt.Errorf("Not a valid integer for property 'max_instances' of 'scalable' capability for node %q derived from 'tosca.nodes.Compute', in deployment %q. Error: %v", nodeName, deploymentId, err)
		} else {
			return true, uint32(val), nil
		}
	}
	// So we have to traverse the hosted on relationships...
	// Lets inspect the requirements to found hosted on relationships
	hostNode, err := GetHostedOnNode(kv, deploymentId, nodeName)
	if err != nil {
		return false, 0, err
	} else if hostNode != "" {
		return GetMaxNbInstancesForNode(kv, deploymentId, hostNode)
	}
	// Not hosted on a tosca.nodes.Compute assume one instance
	return false, 1, nil
}

// GetMinNbInstancesForNode retrieves the minimum number of instances for a given node nodeName in deployment deploymentId.
//
// If the node is or is derived from 'tosca.nodes.Compute' it will look for a property 'min_instances' in the 'scalable' capability of
// this node. Otherwise it will search for any relationship derived from 'tosca.relationships.HostedOn' in node requirements and reiterate
// the process. If a Compute is finally found it returns 'true' and the instances number.
// If there is no 'tosca.nodes.Compute' at the end of the hosted on chain then assume that there is only one instance and return 'false'
func GetMinNbInstancesForNode(kv *api.KV, deploymentId, nodeName string) (bool, uint32, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology", "nodes", nodeName)
	kvp, _, err := kv.Get(nodePath+"/type", nil)
	if err != nil {
		return false, 0, err
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return false, 0, fmt.Errorf("Missing type for node %q, in deployment %q", nodeName, deploymentId)
	}
	nodeType := string(kvp.Value)
	if ok, err := IsNodeTypeDerivedFrom(kv, deploymentId, nodeType, "tosca.nodes.Compute"); err != nil {
		return false, 0, err
	} else if ok {
		//For now we look into default instances in scalable capability but it will be dynamic at runtime we will have to store the
		//current number of instances somewhere else
		kvp, _, err = kv.Get(nodePath+"/capabilities/scalable/properties/min_instances", nil)
		if err != nil {
			return false, 0, err
		}
		if kvp == nil || len(kvp.Value) == 0 {
			log.Debugf("Missing property 'min_instances'")
			return true, 1, nil
		}
		if val, err := strconv.ParseUint(string(kvp.Value), 10, 32); err != nil {
			return false, 0, fmt.Errorf("Not a valid integer for property 'min_instances' of 'scalable' capability for node %q derived from 'tosca.nodes.Compute', in deployment %q. Error: %v", nodeName, deploymentId, err)
		} else {
			return true, uint32(val), nil
		}
	}
	// So we have to traverse the hosted on relationships...
	// Lets inspect the requirements to found hosted on relationships
	hostNode, err := GetHostedOnNode(kv, deploymentId, nodeName)
	if err != nil {
		return false, 0, err
	} else if hostNode != "" {
		return GetMinNbInstancesForNode(kv, deploymentId, hostNode)
	}
	// Not hosted on a tosca.nodes.Compute assume one instance
	return false, 1, nil
}

// GetNbInstancesForNode retrieves the number of instances for a given node nodeName in deployment deploymentId.
func GetNbInstancesForNode(kv *api.KV, deploymentId, nodeName string) (bool, uint32, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology", "nodes", nodeName)
	kvp, _, err := kv.Get(nodePath+"/type", nil)
	if err != nil {
		return false, 0, err
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return false, 0, fmt.Errorf("Missing type for node %q, in deployment %q", nodeName, deploymentId)
	}

	//For now we look into default instances in scalable capability but it will be dynamic at runtime we will have to store the
	//current number of instances somewhere else
	kvp, _, err = kv.Get(nodePath+"/nbInstances", nil)
	if err != nil {
		return false, 0, err
	}
	if kvp == nil || len(kvp.Value) == 0 {
		log.Debugf("Missing property 'nbInstances'")
		return true, 1, nil
	}
	if val, err := strconv.ParseUint(string(kvp.Value), 10, 32); err != nil {
		return false, 0, fmt.Errorf("Not a valid integer for property 'default_instances' of 'scalable' capability for node %q derived from 'tosca.nodes.Compute', in deployment %q. Error: %v", nodeName, deploymentId, err)
	} else {
		return true, uint32(val), nil
	}

	// Not hosted on a tosca.nodes.Compute assume one instance
	return false, 1, nil
}

// SetNbInstancesForNode just update the number of instances for the given nodeName, in the deployment (deploymentId)
func SetNbInstancesForNode(kv *api.KV, deploymentId, nodeName string, newNbOfInstances uint32) error {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology", "nodes", nodeName)
	strInt := fmt.Sprint(newNbOfInstances)

	err := consulutil.StoreConsulKeyAsString(nodePath+"/nbInstances", strInt)
	if err != nil {
		return err
	}

	return nil
}

// GetNodeInstancesIds returns the names of the different instances for a given node.
//
// It may be an empty array if the given node is not HostedOn a scalable node.
func GetNodeInstancesIds(kv *api.KV, deploymentId, nodeName string) ([]string, error) {
	names := make([]string, 0)
	instancesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/instances", nodeName)
	instances, _, err := kv.Keys(instancesPath+"/", "/", nil)
	if err != nil {
		return names, errors.Wrap(err, "Consul communication error")
	}
	for _, instance := range instances {
		names = append(names, path.Base(instance))
	}
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
		return "", errors.Wrap(err, "Consul communication error")
	}
	log.Debugf("Deployment: %q. Node %q. Requirements %v", deploymentID, nodeName, reqKVPs)
	for _, reqKey := range reqKVPs {
		log.Debugf("Deployment: %q. Node %q. Inspecting requirement %q", deploymentID, nodeName, reqKey)
		// Check requirement relationship
		kvp, _, err := kv.Get(path.Join(reqKey, "relationship"), nil)
		if err != nil {
			return "", errors.Wrap(err, "Consul communication error")
		}
		// Is this relationship an HostedOn?
		if kvp.Value != nil {
			if ok, err := IsNodeTypeDerivedFrom(kv, deploymentID, string(kvp.Value), "tosca.relationships.HostedOn"); err != nil {
				return "", err
			} else if ok {
				// An HostedOn! Great! let inspect the target node.
				kvp, _, err := kv.Get(path.Join(reqKey, "node"), nil)
				if err != nil {
					return "", errors.Wrap(err, "Consul communication error")
				}
				if kvp == nil || len(kvp.Value) == 0 {
					return "", errors.Errorf("Missing 'node' attribute for requirement at index %q for node %q in deployement %q", path.Base(reqKey), nodeName, deploymentID)
				}
				return string(kvp.Value), nil
			}
		}
	}
	return "", nil
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

// GetTypeDefaultProperty checks if a type has a default value for a given property.
//
// It returns true if a default value is found false otherwise as first return parameter.
// If no default value is found in a given type then the derived_from hierarchy is explored to find the default value.
func GetTypeDefaultProperty(kv *api.KV, deploymentId, typeName, propertyName string) (bool, string, error) {
	return getTypeDefaultAttributeOrProperty(kv, deploymentId, typeName, propertyName, true)
}

// GetTypeDefaultAttribute checks if a type has a default value for a given attribute.
//
// It returns true if a default value is found false otherwise as first return parameter.
// If no default value is found in a given type then the derived_from hierarchy is explored to find the default value.
func GetTypeDefaultAttribute(kv *api.KV, deploymentId, typeName, attributeName string) (bool, string, error) {
	return getTypeDefaultAttributeOrProperty(kv, deploymentId, typeName, attributeName, false)
}

// GetNodeProperty retrieves the value for a given property in a given node
//
// It returns true if a value is found false otherwise as first return parameter.
// If the property is not found in the node then the type hierarchy is explored to find a default value.
// If the property is still not found then it will explore the HostedOn hierarchy.
func GetNodeProperty(kv *api.KV, deploymentId, nodeName, propertyName string) (bool, string, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology", "nodes", nodeName)
	kvp, _, err := kv.Get(path.Join(nodePath, "properties", propertyName), nil)
	if err != nil {
		return false, "", err
	}
	if kvp != nil {
		return true, string(kvp.Value), nil
	}

	// Not found look at node type
	kvp, _, err = kv.Get(path.Join(nodePath, "type"), nil)
	if err != nil {
		return false, "", err
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return false, "", fmt.Errorf("Missing type for node %q in deployment %q", nodeName, deploymentId)
	}

	ok, value, err := GetTypeDefaultProperty(kv, deploymentId, string(kvp.Value), propertyName)
	if err != nil {
		return false, "", nil
	}
	if ok {
		return true, value, nil
	}
	// No default found in type hierarchy
	// then traverse HostedOn relationships to find the value
	host, err := GetHostedOnNode(kv, deploymentId, nodeName)
	if err != nil {
		return false, "", err
	}
	if host != "" {
		return GetNodeProperty(kv, deploymentId, host, propertyName)
	}
	// Not found anywhere
	return false, "", nil
}

// GetNodeAttributes retrieves the values for a given attribute in a given node.
//
// As a node may have multiple instances and attributes may be instance-scoped, then returned result is a map with the instance name as key
// and the retrieved attributes as values.
//
// It returns true if a value is found false otherwise as first return parameter.
// If the property is not found in the node then the type hierarchy is explored to find a default value.
// If the property is still not found then it will explore the HostedOn hierarchy.
func GetNodeAttributes(kv *api.KV, deploymentId, nodeName, attributeName string) (found bool, attributes map[string]string, err error) {
	found = false
	instances, err := GetNodeInstancesIds(kv, deploymentId, nodeName)
	if err != nil {
		return
	}

	if len(instances) > 0 {
		attributes = make(map[string]string)
		nodeInstancesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology", "instances", nodeName)
		for _, instance := range instances {
			var kvp *api.KVPair
			kvp, _, err = kv.Get(path.Join(nodeInstancesPath, instance, "attributes", attributeName), nil)
			if err != nil {
				return
			}
			if kvp != nil {
				attributes[instance] = string(kvp.Value)
			}
		}
		if len(attributes) > 0 {
			found = true
			return
		}
	}

	// Look at not instance-scoped attribute
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology", "nodes", nodeName)

	kvp, _, err := kv.Get(path.Join(nodePath, "attributes", attributeName), nil)
	if err != nil {
		return
	}
	if kvp != nil {
		if attributes == nil {
			attributes = make(map[string]string)
		}
		if len(instances) > 0 {
			for _, instance := range instances {
				attributes[instance] = string(kvp.Value)
			}
		} else {
			attributes[""] = string(kvp.Value)
		}
		found = true
		return
	}

	// Not found look at node type
	kvp, _, err = kv.Get(path.Join(nodePath, "type"), nil)
	if err != nil {
		return
	}
	if kvp == nil || len(kvp.Value) == 0 {
		err = fmt.Errorf("Missing type for node %q in deployment %q", nodeName, deploymentId)
		return
	}

	ok, defaultValue, err := GetTypeDefaultAttribute(kv, deploymentId, string(kvp.Value), attributeName)
	if err != nil {
		return
	}
	if ok {
		if attributes == nil {
			attributes = make(map[string]string)
		}
		if len(instances) > 0 {
			for _, instance := range instances {
				attributes[instance] = string(defaultValue)
			}
		} else {
			attributes[""] = string(defaultValue)
		}
		found = true
		return
	}
	// No default found in type hierarchy
	// then traverse HostedOn relationships to find the value
	var host string
	host, err = GetHostedOnNode(kv, deploymentId, nodeName)
	if err != nil {
		return
	}
	if host != "" {
		found, attributes, err = GetNodeAttributes(kv, deploymentId, host, attributeName)
		if found || err != nil {
			return
		}
	}

	// Now check properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
	var prop string
	found, prop, err = GetNodeProperty(kv, deploymentId, nodeName, attributeName)
	if !found || err != nil {
		return
	}
	if attributes == nil {
		attributes = make(map[string]string)
	}
	if len(instances) > 0 {
		for _, instance := range instances {
			attributes[instance] = prop
		}
	} else {
		attributes[""] = prop
	}
	return
}

// getTypeDefaultProperty checks if a type has a default value for a given property or attribute.
// It returns true if a default value is found false otherwise as first return parameter.
// If no default value is found in a given type then the derived_from hierarchy is explored to find the default value.
func getTypeDefaultAttributeOrProperty(kv *api.KV, deploymentId, typeName, propertyName string, isProperty bool) (bool, string, error) {
	typePath := path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology", "types", typeName)
	var defaultPath string
	if isProperty {
		defaultPath = path.Join(typePath, "properties", propertyName, "default")
	} else {
		defaultPath = path.Join(typePath, "attributes", propertyName, "default")
	}
	kvp, _, err := kv.Get(defaultPath, nil)
	if err != nil {
		return false, "", err
	}
	if kvp != nil {
		return true, string(kvp.Value), nil
	}
	// No default in this type
	// Lets look at parent type
	kvp, _, err = kv.Get(typePath+"/derived_from", nil)
	if err != nil {
		return false, "", err
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return false, "", nil
	}
	return getTypeDefaultAttributeOrProperty(kv, deploymentId, string(kvp.Value), propertyName, isProperty)
}

// GetNodes returns the names of the different nodes for a given deployment.
func GetNodes(kv *api.KV, deploymentID string) ([]string, error) {
	names := make([]string, 0)
	nodesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes")
	nodes, _, err := kv.Keys(nodesPath+"/", "/", nil)
	if err != nil {
		return names, errors.Wrap(err, "Consul communication error")
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
	return string(kvp.Value), nil
}

// HasScalableCapability check if the given nodeName in the specified deployment, has in this capabilities a Key named scalable
func HasScalableCapability(kv *api.KV, deploymentID, nodeName string) (bool, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	capabilitiesKeys, _, err := kv.Keys(nodePath+"/capabilities/", "/", nil)
	if err != nil {
		return false, err
	}

	for _, val := range capabilitiesKeys {
		if path.Base(val) == "scalable" {
			return true, nil
		}
	}

	return false, nil
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
		storeSubKeysInSet(kv, path.Join(nodeInstancesPath, instance, "attributes"), attributesSet)
		if err != nil {
			return nil, err
		}
	}

	// Look at not instance-scoped attribute
	err = storeSubKeysInSet(kv, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName, "attributes"), attributesSet)
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

// GetTypeAttributesNames returns the list of attributes names found in the type hierarchy
func GetTypeAttributesNames(kv *api.KV, deploymentID, typeName string) ([]string, error) {
	attributesSet := make(map[string]struct{})

	parentType, err := GetParentType(kv, deploymentID, typeName)
	if err != nil {
		return nil, err
	}
	if parentType != "" {
		parentAttrs, err := GetTypeAttributesNames(kv, deploymentID, parentType)
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
		return errors.Wrap(err, "Consul access error: ")
	}
	for _, key := range keys {
		attr := path.Base(key)
		if _, found := set[attr]; !found {
			set[attr] = struct{}{}
		}
	}
	return nil
}

// CreateNewNodeStackInstances create the given number of new instances of the given node and all other nodes hosted on this one
//
// CreateNewNodeStackInstances returns the list of newly created instances IDs
func CreateNewNodeStackInstances(kv *api.KV, deploymentID, nodeName string, instances int) ([]string, error) {
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

	// Now get existing nodes instances ids to have the
	existingIds, err := GetNodeInstancesIds(kv, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	instancesIDs := make([]string, 0)
	initialID := len(existingIds)
	for i := initialID; i < initialID+instances; i++ {
		id := strconv.FormatUint(uint64(i), 10)
		instancesIDs = append(instancesIDs, id)
		for _, stackNode := range stackNodes {

			createNodeInstance(consulStore, deploymentID, stackNode, id)
		}
	}

	return instancesIDs, errors.Wrapf(errGroup.Wait(), "Failed to create instances for node %q", nodeName)

}

// createNodeInstance creates required elements for a new node
func createNodeInstance(consulStore consulutil.ConsulStore, deploymentID, nodeName, instanceName string) {

	instancePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName)

	consulStore.StoreConsulKeyAsString(path.Join(instancePath, instanceName, "attributes/state"), INITIAL.String())
	consulStore.StoreConsulKeyAsString(path.Join(instancePath, instanceName, "attributes/tosca_name"), nodeName)
	consulStore.StoreConsulKeyAsString(path.Join(instancePath, instanceName, "attributes/tosca_id"), nodeName+"-"+instanceName)
}
