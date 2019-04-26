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
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v3/helper/collections"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"path"
	"strings"
)

// GetPoliciesForType retrieves all policies with or derived from policyTypeName
func GetPoliciesForType(kv *api.KV, deploymentID, policyTypeName string) ([]string, error) {
	p := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "policies")
	keys, _, err := kv.Keys(p+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	policies := make([]string, 0)
	for _, key := range keys {
		policyName := path.Base(key)
		kvp, _, err := kv.Get(path.Join(key, "type"), nil)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvp == nil || len(kvp.Value) == 0 {
			return nil, errors.Errorf("Missing mandatory attribute \"type\" for policy %q", path.Base(key))
		}
		policyType := string(kvp.Value)
		// Check policy type
		isType, err := IsTypeDerivedFrom(kv, deploymentID, policyType, policyTypeName)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		// Check policy targets
		if isType {
			policies = append(policies, policyName)
		}
	}
	return policies, nil
}

// GetPoliciesForTypeAndNode retrieves all policies with or derived from policyTypeName and with nodeName as target
func GetPoliciesForTypeAndNode(kv *api.KV, deploymentID, policyTypeName, nodeName string) ([]string, error) {
	policiesForType, err := GetPoliciesForType(kv, deploymentID, policyTypeName)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	policies := make([]string, 0)
	for _, policy := range policiesForType {
		is, err := IsTargetForPolicy(kv, deploymentID, policy, nodeName, false)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
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
func GetPolicyPropertyValue(kv *api.KV, deploymentID, policyName, propertyName string, nestedKeys ...string) (*TOSCAValue, error) {
	policyType, err := GetPolicyType(kv, deploymentID, policyName)
	if err != nil {
		return nil, err
	}
	var propDataType string
	hasProp, err := TypeHasProperty(kv, deploymentID, policyType, propertyName, true)
	if err != nil {
		return nil, err
	}
	if hasProp {
		propDataType, err = GetTypePropertyDataType(kv, deploymentID, policyType, propertyName)
		if err != nil {
			return nil, err
		}
	}
	p := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "policies", policyName)

	result, err := getValueAssignmentWithDataType(kv, deploymentID, path.Join(p, "properties", propertyName), policyName, "", "", propDataType, nestedKeys...)
	if err != nil || result != nil {
		return result, errors.Wrapf(err, "Failed to get property %q for policy %q", propertyName, policyName)
	}
	// Not found look at policy type
	value, isFunction, err := getTypeDefaultProperty(kv, deploymentID, policyType, propertyName, nestedKeys...)
	if err != nil {
		return nil, err
	}
	if value != nil {
		if !isFunction {
			return value, nil
		}
		return resolveValueAssignment(kv, deploymentID, policyName, "", "", value, nestedKeys...)
	}
	// Not found anywhere
	return nil, nil
}

// GetPolicyType returns the type of the policy
func GetPolicyType(kv *api.KV, deploymentID, policyName string) (string, error) {
	p := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "policies", policyName)
	kvp, _, err := kv.Get(path.Join(p, "type"), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", errors.Errorf("Missing mandatory attribute \"type\" for policy %q", policyName)
	}
	return string(kvp.Value), nil
}

// IsTargetForPolicy returns true if the node name is a policy target
func IsTargetForPolicy(kv *api.KV, deploymentID, policyName, nodeName string, recursive bool) (bool, error) {
	targets, err := GetPolicyTargets(kv, deploymentID, policyName)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !recursive {
		return collections.ContainsString(targets, nodeName), nil
	}

	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	policyType, err := GetPolicyType(kv, deploymentID, policyName)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	targets, err = GetPolicyTargetsForType(kv, deploymentID, policyType)
	if err != nil {
		return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, target := range targets {
		is, err := IsTypeDerivedFrom(kv, deploymentID, nodeType, target)
		if err != nil {
			return false, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if is {
			return true, nil
		}
	}
	return false, nil
}

// GetPolicyTargets retrieves the policy template targets
// these targets are node names
func GetPolicyTargets(kv *api.KV, deploymentID, policyName string) ([]string, error) {
	p := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "policies", policyName, "targets")
	kvp, _, err := kv.Get(p, nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil && len(kvp.Value) != 0 {
		return strings.Split(string(kvp.Value), ","), nil
	}
	return nil, nil
}

// GetPolicyTargetsForType retrieves the policy type targets
// this targets are node types
func GetPolicyTargetsForType(kv *api.KV, deploymentID, policyType string) ([]string, error) {
	p := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "types", policyType, "targets")
	kvp, _, err := kv.Get(p, nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil || len(kvp.Value) != 0 {
		return strings.Split(string(kvp.Value), ","), nil
	}

	parentType, err := GetParentType(kv, deploymentID, policyType)
	if err != nil {
		return nil, err
	}
	if parentType == "" {
		return nil, nil
	}
	return GetPolicyTargetsForType(kv, deploymentID, parentType)
}
