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

	"github.com/ystia/yorc/events"

	"github.com/hashicorp/consul/api"

	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/helper/provutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/tasks"
)

// An EnvInput represent a TOSCA operation input
type EnvInput struct {
	Name         string
	Value        string
	InstanceName string
}

func (ei EnvInput) String() string {
	return fmt.Sprintf("EnvInput: [Name: %q, Value: %q, InstanceName: %q]", ei.Name, ei.Value, ei.InstanceName)
}

// ResolveInputs allows to resolve inputs for an operation
func ResolveInputs(kv *api.KV, deploymentID, nodeName, taskID string, operation prov.Operation) ([]*EnvInput, []string, error) {
	sourceInstances, err := tasks_old.GetInstances(kv, taskID, deploymentID, nodeName)
	if err != nil {
		return nil, nil, err
	}

	var targetInstances []string
	if operation.RelOp.IsRelationshipOperation {
		targetInstances, err = tasks_old.GetInstances(kv, taskID, deploymentID, operation.RelOp.TargetNodeName)
		if err != nil {
			return nil, nil, err
		}
	}
	return ResolveInputsWithInstances(kv, deploymentID, nodeName, taskID, operation, sourceInstances, targetInstances)
}

// ResolveInputsWithInstances used to resolve inputs for an operation
func ResolveInputsWithInstances(kv *api.KV, deploymentID, nodeName, taskID string, operation prov.Operation,
	sourceNodeInstances, targetNodeInstances []string) ([]*EnvInput, []string, error) {

	log.Debug("resolving inputs")

	envInputs := make([]*EnvInput, 0)
	varInputsNames := make([]string, 0)

	inputKeys, err := deployments.GetOperationInputs(kv, deploymentID, operation.ImplementedInNodeTemplate, operation.ImplementedInType, operation.Name)
	if err != nil {
		return nil, nil, err
	}

	for _, input := range inputKeys {
		isPropDef, err := deployments.IsOperationInputAPropertyDefinition(kv, deploymentID, operation.ImplementedInNodeTemplate, operation.ImplementedInType, operation.Name, input)
		if err != nil {
			return nil, nil, err
		}

		if isPropDef {
			inputValue, err := tasks_old.GetTaskInput(kv, taskID, input)
			if err != nil {
				if !tasks_old.IsTaskDataNotFoundError(err) {
					return nil, nil, err
				}
				defaultInputValues, err := deployments.GetOperationInputPropertyDefinitionDefault(kv, deploymentID, nodeName, operation, input)
				if err != nil {
					return nil, nil, err
				}
				for i, iv := range defaultInputValues {
					envInputs = append(envInputs, &EnvInput{Name: input, InstanceName: GetInstanceName(iv.NodeName, iv.InstanceName), Value: iv.Value})
					if i == 0 {
						varInputsNames = append(varInputsNames, provutil.SanitizeForShell(input))
					}
				}
				continue
			}
			instances, err := deployments.GetNodeInstancesIds(kv, deploymentID, nodeName)
			if err != nil {
				return nil, nil, err
			}
			for i, ins := range instances {
				envInputs = append(envInputs, &EnvInput{Name: input, InstanceName: GetInstanceName(nodeName, ins), Value: inputValue})
				if i == 0 {
					varInputsNames = append(varInputsNames, provutil.SanitizeForShell(input))
				}
			}
		} else {
			inputValues, err := deployments.GetOperationInput(kv, deploymentID, nodeName, operation, input)
			if err != nil {
				return nil, nil, err
			}
			for i, iv := range inputValues {
				envInputs = append(envInputs, &EnvInput{Name: input, InstanceName: GetInstanceName(iv.NodeName, iv.InstanceName), Value: iv.Value})
				if i == 0 {
					varInputsNames = append(varInputsNames, provutil.SanitizeForShell(input))
				}
			}
		}
	}

	log.Debugf("Resolved env inputs: %s", envInputs)
	return envInputs, varInputsNames, nil
}

