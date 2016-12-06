package deployments

import (
	"strings"

	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
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
	} else {
		op = strings.TrimPrefix(operationName, "tosca.interfaces.node.lifecycle.")
		op = strings.TrimPrefix(op, "tosca.interfaces.relationship.")
		op = strings.TrimPrefix(op, "tosca.interfaces.node.lifecycle.")
		op = strings.TrimPrefix(op, "tosca.interfaces.relationship.")
	}
	op = strings.Replace(op, ".", "/", -1)
	operationPath := path.Join(DeploymentKVPrefix, deploymentID, "topology/types", nodeType, "interfaces", op)
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
