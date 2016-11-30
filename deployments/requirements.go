package deployments

import (
	"fmt"
	"path"
	"sort"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

// GetRequirementsKeysByNameForNode returns paths to requirements whose names matches the given requirementName.
//
// The returned slice may be empty if there is no matching requirements.
func GetRequirementsKeysByNameForNode(kv *api.KV, deploymentID, nodeName, requirementName string) ([]string, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
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
			return nil, fmt.Errorf("Missing mandatory parameter \"name\" for requirement at index %q for node %q deployment %q", path.Base(reqIndexKey), nodeName, deploymentID)
		}
		if string(kvp.Value) == requirementName {
			reqKeys = append(reqKeys, reqIndexKey)
		}
	}
	sort.Strings(reqKeys)
	return reqKeys, nil
}

// GetNbRequirementsForNode returns the number of requirements declared for the given node
func GetNbRequirementsForNode(kv *api.KV, deploymentID, nodeName string) (int, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	reqKVPs, _, err := kv.Keys(path.Join(nodePath, "requirements")+"/", "/", nil)
	if err != nil {
		return 0, errors.Wrapf(err, "Failed to retrieve requirements for node %q", nodeName)
	}
	return len(reqKVPs), nil
}

// GetRequirementByNameAndTargetForNode returns path to requirement which names matches the given requirementName and node matches the given targetName.
//
// The returned string may be empty if there is no matching requirements.
func GetRequirementByNameAndTargetForNode(kv *api.KV, deploymentID, nodeName, requirementName, targetName string) (string, error) {
	nodePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "nodes", nodeName)
	reqKVPs, _, err := kv.Keys(path.Join(nodePath, "requirements")+"/", "/", nil)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get requirement index for node %q, requirement %q, target node %q", nodeName, requirementName, targetName)
	}
	for _, reqIndexKey := range reqKVPs {
		reqIndexKey = path.Clean(reqIndexKey)
		kvp, _, err := kv.Get(path.Join(reqIndexKey, "name"), nil)
		if err != nil {
			return "", errors.Wrapf(err, "Failed to get requirement index for node %q, requirement %q, target node %q", nodeName, requirementName, targetName)
		}
		if kvp == nil || len(kvp.Value) == 0 {
			return "", errors.Errorf("Missing mandatory parameter \"name\" for requirement at index %q for node %q deployment %q", path.Base(reqIndexKey), nodeName, deploymentID)
		}
		if string(kvp.Value) != requirementName {
			continue
		}
		kvp, _, err = kv.Get(path.Join(reqIndexKey, "node"), nil)
		if err != nil {
			return "", errors.Wrapf(err, "Failed to get requirement index for node %q, requirement %q, target node %q", nodeName, requirementName, targetName)
		}
		if kvp == nil || len(kvp.Value) == 0 {
			return "", errors.Errorf("Missing mandatory parameter \"name\" for requirement at index %q for node %q deployment %q", path.Base(reqIndexKey), nodeName, deploymentID)
		}
		if string(kvp.Value) == targetName {
			// Found
			return reqIndexKey, nil
		}
	}
	// Not found
	return "", nil
}
