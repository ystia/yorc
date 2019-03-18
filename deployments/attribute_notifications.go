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
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v3/events"
	"github.com/ystia/yorc/v3/helper/collections"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/tosca"
	"gopkg.in/yaml.v2"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// AttributeData represents the related attribute data
type AttributeData struct {
	DeploymentID     string
	NodeName         string
	InstanceName     string
	Name             string
	Value            string
	CapabilityName   string
	RequirementIndex string
}

// Notifier represents the action of notify it's value change
type Notifier interface {
	NotifyValueChange(kv *api.KV, deploymentID string) error
}

// AttributeNotifier is an attribute notifying its value changes
type AttributeNotifier struct {
	NodeName       string
	InstanceName   string
	AttributeName  string
	CapabilityName string
}

// OperationOutputNotifier is an operation output notifying its value changes
type OperationOutputNotifier struct {
	NodeName      string
	InstanceName  string
	InterfaceName string
	OperationName string
	OutputName    string
}

type attributeNotificationsHandler struct {
	kv           *api.KV
	consulStore  consulutil.ConsulStore
	deploymentID string
	attribute    *notifiedAttribute
}

type notifiedAttribute struct {
	nodeName       string
	instanceName   string
	attributeName  string
	capabilityName string
}

// BuildAttributeDataFromPath allows to return attribute data from path as below:
// - instance attribute:     _yorc/deployments/<DEPLOYMENT_ID>/topology/instances/<NODE_NAME>/<INSTANCE_NAME>/attributes/<ATTRIBUTE_NAME>
// - capability attribute:   _yorc/deployments/<DEPLOYMENT_ID>/topology/instances/<NODE_NAME>/<INSTANCE_NAME>/capabilities/(/*)*/attributes/<ATTRIBUTE_NAME>
// - relationship attribute: _yorc/deployments/<DEPLOYMENT_ID>/topology/relationship_instances/<NODE_NAME>/<REQUIREMENT_INDEX>/<INSTANCE_NAME>/attributes/<ATTRIBUTE_NAME>
func BuildAttributeDataFromPath(aPath string) (*AttributeData, error) {
	// Find instance attribute path
	match := regexp.MustCompile(consulutil.DeploymentKVPrefix + "/([0-9a-zA-Z-_]+)/topology/instances/([0-9a-zA-Z-_]+)/([0-9a-zA-Z-]*)/attributes/(\\w+)").FindStringSubmatch(aPath)
	if match != nil && len(match) == 5 {
		return &AttributeData{
			DeploymentID: match[1],
			NodeName:     match[2],
			InstanceName: match[3],
			Name:         match[4],
		}, nil
	}

	// Find capabilities instance attribute path
	match = regexp.MustCompile(consulutil.DeploymentKVPrefix + "/([0-9a-zA-Z-_]+)/topology/instances/([0-9a-zA-Z-_]+)/([0-9a-zA-Z-]*)/capabilities/([/0-9a-zA-Z]+)/attributes/(\\w+)").FindStringSubmatch(aPath)
	if match != nil && len(match) == 6 {
		return &AttributeData{
			DeploymentID:   match[1],
			NodeName:       match[2],
			InstanceName:   match[3],
			CapabilityName: match[4],
			Name:           match[5],
		}, nil
	}

	// Find relationship instance attribute path
	match = regexp.MustCompile(consulutil.DeploymentKVPrefix + "/([0-9a-zA-Z-_]+)/topology/relationship_instances/([0-9a-zA-Z-_]+)/([0-9a-zA-Z-]+)/([/0-9a-zA-Z]*)/attributes/(\\w+)").FindStringSubmatch(aPath)
	if match != nil && len(match) == 6 {
		return &AttributeData{
			DeploymentID:     match[1],
			NodeName:         match[2],
			RequirementIndex: match[3],
			InstanceName:     match[4],
			Name:             match[5],
		}, nil
	}
	return nil, errors.Errorf("failed to build attribute data from path:%q", aPath)
}

