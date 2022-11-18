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
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tosca"
	"github.com/ystia/yorc/v4/vault"
)

// DefaultVaultClient is the default Vault Client used to resolve get_secret functions
// it is nil by default and should be set by the one who created the client
var DefaultVaultClient vault.Client

const funcKeywordSELF string = "SELF"
const funcKeywordHOST string = "HOST"
const funcKeywordSOURCE string = "SOURCE"
const funcKeywordTARGET string = "TARGET"

// R_TARGET has a special meaning for A4C but we just consider it as an alias for TARGET
const funcKeywordRTARGET string = "R_TARGET"
const funcKeywordREQTARGET string = "REQ_TARGET"

// functionResolver is used to resolve TOSCA functions
type functionResolver struct {
	deploymentID     string
	nodeName         string
	instanceName     string
	requirementIndex string
	inputs           map[string]tosca.ParameterDefinition
}

type resolverContext func(*functionResolver)

func (fr *functionResolver) context(contexts ...resolverContext) *functionResolver {
	for _, ctx := range contexts {
		ctx(fr)
	}
	return fr
}

func resolver(deploymentID string) *functionResolver {
	return &functionResolver{deploymentID: deploymentID}
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

func withInputParameters(inputs map[string]tosca.ParameterDefinition) resolverContext {
	return func(fr *functionResolver) {
		fr.inputs = inputs
	}
}

func (fr *functionResolver) resolveFunction(ctx context.Context, fn *tosca.Function) (*TOSCAValue, error) {
	if fn == nil {
		return nil, errors.Errorf("Trying to resolve a nil function")
	}
	operands := make([]string, len(fn.Operands))
	var hasSecret bool
	for i, op := range fn.Operands {
		if op.IsLiteral() {
			var err error
			s := op.String()
			if isQuoted(s) {
				s, err = strconv.Unquote(s)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to unquote literal operand of function %v", fn)
				}
			}
			operands[i] = s
		} else {
			subFn := op.(*tosca.Function)
			r, err := fr.resolveFunction(ctx, subFn)
			if err != nil {
				return nil, err
			}
			if r != nil {
				if r.IsSecret {
					hasSecret = true
				}
				operands[i] = r.RawString()
			}
		}
	}
	switch fn.Operator {
	case tosca.ConcatOperator:
		return &TOSCAValue{Value: strings.Join(operands, ""), IsSecret: hasSecret}, nil
	case tosca.GetInputOperator:
		res, err := fr.resolveGetInput(ctx, operands)
		return &TOSCAValue{Value: res}, err
	case tosca.GetSecretOperator:
		res, err := fr.resolveGetSecret(operands)
		return &TOSCAValue{Value: res, IsSecret: true}, err
	case tosca.GetOperationOutputOperator:
		res, err := fr.resolveGetOperationOutput(ctx, operands)
		return &TOSCAValue{Value: res}, err
	case tosca.GetPropertyOperator:
		res, err := fr.resolveGetPropertyOrAttribute(ctx, "property", operands)
		if res != nil && hasSecret {
			res.IsSecret = true
		}
		return res, err
	case tosca.GetAttributeOperator:
		res, err := fr.resolveGetPropertyOrAttribute(ctx, "attribute", operands)
		if res != nil && hasSecret {
			res.IsSecret = true
		}
		return res, err
	}
	return nil, errors.Errorf("Unsupported function %q", string(fn.Operator))
}

func (fr *functionResolver) resolveGetInput(ctx context.Context, operands []string) (string, error) {
	if len(operands) < 1 {
		return "", errors.Errorf("expecting at least one parameter for a get_input function")
	}
	args := getFuncNestedArgs(operands...)
	return GetInputValue(ctx, fr.inputs, fr.deploymentID, args[0], args[1:]...)
}

