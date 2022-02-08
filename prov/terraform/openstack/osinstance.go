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

package openstack

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
)

const (
	topologyTree = "topology"
)

type osInstanceOptions struct {
	cfg            config.Configuration
	infrastructure *commons.Infrastructure
	locationProps  config.DynamicMap
	deploymentID   string
	nodeName       string
	instanceName   string
	resourceTypes  map[string]string
}

func (g *osGenerator) generateOSInstance(ctx context.Context, opts osInstanceOptions, outputs map[string]string, env *[]string) error {
	deploymentID := opts.deploymentID
	nodeName := opts.nodeName
	instanceName := opts.instanceName

	nodeType, err := deployments.GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType != "yorc.nodes.openstack.Compute" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}
	instance, err := generateComputeInstance(ctx, opts)
	if err != nil {
		return err
	}

	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, opts.deploymentID, topologyTree, "instances")
	err = generateAttachedVolumes(ctx, opts, instancesPrefix, instance, outputs)
	if err != nil {
		return err
	}

	err = addServerGroupMembership(ctx, deploymentID, nodeName, &instance)
	if err != nil {
		return errors.Wrapf(err, "failed to add serverGroup membership for deploymentID:%q, nodeName:%q, instance:%q",
			deploymentID, nodeName, instanceName)
	}

	instance.Networks, err = getComputeInstanceNetworks(ctx, opts)
	if err != nil {
		return err
	}

	err = computeConnectionSettings(ctx, opts, instancesPrefix, &instance, outputs, env)
	return err
}

func addServerGroupMembership(ctx context.Context, deploymentID, nodeName string, compute *ComputeInstance) error {
	reqs, err := deployments.GetRequirementsByTypeForNode(ctx, deploymentID, nodeName, "group")
	if err != nil {
		return err
	}
	if len(reqs) == 0 {
		return nil
	}

	if len(reqs) == 1 {
		id, err := deployments.LookupInstanceAttributeValue(ctx, deploymentID, reqs[0].Node, deployments.DefaultInstanceName, "id")
		if err != nil {
			return err
		}

		compute.SchedulerHints.Group = id
		return nil
	}

	return errors.Errorf("Only one group requirement can be accepted for OpenStack compute with name:%q", nodeName)

}

func generateComputeInstance(ctx context.Context, opts osInstanceOptions) (ComputeInstance, error) {

	instance := ComputeInstance{}
	nodeType, err := deployments.GetNodeType(ctx, opts.deploymentID, opts.nodeName)
	if err != nil {
		return instance, err
	}
	if nodeType != "yorc.nodes.openstack.Compute" {
		return instance, errors.Errorf("Unsupported node type for %q: %s", opts.nodeName, nodeType)
	}

	instance.Name = opts.cfg.ResourcesPrefix + opts.nodeName + "-" + opts.instanceName

	if instance.BootVolume, err = computeBootVolume(ctx, opts.deploymentID, opts.nodeName); err != nil {
		return instance, err
	}

	if instance.BootVolume == nil {
		// When no boot volume is defined, it is mandatory to define an image
		// or an image name
		if instance.ImageID, instance.ImageName, err = computeInstanceMandatoryAttributeInPair(ctx, opts.deploymentID, opts.nodeName, "image", "imageName"); err != nil {
			return instance, err
		}
	}

	if instance.FlavorID, instance.FlavorName, err = computeInstanceMandatoryAttributeInPair(ctx, opts.deploymentID, opts.nodeName, "flavor", "flavorName"); err != nil {
		return instance, err
	}

	instance.AvailabilityZone, err = deployments.GetStringNodeProperty(ctx, opts.deploymentID,
		opts.nodeName, "availability_zone", false)
	if err != nil {
		return instance, err
	}
	instance.Region, err = deployments.GetStringNodeProperty(ctx, opts.deploymentID, opts.nodeName, "region", false)
	if err != nil {
		return instance, err
	}
	if instance.Region == "" {
		instance.Region = opts.locationProps.GetStringOrDefault("region", defaultOSRegion)
	}

	instance.KeyPair, err = deployments.GetStringNodeProperty(ctx, opts.deploymentID, opts.nodeName, "key_pair", false)
	if err != nil {
		return instance, err
	}

	instance.SecurityGroups = opts.locationProps.GetStringSlice("default_security_groups")
	secGroups, err := deployments.GetStringNodeProperty(ctx, opts.deploymentID, opts.nodeName, "security_groups", false)
	if err != nil {
		return instance, err
	}
	if secGroups != "" {
		for _, secGroup := range strings.Split(strings.NewReplacer("\"", "", "'", "").Replace(secGroups), ",") {
			secGroup = strings.TrimSpace(secGroup)
			instance.SecurityGroups = append(instance.SecurityGroups, secGroup)
		}
	}

	toscaVal, err := deployments.GetNodePropertyValue(ctx, opts.deploymentID, opts.nodeName, "metadata")
	if err != nil {
		return instance, err
	}
	if toscaVal != nil && toscaVal.RawString() != "" {
		err = json.Unmarshal([]byte(toscaVal.RawString()), &instance.Metadata)
		if err != nil {
			return instance, errors.Wrapf(err, "Expected a map of strings for the metadata value of node %s instance %s, got: %s",
				opts.nodeName, opts.instanceName, toscaVal.RawString())
		}
	}

	instance.UserData, err = deployments.GetStringNodeProperty(ctx, opts.deploymentID,
		opts.nodeName, "user_data", false)

	return instance, err
}

