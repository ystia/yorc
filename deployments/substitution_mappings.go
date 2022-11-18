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
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tosca"
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
func isSubstitutableNode(ctx context.Context, deploymentID, nodeName string) (bool, error) {

	node, err := getNodeTemplate(ctx, deploymentID, nodeName)
	if err != nil {
		return false, errors.Wrapf(err, "Can't get directives for node %q", nodeName)
	}

	substitutable := false
	if node.Directives != nil {
		for _, value := range node.Directives {
			if value == directiveSubstitutable {
				substitutable = true
				break
			}
		}
	}
	return substitutable, nil
}

func getDeploymentSubstitutionMapping(ctx context.Context, deploymentID string) (*tosca.SubstitutionMapping, error) {
	return getSubstitutionMappingFromStore(ctx, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology"))
}

func getSubstitutionMappingFromStore(ctx context.Context, prefix string) (*tosca.SubstitutionMapping, error) {
	substitutionPrefix := path.Join(prefix, "substitution_mappings")

	substitutionMapping := new(tosca.SubstitutionMapping)
	exist, err := storage.GetStore(types.StoreTypeDeployment).Get(substitutionPrefix, substitutionMapping)
	if err != nil {
		return nil,
			errors.Wrapf(err, "Can't get node type for substitution at %q", substitutionPrefix)
	}
	if !exist {
		// No mapping defined
		return substitutionMapping, nil
	}
	return substitutionMapping, err
}

// storeSubstitutionMappingAttributeNamesInSet gets capability attributes for capabilities
// exposed in the deployment through substitution mappings, and stores these
// capability attributes as node attributes prefixed by capabilities.<capability name>
// as expected by Alien4Cloud until it implements the mapping of capability attributes
// See http://alien4cloud.github.io/#/documentation/2.0.0/user_guide/services_management.html
func storeSubstitutionMappingAttributeNamesInSet(ctx context.Context, deploymentID, nodeName string, set map[string]struct{}) error {

	substMapping, err := getDeploymentSubstitutionMapping(ctx, deploymentID)
	if err != nil {
		return err
	}

	capabilityToAttrNames := make(map[string][]string)
	exploreParents := true
	// Get the capabilities exposed for this node
	for _, capMapping := range substMapping.Capabilities {
		if len(capMapping.Mapping) > 1 {
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
					attributeNames, err = GetNodeCapabilityAttributeNames(ctx, deploymentID, nodeName, capability, exploreParents)
					if err != nil {
						return err
					}
				}

				capabilityToAttrNames[capability] = attributeNames
			}
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
// this function returns the the value of the corresponding instance capability
// attribute
func getSubstitutionMappingAttribute(ctx context.Context, deploymentID, nodeName, instanceName, attributeName string, nestedKeys ...string) (*TOSCAValue, error) {

	if !isSubstitutionMappingAttribute(attributeName) {
		return nil, nil
	}

	log.Debugf("Attempting to substitute attribute %s in %s %s %s", attributeName, deploymentID, nodeName, instanceName)

	items := strings.Split(attributeName, ".")
	capabilityName := items[1]
	capAttrName := items[2]

	// Check this capability attribute is really exposed before returning its
	// value
	attributesSet := make(map[string]struct{})
	err := storeSubstitutionMappingAttributeNamesInSet(ctx, deploymentID, nodeName, attributesSet)
	if err != nil {
		return nil, err
	}

	if _, ok := attributesSet[attributeName]; ok {
		// This attribute is exposed, returning its value
		log.Debugf("Substituting attribute %s by its instance capability attribute in %s %s %s", attributeName, deploymentID, nodeName, instanceName)
		return GetInstanceCapabilityAttributeValue(ctx, deploymentID, nodeName, instanceName, capabilityName, capAttrName, nestedKeys...)
	}

	return nil, nil
}

// getSubstitutionNodeInstancesIds returns for a substitutable node, a fake
// instance ID, all necessary infos are stored and retrieved at the node level.
func getSubstitutionNodeInstancesIds(ctx context.Context, deploymentID, nodeName string) ([]string, error) {

	var names []string
	substitutable, err := isSubstitutableNode(ctx, deploymentID, nodeName)
	if err == nil && substitutable {
		names = []string{substitutableNodeInstance}
	}
	return names, err
}

func isSubstitutionNodeInstance(ctx context.Context, deploymentID, nodeName, nodeInstance string) (bool, error) {
	if nodeInstance != substitutableNodeInstance {
		return false, nil
	}

	return isSubstitutableNode(ctx, deploymentID, nodeName)
}

// getSubstitutableNodeType returns the node type of a substitutable node.
// There are 2 cases :
//   - either this node a reference to an external service, and the node type
//     here is a real node type (abstract)
//   - or this is a reference to a managed service, and the node type here
//     does not exist, it is the name of a template whose import contains the
//     real node type
func getSubstitutableNodeType(ctx context.Context, deploymentID, nodeName, nodeType string) (string, error) {
	_, err := GetParentType(ctx, deploymentID, nodeType)
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
	imports, err := storage.GetStore(types.StoreTypeDeployment).Keys(importsPath)
	if err != nil {
		return "", err
	}

	var importTemplatePath string
	for _, importPath := range imports {
		metadataPtr := new(map[string]string)
		exist, err := storage.GetStore(types.StoreTypeDeployment).Get(path.Join(importPath, "metadata"), metadataPtr)
		metadata := *metadataPtr
		if err == nil && exist && metadata["template_name"] == nodeType {
			// Found the import
			importTemplatePath = importPath
			break
		}
	}

	if importTemplatePath == "" {
		// No such template found
		return nodeType, nil
	}

	// Check substitution mappings in this import
	mappings, err := getSubstitutionMappingFromStore(ctx, importTemplatePath)
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
func getSubstitutionInstanceCapabilityAttribute(ctx context.Context, deploymentID, nodeName,
	instanceName, capabilityName, attributeType, attributeName string, nestedKeys ...string) (*TOSCAValue, error) {

	nodeAttrName := fmt.Sprintf(capabilityFormat, capabilityName, attributeName)
	return getNodeAttributeValue(ctx, deploymentID, nodeName, instanceName, nodeAttrName, attributeType, nestedKeys...)
}