// GetTargetCapabilityPropertiesAndAttributes retrieves properties and attributes of the target capability of the relationship (if this operation is related to a relationship)
//
// It may happen in rare cases that several capabilities match the same requirement.
// Values are stored in this way:
//   * TARGET_CAPABILITY_NAMES: comma-separated list of matching capabilities names. It could be use to loop over the injected variables
//   * TARGET_CAPABILITY_<capabilityName>_TYPE: actual type of the capability
//   * TARGET_CAPABILITY_TYPE: actual type of the capability of the first matching capability
// 	 * TARGET_CAPABILITY_<capabilityName>_PROPERTY_<propertyName>: value of a property
// 	 * TARGET_CAPABILITY_PROPERTY_<propertyName>: value of a property for the first matching capability
// 	 * TARGET_CAPABILITY_<capabilityName>_<instanceName>_ATTRIBUTE_<attributeName>: value of an attribute of a given instance
// 	 * TARGET_CAPABILITY_<instanceName>_ATTRIBUTE_<attributeName>: value of an attribute of a given instance for the first matching capability
func GetTargetCapabilityPropertiesAndAttributes(ctx context.Context, kv *api.KV, deploymentID, nodeName string, op prov.Operation) (map[string]string, error) {
	// Only for relationship operations
	if !IsRelationshipOperation(op) {
		return nil, nil
	}

	props := make(map[string]string)

	capabilityType, err := deployments.GetCapabilityForRequirement(kv, deploymentID, nodeName, op.RelOp.RequirementIndex)
	if err != nil {
		return nil, err
	}

	targetNodeType, err := deployments.GetNodeType(kv, deploymentID, op.RelOp.TargetNodeName)
	if err != nil {
		return nil, err
	}

	targetInstances, err := deployments.GetNodeInstancesIds(kv, deploymentID, op.RelOp.TargetNodeName)
	if err != nil {
		return nil, err
	}

	capabilities, err := deployments.GetCapabilitiesOfType(kv, deploymentID, targetNodeType, capabilityType)
	for i, capabilityName := range capabilities {
		capabilityType, err := deployments.GetNodeTypeCapabilityType(kv, deploymentID, targetNodeType, capabilityName)
		if err != nil {
			return nil, err
		}
		props["TARGET_CAPABILITY_"+capabilityName+"_TYPE"] = capabilityType
		if i == 0 {
			props["TARGET_CAPABILITY_TYPE"] = capabilityType
		}

		err = setCapabilityProperties(ctx, kv, deploymentID, capabilityName, capabilityType, op, i == 0, props)
		if err != nil {
			return nil, err
		}

		err = setCapabilityAttributes(ctx, kv, deploymentID, capabilityName, capabilityType, op, targetInstances, i == 0, props)
		if err != nil {
			return nil, err
		}
	}
	props["TARGET_CAPABILITY_NAMES"] = strings.Join(capabilities, ",")
	return props, nil
}

func setCapabilityProperties(ctx context.Context, kv *api.KV, deploymentID, capabilityName, capabilityType string, op prov.Operation, isFirst bool, props map[string]string) error {
	capProps, err := deployments.GetTypeProperties(kv, deploymentID, capabilityType, true)
	if err != nil {
		return err
	}
	for _, capProp := range capProps {
		found, value, err := deployments.GetCapabilityProperty(kv, deploymentID, op.RelOp.TargetNodeName, capabilityName, capProp)
		if err != nil {
			return err
		}
		if !found {
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

func setCapabilityAttributes(ctx context.Context, kv *api.KV, deploymentID, capabilityName, capabilityType string, op prov.Operation, targetInstances []string, isFirst bool, props map[string]string) error {
	capAttrs, err := deployments.GetTypeAttributes(kv, deploymentID, capabilityType, true)
	if err != nil {
		return err
	}
	for _, capAttr := range capAttrs {
		for _, instanceID := range targetInstances {
			found, value, err := deployments.GetInstanceCapabilityAttribute(kv, deploymentID, op.RelOp.TargetNodeName, instanceID, capabilityName, capAttr)
			if err != nil {
				return err
			}
			if !found {
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