func computeInstanceMandatoryAttributeInPair(ctx context.Context, deploymentID, nodeName, attr1, attr2 string) (string, string, error) {
	var err error
	var value1, value2 string
	value1, err = deployments.GetStringNodeProperty(ctx, deploymentID, nodeName, attr1, false)
	if err != nil {
		return value1, value2, err
	}
	value2, err = deployments.GetStringNodeProperty(ctx, deploymentID, nodeName, attr2, false)
	if err != nil {
		return value1, value2, err
	}

	if value1 == "" && value2 == "" {
		err = errors.Errorf("Missing mandatory parameter %q or %q node type for %s",
			attr1, attr2, nodeName)
	}

	return value1, value2, err
}

func generateAttachedVolumes(ctx context.Context, opts osInstanceOptions,
	instancesPrefix string, instance ComputeInstance,
	outputs map[string]string) error {

	infrastructure := opts.infrastructure
	deploymentID := opts.deploymentID
	nodeName := opts.nodeName
	instanceName := opts.instanceName

	storageReqs, err := deployments.GetRequirementsByTypeForNode(ctx, deploymentID, nodeName, "local_storage")
	if err != nil {
		return err
	}
	for _, storageReq := range storageReqs {
		if storageReq.Node != "" {
			log.Debugf("Volume attachment required form Volume named %s", storageReq.Node)
			device, err := deployments.GetRelationshipPropertyValueFromRequirement(ctx, deploymentID, nodeName, storageReq.Index, "device")
			if err != nil {
				return err
			}
			volumeID, err := getVolumeID(ctx, deploymentID, storageReq.Node, instanceName)
			if err != nil {
				return err
			}
			volumeAttach := ComputeVolumeAttach{
				Region:   instance.Region,
				VolumeID: volumeID,
				InstanceID: fmt.Sprintf("${%s.%s.id}",
					opts.resourceTypes[computeInstance], instance.Name),
			}
			if device != nil {
				volumeAttach.Device = device.RawString()
			}
			attachName := "Vol" + storageReq.Node + "to" + instance.Name
			commons.AddResource(infrastructure, opts.resourceTypes[computeVolumeAttach],
				attachName, &volumeAttach)

			// retrieve the actual used device as depending on the hypervisor it may not be the one we provided, and if there was no devices provided
			// then we can get it back

			key1 := attachName + "-device"
			commons.AddOutput(infrastructure, key1, &commons.Output{
				Value: fmt.Sprintf("${%s.%s.device}",
					opts.resourceTypes[computeVolumeAttach], attachName)})
			outputs[path.Join(instancesPrefix, storageReq.Node, instanceName, "attributes/device")] = key1
			outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID, topologyTree, "relationship_instances", nodeName, storageReq.Index, instanceName, "attributes/device")] = key1
			outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID, topologyTree, "relationship_instances", storageReq.Node, storageReq.Index, instanceName, "attributes/device")] = key1
		}
	}

	return err
}

func getVolumeID(ctx context.Context, deploymentID, volumeNodeName, instanceName string) (string, error) {
	log.Debugf("Looking for volume_id")
	volumeID, err := deployments.GetStringNodeProperty(ctx, deploymentID, volumeNodeName, "volume_id", false)
	if err != nil || volumeID != "" {
		return volumeID, err
	}

	return deployments.LookupInstanceAttributeValue(ctx, deploymentID, volumeNodeName, instanceName, "volume_id")
}

