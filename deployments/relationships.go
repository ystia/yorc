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
	"fmt"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/helper/consulutil"
)

// GetRelationshipPropertyValueFromRequirement returns the value of a relationship's property identified by a requirement index on a node
func GetRelationshipPropertyValueFromRequirement(ctx context.Context, deploymentID, nodeName, requirementIndex, propertyName string, nestedKeys ...string) (*TOSCAValue, error) {
	relationshipType, err := GetRelationshipForRequirement(ctx, deploymentID, nodeName, requirementIndex)
	if err != nil {
		return nil, err
	}

	var propDataType string
	var hasProp bool
	if relationshipType != "" {
		hasProp, err := TypeHasProperty(ctx, deploymentID, relationshipType, propertyName, true)
		if err != nil {
			return nil, err
		}
		if hasProp {
			propDataType, err = GetTypePropertyDataType(ctx, deploymentID, relationshipType, propertyName)
			if err != nil {
				return nil, err
			}
		}
	}

	// Look at requirement level
	_, req, err := getRequirementByIndex(ctx, deploymentID, nodeName, requirementIndex)
	if err != nil {
		return nil, err
	}
	if req != nil {
		va, is := req.RelationshipProps[propertyName]
		if is && va != nil {
			result, err := getValueAssignment(ctx, deploymentID, nodeName, "", requirementIndex, propDataType, va, nestedKeys...)
			if err != nil || result != nil {
				return result, errors.Wrapf(err, "Failed to get property %q for requirement %q on node %q", propertyName, requirementIndex, nodeName)
			}
		}
	}

	// Look at the relationship type to find a default value
	if relationshipType != "" {
		result, isFunction, err := getTypeDefaultProperty(ctx, deploymentID, relationshipType, propertyName, nestedKeys...)
		if err != nil {
			return nil, err
		}
		if result != nil {
			if !isFunction {
				return result, nil
			}
			return resolveValueAssignment(ctx, deploymentID, nodeName, "", requirementIndex, result, nestedKeys...)
		}
	}

	if hasProp && relationshipType != "" {
		// Check if the whole property is optional
		isRequired, err := IsTypePropertyRequired(ctx, deploymentID, relationshipType, propertyName)
		if err != nil {
			return nil, err
		}
		if !isRequired {
			// For backward compatibility
			// TODO this doesn't look as a good idea to me
			return &TOSCAValue{Value: ""}, nil
		}

		if len(nestedKeys) > 1 && propDataType != "" {
			// Check if nested type is optional
			nestedKeyType, err := GetNestedDataType(ctx, deploymentID, propDataType, nestedKeys[:len(nestedKeys)-1]...)
			if err != nil {
				return nil, err
			}
			isRequired, err = IsTypePropertyRequired(ctx, deploymentID, nestedKeyType, nestedKeys[len(nestedKeys)-1])
			if err != nil {
				return nil, err
			}
			if !isRequired {
				// For backward compatibility
				// TODO this doesn't look as a good idea to me
				return &TOSCAValue{Value: ""}, nil
			}
		}
	}
	return nil, nil
}

// GetRelationshipAttributeValueFromRequirement retrieves the value for a given attribute in a given node instance requirement
//
// It returns true if a value is found false otherwise as first return parameter.
// If the attribute is not found in the node then the type hierarchy is explored to find a default value.
// If still not found check properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
func GetRelationshipAttributeValueFromRequirement(ctx context.Context, deploymentID, nodeName, instanceName, requirementIndex, attributeName string, nestedKeys ...string) (*TOSCAValue, error) {
	relationshipType, err := GetRelationshipForRequirement(ctx, deploymentID, nodeName, requirementIndex)
	if err != nil {
		return nil, err
	}

	// First look at instance scoped attributes
	capAttrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances", nodeName, requirementIndex, instanceName, "attributes", attributeName)
	result, err := getInstanceValueAssignment(ctx, capAttrPath, nestedKeys...)
	if err != nil || result != nil {
		// If there is an error or attribute was found
		return result, errors.Wrapf(err, "Failed to get attribute %q for requirement index %q on node %q (instance %q)", attributeName, requirementIndex, nodeName, instanceName)
	}
	// Now look at relationship type for default
	if relationshipType != "" {
		result, isFunction, err := getTypeDefaultAttribute(ctx, deploymentID, relationshipType, attributeName, nestedKeys...)
		if err != nil {
			return nil, err
		}
		if result != nil {
			if !isFunction {
				return result, nil
			}
			return resolveValueAssignment(ctx, deploymentID, nodeName, instanceName, requirementIndex, result, nestedKeys...)
		}
	}
	// If still not found check properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
	return GetRelationshipPropertyValueFromRequirement(ctx, deploymentID, nodeName, requirementIndex, attributeName, nestedKeys...)
}

// SetInstanceRelationshipAttribute sets a relationship attribute for a given node instance
func SetInstanceRelationshipAttribute(ctx context.Context, deploymentID, nodeName, instanceName, requirementIndex, attributeName, value string) error {
	return SetInstanceRelationshipAttributeComplex(ctx, deploymentID, nodeName, instanceName, requirementIndex, attributeName, value)
}