// NotifyValueChange allows to notify output value change
func (oon *OperationOutputNotifier) NotifyValueChange(kv *api.KV, deploymentID string) error {
	log.Debugf("Received operation output value change notification for [deploymentID:%q, nodeName:%q, instanceName:%q, interfaceName:%q, operationName:%q, outputName:%q", deploymentID, oon.NodeName, oon.InstanceName, oon.InterfaceName, oon.OperationName, oon.OutputName)
	notificationsPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", oon.NodeName, oon.InstanceName, "outputs", oon.InterfaceName, oon.OperationName, "attribute_notifications", oon.OutputName)
	return notifyAttributeOnValueChange(kv, notificationsPath, deploymentID)
}

// NotifyValueChange allows to notify attribute value change
func (an *AttributeNotifier) NotifyValueChange(kv *api.KV, deploymentID string) error {
	log.Debugf("Received instance attribute value change notification for [deploymentID:%q, nodeName:%q, instanceName:%q, capabilityName:%q, attributeName:%q", deploymentID, an.NodeName, an.InstanceName, an.CapabilityName, an.AttributeName)
	var notificationsPath string
	if an.CapabilityName != "" {
		notificationsPath = path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", an.NodeName, an.InstanceName, "capabilities", an.CapabilityName, "attribute_notifications", an.AttributeName)
	} else {
		notificationsPath = path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", an.NodeName, an.InstanceName, "attribute_notifications", an.AttributeName)
	}

	return notifyAttributeOnValueChange(kv, notificationsPath, deploymentID)
}

func notifyAttributeOnValueChange(kv *api.KV, notificationsPath, deploymentID string) error {
	kvps, _, err := kv.List(notificationsPath, nil)
	if err != nil {
		return err
	}
	for _, kvp := range kvps {

		notified, err := getNotifiedAttribute(string(kvp.Value))
		log.Debugf("Need to notify attribute:%+v from attribute/operation output value change", notified)
		if err != nil {
			return err
		}
		if notified.capabilityName != "" {
			value, err := GetInstanceCapabilityAttributeValue(kv, deploymentID, notified.nodeName, notified.instanceName, notified.capabilityName, notified.attributeName)
			if err != nil {
				return err
			}
			if value != nil {
				if err = SetInstanceCapabilityAttribute(deploymentID, notified.nodeName, notified.instanceName, notified.capabilityName, notified.attributeName, value.String()); err != nil {
					return err
				}
			}
			return nil
		}

		value, err := GetInstanceAttributeValue(kv, deploymentID, notified.nodeName, notified.instanceName, notified.attributeName)

		if err != nil {
			return err
		}
		if value != nil {
			if err = SetInstanceAttribute(deploymentID, notified.nodeName, notified.instanceName, notified.attributeName, value.String()); err != nil {
				return err
			}
		}
	}
	return nil
}

