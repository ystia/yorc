package deployments

import (
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

// GetParentType returns the direct parent type of a given type using the 'derived_from' attributes
//
// An empty string denotes a root type
func GetParentType(kv *api.KV, deploymentID, typeName string) (string, error) {
	typePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", typeName)
	// Check if node type exist
	if kvps, _, err := kv.List(typePath+"/", nil); err != nil {
		return "", errors.Wrap(err, "Consul access error: ")
	} else if kvps == nil || len(kvps) == 0 {
		return "", errors.Errorf("Looking for a type %q that do not exists in deployment %q.", typeName, deploymentID)
	}

	kvp, _, err := kv.Get(path.Join(typePath, "derived_from"), nil)
	if err != nil {
		return "", errors.Wrap(err, "Consul access error: ")
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", nil
	}
	return string(kvp.Value), nil
}

// IsNodeTypeDerivedFrom traverses 'derived_from' to check if type derives from another type
func IsNodeTypeDerivedFrom(kv *api.KV, deploymentID, nodeType, derives string) (bool, error) {
	if nodeType == derives {
		return true, nil
	}
	parent, err := GetParentType(kv, deploymentID, nodeType)
	if err != nil || parent == "" {
		return false, err
	}
	return IsNodeTypeDerivedFrom(kv, deploymentID, parent, derives)
}