func (fr *functionResolver) resolveGetOperationOutput(ctx context.Context, operands []string) (string, error) {
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
			return GetOperationOutputForRelationship(ctx, fr.deploymentID, fr.nodeName, fr.instanceName, fr.requirementIndex, ifName, opName, varName)
		}
		// Node case
		// Check entity
		if entity == funcKeywordSOURCE {
			return "", errors.Errorf("Keyword %q not supported for an node expression (only supported in relationships)", funcKeywordSOURCE)
		}
		output, err := GetOperationOutputForNode(ctx, fr.deploymentID, fr.nodeName, fr.instanceName, ifName, opName, varName)
		if err != nil || output != "" {
			return output, err
		}
		// Workaround to be backward compatible lets look at relationships
		return getOperationOutputForRequirements(ctx, fr.deploymentID, fr.nodeName, fr.instanceName, ifName, opName, varName)
	case funcKeywordTARGET, funcKeywordRTARGET:
		if fr.requirementIndex != "" {
			return "", errors.Errorf("Keyword %q not supported for an node expression (only supported in relationships)", funcKeywordTARGET)
		}
		targetNode, err := GetTargetNodeForRequirement(ctx, fr.deploymentID, fr.nodeName, fr.requirementIndex)
		if err != nil {
			return "", err
		}
		return GetOperationOutputForRelationship(ctx, fr.deploymentID, targetNode, fr.instanceName, fr.requirementIndex, ifName, opName, varName)
	default:
		instanceIDs, err := GetNodeInstancesIds(ctx, fr.deploymentID, entity)
		if err != nil {
			return "", err
		}
		if collections.ContainsString(instanceIDs, fr.instanceName) {
			return GetOperationOutputForNode(ctx, fr.deploymentID, entity, fr.instanceName, ifName, opName, varName)
		}
		for _, id := range instanceIDs {
			// by default take the first one
			return GetOperationOutputForNode(ctx, fr.deploymentID, entity, id, ifName, opName, varName)
		}

		return "", errors.Errorf(`Can't resolve "get_operation_output: [%s]" can't find a valid instance for %q`, strings.Join(operands, ", "), entity)
	}
}

