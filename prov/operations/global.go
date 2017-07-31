package operations

import (
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/provutil"
	"strings"
)

func GetInstanceName(nodeName, instanceID string) string {
	return provutil.SanitizeForShell(nodeName + "_" + instanceID)
}

// IsOperationNotImplemented checks if a given error is an error indicating that an operation is not implemented
func IsOperationNotImplemented(err error) bool {
	_, ok := err.(operationNotImplemented)
	return ok
}

type operationNotImplemented struct {
	Msg string
}

func (oni operationNotImplemented) Error() string {
	return oni.Msg
}

// ResolveIsPerInstanceOperation sets e.isPerInstanceOperation to true if the given operationName contains one of the following patterns (case doesn't matter):
//	add_target, remove_target, add_source, target_changed
// And in case of a relationship operation the relationship does not derive from "tosca.relationships.HostedOn" as it makes no sense till we scale at compute level
func ResolveIsPerInstanceOperation(operationName, deploymentID, relationshipType string, kv *api.KV) (bool, error) {
	op := strings.ToLower(operationName)
	if strings.Contains(op, "add_target") || strings.Contains(op, "remove_target") || strings.Contains(op, "target_changed") || strings.Contains(op, "add_source") {
		// Do not call the call the operation several time for an HostedOn relationship (makes no sense till we scale at compute level)
		if hostedOn, err := deployments.IsTypeDerivedFrom(kv, deploymentID, relationshipType, "tosca.relationships.HostedOn"); err != nil || hostedOn {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// IsTargetOperation returns true if the given operationName contains one of the following patterns (case doesn't matter):
//	pre_configure_target, post_configure_target, add_source
func IsTargetOperation(operationName string) bool {
	op := strings.ToLower(operationName)
	if strings.Contains(op, "pre_configure_target") || strings.Contains(op, "post_configure_target") || strings.Contains(op, "add_source") {
		return true
	}
	return false
}
