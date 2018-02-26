// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deployments

import (
	"context"
	"path"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/collections"
	"github.com/ystia/yorc/helper/consulutil"
)

// GetRelationshipPropertyFromRequirement returns the value of a relationship's property identified by a requirement index on a node
func GetRelationshipPropertyFromRequirement(kv *api.KV, deploymentID, nodeName, requirementIndex, propertyName string, nestedKeys ...string) (bool, string, error) {
	relationshipType, err := GetRelationshipForRequirement(kv, deploymentID, nodeName, requirementIndex)
	if err != nil {
		return false, "", err
	}

	var propDataType string
	var hasProp bool
	if relationshipType != "" {
		hasProp, err := TypeHasProperty(kv, deploymentID, relationshipType, propertyName, true)
		if err != nil {
			return false, "", err
		}
		if hasProp {
			propDataType, err = GetTypePropertyDataType(kv, deploymentID, relationshipType, propertyName)
			if err != nil {
				return false, "", err
			}
		}
	}
	reqPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "requirements", requirementIndex)

	found, result, err := getValueAssignmentWithDataType(kv, deploymentID, path.Join(reqPrefix, "properties", propertyName), nodeName, "", requirementIndex, propDataType, nestedKeys...)
	if err != nil {
		return false, "", errors.Wrapf(err, "Failed to get property %q for requirement %q on node %q", propertyName, requirementIndex, nodeName)
	}
	if found {
		return true, result, nil
	}

	// Look at the relationship type to find a default value
	if relationshipType != "" {
		found, result, isFunction, err := getTypeDefaultProperty(kv, deploymentID, relationshipType, propertyName, nestedKeys...)
		if err != nil {
			return false, "", err
		}
		if found {
			if !isFunction {
				return true, result, nil
			}
			result, err = resolveValueAssignmentAsString(kv, deploymentID, nodeName, "", requirementIndex, result, nestedKeys...)
			return true, result, err
		}
	}

	if hasProp && relationshipType != "" {
		// Check if the whole property is optional
		isRequired, err := IsTypePropertyRequired(kv, deploymentID, relationshipType, propertyName)
		if err != nil {
			return false, "", err
		}
		if !isRequired {
			// For backward compatibility
			// TODO this doesn't look as a good idea to me
			return true, "", nil
		}

		if len(nestedKeys) > 1 && propDataType != "" {
			// Check if nested type is optional
			nestedKeyType, err := GetNestedDataType(kv, deploymentID, propDataType, nestedKeys[:len(nestedKeys)-1]...)
			if err != nil {
				return false, "", err
			}
			isRequired, err = IsTypePropertyRequired(kv, deploymentID, nestedKeyType, nestedKeys[len(nestedKeys)-1])
			if err != nil {
				return false, "", err
			}
			if !isRequired {
				// For backward compatibility
				// TODO this doesn't look as a good idea to me
				return true, "", nil
			}
		}
	}
	return false, "", nil
}

// GetRelationshipAttributeFromRequirement retrieves the value for a given attribute in a given node instance requirement
//
// It returns true if a value is found false otherwise as first return parameter.
// If the attribute is not found in the node then the type hierarchy is explored to find a default value.
// If still not found check properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
func GetRelationshipAttributeFromRequirement(kv *api.KV, deploymentID, nodeName, instanceName, requirementIndex, attributeName string, nestedKeys ...string) (bool, string, error) {
	relationshipType, err := GetRelationshipForRequirement(kv, deploymentID, nodeName, requirementIndex)
	if err != nil {
		return false, "", err
	}

	var attrDataType string
	if relationshipType != "" {
		hasProp, err := TypeHasAttribute(kv, deploymentID, relationshipType, attributeName, true)
		if err != nil {
			return false, "", err
		}
		if hasProp {
			attrDataType, err = GetTypeAttributeDataType(kv, deploymentID, relationshipType, attributeName)
			if err != nil {
				return false, "", err
			}
		}
	}

	// First look at instance scoped attributes
	capAttrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances", nodeName, requirementIndex, instanceName, "attributes", attributeName)
	found, result, err := getValueAssignmentWithDataType(kv, deploymentID, capAttrPath, nodeName, instanceName, requirementIndex, attrDataType, nestedKeys...)
	if err != nil || found {
		// If there is an error or attribute was found
		return found, result, errors.Wrapf(err, "Failed to get attribute %q for requirement index %q on node %q (instance %q)", attributeName, requirementIndex, nodeName, instanceName)
	}
	// Now look at relationship type for default
	found, result, isFunction, err := getTypeDefaultAttribute(kv, deploymentID, relationshipType, attributeName, nestedKeys...)
	if err != nil {
		return false, "", err
	}
	if found {
		if !isFunction {
			return true, result, nil
		}
		result, err = resolveValueAssignmentAsString(kv, deploymentID, nodeName, instanceName, requirementIndex, result, nestedKeys...)
		return true, result, err
	}

	// If still not found check properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
	return GetRelationshipPropertyFromRequirement(kv, deploymentID, nodeName, requirementIndex, attributeName, nestedKeys...)
}