func addSubstitutionMappingAttributeHostNotification(kv *api.KV, deploymentID, nodeName, instanceName, capabilityName, attributeName string, anh *attributeNotificationsHandler) error {
	host, err := GetHostedOnNode(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if host != "" {
		var notifier *AttributeNotifier
		capabilityType, err := GetNodeCapabilityType(kv, deploymentID, nodeName, capabilityName)
		if err != nil {
			return err
		}

		isEndpoint, err := IsTypeDerivedFrom(kv, deploymentID, capabilityType,
			tosca.EndpointCapability)
		if err != nil {
			return err
		}

		// TOSCA specification at :
		// http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_TYPE_CAPABILITIES_ENDPOINT
		// describes that the ip_address attribute of an endpoint is the IP address
		// as propagated up by the associated node’s host (Compute) container.
		if isEndpoint && attributeName == "ip_address" {
			notifier = &AttributeNotifier{
				NodeName:      host,
				InstanceName:  anh.attribute.instanceName,
				AttributeName: attributeName,
			}
		} else {
			notifier = &AttributeNotifier{
				NodeName:       host,
				InstanceName:   anh.attribute.instanceName,
				AttributeName:  attributeName,
				CapabilityName: capabilityName,
			}
		}

		log.Debugf("Add substitution attribute %s for %s %s %s with notifier:%+v", attributeName, deploymentID, host, instanceName, notifier)
		err = anh.saveNotification(notifier)
		if err != nil {
			return err
		}
		return addSubstitutionMappingAttributeHostNotification(kv, deploymentID, host, instanceName, capabilityName, attributeName, anh)
	}
	return nil
}

func addSubstitutionMappingAttributeNotification(consulStore consulutil.ConsulStore, kv *api.KV, deploymentID, nodeName, instanceName, attributeName string) error {
	items := strings.Split(attributeName, ".")
	capabilityName := items[1]
	capAttrName := items[2]

	// Check this capability attribute is really exposed before returning its
	// value
	attributesSet := make(map[string]struct{})
	err := storeSubstitutionMappingAttributeNamesInSet(kv, deploymentID, nodeName, attributesSet)
	if err != nil {
		return err
	}
	if _, ok := attributesSet[attributeName]; ok {
		anh := &attributeNotificationsHandler{
			consulStore:  consulStore,
			kv:           kv,
			deploymentID: deploymentID,
			attribute: &notifiedAttribute{
				nodeName:      nodeName,
				instanceName:  instanceName,
				attributeName: attributeName,
			},
		}

		// As we can't say if the capability attribute is related to node nodeName or its host, we add notifications for all
		err = addSubstitutionMappingAttributeHostNotification(kv, deploymentID, nodeName, instanceName, capabilityName, capAttrName, anh)
		if err != nil {
			return err
		}

		notifier := &AttributeNotifier{
			NodeName:       nodeName,
			InstanceName:   anh.attribute.instanceName,
			AttributeName:  capAttrName,
			CapabilityName: capabilityName,
		}
		log.Debugf("Add substitution attribute %s for %s %s %s with notifier:%+v", attributeName, deploymentID, nodeName, instanceName, notifier)
		return anh.saveNotification(notifier)
	}
	return nil
}

// This allows to store notifications for attributes depending on other ones or on operation outputs  in order to ensure events publication when attribute value change
// This allows too to publish initial state for default attribute value
func addAttributeNotifications(consulStore consulutil.ConsulStore, kv *api.KV, deploymentID, nodeName, instanceName, attributeName string) error {
	substitutionInstance, err := isSubstitutionNodeInstance(kv, deploymentID, nodeName, instanceName)
	if err != nil {
		return err
	}

	// Publish attributes for substitution attributes
	if substitutionInstance {
		found, result := getSubstitutionInstanceAttribute(deploymentID, nodeName, instanceName, attributeName)
		if found {
			events.PublishAndLogAttributeValueChange(context.Background(), deploymentID, nodeName, instanceName, attributeName, result, "updated")
			return nil
		}
	}

	if isSubstitutionMappingAttribute(attributeName) && !substitutionInstance {
		return addSubstitutionMappingAttributeNotification(consulStore, kv, deploymentID, nodeName, instanceName, attributeName)
	}

	nodeType, err := GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}

	var attrDataType string
	hasAttr, err := TypeHasAttribute(kv, deploymentID, nodeType, attributeName, true)
	if err != nil {
		return err
	}
	if hasAttr {
		attrDataType, err = GetTypeAttributeDataType(kv, deploymentID, nodeType, attributeName)
		if err != nil {
			return err
		}
	}

	// First look at instance-scoped attributes
	vaPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances", nodeName, instanceName, "attributes", attributeName)
	value, isFunction, err := getValueAssignmentWithoutResolve(kv, deploymentID, vaPath, attrDataType)
	if err != nil || (value != nil && !isFunction) {
		return errors.Wrapf(err, "Failed to add instance attribute notifications %q for node %q (instance %q)", attributeName, nodeName, instanceName)
	}

	// Then look at global node level (not instance-scoped)
	if value == nil {
		vaPath = path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/nodes", nodeName, "attributes", attributeName)
		value, isFunction, err = getValueAssignmentWithoutResolve(kv, deploymentID, vaPath, attrDataType)
		if err != nil || (value != nil && !isFunction) {
			return errors.Wrapf(err, "Failed to add instance attribute notifications %q for node %q (instance %q)", attributeName, nodeName, instanceName)
		}
	}

	// Not found look at node type
	if value == nil {
		value, isFunction, err = getTypeDefaultAttribute(kv, deploymentID, nodeType, attributeName)
		if err != nil {
			return errors.Wrapf(err, "Failed to add instance attribute notifications %q for node %q (instance %q)", attributeName, nodeName, instanceName)
		}
		// Publish default value
		if value != nil && !isFunction {
			events.PublishAndLogAttributeValueChange(context.Background(), deploymentID, nodeName, instanceName, attributeName, value.String(), "default")
			return nil
		}
	}

	if value == nil {
		// No default found in type hierarchy
		// then traverse HostedOn relationships to find the value
		var host string
		host, err = GetHostedOnNode(kv, deploymentID, nodeName)
		if err != nil {
			return errors.Wrapf(err, "Failed to add instance attribute notifications %q for node %q (instance %q)", attributeName, nodeName, instanceName)
		}
		if host != "" {
			addAttributeNotifications(consulStore, kv, deploymentID, host, instanceName, attributeName)
		}
	}

	// all possibilities have been checked at this point: check if any get_attribute function is contained
	if value != nil {
		anh := attributeNotificationsHandler{
			consulStore:  consulStore,
			kv:           kv,
			deploymentID: deploymentID,
			attribute: &notifiedAttribute{
				nodeName:      nodeName,
				instanceName:  instanceName,
				attributeName: attributeName,
			},
		}
		return anh.parseFunction(value.RawString())
	}

	return nil
}

