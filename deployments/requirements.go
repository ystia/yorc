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
	"path"
	"sort"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"
	"vbom.ml/util/sortorder"
)

// GetRequirementKeyByNameForNode returns path to requirement which name match with defined requirementName for a given node name
func GetRequirementKeyByNameForNode(kv *api.KV, deploymentID, nodeName, requirementName string) (string, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	reqKVPs, _, err := kv.Keys(path.Join(nodePath, "requirements")+"/", "/", nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, reqIndexKey := range reqKVPs {
		reqIndexKey = path.Clean(reqIndexKey)
		kvp, _, err := kv.Get(path.Join(reqIndexKey, "name"), nil)
		if err != nil {
			return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp == nil || len(kvp.Value) == 0 {
			return "", errors.Errorf("Missing mandatory parameter \"name\" for requirement at index %q for node %q deployment %q", path.Base(reqIndexKey), nodeName, deploymentID)
		}
		if string(kvp.Value) == requirementName {
			return reqIndexKey, nil
		}
	}
	return "", nil
}

// GetRequirementsKeysByTypeForNode returns paths to requirements whose name or type_requirement match the given requirementType
//
// The returned slice may be empty if there is no matching requirements.
func GetRequirementsKeysByTypeForNode(kv *api.KV, deploymentID, nodeName, requirementType string) ([]string, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	reqKVPs, _, err := kv.Keys(path.Join(nodePath, "requirements")+"/", "/", nil)
	reqKeys := make([]string, 0)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, reqIndexKey := range reqKVPs {
		reqIndexKey = path.Clean(reqIndexKey)

		// Search matching with name
		kvp, _, err := kv.Get(path.Join(reqIndexKey, "name"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp == nil || len(kvp.Value) == 0 {
			return nil, errors.Errorf("Missing mandatory parameter \"name\" for requirement at index %q for node %q deployment %q", path.Base(reqIndexKey), nodeName, deploymentID)
		}
		if string(kvp.Value) == requirementType {
			reqKeys = append(reqKeys, reqIndexKey)
			// Pass to the next index
			continue
		}

		// Search matching with type_requirement
		kvp, _, err = kv.Get(path.Join(reqIndexKey, "type_requirement"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp != nil && string(kvp.Value) == requirementType {
			reqKeys = append(reqKeys, reqIndexKey)
		}
	}
	sort.Sort(sortorder.Natural(reqKeys))
	return reqKeys, nil
}

// GetRequirementIndexFromRequirementKey returns the corresponding requirement index from a given requirement key
// (typically returned by GetRequirementsKeysByTypeForNode)
func GetRequirementIndexFromRequirementKey(requirementKey string) string {
	return path.Base(requirementKey)
}

// GetRequirementIndexByNameForNode returns the requirement index which name match with defined requirementName for a given node name
func GetRequirementIndexByNameForNode(kv *api.KV, deploymentID, nodeName, requirementName string) (string, error) {
	reqPath, err := GetRequirementKeyByNameForNode(kv, deploymentID, nodeName, requirementName)
	if err != nil {
		return "", err
	}
	return path.Base(reqPath), nil
}

// GetRequirementsIndexes returns the list of requirements indexes for a given node
func GetRequirementsIndexes(kv *api.KV, deploymentID, nodeName string) ([]string, error) {
	reqPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName, "requirements")
	reqKVPs, _, err := kv.Keys(reqPath+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrapf(err, consulutil.ConsulGenericErrMsg)
	}
	for i := range reqKVPs {
		reqKVPs[i] = path.Base(reqKVPs[i])
	}
	return reqKVPs, nil
}

// GetNbRequirementsForNode returns the number of requirements declared for the given node
func GetNbRequirementsForNode(kv *api.KV, deploymentID, nodeName string) (int, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	reqKVPs, _, err := kv.Keys(path.Join(nodePath, "requirements")+"/", "/", nil)
	if err != nil {
		return 0, errors.Wrapf(err, "Failed to retrieve requirements for node %q", nodeName)
	}
	return len(reqKVPs), nil
}

// GetRelationshipForRequirement returns the relationship associated with a given requirementIndex for the given nodeName.
//
// If there is no relationship defined for this requirement then an empty string is returned.
func GetRelationshipForRequirement(kv *api.KV, deploymentID, nodeName, requirementIndex string) (string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements", requirementIndex, "relationship"), nil)
	// TODO: explicit naming of the relationship is optional and there is alternative way to retrieve it furthermore it can refer to a relationship_template_name instead of a relationship_type_name
	if err != nil || kvp == nil || len(kvp.Value) == 0 {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return string(kvp.Value), nil
}

// GetCapabilityForRequirement returns the capability associated with a given requirementIndex for the given nodeName.
//
// If there is no capability defined for this requirement then an empty string is returned.
func GetCapabilityForRequirement(kv *api.KV, deploymentID, nodeName, requirementIndex string) (string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements", requirementIndex, "capability"), nil)
	if err != nil || kvp == nil || len(kvp.Value) == 0 {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return string(kvp.Value), nil
}

// GetTargetNodeForRequirement returns the target node associated with a given requirementIndex for the given nodeName.
//
// If there is no node defined for this requirement then an empty string is returned.
func GetTargetNodeForRequirement(kv *api.KV, deploymentID, nodeName, requirementIndex string) (string, error) {
	kvp, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements", requirementIndex, "node"), nil)
	// TODO: explicit naming of the node is optional and there is alternative way to retrieve it furthermore it can refer to a node_template_name instead of a node_type_name
	if err != nil || kvp == nil || len(kvp.Value) == 0 {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	return string(kvp.Value), nil
}

// GetTargetInstanceForRequirement returns the target node and instances
// associated with a given requirementIndex of the given nodeName/instanceName.
//
func GetTargetInstanceForRequirement(kv *api.KV, deploymentID, nodeName, requirementIndex, instanceName string) (string, []string, error) {
	targetPrefix := path.Join(
		consulutil.DeploymentKVPrefix, deploymentID,
		"topology", "relationship_instances", nodeName, requirementIndex,
		instanceName, "target")
	kvp, _, err := kv.Get(path.Join(targetPrefix, "name"), nil)
	if err != nil {
		return "", nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", nil, nil
	}
	targetNodeName := string(kvp.Value)

	kvp, _, err = kv.Get(path.Join(targetPrefix, "instances"), nil)
	if err != nil {
		return "", nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", nil, nil
	}
	instanceIds := strings.Split(string(kvp.Value), ",")
	return targetNodeName, instanceIds, nil
}

// GetTargetNodeForRequirementByName returns the target node associated with a given requirementName for the given nodeName.
//
// If there is no node defined for this requirement then an empty string is returned.
func GetTargetNodeForRequirementByName(kv *api.KV, deploymentID, nodeName, requirementName string) (string, error) {
	reqPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements")
	kvp, _, err := kv.Keys(reqPath+"/", "/", nil)
	// TODO: explicit naming of the node is optional and there is alternative way to retrieve it furthermore it can refer to a node_template_name instead of a node_type_name
	if err != nil || kvp == nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	for _, req := range kvp {
		reqKv, _, err := kv.Get(req+"/type_requirement", nil)
		if err != nil || kv == nil {
			return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}

		if string(reqKv.Value) == requirementName {
			nodeNameKvp, _, err := kv.Get(req+"/node", nil)
			if err != nil || nodeNameKvp == nil {
				return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
			}
			return string(nodeNameKvp.Value), nil
		}

	}
	return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
}

// HasAnyRequirementCapability returns true and the the related node name addressing the capability
// if node with name nodeName has the requirement with the capability type equal or derived from the provided type
// otherwise it returns false and empty string
func HasAnyRequirementCapability(kv *api.KV, deploymentID, nodeName, requirement, capabilityType string) (bool, string, error) {
	reqkKeys, err := GetRequirementsKeysByTypeForNode(kv, deploymentID, nodeName, requirement)
	if err != nil {
		return false, "", err
	}
	for _, reqPrefix := range reqkKeys {
		requirementIndex := GetRequirementIndexFromRequirementKey(reqPrefix)
		capability, err := GetCapabilityForRequirement(kv, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return false, "", err
		}
		relatedNodeName, err := GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return false, "", err
		}

		if capability != "" {
			if capability == capabilityType {
				return true, relatedNodeName, nil
			}
			is, err := IsTypeDerivedFrom(kv, deploymentID, capability, capabilityType)
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
func HasAnyRequirementFromNodeType(kv *api.KV, deploymentID, nodeName, requirement, nodeType string) (bool, string, error) {
	reqkKeys, err := GetRequirementsKeysByTypeForNode(kv, deploymentID, nodeName, requirement)
	if err != nil {
		return false, "", err
	}
	for _, reqPrefix := range reqkKeys {
		requirementIndex := GetRequirementIndexFromRequirementKey(reqPrefix)
		capability, err := GetCapabilityForRequirement(kv, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return false, "", err
		}
		relatedNodeName, err := GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return false, "", err
		}

		if capability != "" {
			is, err := IsNodeDerivedFrom(kv, deploymentID, relatedNodeName, nodeType)
			if err != nil {
				return false, "", err
			} else if is {
				return is, relatedNodeName, nil
			}
		}
	}

	return false, "", nil
}
