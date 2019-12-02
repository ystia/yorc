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
	"encoding/json"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tosca"
)

// SetInstanceStateStringWithContextualLogs stores the state of a given node instance and publishes a status change event
// context is used to carry contextual information for logging (see events package)
func SetInstanceStateStringWithContextualLogs(ctx context.Context, deploymentID, nodeName, instanceName, state string) error {
	err := consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "attributes/state"), state)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	_, err = events.PublishAndLogInstanceStatusChange(ctx, deploymentID, nodeName, instanceName, state)
	if err != nil {
		return err
	}
	return notifyAndPublishAttributeValueChange(ctx, deploymentID, nodeName, instanceName, "state", state)
}

// SetInstanceStateWithContextualLogs stores the state of a given node instance and publishes a status change event
// context is used to carry contextual information for logging (see events package)
func SetInstanceStateWithContextualLogs(ctx context.Context, deploymentID, nodeName, instanceName string, state tosca.NodeState) error {
	return SetInstanceStateStringWithContextualLogs(ctx, deploymentID, nodeName, instanceName, state.String())
}

// GetInstanceState retrieves the state of a given node instance
func GetInstanceState(ctx context.Context, deploymentID, nodeName, instanceName string) (tosca.NodeState, error) {
	stringStateValue, err := GetInstanceStateString(ctx, deploymentID, nodeName, instanceName)
	if err != nil {
		return tosca.NodeStateError, err
	}
	state, err := tosca.NodeStateString(stringStateValue)
	if err != nil {
		return tosca.NodeStateError, err
	}
	return state, nil
}

// GetInstanceStateString retrieves the string value of the state attribute of a given node instance
func GetInstanceStateString(ctx context.Context, deploymentID, nodeName, instanceName string) (string, error) {
	exist, value, err := consulutil.GetStringValue(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "attributes/state"))
	if err != nil {
		return "", errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if !exist || value == "" {
		return "", errors.Errorf("Missing mandatory attribute \"state\" on instance %q for node %q", instanceName, nodeName)
	}
	return value, nil
}

// DeleteInstance deletes the given instance of the given node from the Consul store
func DeleteInstance(ctx context.Context, deploymentID, nodeName, instanceName string) error {
	return consulutil.Delete(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName)+"/", true)
}

// DeleteAllInstances deletes all instances of the given node from the Consul store
func DeleteAllInstances(ctx context.Context, deploymentID, nodeName string) error {
	return consulutil.Delete(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName)+"/", true)
}