// This is looking for Tosca get_attribute and get_operation_output functions
func (anh *attributeNotificationsHandler) parseFunction(rawFunction string) error {
	// Function
	va := &tosca.ValueAssignment{}
	err := yaml.Unmarshal([]byte(rawFunction), va)
	if err != nil {
		return errors.Wrapf(err, "Failed to parse TOSCA function %q for node %q", rawFunction, anh.attribute.nodeName)
	}
	log.Debugf("function = %+v", va.GetFunction())

	f := va.GetFunction()

	fcts := f.GetFunctionsByOperator(tosca.GetAttributeOperator)
	for _, fct := range fcts {
		// Find related notifier
		operands := make([]string, len(fct.Operands))
		for i, op := range fct.Operands {
			operands[i] = op.String()
		}
		notifier, err := anh.findAttributeNotifier(operands)
		if err != nil {
			return errors.Wrapf(err, "Failed to find get_attribute notifier for function: %q and node %q", fct, anh.attribute.nodeName)
		}

		// Store notification
		err = anh.saveNotification(notifier)
		if err != nil {
			return errors.Wrapf(err, "Failed to save notification from notifier:%+v and notified %+v", notifier, anh.attribute)
		}
	}

	fcts = f.GetFunctionsByOperator(tosca.GetOperationOutputOperator)
	for _, fct := range fcts {
		// Find related notifier
		operands := make([]string, len(fct.Operands))
		for i, op := range fct.Operands {
			operands[i] = op.String()
		}
		notifier, err := anh.findOperationOutputNotifier(operands)
		if err != nil {
			return errors.Wrapf(err, "Failed to find get_attribute notifier for function: %q and node %q", fct, anh.attribute.nodeName)
		}

		// Store notification
		err = anh.saveNotification(notifier)
		if err != nil {
			return errors.Wrapf(err, "Failed to save notification from notifier:%+v and notified %+v", notifier, anh.attribute)
		}
	}
	return nil
}

func (anh *attributeNotificationsHandler) findOperationOutputNotifier(operands []string) (Notifier, error) {
	funcString := fmt.Sprintf("get_operation_output: [%s]", strings.Join(operands, ", "))
	if len(operands) != 4 {
		return nil, errors.Errorf("expecting exactly four parameters for a get_operation_output function (%s)", funcString)
	}
	return &OperationOutputNotifier{
		InstanceName:  anh.attribute.instanceName,
		NodeName:      anh.attribute.nodeName,
		OperationName: strings.ToLower(operands[2]),
		InterfaceName: strings.ToLower(operands[1]),
		OutputName:    operands[3],
	}, nil
}

func (anh *attributeNotificationsHandler) findAttributeNotifier(operands []string) (Notifier, error) {
	funcString := fmt.Sprintf("get_attribute: [%s]", strings.Join(operands, ", "))
	if len(operands) < 2 || len(operands) > 3 {
		return nil, errors.Errorf("expecting two or three parameters for a non-relationship context get_attribute function (%s)", funcString)
	}

	var node, capName, attrName string
	var err error

	if len(operands) == 2 {
		attrName = operands[1]
	} else {
		attrName = operands[2]
		capName = operands[1]
	}

	switch operands[0] {
	case funcKeywordSELF:
		node = anh.attribute.nodeName
	case funcKeywordHOST:
		node, err = resolveHostNotifier(anh.kv, anh.deploymentID, anh.attribute.nodeName, attrName)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.Errorf("unexpected keyword:%q in get_attribute function (%s)", operands[0], funcString)
	}

	if node == "" {
		return nil, errors.Errorf("unable to find node name related to get_attribute function (%s)", funcString)
	}

	notifier := &AttributeNotifier{
		NodeName:       node,
		InstanceName:   anh.attribute.instanceName,
		AttributeName:  attrName,
		CapabilityName: capName,
	}
	return notifier, nil
}

