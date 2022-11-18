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
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tosca"
)

// Requirement describes a requirement assignment with its name and index
type Requirement struct {
	tosca.RequirementAssignment
	Name  string
	Index string
}

func getRequirements(ctx context.Context, deploymentID, nodeName string) ([]tosca.RequirementAssignmentMap, error) {
	node, err := getNodeTemplate(ctx, deploymentID, nodeName)
	if err != nil {
		if IsNodeNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	return node.Requirements, nil
}

// GetRequirementsByTypeForNode returns requirements for a defined type for a defined node.
func GetRequirementsByTypeForNode(ctx context.Context, deploymentID, nodeName, requirementType string) ([]Requirement, error) {
	reqs := make([]Requirement, 0)
	reqMap, err := getRequirements(ctx, deploymentID, nodeName)
	if err != nil || reqMap == nil {
		return nil, err
	}
	for i, reqList := range reqMap {
		for name, reqAssignment := range reqList {
			// Search matching with name or type_requirement
			if name == requirementType || reqAssignment.TypeRequirement == requirementType {
				reqs = append(reqs, Requirement{reqAssignment, name, strconv.Itoa(i)})
			}
		}
	}
	return reqs, nil
}

// GetRequirementIndexByNameForNode returns the requirement index which name match with defined requirementName for a given node name
func GetRequirementIndexByNameForNode(ctx context.Context, deploymentID, nodeName, requirementName string) (string, error) {
	reqMap, err := getRequirements(ctx, deploymentID, nodeName)
	if err != nil || reqMap == nil {
		return "", err
	}
	for i, req := range reqMap {
		for name := range req {
			if requirementName == name {
				return strconv.Itoa(i), nil
			}
		}
	}
	return "", nil
}

// GetRequirementNameByIndexForNode returns the requirement name for a given node and requirement index
func GetRequirementNameByIndexForNode(ctx context.Context, deploymentID, nodeName, requirementIndex string) (string, error) {
	name, req, err := getRequirementByIndex(ctx, deploymentID, nodeName, requirementIndex)
	if err != nil || req == nil {
		return "", err
	}
	return name, nil
}

// GetRequirementsIndexes returns the list of requirements indexes for a given node
func GetRequirementsIndexes(ctx context.Context, deploymentID, nodeName string) ([]string, error) {
	indexes := make([]string, 0)
	reqMap, err := getRequirements(ctx, deploymentID, nodeName)
	if err != nil || reqMap == nil {
		return nil, err
	}
	for i := range reqMap {
		indexes = append(indexes, strconv.Itoa(i))
	}
	return indexes, nil
}

// GetNbRequirementsForNode returns the number of requirements declared for the given node
func GetNbRequirementsForNode(ctx context.Context, deploymentID, nodeName string) (int, error) {
	reqMap, err := getRequirements(ctx, deploymentID, nodeName)
	if err != nil || reqMap == nil {
		return 0, err
	}
	return len(reqMap), err
}

// GetRelationshipForRequirement returns the relationship associated with a given requirementIndex for the given nodeName.
//
// If there is no relationship defined for this requirement then an empty string is returned.
func GetRelationshipForRequirement(ctx context.Context, deploymentID, nodeName, requirementIndex string) (string, error) {
	name, req, err := getRequirementByIndex(ctx, deploymentID, nodeName, requirementIndex)
	if err != nil || req == nil || name == "" {
		return "", err
	}
	return req.Relationship, nil
}

// GetCapabilityForRequirement returns the capability associated with a given requirementIndex for the given nodeName.
//
// If there is no capability defined for this requirement then an empty string is returned.
func GetCapabilityForRequirement(ctx context.Context, deploymentID, nodeName, requirementIndex string) (string, error) {
	name, req, err := getRequirementByIndex(ctx, deploymentID, nodeName, requirementIndex)
	if err != nil || req == nil || name == "" {
		return "", err
	}
	return req.Capability, nil
}

// GetTargetNodeForRequirement returns the target node associated with a given requirementIndex for the given nodeName.
//
// If there is no node defined for this requirement then an empty string is returned.
func GetTargetNodeForRequirement(ctx context.Context, deploymentID, nodeName, requirementIndex string) (string, error) {
	name, req, err := getRequirementByIndex(ctx, deploymentID, nodeName, requirementIndex)
	if err != nil || req == nil || name == "" {
		return "", err
	}
	return req.Node, nil
}

// Return req name and requirement assignment
func getRequirementByIndex(ctx context.Context, deploymentID, nodeName, requirementIndex string) (string, *tosca.RequirementAssignment, error) {
	if requirementIndex == "" {
		return "", nil, nil
	}
	reqMap, err := getRequirements(ctx, deploymentID, nodeName)
	if err != nil || reqMap == nil {
		return "", nil, err
	}

	ind, err := strconv.Atoi(requirementIndex)
	if err != nil {
		return "", nil, errors.Wrapf(err, "requirement index %q is not a valid index", requirementIndex)
	}

	if ind+1 > len(reqMap) {
		return "", nil, errors.Wrapf(err, "requirement index %q is not a valid index as node with name %q has only %d requirements", requirementIndex, nodeName, len(reqMap))
	}

	// Only one requirement Assignment is expected
	if len(reqMap[ind]) > 1 {
		return "", nil, errors.Wrapf(err, "more than one requirement assignment for node:%q, index:%q", nodeName, requirementIndex)
	}
	for name, req := range reqMap[ind] {
		return name, &req, nil
	}
	return "", nil, nil
}

// GetTargetInstanceForRequirement returns the target node and instances
// associated with a given requirementIndex of the given nodeName/instanceName.
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
	reqMap, err := getRequirements(ctx, deploymentID, nodeName)
	if err != nil || reqMap == nil {
		return "", err
	}

	for _, reqList := range reqMap {
		for name, reqAssignment := range reqList {
			// Search matching with name or type_requirement
			if name == requirementName || reqAssignment.TypeRequirement == requirementName {
				return reqAssignment.Node, nil
			}
		}
	}
	return "", nil
}

