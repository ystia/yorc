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
	"fmt"
	"strconv"
	"strings"

	"github.com/ystia/yorc/helper/collections"
	"github.com/ystia/yorc/log"
	yaml "gopkg.in/yaml.v2"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/tosca"
)

const funcKeywordSELF string = "SELF"
const funcKeywordHOST string = "HOST"
const funcKeywordSOURCE string = "SOURCE"
const funcKeywordTARGET string = "TARGET"

// R_TARGET has a special meaning for A4C but we just consider it as an alias for TARGET
const funcKeywordRTARGET string = "R_TARGET"
const funcKeywordREQTARGET string = "REQ_TARGET"

// functionResolver is used to resolve TOSCA functions
type functionResolver struct {
	kv               *api.KV
	deploymentID     string
	nodeName         string
	instanceName     string
	requirementIndex string
}

type resolverContext func(*functionResolver)

func (fr *functionResolver) context(contexts ...resolverContext) *functionResolver {
	for _, ctx := range contexts {
		ctx(fr)
	}
	return fr
}

func resolver(kv *api.KV, deploymentID string) *functionResolver {
	return &functionResolver{kv: kv, deploymentID: deploymentID}
}

func withNodeName(nodeName string) resolverContext {
	return func(fr *functionResolver) {
		fr.nodeName = nodeName
	}
}

func withInstanceName(instanceName string) resolverContext {
	return func(fr *functionResolver) {
		fr.instanceName = instanceName
	}
}

func withRequirementIndex(reqIndex string) resolverContext {
	return func(fr *functionResolver) {
		fr.requirementIndex = reqIndex
	}
}

func (fr *functionResolver) resolveFunction(fn *tosca.Function) (string, error) {
	if fn == nil {
		return "", errors.Errorf("Trying to resolve a nil function")
	}
	operands := make([]string, len(fn.Operands))
	for i, op := range fn.Operands {
		if op.IsLiteral() {
			var err error
			s := op.String()
			if isQuoted(s) {
				s, err = strconv.Unquote(s)
				if err != nil {
					return "", errors.Wrapf(err, "failed to unquote literal operand of function %v", fn)
				}
			}
			operands[i] = s
		} else {
			subFn := op.(*tosca.Function)
			r, err := fr.resolveFunction(subFn)
			if err != nil {
				return "", err
			}
			operands[i] = r
		}
	}
	switch fn.Operator {
	case tosca.ConcatOperator:
		return strings.Join(operands, ""), nil
	case tosca.GetInputOperator:
		return fr.resolveGetInput(operands)
	case tosca.GetOperationOutputOperator:
		return fr.resolveGetOperationOutput(operands)
	case tosca.GetPropertyOperator:
		return fr.resolveGetPropertyOrAttribute("property", operands)
	case tosca.GetAttributeOperator:
		return fr.resolveGetPropertyOrAttribute("attribute", operands)
	}
	return "", errors.Errorf("Unsupported function %q", string(fn.Operator))
}

func (fr *functionResolver) resolveGetInput(operands []string) (string, error) {
	if len(operands) < 1 {
		return "", errors.Errorf("expecting at least one parameter for a get_input function")
	}
	args := getFuncNestedArgs(operands...)
	return GetInputValue(fr.kv, fr.deploymentID, args[0], args[1:]...)
}

func (fr *functionResolver) resolveGetOperationOutput(operands []string) (string, error) {
	if len(operands) != 4 {
		return "", errors.Errorf("expecting exactly four parameters for a get_operation_output function")
	}
	entity := operands[0]
	ifName := operands[1]
	opName := operands[2]
	varName := operands[3]
	if fr.instanceName == "" {
		return "", errors.Errorf(`Can't resolve "get_operation_output: [%s]" without a specified instance name`, strings.Join(operands, ", "))
	}
	switch entity {
	case funcKeywordHOST:
		return "", errors.Errorf("Keyword %q not supported for function get_operation_output", funcKeywordHOST)
	case funcKeywordSELF, funcKeywordSOURCE:
		if fr.requirementIndex != "" {
			// Relationship case
			return GetOperationOutputForRelationship(fr.kv, fr.deploymentID, fr.nodeName, fr.instanceName, fr.requirementIndex, ifName, opName, varName)
		}
		// Node case
		// Check entity
		if entity == funcKeywordSOURCE {
			return "", errors.Errorf("Keyword %q not supported for an node expression (only supported in relationships)", funcKeywordSOURCE)
		}
		output, err := GetOperationOutputForNode(fr.kv, fr.deploymentID, fr.nodeName, fr.instanceName, ifName, opName, varName)
		if err != nil || output != "" {
			return output, err
		}
		// Workaround to be backward compatible lets look at relationships
		return getOperationOutputForRequirements(fr.kv, fr.deploymentID, fr.nodeName, fr.instanceName, ifName, opName, varName)
	case funcKeywordTARGET, funcKeywordRTARGET:
		if fr.requirementIndex != "" {
			return "", errors.Errorf("Keyword %q not supported for an node expression (only supported in relationships)", funcKeywordTARGET)
		}
		targetNode, err := GetTargetNodeForRequirement(fr.kv, fr.deploymentID, fr.nodeName, fr.requirementIndex)
		if err != nil {
			return "", err
		}
		return GetOperationOutputForRelationship(fr.kv, fr.deploymentID, targetNode, fr.instanceName, fr.requirementIndex, ifName, opName, varName)
	default:
		instanceIDs, err := GetNodeInstancesIds(fr.kv, fr.deploymentID, entity)
		if err != nil {
			return "", err
		}
		if collections.ContainsString(instanceIDs, fr.instanceName) {
			return GetOperationOutputForNode(fr.kv, fr.deploymentID, entity, fr.instanceName, ifName, opName, varName)
		}
		for _, id := range instanceIDs {
			// by default take the first one
			return GetOperationOutputForNode(fr.kv, fr.deploymentID, entity, id, ifName, opName, varName)
		}

		return "", errors.Errorf(`Can't resolve "get_operation_output: [%s]" can't find a valid instance for %q`, strings.Join(operands, ", "), entity)
	}
}

