package deployments

import (
	"path/filepath"
	"strings"

	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

// GetOperationPathAndPrimaryImplementationForNodeType traverses the type hierarchy to find an operation matching the given operationName.
//
// Once found it returns the path to the operation and the value of its primary implementation.
// If the operation is not found in the type hierarchy then empty strings are returned.
func GetOperationPathAndPrimaryImplementationForNodeType(kv *api.KV, deploymentID, nodeType, operationName string) (string, string, error) {
	// First check if operation exists in current nodeType
	var op string
	if idx := strings.Index(operationName, "configure."); idx >= 0 {
		op = operationName[idx:]
	} else if idx := strings.Index(operationName, "standard."); idx >= 0 {
		op = operationName[idx:]
	} else if idx := strings.Index(operationName, "custom."); idx >= 0 {
		op = operationName[idx:]
	} else {
		op = strings.TrimPrefix(operationName, "tosca.interfaces.node.lifecycle.")
		op = strings.TrimPrefix(op, "tosca.interfaces.relationship.")
		op = strings.TrimPrefix(op, "tosca.interfaces.node.lifecycle.")
		op = strings.TrimPrefix(op, "tosca.interfaces.relationship.")
	}
	op = strings.Replace(op, ".", "/", -1)
	operationPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", nodeType, "interfaces", op)
	kvp, _, err := kv.Get(path.Join(operationPath, "implementation/primary"), nil)
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed to retrieve primary implementation for operation %q on type %q", operationName, nodeType)
	}
	if kvp != nil && len(kvp.Value) > 0 {
		return operationPath, string(kvp.Value), nil
	}

	// Not found here check the type hierarchy
	parentType, err := GetParentType(kv, deploymentID, nodeType)
	if err != nil || parentType == "" {
		return "", "", err
	}

	return GetOperationPathAndPrimaryImplementationForNodeType(kv, deploymentID, parentType, operationName)
}

// IsNormativeOperation checks if a given operationName is known as a normative operation.
//
// The given operationName should be the fully qualified operation name composed of the <interface_type_name>.<operation_name>
// Basically this function checks if operationName starts with either tosca.interfaces.node.lifecycle.Standard or tosca.interfaces.relationship.Configure (the case is ignored)
func IsNormativeOperation(kv *api.KV, deploymentID, operationName string) bool {
	operationName = strings.ToLower(operationName)
	return strings.HasPrefix(operationName, "tosca.interfaces.relationship.configure") || strings.HasPrefix(operationName, "tosca.interfaces.node.lifecycle.standard")
}

// IsRelationshipOperationOnTargetNode returns true if the given operationName contains one of the following patterns (case doesn't matter):
//		pre_configure_target, post_configure_target, add_source
// Those patterns indicates that a relationship operation executes on the target node
func IsRelationshipOperationOnTargetNode(operationName string) bool {
	op := strings.ToLower(operationName)
	if strings.Contains(op, "pre_configure_target") || strings.Contains(op, "post_configure_target") || strings.Contains(op, "add_source") {
		return true
	}
	return false
}

// DecodeOperation takes a given operationName that should be formated as <fully_qualified_operation_name> or <fully_qualified_relationship_operation_name>/<requirementIndex> or <fully_qualified_relationship_operation_name>/<requirementName>/<targetNodeName>
// and extract the revelant information
//
// * isRelationshipOp indicates if operationName follows one of the relationship operation format
// * operationRealName extracts the fully_qualified_operation_name (identical to operationName if isRelationshipOp==false)
// * requirementIndex is the index of the requirement for this relationship operation (empty if isRelationshipOp==false)
// * targetNodeName is the name of the target node for this relationship operation (empty if isRelationshipOp==false)
func DecodeOperation(kv *api.KV, deploymentID, nodeName, operationName string) (isRelationshipOp bool, operationRealName, requirementIndex, targetNodeName string, err error) {
	opParts := strings.Split(operationName, "/")
	if len(opParts) == 1 {
		// not a relationship use default for return values
		operationRealName = operationName
		return
	} else if len(opParts) == 2 {
		isRelationshipOp = true
		operationRealName = opParts[0]
		requirementIndex = opParts[1]

		targetNodeName, err = GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)
		return
	} else if len(opParts) == 3 {
		isRelationshipOp = true
		operationRealName = opParts[0]
		requirementName := opParts[1]
		targetNodeName = opParts[2]
		var requirementPath string
		requirementPath, err = GetRequirementByNameAndTargetForNode(kv, deploymentID, nodeName, requirementName, targetNodeName)
		if err != nil {
			return
		}
		if requirementPath == "" {
			err = errors.Errorf("Unable to find a matching requirement for this relationship operation %q, source node %q, requirement name %q, target node %q", operationName, nodeName, requirementName, targetNodeName)
			return
		}
		requirementIndex = path.Base(requirementPath)
		return
	}
	err = errors.Errorf("operation %q doesn't follow the format <fully_qualified_operation_name>/<requirementIndex> or <fully_qualified_operation_name>/<requirementName>/<targetNodeName>", operationName)
	return
}

// GetOperationOutputForNode return a map with in index the instance number and in value the result of the output
// The "params" parameter is necessary to pass the path of the output
func GetOperationOutputForNode(kv *api.KV, deploymentID, nodeName, instanceName, interfaceName, operationName, outputName string) (string, error) {
	instancesPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName)

	output, _, err := kv.Get(filepath.Join(instancesPath, instanceName, "outputs", strings.ToLower(interfaceName), strings.ToLower(operationName), outputName), nil)
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if output != nil && len(output.Value) > 0 {
		return string(output.Value), nil
	}
	// Look at host node
	var host string
	host, err = GetHostedOnNode(kv, deploymentID, nodeName)
	if err != nil {
		return "", err
	}
	if host != "" {
		// TODO we consider that instance name is the same for the host but we should not
		return GetOperationOutputForNode(kv, deploymentID, host, instanceName, interfaceName, operationName, outputName)
	}
	return "", nil
}

// GetOperationOutputForRelationship retrieves an operation output for a relationship
// The returned value may be empty if the operation output could not be retrieved
func GetOperationOutputForRelationship(kv *api.KV, deploymentID, nodeName, instanceName, requirementIndex, interfaceName, operationName, outputName string) (string, error) {
	relationshipType, err := GetRelationshipForRequirement(kv, deploymentID, nodeName, requirementIndex)
	if err != nil {
		return "", err
	}
	node := nodeName
	if IsRelationshipOperationOnTargetNode(operationName) {
		node, err = GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return "", err
		}
	}
	result, _, err := kv.Get(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances", node, relationshipType, instanceName, "outputs", strings.ToLower(path.Join(interfaceName, operationName)), outputName), nil)
	if err != nil {
		return "", err
	}

	if result == nil || len(result.Value) == 0 {
		return "", nil
	}
	return string(result.Value), nil
}

func getOperationOutputForRequirements(kv *api.KV, deploymentID, nodeName, instanceName, interfaceName, operationName, outputName string) (string, error) {
	reqIndexes, err := GetRequirementsIndexes(kv, deploymentID, nodeName)
	if err != nil {
		return "", err
	}
	for _, reqIndex := range reqIndexes {
		result, err := GetOperationOutputForRelationship(kv, deploymentID, nodeName, instanceName, reqIndex, interfaceName, operationName, outputName)
		if err != nil || result != "" {
			return result, err
		}
	}
	return "", nil
}