// HasAnyRequirementCapability returns true and the related target node name addressing the capability
// if node with name nodeName has the requirement requirement with the capability type equal or derived from the provided type
// otherwise it returns false and empty string
func HasAnyRequirementCapability(ctx context.Context, deploymentID, nodeName, requirement, capabilityType string) (bool, string, error) {
	reqs, err := GetRequirementsByTypeForNode(ctx, deploymentID, nodeName, requirement)
	if err != nil || reqs == nil {
		return false, "", err
	}
	for _, req := range reqs {
		if req.Capability == capabilityType {
			return true, req.Node, nil
		}
		is, err := IsTypeDerivedFrom(ctx, deploymentID, req.Capability, capabilityType)
		if err != nil {
			return false, "", err
		}
		if is {
			return true, req.Node, nil
		}
	}
	return false, "", nil
}

// HasAnyRequirementFromNodeType returns true and the related node name addressing the capability
// if node with name nodeName has the requirement with the node type equal or derived from the provided type
// otherwise it returns false and empty string
func HasAnyRequirementFromNodeType(ctx context.Context, deploymentID, nodeName, requirement, nodeType string) (bool, string, error) {
	reqs, err := GetRequirementsByTypeForNode(ctx, deploymentID, nodeName, requirement)
	if err != nil || reqs == nil {
		return false, "", err
	}

	for _, req := range reqs {
		if req.Capability != "" {
			is, err := IsNodeDerivedFrom(ctx, deploymentID, req.Node, nodeType)
			if err != nil {
				return false, "", err
			} else if is {
				return is, req.Node, nil
			}
		}
	}
	return false, "", nil
}

func GetRequirementDefinitionOnTypeByName(ctx context.Context, deploymentID, nodeType, requirementName string) (*tosca.RequirementDefinition, error) {
	log.Debugf("-> searching for requirement %q on node type %q\n", requirementName, nodeType)
	node := new(tosca.NodeType)
	err := getExpectedTypeFromName(ctx, deploymentID, nodeType, node)
	if err != nil {
		return nil, err
	}
	for _, reqMap := range node.Requirements {
		req, ok := reqMap[requirementName]
		if ok {
			log.Debugf("-> found requirement %q on node type %q\n", requirementName, nodeType)
			return &req, nil
		}
	}

	if node.DerivedFrom == "" {
		log.Debugf("-> requirement %q not found on node type %q and no parent type\n", requirementName, nodeType)
		return nil, nil
	}
	return GetRequirementDefinitionOnTypeByName(ctx, deploymentID, node.DerivedFrom, requirementName)
}
