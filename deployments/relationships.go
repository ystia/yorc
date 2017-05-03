package deployments

import (
	"path"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/collections"
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

// This function create an instance of each relationship and reference who is the target and the instanceID of this one
func createRelationshipInstances(consulStore consulutil.ConsulStore, kv *api.KV, deploymentID, nodeName string) error {
	relInstancePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances")
	reqKeys, err := GetRequirementsIndexes(kv, deploymentID, nodeName)
	nodeInstanceIds, err := GetNodeInstancesIds(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	for _, req := range reqKeys {
		reqType, err := GetRelationshipForRequirement(kv, deploymentID, nodeName, req)
		if err != nil {
			return err
		}

		if reqType == "" {
			continue
		}

		targetName, err := GetTargetNodeForRequirement(kv, deploymentID, nodeName, req)
		if err != nil {
			return err
		}

		// TODO for now we consider only relationships for every source instances to every target instances
		targetInstanceIds, err := GetNodeInstancesIds(kv, deploymentID, targetName)
		if err != nil {
			return err
		}
		targetInstanceIdsString := strings.Join(targetInstanceIds, ",")
		for _, instanceID := range nodeInstanceIds {
			consulStore.StoreConsulKeyAsString(path.Join(relInstancePath, nodeName, reqType, instanceID, "target/name"), targetName)
			consulStore.StoreConsulKeyAsString(path.Join(relInstancePath, nodeName, reqType, instanceID, "target/instances"), targetInstanceIdsString)
		}
	}
	return nil
}

func addOrRemoveInstanceFromTargetRelationship(kv *api.KV, deploymentID, nodeName, instanceName string, add bool) error {
	relInstancePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances")
	relInstKVPairs, _, err := kv.List(relInstancePath, nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, relInstKVPair := range relInstKVPairs {
		if strings.HasSuffix(relInstKVPair.Key, "target/name") {
			if string(relInstKVPair.Value) == nodeName {
				instPath := path.Join(relInstKVPair.Key, "../instances")
				kvp, _, err := kv.Get(instPath, nil)
				if err != nil {
					return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
				}
				if kvp.Value == nil {
					return errors.Errorf("Missing key %q", instPath)
				}
				instances := strings.Split(string(kvp.Value), ",")
				if add {
					// TODO for now we consider only relationships for every source instances to every target instances
					if !collections.ContainsString(instances, instanceName) {
						instances = append(instances, instanceName)
					}
				} else {
					newInstances := instances[:0]
					for i := range instances {
						if instances[i] != instanceName {
							newInstances = append(newInstances, instances[i])
						}
					}
					instances = newInstances
				}
				kvp.Value = []byte(strings.Join(instances, ","))
				_, err = kv.Put(kvp, nil)
				if err != nil {
					return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
				}

			}
		}
	}
	return nil
}

// DeleteRelationshipInstance deletes the instance from relationship instances stored in consul
func DeleteRelationshipInstance(kv *api.KV, deploymentID, nodeName, instanceName string) error {
	relInstancePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances")
	nodeRelInstancePath := path.Join(relInstancePath, nodeName)
	reqTypes, _, err := kv.Keys(nodeRelInstancePath+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, reqType := range reqTypes {
		_, err := kv.DeleteTree(path.Join(reqType, instanceName), nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
	}

	// now delete from targets in relationships instances
	addOrRemoveInstanceFromTargetRelationship(kv, deploymentID, nodeName, instanceName, false)

	return nil
}
