package operations

import (
	"novaforge.bull.com/starlings-janus/janus/helper/provutil"
)

// GetInstanceName returns the built instance name from nodeName and instanceID
func GetInstanceName(nodeName, instanceID string) string {
	return provutil.SanitizeForShell(nodeName + "_" + instanceID)
}
