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

package operations

import (
	"context"
	"fmt"
	"strings"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/provutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/tasks"
)

// TODO: why do we have both EnvInput and deployments.OperationInputResult? Can't we factorize them?

// An EnvInput represent a TOSCA operation input
type EnvInput struct {
	Name         string
	Value        string
	InstanceName string
	IsSecret     bool
}

func (ei EnvInput) String() string {
	value := ei.Value
	if ei.IsSecret {
		value = "<secret value redacted>"
	}
	return fmt.Sprintf("EnvInput: [Name: %q, Value: %q, InstanceName: %q, IsSecret: %t]", ei.Name, value, ei.InstanceName, ei.IsSecret)
}

// ResolveInputs allows to resolve inputs for an operation
func ResolveInputs(ctx context.Context, deploymentID, nodeName, taskID string, operation prov.Operation) ([]*EnvInput, []string, error) {
	sourceInstances, err := tasks.GetInstances(ctx, taskID, deploymentID, nodeName)
	if err != nil {
		return nil, nil, err
	}

	var targetInstances []string
	if operation.RelOp.IsRelationshipOperation {
		targetInstances, err = tasks.GetInstances(ctx, taskID, deploymentID, operation.RelOp.TargetNodeName)
		if err != nil {
			return nil, nil, err
		}
	}
	return ResolveInputsWithInstances(ctx, deploymentID, nodeName, taskID, operation, sourceInstances, targetInstances)
}

// ResolveInputsWithInstances used to resolve inputs for an operation
func ResolveInputsWithInstances(ctx context.Context, deploymentID, nodeName, taskID string, operation prov.Operation, sourceNodeInstances, targetNodeInstances []string) ([]*EnvInput, []string, error) {

	log.Debug("resolving inputs")

	envInputs := make([]*EnvInput, 0)
	varInputsNames := make([]string, 0)

	inputKeys, err := deployments.GetOperationInputs(ctx, deploymentID, operation.ImplementedInNodeTemplate, operation.ImplementedInType, operation.Name)
	if err != nil {
		return nil, nil, err
	}

	for _, input := range inputKeys {
		isPropDef, err := deployments.IsOperationInputAPropertyDefinition(ctx, deploymentID, nodeName, operation.ImplementedInType, operation.Name, input)
		if err != nil {
			return nil, nil, err
		}

		if isPropDef {
			_, found := operation.Inputs[input]
			if !found {

				// Rely on default values if any
				defaultInputValues, err := deployments.GetOperationInputPropertyDefinitionDefault(ctx, deploymentID, nodeName, operation, input)
				if err != nil {
					return nil, nil, err
				}
				for i, iv := range defaultInputValues {
					envInputs = append(envInputs, &EnvInput{Name: input, InstanceName: GetInstanceName(iv.NodeName, iv.InstanceName), Value: iv.Value, IsSecret: iv.IsSecret})
					if i == 0 {
						varInputsNames = append(varInputsNames, provutil.SanitizeForShell(input))
					}
				}
				continue
			}
		}

		inputValues, err := deployments.GetOperationInput(ctx, deploymentID, nodeName, operation, input)
		if err != nil {
			return nil, nil, err
		}
		for i, iv := range inputValues {
			envInputs = append(envInputs, &EnvInput{Name: input, InstanceName: GetInstanceName(iv.NodeName, iv.InstanceName), Value: iv.Value, IsSecret: iv.IsSecret})
			if i == 0 {
				varInputsNames = append(varInputsNames, provutil.SanitizeForShell(input))
			}
		}
	}
	return envInputs, varInputsNames, nil
}

