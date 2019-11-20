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
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
	"path"
)

func getPolicyTypeStruct(ctx context.Context, deploymentID, policyTypeName string) (*tosca.PolicyType, error) {
	policyTyp := new(tosca.PolicyType)
	err := getTypeStruct(deploymentID, policyTypeName, policyTyp)
	if err != nil {
		return nil, err
	}
	return policyTyp, nil
}

func getPolicyStruct(ctx context.Context, deploymentID, policyName string) (*tosca.Policy, error) {
	key := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "policies", policyName)
	policy := new(tosca.Policy)
	exist, err := storage.GetStore(types.StoreTypeDeployment).Get(key, policy)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.Errorf("No policy with name %q found", policyName)
	}
	return policy, nil
}

// GetPoliciesForType retrieves all policies with or derived from policyTypeName
func GetPoliciesForType(ctx context.Context, deploymentID, policyTypeName string) ([]string, error) {
	p := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "policies")
	keys, err := consulutil.GetKeys(p)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	policies := make([]string, 0)
	for _, key := range keys {
		policyName := path.Base(key)
		policy, err := getPolicyStruct(ctx, deploymentID, policyName)
		if err != nil {
			return nil, err
		}
		if policy.Type == "" {
			return nil, errors.Errorf("Missing mandatory attribute \"type\" for policy %q", path.Base(key))
		}
		// Check policy type
		isType, err := IsTypeDerivedFrom(ctx, deploymentID, policy.Type, policyTypeName)
		if err != nil {
			return nil, err
		}
		// Check policy targets
		if isType {
			policies = append(policies, policyName)
		}
	}
	return policies, nil
}

// GetPoliciesForTypeAndNode retrieves all policies with or derived from policyTypeName and with nodeName as target
func GetPoliciesForTypeAndNode(ctx context.Context, deploymentID, policyTypeName, nodeName string) ([]string, error) {
	policiesForType, err := GetPoliciesForType(ctx, deploymentID, policyTypeName)
	if err != nil {
		return nil, err
	}
	policies := make([]string, 0)
	for _, policy := range policiesForType {
		is, err := IsTargetForPolicy(ctx, deploymentID, policy, nodeName, false)
		if err != nil {
			return nil, err
		}
		if is {
			policies = append(policies, policy)
		}
	}
	return policies, nil
}

// GetPolicyPropertyValue retrieves the value for a given property in a given policy
//
// It returns true if a value is found false otherwise as first return parameter.
// If the property is not found in the policy then the type hierarchy is explored to find a default value.
func GetPolicyPropertyValue(ctx context.Context, deploymentID, policyName, propertyName string, nestedKeys ...string) (*TOSCAValue, error) {
	policy, err := getPolicyStruct(ctx, deploymentID, policyName)
	if err != nil {
		return nil, err
	}

	// Check if the node template property is set and doesn't need to be resolved
	va, is := policy.Properties[propertyName]
	if is && va != nil && va.Type != tosca.ValueAssignmentFunction {
		return readComplexVA(ctx, va, nestedKeys...), nil
	}

	// Retrieve related propertyDefinition with default property
	propDef, err := getTypePropertyDefinition(ctx, deploymentID, policy.Type, "policy", propertyName)
	if err != nil {
		return nil, err
	}
	if propDef != nil {
		return getValueAssignment(ctx, deploymentID, policyName, "", "", va, propDef.Default, nestedKeys...)
	}

	// Not found anywhere
	return nil, nil
}

// GetPolicyType returns the type of the policy
func GetPolicyType(ctx context.Context, deploymentID, policyName string) (string, error) {
	policy, err := getPolicyStruct(ctx, deploymentID, policyName)
	if err != nil {
		return "", err
	}
	if policy.Type == "" {
		return "", errors.Errorf("Missing mandatory attribute \"type\" for policy %q", policyName)
	}
	return policy.Type, nil
}

// IsTargetForPolicy returns true if the node name is a policy target
func IsTargetForPolicy(ctx context.Context, deploymentID, policyName, nodeName string, recursive bool) (bool, error) {
	targets, err := GetPolicyTargets(ctx, deploymentID, policyName)
	if err != nil {
		return false, err
	}
	if !recursive {
		return collections.ContainsString(targets, nodeName), nil
	}

	nodeType, err := GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return false, err
	}

	policyType, err := GetPolicyType(ctx, deploymentID, policyName)
	if err != nil {
		return false, err
	}
	targets, err = GetPolicyTargetsForType(ctx, deploymentID, policyType)
	if err != nil {
		return false, err
	}
	for _, target := range targets {
		is, err := IsTypeDerivedFrom(ctx, deploymentID, nodeType, target)
		if err != nil {
			return false, err
		}
		if is {
			return true, nil
		}
	}
	return false, nil
}

// GetPolicyTargets retrieves the policy template targets
// these targets are node names
func GetPolicyTargets(ctx context.Context, deploymentID, policyName string) ([]string, error) {
	policy, err := getPolicyStruct(ctx, deploymentID, policyName)
	if err != nil {
		return nil, err
	}
	return policy.Targets, nil
}

// GetPolicyTargetsForType retrieves the policy type targets
// this targets are node types
func GetPolicyTargetsForType(ctx context.Context, deploymentID, policyTypeName string) ([]string, error) {
	policyType, err := getPolicyTypeStruct(ctx, deploymentID, policyTypeName)
	if err != nil {
		return nil, err
	}
	if policyType.Targets != nil {
		return policyType.Targets, nil
	}

	parentType, err := GetParentType(ctx, deploymentID, policyTypeName)
	if err != nil {
		return nil, err
	}
	if parentType == "" {
		return nil, nil
	}
	return GetPolicyTargetsForType(ctx, deploymentID, parentType)
}