func getComputeInstanceNetworks(ctx context.Context, opts osInstanceOptions) ([]ComputeNetwork, error) {
	deploymentID := opts.deploymentID
	nodeName := opts.nodeName

	var networkSlice []ComputeNetwork
	networkName, err := deployments.GetCapabilityPropertyValue(ctx, deploymentID, nodeName, "endpoint", "network_name")
	if err != nil {
		return networkSlice, err
	}
	defaultPrivateNetName := opts.locationProps.GetString("private_network_name")
	if networkName != nil && networkName.RawString() != "" {
		// Should Deal here with networks aliases (PUBLIC)
		if strings.EqualFold(networkName.RawString(), "private") {
			if defaultPrivateNetName == "" {
				return networkSlice, errors.Errorf(
					"You should either specify a default private network name using "+
						`the "private_network_name" configuration parameter for the "openstack" `+
						`infrastructure or specify a "network_name" property in the "endpoint" capability of node %q`,
					nodeName)
			}
			networkSlice = append(networkSlice,
				ComputeNetwork{Name: defaultPrivateNetName, AccessNetwork: true})
		} else if strings.EqualFold(networkName.RawString(), "public") {
			return networkSlice, errors.Errorf("Public Network aliases currently not supported")
		} else {
			networkSlice = append(networkSlice, ComputeNetwork{Name: networkName.RawString(), AccessNetwork: true})
		}
	} else {
		// Use a default
		if defaultPrivateNetName == "" {
			return networkSlice, errors.Errorf(
				"You should either specify a default private network name using the "+
					`private_network_name" configuration parameter for the "openstack" `+
					`infrastructure or specify a "network_name" property in the "endpoint" `+
					"capability of node %q`",
				nodeName)
		}
		networkSlice = append(networkSlice, ComputeNetwork{Name: defaultPrivateNetName, AccessNetwork: true})
	}

	return networkSlice, err
}

func computeConnectionSettings(ctx context.Context, opts osInstanceOptions,
	instancesPrefix string, instance *ComputeInstance, outputs map[string]string, env *[]string) error {
	deploymentID := opts.deploymentID
	nodeName := opts.nodeName

	networkReqs, err := deployments.GetRequirementsByTypeForNode(ctx, deploymentID, nodeName, "network")
	if err != nil {
		return err
	}
	var fipAssociateName string
	instancesKey := path.Join(instancesPrefix, nodeName)
	for _, networkReq := range networkReqs {
		var isFip bool
		if networkReq.Capability != "" {
			isFip, err = deployments.IsTypeDerivedFrom(ctx, deploymentID, networkReq.Capability, "yorc.capabilities.openstack.FIPConnectivity")
			if err != nil {
				return err
			}
		}

		if isFip {
			fipAssociateName = "FIP" + instance.Name
			err = computeFloatingIPAddress(ctx, opts, fipAssociateName, networkReq.Node, instancesKey, instance, outputs)
		} else {
			err = computeNetworkAttributes(ctx, opts, networkReq.Node, instancesKey, instance, outputs)

		}
		if err != nil {
			return err
		}
	}

	return addResources(ctx, opts, fipAssociateName, instancesKey, instance, outputs, env)
}

func addResources(ctx context.Context, opts osInstanceOptions, fipAssociateName, instancesKey string, instance *ComputeInstance, outputs map[string]string,
	env *[]string) error {

	commons.AddResource(opts.infrastructure, opts.resourceTypes[computeInstance], instance.Name, instance)

	var accessIP string
	if fipAssociateName != "" && opts.locationProps.GetBool(
		"provisioning_over_fip_allowed") {

		// Use Floating IP for provisioning
		accessIP = fmt.Sprintf("${%s.%s.floating_ip}",
			opts.resourceTypes[computeFloatingIPAssociate], fipAssociateName)
	} else {
		accessIP = fmt.Sprintf("${%s.%s.network.0.fixed_ip_v4}",
			opts.resourceTypes[computeInstance], instance.Name)
	}

	// Provide output for access IP and private IP
	accessIPKey := opts.nodeName + "-" + opts.instanceName + "-IPAddress"
	commons.AddOutput(opts.infrastructure, accessIPKey, &commons.Output{Value: accessIP})
	outputs[path.Join(instancesKey, opts.instanceName,
		"/capabilities/endpoint/attributes/ip_address")] = accessIPKey
	outputs[path.Join(instancesKey, opts.instanceName,
		"/attributes/ip_address")] = accessIPKey

	privateIPKey := opts.nodeName + "-" + opts.instanceName + "-privateIP"
	privateIP := fmt.Sprintf("${%s.%s.network.%d.fixed_ip_v4}",
		opts.resourceTypes[computeInstance], instance.Name,
		len(instance.Networks)-1) // Use latest provisioned network for private access
	commons.AddOutput(opts.infrastructure, privateIPKey, &commons.Output{Value: privateIP})
	outputs[path.Join(instancesKey, opts.instanceName,
		"/attributes/private_address")] = privateIPKey

	// Get connection info (user, private key)
	user, pk, err := commons.GetConnInfoFromEndpointCredentials(ctx, opts.deploymentID, opts.nodeName)
	if err != nil {
		return err
	}

	return commons.AddConnectionCheckResource(ctx, opts.deploymentID, opts.nodeName, opts.infrastructure, user,
		pk, accessIP, instance.Name, env)

}