// GetTargetCapabilityPropertiesAndAttributesValues retrieves properties and attributes of the target capability of the relationship (if this operation is related to a relationship)
//
// It may happen in rare cases that several capabilities match the same requirement.
// Values are stored in this way:
//   - TARGET_CAPABILITY_NAMES: comma-separated list of matching capabilities names. It could be use to loop over the injected variables
//   - TARGET_CAPABILITY_<capabilityName>_TYPE: actual type of the capability
//   - TARGET_CAPABILITY_TYPE: actual type of the capability of the first matching capability
//   - TARGET_CAPABILITY_<capabilityName>_PROPERTY_<propertyName>: value of a property
//   - TARGET_CAPABILITY_PROPERTY_<propertyName>: value of a property for the first matching capability
//   - TARGET_CAPABILITY_<capabilityName>_<instanceName>_ATTRIBUTE_<attributeName>: value of an attribute of a given instance
//   - TARGET_CAPABILITY_<instanceName>_ATTRIBUTE_<attributeName>: value of an attribute of a given instance for the first matching capability
func GetTargetCapabilityPropertiesAndAttributesValues(ctx context.Context, deploymentID, nodeName string, op prov.Operation) (map[string]*deployments.TOSCAValue, error) {
	// Only for relationship operations
	if !IsRelationshipOperation(op) {
		return nil, nil
	}

	props := make(map[string]*deployments.TOSCAValue)

	capabilityType, err := deployments.GetCapabilityForRequirement(ctx, deploymentID, nodeName, op.RelOp.RequirementIndex)
	if err != nil {
		return nil, err
	}

	targetNodeType, err := deployments.GetNodeType(ctx, deploymentID, op.RelOp.TargetNodeName)
	if err != nil {
		return nil, err
	}

	targetInstances, err := deployments.GetNodeInstancesIds(ctx, deploymentID, op.RelOp.TargetNodeName)
	if err != nil {
		return nil, err
	}

	capabilities, err := deployments.GetCapabilitiesOfType(ctx, deploymentID, targetNodeType, capabilityType)
	for i, capabilityName := range capabilities {
		capabilityType, err := deployments.GetNodeTypeCapabilityType(ctx, deploymentID, targetNodeType, capabilityName)
		if err != nil {
			return nil, err
		}
		props["TARGET_CAPABILITY_"+capabilityName+"_TYPE"] = &deployments.TOSCAValue{Value: capabilityType}
		if i == 0 {
			props["TARGET_CAPABILITY_TYPE"] = &deployments.TOSCAValue{Value: capabilityType}
		}

		err = setCapabilityProperties(ctx, deploymentID, capabilityName, capabilityType, op, i == 0, props)
		if err != nil {
			return nil, err
		}

		err = setCapabilityAttributes(ctx, deploymentID, capabilityName, capabilityType, op, targetInstances, i == 0, props)
		if err != nil {
			return nil, err
		}
	}
	props["TARGET_CAPABILITY_NAMES"] = &deployments.TOSCAValue{Value: strings.Join(capabilities, ",")}
	return props, nil
}

func setCapabilityProperties(ctx context.Context, deploymentID, capabilityName, capabilityType string, op prov.Operation, isFirst bool, props map[string]*deployments.TOSCAValue) error {
	capProps, err := deployments.GetTypeProperties(ctx, deploymentID, capabilityType, true)
	if err != nil {
		return err
	}
	for _, capProp := range capProps {
		value, err := deployments.GetCapabilityPropertyValue(ctx, deploymentID, op.RelOp.TargetNodeName, capabilityName, capProp)
		if err != nil {
			return err
		}
		if value == nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("failed to retrieve property %q for capability %q on node %q. It will not be injected in operation context.", capProp, capabilityName, op.RelOp.TargetNodeName)
			continue
		}
		props["TARGET_CAPABILITY_"+capabilityName+"_PROPERTY_"+capProp] = value
		if isFirst {
			props["TARGET_CAPABILITY_PROPERTY_"+capProp] = value
		}
	}
	return nil
}

func setCapabilityAttributes(ctx context.Context, deploymentID, capabilityName, capabilityType string, op prov.Operation, targetInstances []string, isFirst bool, props map[string]*deployments.TOSCAValue) error {
	capAttrs, err := deployments.GetTypeAttributes(ctx, deploymentID, capabilityType, true)
	if err != nil {
		return err
	}
	for _, capAttr := range capAttrs {
		for _, instanceID := range targetInstances {
			value, err := deployments.GetInstanceCapabilityAttributeValue(ctx, deploymentID, op.RelOp.TargetNodeName, instanceID, capabilityName, capAttr)
			if err != nil {
				return err
			}
			if value == nil {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("failed to retrieve attribute %q for capability %q on node %q instance %q. It will not be injected in operation context.", capAttr, capabilityName, op.RelOp.TargetNodeName, instanceID)
				continue
			}
			instanceName := GetInstanceName(op.RelOp.TargetNodeName, instanceID)
			props[fmt.Sprintf("TARGET_CAPABILITY_%s_%s_ATTRIBUTE_%s", capabilityName, instanceName, capAttr)] = value
			if isFirst {
				props[fmt.Sprintf("TARGET_CAPABILITY_%s_ATTRIBUTE_%s", instanceName, capAttr)] = value
			}
		}
	}
	return nil
}
