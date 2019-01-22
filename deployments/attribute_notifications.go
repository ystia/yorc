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
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/tosca"
	"gopkg.in/yaml.v2"
	"path"
	"strconv"
	"strings"
)

type attributeNotificationsHandler struct {
	kv           *api.KV
	consulStore  consulutil.ConsulStore
	deploymentID string
	attribute    *attributeData
}

type attributeData struct {
	nodeName       string
	instanceName   string
	attributeName  string
	capabilityName string
}

// Get related instance capability attribute notifications to update notified attributes
func notifyInstanceCapabilityAttributeValueChange(kv *api.KV, deploymentID, nodeName, instanceName, capabilityName, attributeName string) error {
	notificationsPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName, instanceName, "capabilities", capabilityName, "attribute_notifications", attributeName)
	return notifyAttributeValueChange(kv, notificationsPath, deploymentID, nodeName, instanceName, capabilityName, attributeName)
}

// Get related instance attribute notifications to update notified attributes
func notifyInstanceAttributeValueChange(kv *api.KV, deploymentID, nodeName, instanceName, attributeName string) error {
	notificationsPath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances", nodeName, instanceName, "attribute_notifications", attributeName)
	return notifyAttributeValueChange(kv, notificationsPath, deploymentID, nodeName, instanceName, "", attributeName)
}

func notifyAttributeValueChange(kv *api.KV, notificationsPath, deploymentID, nodeName, instanceName, capabilityName, attributeName string) error {
	kvps, _, err := kv.List(notificationsPath, nil)
	if err != nil {
		return err
	}
	for _, kvp := range kvps {
		notified, err := getNotifiedAttribute(string(kvp.Value))
		log.Debugf("Need to notify attribute:%+v from attribute value change:[nodeName:%q, instanceName:%q, capabilityName:%q, attributeName:%q]", notified, nodeName, instanceName, capabilityName, attributeName)
		if err != nil {
			return err
		}
		if notified.capabilityName != "" {
			value, err := GetInstanceCapabilityAttributeValue(kv, deploymentID, notified.nodeName, notified.instanceName, notified.capabilityName, notified.attributeName)
			if err != nil {
				return err
			}
			if err = SetInstanceCapabilityAttribute(deploymentID, notified.nodeName, notified.instanceName, notified.capabilityName, notified.attributeName, value.String()); err != nil {
				return err
			}
			return nil
		}

		value, err := GetInstanceAttributeValue(kv, deploymentID, notified.nodeName, notified.instanceName, notified.attributeName)
		if err != nil {
			return err
		}
		if err = SetInstanceAttribute(deploymentID, notified.nodeName, notified.instanceName, notified.attributeName, value.String()); err != nil {
			return err
		}
	}
	return nil
}

func addAttributeNotifications(consulStore consulutil.ConsulStore, kv *api.KV, deploymentID, nodeName, instanceName, attributeName string) error {
	substitutionInstance, err := isSubstitutionNodeInstance(kv, deploymentID, nodeName, instanceName)
	if err != nil {
		return err
	}

	// Nothing to do (TBC)
	if substitutionInstance || isSubstitutionMappingAttribute(attributeName) {
		return nil
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
		if err != nil || (value != nil && !isFunction) {
			return errors.Wrapf(err, "Failed to add instance attribute notifications %q for node %q (instance %q)", attributeName, nodeName, instanceName)
		}
	}

	// all possibilities have been checked at this point: check if any get_attribute function is contained
	if value != nil {
		anh := attributeNotificationsHandler{
			consulStore:  consulStore,
			kv:           kv,
			deploymentID: deploymentID,
			attribute: &attributeData{
				nodeName:      nodeName,
				instanceName:  instanceName,
				attributeName: attributeName,
			},
		}
		return anh.parseFunction(value.RawString())
	}

	return nil
}

