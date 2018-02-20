package operations

import (
	"github.com/ystia/yorc/helper/provutil"
)

// GetInstanceName returns the built instance name from nodeName and instanceID
func GetInstanceName(nodeName, instanceID string) string {
	return provutil.SanitizeForShell(nodeName + "_" + instanceID)
}