func (fr *functionResolver) resolveGetPropertyOrAttribute(rType string, operands []string) (string, error) {
	funcString := fmt.Sprintf("get_%s: [%s]", rType, strings.Join(operands, ", "))
	if len(operands) < 2 {
		return "", errors.Errorf("expecting at least two parameters for a get_%s function (%s)", rType, funcString)
	}
	if rType == "attribute" && fr.instanceName == "" {
		return "", errors.Errorf(`Can't resolve %q without a specified instance name`, funcString)
	}
	entity := operands[0]

	if entity == funcKeywordHOST && fr.requirementIndex != "" {
		return "", errors.Errorf(`Can't resolve %q %s keyword is not supported in the context of a relationship`, funcString, funcKeywordHOST)
	} else if fr.requirementIndex == "" && (entity == funcKeywordSOURCE || entity == funcKeywordTARGET || entity == funcKeywordRTARGET) {
		return "", errors.Errorf(`Can't resolve %q %s keyword is supported only in the context of a relationship`, funcString, entity)
	}
	// First get the node on which we should resolve the get_property

	var err error
	var actualNode string
	switch entity {
	case funcKeywordSELF, funcKeywordSOURCE:
		actualNode = fr.nodeName
	case funcKeywordHOST:
		actualNode, err = GetHostedOnNode(fr.kv, fr.deploymentID, fr.nodeName)
		if err != nil {
			return "", err
		}
	case funcKeywordTARGET, funcKeywordRTARGET:
		actualNode, err = GetTargetNodeForRequirement(fr.kv, fr.deploymentID, fr.nodeName, fr.requirementIndex)
		if err != nil {
			return "", err
		}
	case funcKeywordREQTARGET:
		actualNode, err = GetTargetNodeForRequirementByName(fr.kv, fr.deploymentID, fr.nodeName, operands[1])
		if err != nil {
			return "", err
		}
	default:
		actualNode = entity
	}

	if actualNode == "" {
		return "", errors.Errorf(`Can't resolve %q without a specified node name`, funcString)
	}
	var args []string
	var found bool
	var result string
	// Check if second param is a capability or requirement
	// this does not makes sens for a REQTARGET
	if entity != funcKeywordREQTARGET {
		if len(operands) > 2 {
			cap, err := GetNodeCapabilityType(fr.kv, fr.deploymentID, actualNode, operands[1])
			if err != nil {
				return "", err
			}
			if cap != "" {
				args := getFuncNestedArgs(operands[2:]...)
				if rType == "attribute" {
					found, result, err = GetInstanceCapabilityAttribute(fr.kv, fr.deploymentID, actualNode, fr.instanceName, operands[1], args[0], args[1:]...)
				} else {
					found, result, err = GetCapabilityProperty(fr.kv, fr.deploymentID, actualNode, operands[1], args[0], args[1:]...)
				}
				if err != nil || found {
					return result, err
				}
				// Else lets continue
			}
			// TODO requirement not supported in Alien
		}
		args = getFuncNestedArgs(operands[1:]...)
	} else {
		args = getFuncNestedArgs(operands[2:]...)
	}

	if rType == "attribute" {
		if entity == funcKeywordSELF && fr.requirementIndex != "" {
			found, result, err = GetRelationshipAttributeFromRequirement(fr.kv, fr.deploymentID, actualNode, fr.instanceName, fr.requirementIndex, args[0], args[1:]...)
		} else {
			found, result, err = GetInstanceAttribute(fr.kv, fr.deploymentID, actualNode, fr.instanceName, args[0], args[1:]...)
		}
	} else {
		if entity == funcKeywordSELF && fr.requirementIndex != "" {
			found, result, err = GetRelationshipPropertyFromRequirement(fr.kv, fr.deploymentID, actualNode, fr.requirementIndex, args[0], args[1:]...)
		} else {
			found, result, err = GetNodeProperty(fr.kv, fr.deploymentID, actualNode, args[0], args[1:]...)
		}
	}
	if err != nil {
		return "", err
	}
	if !found {
		log.Debugf("Deployment %q, node %q, can't resolve expression %q", fr.deploymentID, fr.nodeName, funcString)
		return "", errors.Errorf("Can't resolve expression %q", funcString)
	}
	return result, nil
}

func getFuncNestedArgs(nestedKeys ...string) []string {
	res := make([]string, 0, len(nestedKeys))
	for _, key := range nestedKeys {
		// Alien way for nested keys . to separe keys and [index] for arrays
		// ex: complex_prop.nested_array[0]
		res = append(res, strings.Split(strings.Replace(strings.Replace(key, "]", "", -1), "[", ".", -1), ".")...)
	}
	return res
}

func resolveValueAssignmentAsString(kv *api.KV, deploymentID, nodeName, instanceName, requirementIndex, valueAssignment string, nestedKeys ...string) (string, error) {
	// Function
	va := &tosca.ValueAssignment{}
	err := yaml.Unmarshal([]byte(valueAssignment), va)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to parse TOSCA function %q for node %q", valueAssignment, nodeName)
	}
	r := resolver(kv, deploymentID).context(withNodeName(nodeName), withInstanceName(instanceName), withRequirementIndex(requirementIndex))
	valueAssignment, err = r.resolveFunction(va.GetFunction())
	if err != nil {
		return "", err
	}
	return valueAssignment, nil
}

func isQuoted(s string) bool {
	return len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"'
}
