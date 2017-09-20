package operations

import (
	"fmt"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"gopkg.in/yaml.v2"

	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/provutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/tasks"
	"novaforge.bull.com/starlings-janus/janus/tosca"
)

// An EnvInput represent a TOSCA operation input
//
// This element is exported in order to be used by text.Template but should be consider as internal
type EnvInput struct {
	Name           string
	Value          string
	InstanceName   string
	IsTargetScoped bool
}

func (ei EnvInput) String() string {
	return fmt.Sprintf("EnvInput: [Name: %q, Value: %q, InstanceName: %q, IsTargetScoped: \"%t\"]", ei.Name, ei.Value, ei.InstanceName, ei.IsTargetScoped)
}

//
func ResolveInputs(kv *api.KV, deploymentID, nodeName, taskID string, operation prov.Operation) ([]*EnvInput, []string, error) {
	sourceInstances, err := tasks.GetInstances(kv, taskID, deploymentID, nodeName)
	if err != nil {
		return nil, nil, err
	}

	var targetInstances []string
	if operation.RelOp.IsRelationshipOperation {
		targetInstances, err = tasks.GetInstances(kv, taskID, deploymentID, operation.RelOp.TargetNodeName)
		if err != nil {
			return nil, nil, err
		}
	}
	return ResolveInputsWithInstances(kv, deploymentID, nodeName, taskID, operation, sourceInstances, targetInstances)
}

// InputsResolver used to resolve inputs for Kubernetes support
func ResolveInputsWithInstances(kv *api.KV, deploymentID, nodeName, taskID string, operation prov.Operation,
	sourceNodeInstances, targetNodeInstances []string) ([]*EnvInput, []string, error) {

	log.Debug("resolving inputs")
	resolver := deployments.NewResolver(kv, deploymentID)

	envInputs := make([]*EnvInput, 0)
	varInputsNames := make([]string, 0)

	inputKeys, err := deployments.GetOperationInputs(kv, deploymentID, operation.ImplementedInType, operation.Name)
	if err != nil {
		return nil, nil, err
	}

	for _, input := range inputKeys {
		isPropDef, err := deployments.IsOperationInputAPropertyDefinition(kv, deploymentID, operation.ImplementedInType, operation.Name, input)
		if err != nil {
			return nil, nil, err
		}

		va := tosca.ValueAssignment{}
		var targetContext bool
		if !isPropDef {
			expr, err := deployments.GetOperationInputExpression(kv, deploymentID, operation.ImplementedInType, operation.Name, input)
			if err != nil {
				return nil, nil, err
			}
			// TODO if expr == ""
			err = yaml.Unmarshal([]byte(expr), &va)
			if err != nil {
				return nil, nil, errors.Wrap(err, "Failed to resolve operation inputs, unable to unmarshal yaml expression: ")
			}
			targetContext = va.Expression.IsTargetContext()
		}

		var instancesIds []string
		if operation.RelOp.IsRelationshipOperation && targetContext {
			instancesIds = targetNodeInstances
		} else {
			instancesIds = sourceNodeInstances
		}
		var inputValue string
		for i, instanceID := range instancesIds {
			envI := &EnvInput{Name: input, IsTargetScoped: targetContext}
			if operation.RelOp.IsRelationshipOperation && targetContext {
				envI.InstanceName = GetInstanceName(operation.RelOp.TargetNodeName, instanceID)
			} else {
				envI.InstanceName = GetInstanceName(nodeName, instanceID)
			}
			if operation.RelOp.IsRelationshipOperation {
				inputValue, err = resolver.ResolveExpressionForRelationship(va.Expression, nodeName, operation.RelOp.TargetNodeName, operation.RelOp.RequirementIndex, instanceID)
			} else if isPropDef {
				inputValue, err = tasks.GetTaskInput(kv, taskID, input)
			} else {
				inputValue, err = resolver.ResolveExpressionForNode(va.Expression, nodeName, instanceID)
			}
			if err != nil {
				return nil, nil, err
			}
			envI.Value = inputValue
			envInputs = append(envInputs, envI)
			if i == 0 {
				varInputsNames = append(varInputsNames, provutil.SanitizeForShell(input))
			}
		}
	}

	log.Debugf("Resolved env inputs: %s", envInputs)
	return envInputs, varInputsNames, nil
}