func computeFloatingIPAddress(ctx context.Context, opts osInstanceOptions,
	fipAssociateName, networkNodeName, instancesKey string,
	instance *ComputeInstance, outputs map[string]string) error {

	deploymentID := opts.deploymentID
	infrastructure := opts.infrastructure
	nodeName := opts.nodeName
	instanceName := opts.instanceName

	log.Debugf("Looking for Floating IP")

	floatingIP, err := deployments.LookupInstanceCapabilityAttributeValue(ctx, deploymentID, networkNodeName, opts.instanceName, "endpoint", "floating_ip_address")
	if err != nil {
		return err
	}

	floatingIPAssociate := ComputeFloatingIPAssociate{
		Region:     instance.Region,
		FloatingIP: floatingIP,
		InstanceID: fmt.Sprintf("${%s.%s.id}",
			opts.resourceTypes[computeInstance], instance.Name),
	}
	commons.AddResource(infrastructure, opts.resourceTypes[computeFloatingIPAssociate],
		fipAssociateName, &floatingIPAssociate)

	// Provide output for public IP as floating IP
	publicIPKey := nodeName + "-" + instanceName + "-publicIP"
	commons.AddOutput(infrastructure, publicIPKey, &commons.Output{Value: floatingIP})
	outputs[path.Join(instancesKey, instanceName, "/attributes/public_address")] = publicIPKey
	// In order to be backward compatible to components developed for Alien (only the above is standard)
	outputs[path.Join(instancesKey, instanceName, "/attributes/public_ip_address")] = publicIPKey

	return nil
}

func computeNetworkAttributes(ctx context.Context, opts osInstanceOptions,
	networkNodeName, instancesKey string,
	instance *ComputeInstance, outputs map[string]string) error {

	log.Debugf("Looking for Network id for %q", networkNodeName)
	networkID, err := deployments.LookupInstanceAttributeValue(ctx, opts.deploymentID, networkNodeName, deployments.DefaultInstanceName, "network_id")
	if err != nil {
		return err
	}

	cn := ComputeNetwork{UUID: networkID, AccessNetwork: false}
	i := len(instance.Networks)
	if instance.Networks == nil {
		instance.Networks = make([]ComputeNetwork, 0)
	}
	instance.Networks = append(instance.Networks, cn)

	// Provide output for network_name, network_id, addresses attributes
	networkIDKey := opts.nodeName + "-" + opts.instanceName + "-networkID"
	networkNameKey := opts.nodeName + "-" + opts.instanceName + "-networkName"
	networkAddressesKey := opts.nodeName + "-" + opts.instanceName + "-addresses"
	commons.AddOutput(opts.infrastructure, networkIDKey, &commons.Output{
		Value: fmt.Sprintf("${%s.%s.network.%d.uuid}",
			opts.resourceTypes[computeInstance], instance.Name, i)})
	commons.AddOutput(opts.infrastructure, networkNameKey, &commons.Output{
		Value: fmt.Sprintf("${%s.%s.network.%d.name}",
			opts.resourceTypes[computeInstance], instance.Name, i)})
	commons.AddOutput(opts.infrastructure, networkAddressesKey, &commons.Output{
		Value: fmt.Sprintf("[ ${%s.%s.network.%d.fixed_ip_v4} ]",
			opts.resourceTypes[computeInstance], instance.Name, i)})

	prefix := path.Join(instancesKey, opts.instanceName, "attributes/networks", strconv.Itoa(i))
	outputs[path.Join(prefix, "network_name")] = networkNameKey
	outputs[path.Join(prefix, "network_id")] = networkIDKey
	outputs[path.Join(prefix, "addresses")] = networkAddressesKey
	return nil
}
