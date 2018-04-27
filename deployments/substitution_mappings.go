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

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tosca"
)

const (
	// Alien4Cloud is managing capability attributes of Managed serviced
	// referenced in a deployment as node templates attributes with
	// the format capabilities.<capability name>.<attribute name>
	// See http://alien4cloud.github.io/#/documentation/2.0.0/user_guide/services_management.html
	capabilityFormat = "capabilities.%s.%s"

	// directiveSubstitutable is a directive to the Orchestrator that a node
	// type is substitutable, ie. this node type is either abstract or a
	// reference to another topology template providing substitution mappings
	directiveSubstitutable = "substitutable"

	// Name of a fake instance for a substitutable node
	// Using an integer value as this is expected by the Alien4cloud Yorc
	// Orchestrator plugin, and it is has to be 0 so that the plugin can manage
	// a state change event started and display the instance as started.
	substitutableNodeInstance = "0"
)

// isSubstitutableNode returns true if a node contains an Orchestrator directive
// that it is substitutable
func isSubstitutableNode(kv *api.KV, deploymentID, nodeName string) (bool, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "directives"), nil)
	if err != nil {
		return false, errors.Wrapf(err, "Can't get directives for node %q", nodeName)
	}

	substitutable := false
	if kvp != nil && kvp.Value != nil {
		values := strings.Split(string(kvp.Value), ",")
		for _, value := range values {
			if value == directiveSubstitutable {
				substitutable = true
				break
			}
		}
	}
	return substitutable, nil
}

func storeSubstitutionMappings(ctx context.Context, topology tosca.Topology, topologyPrefix string) {
	consulStore := ctx.Value(consulStoreKey).(consulutil.ConsulStore)
	substitutionPrefix := path.Join(topologyPrefix, "substitution_mappings")
	substitution := topology.TopologyTemplate.SubstitionMappings
	if substitution != nil {
		consulStore.StoreConsulKeyAsString(path.Join(substitutionPrefix, "node_type"),
			substitution.NodeType)
		storePropAttrMappings(consulStore, path.Join(substitutionPrefix, "properties"),
			substitution.Properties)
		storePropAttrMappings(consulStore, path.Join(substitutionPrefix, "attributes"),
			substitution.Attributes)
		storeCapReqMappings(consulStore, path.Join(substitutionPrefix, "capabilities"),
			substitution.Capabilities)
		storeCapReqMappings(consulStore, path.Join(substitutionPrefix, "requirements"),
			substitution.Requirements)
		storeInterfaceMappings(consulStore, path.Join(substitutionPrefix, "interfaces"),
			substitution.Interfaces)
	}
}

func storePropAttrMappings(
	consulStore consulutil.ConsulStore,
	prefix string,
	mappings map[string]tosca.PropAttrMapping) {

	if mappings != nil {

		for name, propAttrMapping := range mappings {

			propAttrPrefix := path.Join(prefix, name)
			if propAttrMapping.Mapping != nil {
				consulStore.StoreConsulKeyAsString(
					path.Join(propAttrPrefix, "mapping"),
					strings.Join(propAttrMapping.Mapping, ","))
			} else {
				storeValueAssignment(consulStore, path.Join(propAttrPrefix, "value"), propAttrMapping.Value)
			}
		}
	}
}

func storeCapReqMappings(
	consulStore consulutil.ConsulStore,
	prefix string,
	mappings map[string]tosca.CapReqMapping) {

	if mappings != nil {

		for name, capReqMapping := range mappings {

			capReqPrefix := path.Join(prefix, name)

			if capReqMapping.Mapping != nil {
				consulStore.StoreConsulKeyAsString(
					path.Join(capReqPrefix, "mapping"),
					strings.Join(capReqMapping.Mapping, ","))
			}

			if capReqMapping.Properties != nil {
				propPrefix := path.Join(capReqPrefix, "properties")
				for name, value := range capReqMapping.Properties {
					storeValueAssignment(
						consulStore, path.Join(propPrefix, name),
						value)
				}
			}

			if capReqMapping.Attributes != nil {
				attrPrefix := path.Join(capReqPrefix, "attributes")
				for name, value := range capReqMapping.Attributes {
					storeValueAssignment(
						consulStore, path.Join(attrPrefix, name),
						value)
				}
			}
		}
	}
}

func storeInterfaceMappings(
	consulStore consulutil.ConsulStore,
	prefix string,
	mappings map[string]string) {

	if mappings != nil {
		for operationName, workflowName := range mappings {
			consulStore.StoreConsulKeyAsString(path.Join(prefix, operationName), workflowName)
		}
	}
}