// LookupInstanceAttributeValue executes a lookup to retrieve instance attribute value when attribute can be long to retrieve
func LookupInstanceAttributeValue(ctx context.Context, deploymentID, nodeName, instanceName, attribute string, nestedKeys ...string) (string, error) {
	log.Debugf("Attribute:%q lookup for deploymentID:%q, node name:%q, instance:%q", attribute, deploymentID, nodeName, instanceName)
	res := make(chan string, 1)
	go func() {
		for {
			if attr, _ := GetInstanceAttributeValue(ctx, deploymentID, nodeName, instanceName, attribute, nestedKeys...); attr != nil && attr.RawString() != "" {
				if attr != nil && attr.RawString() != "" {
					res <- attr.RawString()
					return
				}
			}

			select {
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				return
			}
		}
	}()

	select {
	case val := <-res:
		return val, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// updateInstanceAttributeValue allows to update instance attribute value if possible
// it resolves instance attribute value skipping existing one
func updateInstanceAttributeValue(ctx context.Context, deploymentID, nodeName, instanceName, attributeName string, nestedKeys ...string) error {
	value, err := getInstanceAttributeValue(ctx, deploymentID, nodeName, instanceName, attributeName, true, nestedKeys...)
	if err != nil {
		return nil
	}
	if value != nil && strings.TrimSpace(value.String()) != "" {
		return SetInstanceAttribute(ctx, deploymentID, nodeName, instanceName, attributeName, value.String())
	}
	return nil
}

// GetInstanceAttributeValue retrieves the given attribute for a node instance
//
// It returns true if a value is found false otherwise as first return parameter.
// If the attribute is not found in the node then the type hierarchy is explored to find a default value.
// If the attribute is still not found then it will explore the HostedOn hierarchy.
// If still not found then it will check node properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
func GetInstanceAttributeValue(ctx context.Context, deploymentID, nodeName, instanceName, attributeName string, nestedKeys ...string) (*TOSCAValue, error) {
	return getInstanceAttributeValue(ctx, deploymentID, nodeName, instanceName, attributeName, false, nestedKeys...)
}

func getInstanceAttributeValue(ctx context.Context, deploymentID, nodeName, instanceName, attributeName string, skipInstanceLevel bool, nestedKeys ...string) (*TOSCAValue, error) {

	substitutionInstance, err := isSubstitutionNodeInstance(ctx, deploymentID, nodeName, instanceName)
	if err != nil {
		return nil, err
	}
	if isSubstitutionMappingAttribute(attributeName) {

		// Alien4Cloud did not yet implement the management of capability attributes
		// in substitution mappings. It is using node attributes names with a
		// capabilities prefix.
		// These attributes will be available :
		// - in the case of the source application providing this attribute,
		//   the call here should be redirected to the call getting
		//   instance capability attribute
		// - in the case of a substitutable node, the attribute is available in
		//   the node attributes (managed in a later block in this function)
		if !substitutionInstance {
			result, err := getSubstitutionMappingAttribute(ctx, deploymentID, nodeName, instanceName, attributeName, nestedKeys...)
			if err != nil || result != nil {
				return result, err
			}
		}
	}

	nodeType, err := GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}

	if !skipInstanceLevel {
		// First look at instance-scoped attributes
		// except if this is a substitutable node instance, in which case
		// attributes are stored at the node level
		if substitutionInstance {
			found, result := getSubstitutionInstanceAttribute(deploymentID, nodeName, instanceName, attributeName)
			if found {
				return &TOSCAValue{Value: result}, nil
			}
		} else {
			result, err := getInstanceValueAssignment(ctx, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "attributes", attributeName), nestedKeys...)
			if err != nil || result != nil {
				return result, errors.Wrapf(err, "Failed to get attribute %q for node %q (instance %q)",
					attributeName, nodeName, instanceName)
			}
		}
	}

	// Then look at global node level (not instance-scoped)
	result, err := getNodeAttributeValue(ctx, deploymentID, nodeName, instanceName, attributeName, nestedKeys...)
	if err != nil || result != nil {
		return result, errors.Wrapf(err, "Failed to get attribute %q for node: %q, instance:%q", attributeName, nodeName, instanceName)
	}

	// Not found look at node type
	defaultValue, isFunction, err := getTypeDefaultAttribute(ctx, deploymentID, nodeType, attributeName, nestedKeys...)
	if err != nil {
		return nil, err
	}
	if defaultValue != nil {
		if !isFunction {
			return defaultValue, nil
		}
		return resolveValueAssignment(ctx, deploymentID, nodeName, instanceName, "", defaultValue, nestedKeys...)
	}

	// No default found in type hierarchy
	// then traverse HostedOn relationships to find the value
	var host string
	host, err = GetHostedOnNode(ctx, deploymentID, nodeName)
	if err != nil {
		return nil, err
	}
	if host != "" {
		hostValue, err := GetInstanceAttributeValue(ctx, deploymentID, host, instanceName, attributeName, nestedKeys...)
		if hostValue != nil || err != nil {
			return hostValue, err
		}
	}

	// Now check properties as the spec states "TOSCA orchestrators will automatically reflect (i.e., make available) any property defined on an entity making it available as an attribute of the entity with the same name as the property."
	return GetNodePropertyValue(ctx, deploymentID, nodeName, attributeName, nestedKeys...)
}

// SetInstanceAttribute sets an instance attribute
func SetInstanceAttribute(ctx context.Context, deploymentID, nodeName, instanceName, attributeName, attributeValue string) error {
	return SetInstanceAttributeComplex(ctx, deploymentID, nodeName, instanceName, attributeName, attributeValue)

}

