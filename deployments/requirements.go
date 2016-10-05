package deployments

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"path"
	"sort"
)

func GetRequirementsKeysByNameForNode(kv *api.KV, deploymentId, nodeName, requirementName string) ([]string, error) {
	nodePath := path.Join(DeploymentKVPrefix, deploymentId, "topology", "nodes", nodeName)
	reqKVPs, _, err := kv.Keys(path.Join(nodePath, "requirements")+"/", "/", nil)
	reqKeys := make([]string, 0)
	if err != nil {
		return nil, err
	}
	for _, reqIndexKey := range reqKVPs {
		reqIndexKey = path.Clean(reqIndexKey)
		kvp, _, err := kv.Get(path.Join(reqIndexKey, "name"), nil)
		if err != nil {
			return nil, err
		}
		if kvp == nil || len(kvp.Value) == 0 {
			return nil, fmt.Errorf("Missing mandatory parameter \"name\" for requirement at index %q for node %q deployment %q", path.Base(reqIndexKey), nodeName, deploymentId)
		}
		if string(kvp.Value) == requirementName {
			reqKeys = append(reqKeys, reqIndexKey)
		}
	}
	sort.Strings(reqKeys)
	return reqKeys, nil
}