// SetInstanceRelationshipAttributeComplex sets an instance relationship attribute that may be a literal or a complex data type
func SetInstanceRelationshipAttributeComplex(ctx context.Context, deploymentID, nodeName, instanceName, requirementIndex, attributeName string, attributeValue interface{}) error {
	attrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances", nodeName, requirementIndex, instanceName, "attributes", attributeName)
	err := storage.GetStore(types.StoreTypeDeployment).Set(attrPath, attributeValue)
	if err != nil {
		return err
	}
	err = publishRelationshipAttributeValueChange(ctx, deploymentID, nodeName, instanceName, requirementIndex, attributeName, attributeValue)
	if err != nil {
		return err
	}

	return nil
}

// SetRelationshipAttributeForAllInstances sets the same relationship attribute value to all instances of a given node.
//
// It does the same thing than iterating over instances ids and calling SetInstanceRelationshipAttribute but use
// a consulutil.ConsulStore to do it in parallel. We can expect better performances with a large number of instances
func SetRelationshipAttributeForAllInstances(ctx context.Context, deploymentID, nodeName, requirementIndex, attributeName, attributeValue string) error {
	return SetRelationshipAttributeComplexForAllInstances(ctx, deploymentID, nodeName, requirementIndex, attributeName, attributeValue)
}

// SetRelationshipAttributeComplexForAllInstances sets the same relationship attribute value  that may be a literal or a complex data type to all instances of a given node.
//
// It does the same thing than iterating over instances ids and calling SetInstanceRelationshipAttributeComplex but use
// a consulutil.ConsulStore to do it in parallel. We can expect better performances with a large number of instances
func SetRelationshipAttributeComplexForAllInstances(ctx context.Context, deploymentID, nodeName, requirementIndex, attributeName string, attributeValue interface{}) error {
	ids, err := GetNodeInstancesIds(ctx, deploymentID, nodeName)
	if err != nil {
		return err
	}
	for _, instanceName := range ids {
		attrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances", nodeName, requirementIndex, instanceName, "attributes", attributeName)
		err := storage.GetStore(types.StoreTypeDeployment).Set(attrPath, attributeValue)
		if err != nil {
			return err
		}
		err = publishRelationshipAttributeValueChange(ctx, deploymentID, nodeName, instanceName, requirementIndex, attributeName, attributeValue)
		if err != nil {
			return err
		}
	}
	return nil
}

// This function create an instance of each relationship and reference who is the target and the instanceID of this one
func createRelationshipInstances(ctx context.Context, consulStore consulutil.ConsulStore, deploymentID, nodeName string) error {
	relInstancePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances")
	reqKeys, err := GetRequirementsIndexes(ctx, deploymentID, nodeName)
	if err != nil {
		return err
	}
	nodeInstanceIds, err := GetNodeInstancesIds(ctx, deploymentID, nodeName)
	if err != nil {
		return err
	}
	for _, req := range reqKeys {
		reqType, err := GetRelationshipForRequirement(ctx, deploymentID, nodeName, req)
		if err != nil {
			return err
		}

		if reqType == "" {
			continue
		}

		targetName, err := GetTargetNodeForRequirement(ctx, deploymentID, nodeName, req)
		if err != nil {
			return err
		}

		// TODO for now we consider only relationships for every source instances to every target instances
		targetInstanceIds, err := GetNodeInstancesIds(ctx, deploymentID, targetName)
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

func addOrRemoveInstanceFromTargetRelationship(ctx context.Context, deploymentID, nodeName, instanceName string, add bool) error {
	relInstancePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances")
	// Appending a final "/" here is not necessary has there is no other keys starting with "relationship_instances" prefix
	relInstKVPairs, err := consulutil.List(relInstancePath)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	_, errGrp, store := consulutil.WithContext(context.Background())
	for key, value := range relInstKVPairs {
		if strings.HasSuffix(key, "target/name") {
			if string(value) == nodeName {
				instPath := path.Join(key, "../instances")
				exist, value, err := consulutil.GetStringValue(instPath)
				if err != nil {
					return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
				}
				if !exist || value == "" {
					return errors.Errorf("Missing key %q", instPath)
				}
				instances := strings.Split(value, ",")
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
				store.StoreConsulKeyAsString(instPath, strings.Join(instances, ","))
			}
		}
	}
	return errGrp.Wait()
}

// DeleteRelationshipInstance deletes the instance from relationship instances stored in consul
func DeleteRelationshipInstance(ctx context.Context, deploymentID, nodeName, instanceName string) error {
	relInstancePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/relationship_instances")
	nodeRelInstancePath := path.Join(relInstancePath, nodeName)
	reqIndices, err := consulutil.GetKeys(nodeRelInstancePath)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, reqindex := range reqIndices {
		err = consulutil.Delete(path.Join(reqindex, instanceName)+"/", true)
		if err != nil {
			return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
	}

	// now delete from targets in relationships instances
	return addOrRemoveInstanceFromTargetRelationship(ctx, deploymentID, nodeName, instanceName, false)
}

func publishRelationshipAttributeValueChange(ctx context.Context, deploymentID, nodeName, instanceName, requirementIndex, attributeName string, attributeValue interface{}) error {
	sValue, ok := attributeValue.(string)
	if ok {
		// Publish the relationship attribute with the related requirement name
		requirementName, err := GetRequirementNameByIndexForNode(ctx, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return err
		}
		relationshipAttribute := fmt.Sprintf("relationship.%s.%s", requirementName, attributeName)
		_, err = events.PublishAndLogAttributeValueChange(context.Background(), deploymentID, nodeName, instanceName, relationshipAttribute, sValue, "updated")
		if err != nil {
			return err
		}
	}
	return nil
}
