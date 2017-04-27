package operations

import (
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/helper/provutil"
	"novaforge.bull.com/starlings-janus/janus/prov/structs"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"novaforge.bull.com/starlings-janus/janus/tosca"
	"strconv"
)

func InputsResolver(kv *api.KV, operationPath, deploymentID, nodeName, taskID, operation string) ([]*structs.EnvInput, []string, error) {

	resolver := deployments.NewResolver(kv, deploymentID)
	EnvInputs := make([]*structs.EnvInput, 0)
	VarInputsNames := make([]string, 0)

	inputKeys, _, err := kv.Keys(operationPath+"/inputs/", "/", nil)
	if err != nil {
		return EnvInputs, VarInputsNames, err
	}

	isRelationshipOperation, _, requirementIndex, relationshipTargetName, err := deployments.DecodeOperation(kv, deploymentID, nodeName, operation)
	if err != nil {
		return EnvInputs, VarInputsNames, err
	}

	sourceInstances, err := tasks.GetInstances(kv, taskID, deploymentID, nodeName)
	if err != nil {
		return EnvInputs, VarInputsNames, err
	}

	targetInstances, err := tasks.GetInstances(kv, taskID, deploymentID, relationshipTargetName)
	if err != nil {
		return EnvInputs, VarInputsNames, err
	}

	for _, input := range inputKeys {
		kvPair, _, err := kv.Get(input+"/name", nil)
		if err != nil {
			return EnvInputs, VarInputsNames, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if kvPair == nil {
			return EnvInputs, VarInputsNames, errors.Errorf("%s/name missing", input)
		}
		inputName := string(kvPair.Value)

		kvPair, _, err = kv.Get(input+"/is_property_definition", nil)
		if err != nil {
			return EnvInputs, VarInputsNames, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		isPropDef, err := strconv.ParseBool(string(kvPair.Value))
		if err != nil {
			return EnvInputs, VarInputsNames, err
		}

		va := tosca.ValueAssignment{}
		var targetContext bool
		if !isPropDef {
			kvPair, _, err = kv.Get(input+"/expression", nil)
			if err != nil {
				return EnvInputs, VarInputsNames, err
			}
			if kvPair == nil {
				return EnvInputs, VarInputsNames, errors.Errorf("%s/expression missing", input)
			}

			err = yaml.Unmarshal(kvPair.Value, &va)
			if err != nil {
				return EnvInputs, VarInputsNames, errors.Wrap(err, "Failed to resolve operation inputs, unable to unmarshal yaml expression: ")
			}
			targetContext = va.Expression.IsTargetContext()
		}

		var instancesIds []string
		if targetContext {
			instancesIds = targetInstances
		} else {
			instancesIds = sourceInstances
		}

		var inputValue string
		for i, instanceID := range instancesIds {
			envI := &structs.EnvInput{Name: inputName, IsTargetScoped: targetContext}
			if isRelationshipOperation && targetContext {
				envI.InstanceName = GetInstanceName(relationshipTargetName, instanceID)
			} else {
				envI.InstanceName = GetInstanceName(nodeName, instanceID)
			}
			if isRelationshipOperation {
				inputValue, err = resolver.ResolveExpressionForRelationship(va.Expression, nodeName, relationshipTargetName, requirementIndex, instanceID)
			} else if isPropDef {
				inputValue, err = tasks.GetTaskInput(kv, taskID, inputName)
			} else {
				inputValue, err = resolver.ResolveExpressionForNode(va.Expression, nodeName, instanceID)
			}
			if err != nil {
				return EnvInputs, VarInputsNames, err
			}

			envI.Value = inputValue
			EnvInputs = append(EnvInputs, envI)
			if i == 0 {
				VarInputsNames = append(VarInputsNames, provutil.SanitizeForShell(inputName))
			}
		}
	}

	return EnvInputs, VarInputsNames, nil
}