func getDeploymentSubstitutionMapping(kv *api.KV, deploymentID string) (tosca.SubstitutionMapping, error) {
	return getSubstitutionMappingFromStore(kv,
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology"))
}

func getSubstitutionMappingFromStore(kv *api.KV, prefix string) (tosca.SubstitutionMapping, error) {
	substitutionPrefix := path.Join(prefix, "substitution_mappings")
	var substitutionMapping tosca.SubstitutionMapping

	kvp, _, err := kv.Get(path.Join(substitutionPrefix, "node_type"), nil)
	if err != nil {
		return substitutionMapping,
			errors.Wrapf(err, "Can't get node type for substitution at %q", substitutionPrefix)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		// No mapping defined
		return substitutionMapping, nil
	}
	substitutionMapping.NodeType = string(kvp.Value)
	substitutionMapping.Capabilities, err = getCapReqMappingFromStore(
		kv, path.Join(substitutionPrefix, "capabilities"))
	if err != nil {
		return substitutionMapping, err
	}

	substitutionMapping.Requirements, err = getCapReqMappingFromStore(
		kv, path.Join(substitutionPrefix, "requirements"))
	if err != nil {
		return substitutionMapping, err
	}

	substitutionMapping.Properties, err = getPropAttrMappingFromStore(
		kv, path.Join(substitutionPrefix, "properties"))
	if err != nil {
		return substitutionMapping, err
	}

	substitutionMapping.Attributes, err = getPropAttrMappingFromStore(
		kv, path.Join(substitutionPrefix, "attributes"))
	if err != nil {
		return substitutionMapping, err
	}

	substitutionMapping.Interfaces, err = getInterfaceMappingFromStore(
		kv, path.Join(substitutionPrefix, "interfaces"))

	return substitutionMapping, err
}

func getCapReqMappingFromStore(kv *api.KV, prefix string) (map[string]tosca.CapReqMapping, error) {

	capabilityPaths, _, err := kv.Keys(prefix+"/", "/", nil)
	if err != nil {
		return nil, err
	}

	capabilities := make(map[string]tosca.CapReqMapping)

	for _, capPath := range capabilityPaths {

		capName := path.Base(capPath)

		kvp, _, err := kv.Get(path.Join(capPath, "mapping"), nil)
		if err != nil {
			return capabilities,
				errors.Wrapf(err, "Can't get mapping for capability at %q", capPath)
		}
		var capMapping tosca.CapReqMapping
		if kvp != nil && len(kvp.Value) != 0 {
			capMapping.Mapping = strings.Split(string(kvp.Value), ",")
			// When a mapping is defined, there is no property/attribute
			// definition, so the capability definition is complete at this point
			capabilities[capName] = capMapping
			continue
		}

		capMapping.Properties, err = getPropertiesOrAttributesFromStore(kv,
			path.Join(capPath, "properties"))
		if err != nil {
			return capabilities, err
		}

		capMapping.Attributes, err = getPropertiesOrAttributesFromStore(kv,
			path.Join(capPath, "attributes"))
		if err != nil {
			return capabilities, err
		}

		capabilities[capName] = capMapping
	}

	return capabilities, nil
}

func getPropertiesOrAttributesFromStore(kv *api.KV, prefix string) (map[string]*tosca.ValueAssignment, error) {
	propPaths, _, err := kv.Keys(prefix+"/", "/", nil)
	if err != nil {
		return nil, err
	}

	props := make(map[string]*tosca.ValueAssignment)

	for _, propPath := range propPaths {

		propName := path.Base(propPath)

		val, err := getValueAssignmentFromStore(kv, propPath)
		if err != nil {
			return props,
				errors.Wrapf(err, "Can't get value for property or attribute at %q", propPath)
		}
		if val != nil {
			props[propName] = val
		}
	}

	return props, nil
}

func getPropAttrMappingFromStore(kv *api.KV, prefix string) (map[string]tosca.PropAttrMapping, error) {

	propPaths, _, err := kv.Keys(prefix+"/", "/", nil)
	if err != nil {
		return nil, err
	}

	props := make(map[string]tosca.PropAttrMapping)

	for _, propPath := range propPaths {

		propName := path.Base(propPath)

		var propMapping tosca.PropAttrMapping
		kvp, _, err := kv.Get(path.Join(propPath, "mapping"), nil)
		if err != nil {
			return props,
				errors.Wrapf(err, "Can't get mapping for properties or attributes at %q", propPath)
		}
		if kvp != nil && len(kvp.Value) != 0 {
			propMapping.Mapping = strings.Split(string(kvp.Value), ",")
		} else {
			propMapping.Value, err = getValueAssignmentFromStore(
				kv, path.Join(propPath, "value"))
			if err != nil {
				return props,
					errors.Wrapf(err, "Can't get mapping for properties or attributes at %q", propPath)
			}
		}
		props[propName] = propMapping
	}

	return props, nil
}

func getValueAssignmentFromStore(kv *api.KV, valPath string) (*tosca.ValueAssignment, error) {
	kvp, _, err := kv.Get(valPath, nil)
	if err != nil {
		return nil, err
	}
	var val tosca.ValueAssignment
	if kvp != nil {
		val.Type = tosca.ValueAssignmentType(kvp.Flags)
		if err != nil {
			return nil, err
		}

		switch val.Type {
		case tosca.ValueAssignmentLiteral, tosca.ValueAssignmentFunction:
			val.Value = kvp.Value
		case tosca.ValueAssignmentList, tosca.ValueAssignmentMap:
			val.Value, err = readComplexVA(kv, val.Type, "", valPath, "")
			if err != nil {
				return nil, err
			}
		}
	}

	return &val, nil
}

func getInterfaceMappingFromStore(kv *api.KV, prefix string) (map[string]string, error) {

	opPaths, _, err := kv.Keys(prefix+"/", "/", nil)
	if err != nil {
		return nil, err
	}

	itfs := make(map[string]string)

	for _, opPath := range opPaths {

		opName := path.Base(opPath)

		kvp, _, err := kv.Get(path.Join(opPath), nil)
		if err != nil {
			return itfs,
				errors.Wrapf(err, "Can't get mapping for interface at %q", opPath)
		}
		if kvp != nil && len(kvp.Value) != 0 {
			itfs[opName] = string(kvp.Value)
		}
	}

	return itfs, nil
}

// storeSubstitutionMappingAttributeNamesInSet gets capability attributes for capabilities
// exposed in the deployment through substitution mappings, and stores these
// capability attributes as node attributes prefixed by capabilities.<capability name>
// as expected by Alien4Cloud until it implements the mapping of capability attributes
// See http://alien4cloud.github.io/#/documentation/2.0.0/user_guide/services_management.html
func storeSubstitutionMappingAttributeNamesInSet(kv *api.KV, deploymentID, nodeName string, set map[string]struct{}) error {

	substMapping, err := getDeploymentSubstitutionMapping(kv, deploymentID)
	if err != nil {
		return err
	}

	capabilityToAttrNames := make(map[string][]string)
	exploreParents := true
	// Get the capabilities exposed for this node
	for _, capMapping := range substMapping.Capabilities {

		capability := capMapping.Mapping[1]
		var attributeNames []string

		if capMapping.Mapping[0] == nodeName {
			if capMapping.Attributes != nil {
				attributeNames := make([]string, len(capMapping.Attributes))
				i := 0
				for name := range capMapping.Attributes {
					attributeNames[i] = name
					i++
				}
			} else {
				// Expose all attributes
				attributeNames, err = GetNodeCapabilityAttributeNames(
					kv, deploymentID, nodeName, capability, exploreParents)
				if err != nil {
					return err
				}
			}

			capabilityToAttrNames[capability] = attributeNames
		}
	}

	// See http://alien4cloud.github.io/#/documentation/2.0.0/user_guide/services_management.html
	// Alien4Cloud is managing capability attributes  as node template attributes with
	// the format capabilities.<capability name>.<attribute name>
	for capability, names := range capabilityToAttrNames {
		for _, attr := range names {
			set[fmt.Sprintf(capabilityFormat, capability, attr)] = struct{}{}
		}
	}

	return nil
}

func isSubstitutionMappingAttribute(attributeName string) bool {
	items := strings.Split(attributeName, ".")
	return len(items) == 3 && items[0] == "capabilities"
}

// getSubstitutionMappingAttribute retrieves the given attribute for a capability
//
// It returns true if a value is found false otherwise as first return parameter.
// If the attribute is a substitution mapping capability attribute as provided
// by Alien4Cloud, using the format capabilities.<capability name>.<attributr name>,
// this function returns the the vqlue of the corresponding instance capability
// attribute
func getSubstitutionMappingAttribute(kv *api.KV, deploymentID, nodeName, instanceName, attributeName string, nestedKeys ...string) (bool, string, error) {

	if !isSubstitutionMappingAttribute(attributeName) {
		return false, "", nil
	}

	log.Debugf("Attempting to substitute attribute %s in %s %s %s", attributeName, deploymentID, nodeName, instanceName)

	items := strings.Split(attributeName, ".")
	capabilityName := items[1]
	capAttrName := items[2]

	// Check this capability attribute is really exposed before returning its
	// value
	attributesSet := make(map[string]struct{})
	err := storeSubstitutionMappingAttributeNamesInSet(kv, deploymentID, nodeName, attributesSet)
	if err != nil {
		return false, "", err
	}

	if _, ok := attributesSet[attributeName]; ok {
		// This attribute is exposed, returning its value
		log.Debugf("Substituting attribute %s by its instance capability attribute in %s %s %s %s", attributeName, deploymentID, nodeName, instanceName)
		return GetInstanceCapabilityAttribute(
			kv, deploymentID, nodeName, instanceName, capabilityName, capAttrName, nestedKeys...)
	}

	return false, "", nil
}

// getSubstitutionNodeInstancesIds returns for a substitutable node, a fake
// instance ID, all necessary infos are stored and retrieved at the node level.
func getSubstitutionNodeInstancesIds(kv *api.KV, deploymentID, nodeName string) ([]string, error) {

	var names []string
	substitutable, err := isSubstitutableNode(kv, deploymentID, nodeName)
	if err == nil && substitutable {
		names = []string{substitutableNodeInstance}
	}
	return names, err
}

func isSubstitutionNodeInstance(kv *api.KV, deploymentID, nodeName, nodeInstance string) (bool, error) {
	if nodeInstance != substitutableNodeInstance {
		return false, nil
	}

	return isSubstitutableNode(kv, deploymentID, nodeName)
}

// getSubstitutableNodeType returns the node type of a substitutable node.
// There are 2 cases :
// - either this node a reference to an external service, and the node type
//   here is a real node type (abstract)
// - or this is a reference to a managed service, and the node type here
//   does not exist, it is the name of a template whose import contains the
//    real node type
func getSubstitutableNodeType(kv *api.KV, deploymentID, nodeName, nodeType string) (string, error) {
	_, err := GetParentType(kv, deploymentID, nodeType)
	if err == nil {
		// Reference to an external service, this node type is a real abstract
		// node type
		return nodeType, nil
	} else if !IsTypeMissingError(err) {
		// Unexpected error
		return nodeType, err
	}

	// The node type does not exist. This is the  reference to an application
	// from another deployment.
	// The real node type has to be found in subsitution mappings of an imported
	// file whose metadata template name is the nodeType here.
	importsPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/imports")
	imports, _, err := kv.Keys(importsPath+"/", "/", nil)
	if err != nil {
		return "", errors.Wrap(err, "Consul communication error")
	}

	var importTemplatePath string
	for _, importPath := range imports {
		kvp, _, err := kv.Get(path.Join(importPath, "metadata/template_name"), nil)
		if err == nil && kvp != nil && len(kvp.Value) != 0 && string(kvp.Value) == nodeType {
			// Found the import
			importTemplatePath = importPath
			break
		}
	}

	if importTemplatePath == "" {
		// No such template found
		return nodeType, nil
	}

	// Check subsitution mappings in this import
	mappings, err := getSubstitutionMappingFromStore(kv, importTemplatePath)
	if err == nil && mappings.NodeType != "" {
		log.Debugf("Substituting type %s by type %s for %s %s", nodeType, mappings.NodeType, deploymentID, nodeName)
		return mappings.NodeType, err
	}
	// Found no substitution type
	return nodeType, nil

}

// getSubstitutionInstanceAttribute returns the value of generic attributes
// associated to any instance, here a fake instance for a substitutable node
func getSubstitutionInstanceAttribute(deploymentID, nodeName, instanceName, attributeName string) (bool, string) {
	switch attributeName {
	case "tosca_name":
		return true, nodeName
	case "tosca_id":
		return true, nodeName + "-" + instanceName
	default:
		return false, ""
	}
}

// getSubstitutionInstanceCapabilityAttribute returns the value of a capability
// attribute for a Service Instance not deployed in this deployment.
// In this case, the capability attribute is provided as an attribute in the Node
// template, the attribute having this format:
// capabilities.<capability name>.<attribute name>
// See http://alien4cloud.github.io/#/documentation/2.0.0/user_guide/services_management.html
func getSubstitutionInstanceCapabilityAttribute(kv *api.KV, deploymentID, nodeName,
	instanceName, capabilityName, attributeType, attributeName string, nestedKeys ...string) (bool, string, error) {

	nodeAttrName := fmt.Sprintf(capabilityFormat, capabilityName, attributeName)
	found, result, err := getValueAssignmentWithDataType(kv, deploymentID,
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes",
			nodeName, "attributes", nodeAttrName),
		nodeName, instanceName, "", attributeType, nestedKeys...)
	if err != nil {
		return false, "", errors.Wrapf(err, "Failed to get attribute %q for node %q", nodeAttrName, nodeName)
	}

	return found, result, nil
}
