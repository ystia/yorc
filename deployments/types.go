package deployments

import (
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

// IsNodeTypeDerivedFrom traverses 'derived_from' to check if type derives from another type
func IsNodeTypeDerivedFrom(kv *api.KV, deploymentID, nodeType, derives string) (bool, error) {
	if nodeType == derives {
		return true, nil
	}
	parent, err := GetParentNodeType(kv, deploymentID, nodeType)
	if err != nil || parent == "" {
		return false, err
	}
	return IsNodeTypeDerivedFrom(kv, deploymentID, parent, derives)
}

// GetParentNodeType looks at the 'derived_from' from property of a node type to retrieve its parent.
//
// In case of a root type an empty string is returned.
func GetParentNodeType(kv *api.KV, deploymentID, nodeType string) (string, error) {
	nodeTypePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "types", nodeType)
	// Check if node type exist
	if kvp, _, err := kv.Get(nodeTypePath+"/name", nil); err != nil {
		return "", errors.Wrapf(err, "Failed to get parent type for node type %q", nodeType)
	} else if kvp == nil || len(kvp.Value) == 0 {
		return "", errors.Errorf("Looking for a node type %q that do not exists.", nodeType)
	}

	kvp, _, err := kv.Get(nodeTypePath+"/derived_from", nil)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get parent type for node type %q", nodeType)
	}
	if kvp == nil || len(kvp.Value) == 0 {
		// This is a root type
		return "", nil
	}
	return string(kvp.Value), nil
}