func (fr *functionResolver) resolveGetPropertyOrAttribute(ctx context.Context, rType string, operands []string) (*TOSCAValue, error) {
	funcString := fmt.Sprintf("get_%s: [%s]", rType, strings.Join(operands, ", "))
	if len(operands) < 2 {
		return nil, errors.Errorf("expecting at least two parameters for a get_%s function (%s)", rType, funcString)
	}
	if rType == "attribute" && fr.instanceName == "" {
		return nil, errors.Errorf(`Can't resolve %q without a specified instance name`, funcString)
	}
	entity := operands[0]

	if entity == funcKeywordHOST && fr.requirementIndex != "" {
		return nil, errors.Errorf(`Can't resolve %q %s keyword is not supported in the context of a relationship`, funcString, funcKeywordHOST)
	} else if fr.requirementIndex == "" && (entity == funcKeywordSOURCE || entity == funcKeywordTARGET || entity == funcKeywordRTARGET) {
		return nil, errors.Errorf(`Can't resolve %q %s keyword is supported only in the context of a relationship`, funcString, entity)
	}
	// First get the node on which we should resolve the get_property

	var err error
	var actualNode string
	switch entity {
	case funcKeywordSELF, funcKeywordSOURCE:
		actualNode = fr.nodeName
	case funcKeywordHOST:
		actualNode, err = GetHostedOnNode(ctx, fr.deploymentID, fr.nodeName)
		if err != nil {
			return nil, err
		}
	case funcKeywordTARGET, funcKeywordRTARGET:
		actualNode, err = GetTargetNodeForRequirement(ctx, fr.deploymentID, fr.nodeName, fr.requirementIndex)
		if err != nil {
			return nil, err
		}
	case funcKeywordREQTARGET:
		actualNode, err = GetTargetNodeForRequirementByName(ctx, fr.deploymentID, fr.nodeName, operands[1])
		if err != nil {
			return nil, err
		}
		if actualNode == "" {
			nodeType, err := GetNodeType(ctx, fr.deploymentID, fr.nodeName)
			if err != nil {
				return nil, err
			}
			req, err := GetRequirementDefinitionOnTypeByName(ctx, fr.deploymentID, nodeType, operands[1])
			if err != nil {
				return nil, err
			}
			if req == nil {
				return nil, errors.Errorf("missing requirement %q on node %q", operands[1], fr.nodeName)
			}
			if req.Occurrences.LowerBound != 0 {
				return nil, errors.Errorf("requirement %q on node %q not found while not optional (occurrences lower bound set to %d)", operands[1], fr.nodeName, req.Occurrences.LowerBound)
			}
			// this means that the relationship between those 2 nodes does not exist
			// and the requirement definition occurrence lower bound is set to 0 meaning it is optional.
			return nil, nil
		}
	default:
		actualNode = entity
	}

	if actualNode == "" {
		return nil, errors.Errorf(`Can't resolve %q without a specified node name`, funcString)
	}
	var args []string
	var result *TOSCAValue
	// Check if second param is a capability or requirement
	// this does not makes sense for a REQTARGET
	if entity != funcKeywordREQTARGET {
		if len(operands) > 2 {
			cap, err := GetNodeCapabilityType(ctx, fr.deploymentID, actualNode, operands[1])
			if err != nil {
				return nil, err
			}
			if cap != "" {
				args := getFuncNestedArgs(operands[2:]...)
				if rType == "attribute" {
					result, err = GetInstanceCapabilityAttributeValue(ctx, fr.deploymentID, actualNode, fr.instanceName, operands[1], args[0], args[1:]...)
				} else {
					result, err = GetCapabilityPropertyValue(ctx, fr.deploymentID, actualNode, operands[1], args[0], args[1:]...)
				}
				if err != nil || result != nil {
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
			result, err = GetRelationshipAttributeValueFromRequirement(ctx, fr.deploymentID, actualNode, fr.instanceName, fr.requirementIndex, args[0], args[1:]...)
		} else {
			result, err = GetInstanceAttributeValue(ctx, fr.deploymentID, actualNode, fr.instanceName, args[0], args[1:]...)
		}
	} else {
		if entity == funcKeywordSELF && fr.requirementIndex != "" {
			result, err = GetRelationshipPropertyValueFromRequirement(ctx, fr.deploymentID, actualNode, fr.requirementIndex, args[0], args[1:]...)
		} else {
			result, err = GetNodePropertyValue(ctx, fr.deploymentID, actualNode, args[0], args[1:]...)
		}
	}
	if err != nil || result != nil {
		return result, err
	}

	// A not found attribute is considered as acceptable for GetAttribute function and so doesn't return any error
	if rType == "attribute" {
		ctx := events.NewContext(context.Background(), events.LogOptionalFields{events.NodeID: fr.nodeName, events.InstanceID: fr.instanceName})
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, fr.deploymentID).Registerf("[WARNING] The attribute %q hasn't be found for deployment: %q, node: %q, instance: %q in expression %q", args[0], fr.deploymentID, fr.nodeName, fr.instanceName, funcString)
		return nil, nil
	}
	log.Debugf("Deployment %q, node %q, can't resolve expression %q", fr.deploymentID, fr.nodeName, funcString)
	return nil, errors.Errorf("Can't resolve expression %q", funcString)
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

func resolveValueAssignmentAsString(ctx context.Context, deploymentID, nodeName, instanceName, requirementIndex, valueAssignment string, nestedKeys ...string) (*TOSCAValue, error) {
	return resolveValueAssignment(ctx, deploymentID, nodeName, instanceName, requirementIndex, &TOSCAValue{Value: valueAssignment}, nestedKeys...)
}

func resolveValueAssignment(ctx context.Context, deploymentID, nodeName, instanceName, requirementIndex string, valueAssignment *TOSCAValue, nestedKeys ...string) (*TOSCAValue, error) {
	// Function
	fromSecret := valueAssignment.IsSecret
	f, err := tosca.ParseFunction(valueAssignment.RawString())
	if err != nil {
		return nil, err
	}
	r := resolver(deploymentID).context(withNodeName(nodeName), withInstanceName(instanceName), withRequirementIndex(requirementIndex))
	valueAssignment, err = r.resolveFunction(ctx, f)
	if err != nil {
		return nil, err
	}
	if valueAssignment == nil {
		ctx := events.NewContext(context.Background(), events.LogOptionalFields{events.NodeID: nodeName, events.InstanceID: instanceName})
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).Registerf("[WARNING] The value assignment %q hasn't be resolved for deployment: %q, node: %q, instance: %q. An empty string is returned instead", valueAssignment, deploymentID, nodeName, instanceName)
		valueAssignment = &TOSCAValue{Value: ""}
	}
	if fromSecret {
		valueAssignment.IsSecret = true
	}

	return valueAssignment, nil
}

func isQuoted(s string) bool {
	return len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"'
}

func (fr *functionResolver) resolveGetSecret(operands []string) (string, error) {
	if len(operands) < 1 {
		return "", errors.New("expecting at least one parameter for a get_secret function")
	}

	if DefaultVaultClient == nil {
		return "", errors.New("can't resolve get_secret function there is no vault client configured")
	}
	var options []string
	if len(operands) > 1 {
		options = operands[1:]
	}
	secret, err := DefaultVaultClient.GetSecret(operands[0], options...)
	if err != nil {
		return "", err
	}
	return secret.String(), nil
}