func (anh *attributeNotificationsHandler) saveNotification(notifier Notifier) error {
	var notificationsPath string
	switch n := notifier.(type) {
	case *AttributeNotifier:
		if n.CapabilityName != "" {
			notificationsPath = path.Join(consulutil.DeploymentKVPrefix, anh.deploymentID, "topology", "instances", n.NodeName, anh.attribute.instanceName, "capabilities", n.CapabilityName, "attribute_notifications", n.AttributeName)

		} else {
			notificationsPath = path.Join(consulutil.DeploymentKVPrefix, anh.deploymentID, "topology", "instances", n.NodeName, anh.attribute.instanceName, "attribute_notifications", n.AttributeName)
		}
	case *OperationOutputNotifier:
		notificationsPath = path.Join(consulutil.DeploymentKVPrefix, anh.deploymentID, "topology", "instances", n.NodeName, n.InstanceName, "outputs", n.InterfaceName, n.OperationName, "attribute_notifications", n.OutputName)

	default:
		return errors.Errorf("Unexpected type %T for saving notifications", n)
	}

	notifs, _, err := anh.kv.Keys(notificationsPath+"/", "/", nil)
	if err != nil {
		return err
	}
	var index int
	if notifs != nil {
		index = len(notifs)
	}

	key := path.Join(notificationsPath, strconv.Itoa(index))
	val := buildNotificationValue(anh.attribute.nodeName, anh.attribute.instanceName, anh.attribute.capabilityName, anh.attribute.attributeName)
	log.Debugf("store notification with[key=%q, value:%q", key, val)
	anh.consulStore.StoreConsulKeyAsString(key, val)
	return nil
}

// notification value is path-based as: "<NODE_NAME>/<INSTANCE_NAME>/attributes/<ATTRIBUTE_NAME>
// or "<NODE_NAME>/<INSTANCE_NAME>/capabilities/<CAPABILITY_NAME>/attributes/<ATTRIBUTE_NAME>"
func buildNotificationValue(nodeName, instanceName, capabilityName, attributeName string) string {
	if capabilityName != "" {
		return path.Join(nodeName, instanceName, "capabilities", capabilityName, "attributes", attributeName)
	}
	return path.Join(nodeName, instanceName, "attributes", attributeName)
}

func getNotifiedAttribute(notification string) (*notifiedAttribute, error) {
	notified := strings.Split(notification, "/")
	if len(notified) != 4 && len(notified) != 6 {
		return nil, errors.Errorf("unexpected format %q for notification", notification)
	}
	var attribData *notifiedAttribute
	if len(notified) == 4 {
		attribData = &notifiedAttribute{
			nodeName:      notified[0],
			instanceName:  notified[1],
			attributeName: notified[3],
		}
	} else {
		attribData = &notifiedAttribute{
			nodeName:       notified[0],
			instanceName:   notified[1],
			capabilityName: notified[3],
			attributeName:  notified[5],
		}
	}
	return attribData, nil
}

// resolveHostNotifier retrieves the node name hosting the provided nodeName having the provided attributeName in the "HostedOn" relationship stack
// If no host node is found with the related attributeName, root hosting node (compute) is returned as attribute can not be defined in Tosca (as public_ip_address for compatibility)
func resolveHostNotifier(kv *api.KV, deploymentID, nodeName, attributeName string) (string, error) {
	hostNode, err := GetHostedOnNode(kv, deploymentID, nodeName)
	if err != nil {
		return "", err
	}
	if hostNode == "" {
		return nodeName, nil
	}
	attributes, err := GetNodeAttributesNames(kv, deploymentID, hostNode)
	if err != nil {
		return "", err
	}
	if collections.ContainsString(attributes, attributeName) {
		return hostNode, nil
	}

	return resolveHostNotifier(kv, deploymentID, hostNode, attributeName)
}