// This is looking for Tosca get_attribute functions
func (anh *attributeNotificationsHandler) parseFunction(rawFunction string) error {
	// Function
	va := &tosca.ValueAssignment{}
	err := yaml.Unmarshal([]byte(rawFunction), va)
	if err != nil {
		return errors.Wrapf(err, "Failed to parse TOSCA function %q for node %q", rawFunction, anh.attribute.nodeName)
	}
	log.Debugf("function = %+v", va.GetFunction())

	f := va.GetFunction()

	switch f.Operator {
	case tosca.ConcatOperator:
		for _, op := range f.Operands {
			if !op.IsLiteral() {
				// Recursive function parsing
				anh.parseFunction(op.String())
			}
		}
	case tosca.GetAttributeOperator:
		// Find related notifier
		operands := make([]string, len(f.Operands))
		for i, op := range f.Operands {
			operands[i] = op.String()
		}
		notifier, err := anh.findGetAttributeNotifier(operands)
		if err != nil {
			return errors.Wrapf(err, "Failed to find get_attribute notifier for function: %q and node %q", rawFunction, anh.attribute.nodeName)
		}

		// Store notification
		return anh.saveNotification(notifier)
	}
	return nil
}

func (anh *attributeNotificationsHandler) findGetAttributeNotifier(operands []string) (*attributeData, error) {
	funcString := fmt.Sprintf("get_attribute: [%s]", strings.Join(operands, ", "))
	if len(operands) < 2 || len(operands) > 3 {
		return nil, errors.Errorf("expecting two or three parameters for a non-relationship context get_attribute function (%s)", funcString)
	}

	var node string
	var err error
	switch operands[0] {
	case funcKeywordSELF:
		node = anh.attribute.nodeName
	case funcKeywordHOST:
		node, err = GetHostedOnNode(anh.kv, anh.deploymentID, anh.attribute.nodeName)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.Errorf("unexpected keyword:%q in get_attribute function (%s)", operands[0], funcString)
	}

	if node == "" {
		return nil, errors.Errorf("unable to find node name related to get_attribute function (%s)", funcString)
	}

	attribData := &attributeData{
		nodeName: node,
	}
	if len(operands) == 2 {
		attribData.attributeName = operands[1]
	} else {
		attribData.capabilityName = operands[1]
		attribData.attributeName = operands[2]
	}
	return attribData, nil
}

func (anh *attributeNotificationsHandler) saveNotification(notifier *attributeData) error {
	var notificationsPath string
	if notifier.capabilityName != "" {
		notificationsPath = path.Join(consulutil.DeploymentKVPrefix, anh.deploymentID, "topology", "instances", notifier.nodeName, anh.attribute.instanceName, "capabilities", notifier.capabilityName, "attribute_notifications", notifier.attributeName)

	} else {
		notificationsPath = path.Join(consulutil.DeploymentKVPrefix, anh.deploymentID, "topology", "instances", notifier.nodeName, anh.attribute.instanceName, "attribute_notifications", notifier.attributeName)
	}
	notifs, _, err := anh.kv.Keys(notificationsPath+"/", "/", nil)
	if err != nil {
		return err
	}
	var index int
	if notifs != nil {
		index = len(notifs)
	}

	anh.consulStore.StoreConsulKeyAsString(path.Join(notificationsPath, strconv.Itoa(index)), buildNotificationValue(anh.attribute.nodeName, anh.attribute.instanceName, anh.attribute.capabilityName, anh.attribute.attributeName))
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

func getNotifiedAttribute(notification string) (*attributeData, error) {
	notified := strings.Split(notification, "/")
	if len(notified) != 4 && len(notified) != 6 {
		return nil, errors.Errorf("unexpected format %q for notification", notification)
	}
	var attribData *attributeData
	if len(notified) == 4 {
		attribData = &attributeData{
			nodeName:      notified[0],
			instanceName:  notified[1],
			attributeName: notified[3],
		}
	} else {
		attribData = &attributeData{
			nodeName:       notified[0],
			instanceName:   notified[1],
			capabilityName: notified[3],
			attributeName:  notified[5],
		}
	}
	return attribData, nil
}