// SetInstanceRelationshipAttribute sets a relationship attribute for a given node instance
func SetInstanceRelationshipAttribute(deploymentID, nodeName, instanceName, requirementIndex, attributeName, value string) error {
	return SetInstanceRelationshipAttributeComplex(deploymentID, nodeName, instanceName, requirementIndex, attributeName, value)
}

// SetInstanceRelationshipAttributeComplex sets an instance relationship attribute that may be a literal or a complex data type
func SetInstanceRelationshipAttributeComplex(deploymentID, nodeName, instanceName, requirementIndex, attributeName string, attributeValue interface{}) error {
	attrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances", nodeName, requirementIndex, instanceName, "attributes", attributeName)
	_, errGrp, store := consulutil.WithContext(context.Background())
	storeComplexType(store, attrPath, attributeValue)
	return errGrp.Wait()
}

// SetRelationshipAttributeForAllInstances sets the same relationship attribute value to all instances of a given node.
//
// It does the same thing than iterating over instances ids and calling SetInstanceRelationshipAttribute but use
// a consulutil.ConsulStore to do it in parallel. We can expect better performances with a large number of instances
func SetRelationshipAttributeForAllInstances(kv *api.KV, deploymentID, nodeName, requirementIndex, attributeName, attributeValue string) error {
	return SetRelationshipAttributeComplexForAllInstances(kv, deploymentID, nodeName, requirementIndex, attributeName, attributeValue)
}

// SetRelationshipAttributeComplexForAllInstances sets the same relationship attribute value  that may be a literal or a complex data type to all instances of a given node.
//
// It does the same thing than iterating over instances ids and calling SetInstanceRelationshipAttributeComplex but use
// a consulutil.ConsulStore to do it in parallel. We can expect better performances with a large number of instances
func SetRelationshipAttributeComplexForAllInstances(kv *api.KV, deploymentID, nodeName, requirementIndex, attributeName string, attributeValue interface{}) error {
	ids, err := GetNodeInstancesIds(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	_, errGrp, store := consulutil.WithContext(context.Background())
	for _, instanceName := range ids {
		attrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances", nodeName, requirementIndex, instanceName, "attributes", attributeName)
		storeComplexType(store, attrPath, attributeValue)
	}
	return errGrp.Wait()
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
			consulStore.StoreConsulKeyAsString(path.Join(relInstancePath, nodeName, req, instanceID, "target/name"), targetName)
			consulStore.StoreConsulKeyAsString(path.Join(relInstancePath, nodeName, req, instanceID, "target/instances"), targetInstanceIdsString)
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
	reqIndices, _, err := kv.Keys(nodeRelInstancePath+"/", "/", nil)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, reqindex := range reqIndices {
		_, err := kv.DeleteTree(path.Join(reqindex, instanceName), nil)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
	}

	// now delete from targets in relationships instances
	addOrRemoveInstanceFromTargetRelationship(kv, deploymentID, nodeName, instanceName, false)

	return nil
}
