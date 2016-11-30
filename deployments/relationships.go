package deployments

import (
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"path"
)

func GetRelationshipPropertyFromRequirement(kv *api.KV, deploymentId, nodeName, requirementIndex, propertyName string) (bool, string, error) {
	reqPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeName, "requirements", requirementIndex)

	kvp, _, err := kv.Get(path.Join(reqPrefix, "properties", propertyName), nil)
	if err != nil {
		return false, "", err
	}
	if kvp != nil {
		return true, string(kvp.Value), nil
	}

	// Look at the relationship type to find a default value
	kvp, _, err = kv.Get(path.Join(reqPrefix, "relationship"), nil)
	if err != nil {
		return false, "", err
	}

	if kvp != nil && len(kvp.Value) > 0 {

		return GetTypeDefaultProperty(kv, deploymentId, string(kvp.Value), propertyName)
	}
	return false, "", nil
}
