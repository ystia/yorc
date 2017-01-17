package deployments

import (
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

// GetRelationshipPropertyFromRequirement returns the value of a relationship's property identified by a requirement index on a node
func GetRelationshipPropertyFromRequirement(kv *api.KV, deploymentID, nodeName, requirementIndex, propertyName string) (bool, string, error) {
	reqPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements", requirementIndex)

	kvp, _, err := kv.Get(path.Join(reqPrefix, "properties", propertyName), nil)
	if err != nil {
		return false, "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if kvp != nil {
		return true, string(kvp.Value), nil
	}

	// Look at the relationship type to find a default value
	kvp, _, err = kv.Get(path.Join(reqPrefix, "relationship"), nil)
	if err != nil {
		return false, "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}

	if kvp != nil && len(kvp.Value) > 0 {

		return GetTypeDefaultProperty(kv, deploymentID, string(kvp.Value), propertyName)
	}
	return false, "", nil
}
