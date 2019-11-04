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
	"strings"

	"github.com/pkg/errors"
	"vbom.ml/util/sortorder"

	"github.com/ystia/yorc/v4/helper/consulutil"
)

// GetRequirementKeyByNameForNode returns path to requirement which name match with defined requirementName for a given node name
func GetRequirementKeyByNameForNode(ctx context.Context, deploymentID, nodeName, requirementName string) (string, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	reqKVPs, err := consulutil.GetKeys(path.Join(nodePath, "requirements"))
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, reqIndexKey := range reqKVPs {
		reqIndexKey = path.Clean(reqIndexKey)
		exist, value, err := consulutil.GetStringValue(path.Join(reqIndexKey, "name"))
		if err != nil {
			return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if !exist || value == "" {
			return "", errors.Errorf("Missing mandatory parameter \"name\" for requirement at index %q for node %q deployment %q", path.Base(reqIndexKey), nodeName, deploymentID)
		}
		if value == requirementName {
			return reqIndexKey, nil
		}
	}
	return "", nil
}

// GetRequirementsKeysByTypeForNode returns paths to requirements whose name or type_requirement match the given requirementType
//
// The returned slice may be empty if there is no matching requirements.
func GetRequirementsKeysByTypeForNode(ctx context.Context, deploymentID, nodeName, requirementType string) ([]string, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	reqKVPs, err := consulutil.GetKeys(path.Join(nodePath, "requirements"))
	reqKeys := make([]string, 0)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, reqIndexKey := range reqKVPs {
		reqIndexKey = path.Clean(reqIndexKey)

		// Search matching with name
		exist, value, err := consulutil.GetStringValue(path.Join(reqIndexKey, "name"))
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if !exist || value == "" {
			return nil, errors.Errorf("Missing mandatory parameter \"name\" for requirement at index %q for node %q deployment %q", path.Base(reqIndexKey), nodeName, deploymentID)
		}
		if value == requirementType {
			reqKeys = append(reqKeys, reqIndexKey)
			// Pass to the next index
			continue
		}

		// Search matching with type_requirement
		exist, value, err = consulutil.GetStringValue(path.Join(reqIndexKey, "type_requirement"))
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if exist && value == requirementType {
			reqKeys = append(reqKeys, reqIndexKey)
		}
	}
	sort.Sort(sortorder.Natural(reqKeys))
	return reqKeys, nil
}

// GetRequirementIndexFromRequirementKey returns the corresponding requirement index from a given requirement key
// (typically returned by GetRequirementsKeysByTypeForNode)
func GetRequirementIndexFromRequirementKey(ctx context.Context, requirementKey string) string {
	return path.Base(requirementKey)
}

// GetRequirementIndexByNameForNode returns the requirement index which name match with defined requirementName for a given node name
func GetRequirementIndexByNameForNode(ctx context.Context, deploymentID, nodeName, requirementName string) (string, error) {
	reqPath, err := GetRequirementKeyByNameForNode(ctx, deploymentID, nodeName, requirementName)
	if err != nil {
		return "", err
	}
	return path.Base(reqPath), nil
}

// GetRequirementNameByIndexForNode returns the requirement name for a given node and requirement index
func GetRequirementNameByIndexForNode(ctx context.Context, deploymentID, nodeName, requirementIndex string) (string, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements", requirementIndex, "name"))
	if err != nil || !exist || value == "" {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return value, nil
}

// GetRequirementsIndexes returns the list of requirements indexes for a given node
func GetRequirementsIndexes(ctx context.Context, deploymentID, nodeName string) ([]string, error) {
	reqPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName, "requirements")
	reqKVPs, err := consulutil.GetKeys(reqPath)
	if err != nil {
		return nil, errors.Wrapf(err, consulutil.ConsulGenericErrMsg)
	}
	for i := range reqKVPs {
		reqKVPs[i] = path.Base(reqKVPs[i])
	}
	return reqKVPs, nil
}

// GetNbRequirementsForNode returns the number of requirements declared for the given node
func GetNbRequirementsForNode(ctx context.Context, deploymentID, nodeName string) (int, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	reqKVPs, err := consulutil.GetKeys(path.Join(nodePath, "requirements"))
	if err != nil {
		return 0, errors.Wrapf(err, "Failed to retrieve requirements for node %q", nodeName)
	}
	return len(reqKVPs), nil
}

// GetRelationshipForRequirement returns the relationship associated with a given requirementIndex for the given nodeName.
//
// If there is no relationship defined for this requirement then an empty string is returned.
func GetRelationshipForRequirement(ctx context.Context, deploymentID, nodeName, requirementIndex string) (string, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements", requirementIndex, "relationship"))
	// TODO: explicit naming of the relationship is optional and there is alternative way to retrieve it furthermore it can refer to a relationship_template_name instead of a relationship_type_name
	if err != nil || !exist || value == "" {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return value, nil
}

// GetCapabilityForRequirement returns the capability associated with a given requirementIndex for the given nodeName.
//
// If there is no capability defined for this requirement then an empty string is returned.
func GetCapabilityForRequirement(ctx context.Context, deploymentID, nodeName, requirementIndex string) (string, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements", requirementIndex, "capability"))
	if err != nil || !exist || value == "" {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return value, nil
}

// GetTargetNodeForRequirement returns the target node associated with a given requirementIndex for the given nodeName.
//
// If there is no node defined for this requirement then an empty string is returned.
func GetTargetNodeForRequirement(ctx context.Context, deploymentID, nodeName, requirementIndex string) (string, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements", requirementIndex, "node"))
	// TODO: explicit naming of the node is optional and there is alternative way to retrieve it furthermore it can refer to a node_template_name instead of a node_type_name
	if err != nil || !exist || value == "" {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return value, nil
}

// GetTargetInstanceForRequirement returns the target node and instances
// associated with a given requirementIndex of the given nodeName/instanceName.
//
func GetTargetInstanceForRequirement(ctx context.Context, deploymentID, nodeName, requirementIndex, instanceName string) (string, []string, error) {
	targetPrefix := path.Join(
		consulutil.DeploymentKVPrefix, deploymentID,
		"topology", "relationship_instances", nodeName, requirementIndex,
		instanceName, "target")
	exist, targetNodeName, err := consulutil.GetStringValue(path.Join(targetPrefix, "name"))
	if err != nil {
		return "", nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist || targetNodeName == "" {
		return "", nil, nil
	}

	exist, value, err := consulutil.GetStringValue(path.Join(targetPrefix, "instances"))
	if err != nil {
		return "", nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist || value == "" {
		return "", nil, nil
	}
	instanceIds := strings.Split(value, ",")
	return targetNodeName, instanceIds, nil
}

// GetTargetNodeForRequirementByName returns the target node associated with a given requirementName for the given nodeName.
//
// If there is no node defined for this requirement then an empty string is returned.
func GetTargetNodeForRequirementByName(ctx context.Context, deploymentID, nodeName, requirementName string) (string, error) {
	reqPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements")
	kvp, err := consulutil.GetKeys(reqPath)
	// TODO: explicit naming of the node is optional and there is alternative way to retrieve it furthermore it can refer to a node_template_name instead of a node_type_name
	if err != nil || kvp == nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	for _, req := range kvp {
		existR, value, err := consulutil.GetStringValue(req + "/type_requirement")
		if err != nil || !existR {
			return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}

		if value == requirementName {
			existN, node, err := consulutil.GetStringValue(req + "/node")
			if err != nil || !existN {
				return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
			}
			return node, nil
		}

	}
	return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
}

// HasAnyRequirementCapability returns true and the the related node name addressing the capability
// if node with name nodeName has the requirement with the capability type equal or derived from the provided type
// otherwise it returns false and empty string
func HasAnyRequirementCapability(ctx context.Context, deploymentID, nodeName, requirement, capabilityType string) (bool, string, error) {
	reqkKeys, err := GetRequirementsKeysByTypeForNode(ctx, deploymentID, nodeName, requirement)
	if err != nil {
		return false, "", err
	}
	for _, reqPrefix := range reqkKeys {
		requirementIndex := GetRequirementIndexFromRequirementKey(ctx, reqPrefix)
		capability, err := GetCapabilityForRequirement(ctx, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return false, "", err
		}
		relatedNodeName, err := GetTargetNodeForRequirement(ctx, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return false, "", err
		}

		if capability != "" {
			if capability == capabilityType {
				return true, relatedNodeName, nil
			}
			is, err := IsTypeDerivedFrom(ctx, deploymentID, capability, capabilityType)
			if err != nil {
				return false, "", err
			}
			return is, relatedNodeName, nil
		}
	}

	return false, "", nil
}

// HasAnyRequirementFromNodeType returns true and the the related node name addressing the capability
// if node with name nodeName has the requirement with the node type equal or derived from the provided type
// otherwise it returns false and empty string
func HasAnyRequirementFromNodeType(ctx context.Context, deploymentID, nodeName, requirement, nodeType string) (bool, string, error) {
	reqkKeys, err := GetRequirementsKeysByTypeForNode(ctx, deploymentID, nodeName, requirement)
	if err != nil {
		return false, "", err
	}
	for _, reqPrefix := range reqkKeys {
		requirementIndex := GetRequirementIndexFromRequirementKey(ctx, reqPrefix)
		capability, err := GetCapabilityForRequirement(ctx, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return false, "", err
		}
		relatedNodeName, err := GetTargetNodeForRequirement(ctx, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return false, "", err
		}

		if capability != "" {
			is, err := IsNodeDerivedFrom(ctx, deploymentID, relatedNodeName, nodeType)
			if err != nil {
				return false, "", err
			} else if is {
				return is, relatedNodeName, nil
			}
		}
	}

	return false, "", nil
}