// SetInstanceListAttributes sets a list of instance attributes in order to store all attributes together then publish.
// This is done for avoiding inconsistency at publish time
func SetInstanceListAttributes(ctx context.Context, attributes []*AttributeData) error {
	return SetInstanceListAttributesComplex(ctx, attributes)

}

// SetInstanceAttributeComplex sets an instance attribute that may be a literal or a complex data type
func SetInstanceAttributeComplex(ctx context.Context, deploymentID, nodeName, instanceName, attributeName string, attributeValue interface{}) error {
	attrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "attributes", attributeName)
	data, err := json.Marshal(attributeValue)
	if err != nil {
		return err
	}

	err = consulutil.StoreConsulKey(attrPath, data)
	if err != nil {
		return err
	}

	err = notifyAndPublishAttributeValueChange(ctx, deploymentID, nodeName, instanceName, attributeName, attributeValue)
	if err != nil {
		return err
	}
	return nil
}

// SetInstanceListAttributesComplex sets an instance list of attributes that may be a literal or a complex data type
// All attributes are stored together
// Then all notifications are published
func SetInstanceListAttributesComplex(ctx context.Context, attributes []*AttributeData) error {
	for _, attribute := range attributes {
		attrPath := path.Join(consulutil.DeploymentKVPrefix, attribute.DeploymentID, "topology/instances", attribute.NodeName, attribute.InstanceName, "attributes", attribute.Name)
		data, err := json.Marshal(attribute.Value)
		if err != nil {
			return err
		}

		err = consulutil.StoreConsulKey(attrPath, data)
		if err != nil {
			return err
		}
	}
	for _, attribute := range attributes {
		err := notifyAndPublishAttributeValueChange(ctx, attribute.DeploymentID, attribute.NodeName, attribute.InstanceName, attribute.Name, attribute.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetAttributeForAllInstances sets the same attribute value to all instances of a given node.
//
// It does the same thing than iterating over instances ids and calling SetInstanceAttribute but use
// a consulutil.ConsulStore to do it in parallel. We can expect better performances with a large number of instances
func SetAttributeForAllInstances(ctx context.Context, deploymentID, nodeName, attributeName, attributeValue string) error {
	return SetAttributeComplexForAllInstances(ctx, deploymentID, nodeName, attributeName, attributeValue)
}

// SetAttributeComplexForAllInstances sets the same attribute value to all instances of a given node.
//
// It does the same thing than iterating over instances ids and calling SetInstanceAttributeComplex but use
// a consulutil.ConsulStore to do it in parallel. We can expect better performances with a large number of instances
func SetAttributeComplexForAllInstances(ctx context.Context, deploymentID, nodeName, attributeName string, attributeValue interface{}) error {
	ids, err := GetNodeInstancesIds(ctx, deploymentID, nodeName)
	if err != nil {
		return err
	}
	for _, instanceName := range ids {
		attrPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "attributes", attributeName)
		data, err := json.Marshal(attributeValue)
		if err != nil {
			return err
		}

		err = consulutil.StoreConsulKey(attrPath, data)
		if err != nil {
			return err
		}

		err = notifyAndPublishAttributeValueChange(ctx, deploymentID, nodeName, instanceName, attributeName, attributeValue)
		if err != nil {
			return err
		}
	}
	return nil
}

func notifyAndPublishAttributeValueChange(ctx context.Context, deploymentID, nodeName, instanceName, attributeName string, attributeValue interface{}) error {
	// First, Publish event
	sValue, ok := attributeValue.(string)
	if ok {
		_, err := events.PublishAndLogAttributeValueChange(context.Background(), deploymentID, nodeName, instanceName, attributeName, sValue, "updated")
		if err != nil {
			return err
		}
	}

	// Next, notify dependent attributes if existing
	an := &AttributeNotifier{
		NodeName:      nodeName,
		InstanceName:  instanceName,
		AttributeName: attributeName,
	}
	return an.NotifyValueChange(ctx, deploymentID)
}
