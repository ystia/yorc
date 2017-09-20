package operations

import (
	"novaforge.bull.com/starlings-janus/janus/helper/provutil"
)

func GetInstanceName(nodeName, instanceID string) string {
	return provutil.SanitizeForShell(nodeName + "_" + instanceID)
}
