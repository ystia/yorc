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
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
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
	kv             *api.KV
	cfg            config.Configuration
	infrastructure *commons.Infrastructure
	deploymentID   string
	nodeName       string
	instanceName   string
	resourceTypes  map[string]string
}

func (g *osGenerator) generateOSInstance(ctx context.Context, opts osInstanceOptions, outputs map[string]string, env *[]string) error {
	kv := opts.kv
	deploymentID := opts.deploymentID
	nodeName := opts.nodeName
	instanceName := opts.instanceName

	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType != "yorc.nodes.openstack.Compute" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}
	instance, err := generateComputeInstance(opts)
	if err != nil {
		return err
	}

	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, opts.deploymentID, topologyTree, "instances")
	err = generateAttachedVolumes(ctx, opts, instancesPrefix, instance, outputs)
	if err != nil {
		return err
	}

	err = addServerGroupMembership(ctx, kv, deploymentID, nodeName, &instance)
	if err != nil {
		return errors.Wrapf(err, "failed to add serverGroup membership for deploymentID:%q, nodeName:%q, instance:%q",
			deploymentID, nodeName, instanceName)
	}

	instanceKey := path.Join(instancesPrefix, nodeName)
	var accessIP string
	instance.Networks, accessIP, err = generateInstanceNetworking(ctx, opts, instanceKey, &instance, outputs)

	addResources(ctx, opts, accessIP, instanceKey, &instance, outputs, env)
	return err
}

func addServerGroupMembership(ctx context.Context, kv *api.KV, deploymentID, nodeName string, compute *ComputeInstance) error {
	keys, err := deployments.GetRequirementsKeysByTypeForNode(kv, deploymentID, nodeName, "group")
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}

	if len(keys) == 1 {
		requirementIndex := deployments.GetRequirementIndexFromRequirementKey(keys[0])
		serverGroup, err := deployments.GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return err
		}

		id, err := deployments.LookupInstanceAttributeValue(ctx, kv, deploymentID, serverGroup, "0", "id")
		if err != nil {
			return err
		}

		compute.SchedulerHints.Group = id
		return nil
	}

	return errors.Errorf("Only one group requirement can be accepted for OpenStack compute with name:%q", nodeName)

}

func generateComputeInstance(opts osInstanceOptions) (ComputeInstance, error) {

	instance := ComputeInstance{}
	nodeType, err := deployments.GetNodeType(opts.kv, opts.deploymentID, opts.nodeName)
	if err != nil {
		return instance, err
	}
	if nodeType != "yorc.nodes.openstack.Compute" {
		return instance, errors.Errorf("Unsupported node type for %q: %s", opts.nodeName, nodeType)
	}

	instance.Name = opts.cfg.ResourcesPrefix + opts.nodeName + "-" + opts.instanceName

	if instance.BootVolume, err = computeBootVolume(opts.kv, opts.deploymentID, opts.nodeName); err != nil {
		return instance, err
	}

	if instance.BootVolume == nil {
		// When no boot volume is defined, it is mandatory to define an image
		// or an image name
		if instance.ImageID, instance.ImageName, err = computeInstanceMandatoryAttributeInPair(
			opts.kv, opts.deploymentID, opts.nodeName, "image", "imageName"); err != nil {
			return instance, err
		}
	}

	if instance.FlavorID, instance.FlavorName, err = computeInstanceMandatoryAttributeInPair(
		opts.kv, opts.deploymentID, opts.nodeName, "flavor", "flavorName"); err != nil {
		return instance, err
	}

	instance.AvailabilityZone, err = deployments.GetStringNodeProperty(opts.kv, opts.deploymentID,
		opts.nodeName, "availability_zone", false)
	if err != nil {
		return instance, err
	}
	instance.Region, err = deployments.GetStringNodeProperty(opts.kv, opts.deploymentID, opts.nodeName, "region", false)
	if err != nil {
		return instance, err
	}
	if instance.Region == "" {
		instance.Region = opts.cfg.Infrastructures[infrastructureName].GetStringOrDefault("region", defaultOSRegion)
	}

	instance.KeyPair, err = deployments.GetStringNodeProperty(opts.kv, opts.deploymentID, opts.nodeName, "key_pair", false)
	if err != nil {
		return instance, err
	}

	instance.SecurityGroups = opts.cfg.Infrastructures[infrastructureName].GetStringSlice("default_security_groups")
	secGroups, err := deployments.GetStringNodeProperty(opts.kv, opts.deploymentID, opts.nodeName, "security_groups", false)
	if err != nil {
		return instance, err
	}
	if secGroups != "" {
		for _, secGroup := range strings.Split(strings.NewReplacer("\"", "", "'", "").Replace(secGroups), ",") {
			secGroup = strings.TrimSpace(secGroup)
			instance.SecurityGroups = append(instance.SecurityGroups, secGroup)
		}
	}

	return instance, err
}

func computeInstanceMandatoryAttributeInPair(kv *api.KV, deploymentID, nodeName,
	attr1, attr2 string) (string, string, error) {
	var err error
	var value1, value2 string
	value1, err = deployments.GetStringNodeProperty(kv, deploymentID, nodeName, attr1, false)
	if err != nil {
		return value1, value2, err
	}
	value2, err = deployments.GetStringNodeProperty(kv, deploymentID, nodeName, attr2, false)
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

	kv := opts.kv
	infrastructure := opts.infrastructure
	deploymentID := opts.deploymentID
	nodeName := opts.nodeName
	instanceName := opts.instanceName

	storageKeys, err := deployments.GetRequirementsKeysByTypeForNode(kv, deploymentID, nodeName, "local_storage")
	if err != nil {
		return err
	}
	for _, storagePrefix := range storageKeys {
		requirementIndex := deployments.GetRequirementIndexFromRequirementKey(storagePrefix)
		volumeNodeName, err := deployments.GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return err
		} else if volumeNodeName != "" {
			log.Debugf("Volume attachment required form Volume named %s", volumeNodeName)
			device, err := deployments.GetRelationshipPropertyValueFromRequirement(kv, deploymentID, nodeName, requirementIndex, "device")
			if err != nil {
				return err
			}
			volumeID, err := getVolumeID(ctx, kv, deploymentID, volumeNodeName, instanceName)
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
			attachName := "Vol" + volumeNodeName + "to" + instance.Name
			commons.AddResource(infrastructure, opts.resourceTypes[computeVolumeAttach],
				attachName, &volumeAttach)

			// retrieve the actual used device as depending on the hypervisor it may not be the one we provided, and if there was no devices provided
			// then we can get it back

			key1 := attachName + "-device"
			commons.AddOutput(infrastructure, key1, &commons.Output{
				Value: fmt.Sprintf("${%s.%s.device}",
					opts.resourceTypes[computeVolumeAttach], attachName)})
			outputs[path.Join(instancesPrefix, volumeNodeName, instanceName, "attributes/device")] = key1
			outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID, topologyTree, "relationship_instances", nodeName, requirementIndex, instanceName, "attributes/device")] = key1
			outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID, topologyTree, "relationship_instances", volumeNodeName, requirementIndex, instanceName, "attributes/device")] = key1
		}
	}

	return err
}

func getVolumeID(ctx context.Context, kv *api.KV,
	deploymentID, volumeNodeName, instanceName string) (string, error) {

	var volumeID string
	log.Debugf("Looking for volume_id")
	volumeIDValue, err := deployments.GetNodePropertyValue(kv, deploymentID, volumeNodeName, "volume_id")
	if err != nil {
		return volumeID, err
	}
	if volumeIDValue == nil || volumeIDValue.RawString() == "" {
		resultChan := make(chan string, 1)
		go func() {
			for {
				// ignore errors and retry
				volID, _ := deployments.GetInstanceAttributeValue(kv, deploymentID, volumeNodeName, instanceName, "volume_id")
				// As volumeID is an optional property GetInstanceAttribute then GetProperty
				// may return an empty volumeID so keep checking as long as we have it
				if volID != nil && volID.RawString() != "" {
					resultChan <- volID.RawString()
					return
				}
				select {
				case <-time.After(1 * time.Second):
				case <-ctx.Done():
					// context cancelled, give up!
					return
				}
			}
		}()
		select {
		case volumeID = <-resultChan:
		case <-ctx.Done():
			return volumeID, ctx.Err()

		}
	} else {
		volumeID = volumeIDValue.RawString()
	}

	return volumeID, err
}

// generateInstanceNetworking generates attachments to private networks, associations of
// floating ips and determines the access ip.
// It returns an array of ComputeNetworks that should get assigned to the ComputeInstance and the access ip.
// FIP associations are added as resources to the infrastructure directly.
//
// The access ip is determined by this strategy: If provisioning using floating ips is allowed, the first fip will
// be used. Otherwise the network configured in the admin endpoint will be used. If the network is set to "PRIVATE",
// the default private network of the infrastructure configuration will be used. If the desired network is not
// configured explicitly, it will be added to the configured networks for convenience.
func generateInstanceNetworking(ctx context.Context, opts osInstanceOptions, instanceKey string,
	instance *ComputeInstance, outputs map[string]string) (networks []ComputeNetwork, accessIP string, err error) {

	defAdminEndpointNwName, err := deployments.GetCapabilityPropertyValue(opts.kv, opts.deploymentID, opts.nodeName,
		"endpoint", "network_name")
	if err != nil {
		return nil, "", err
	}
	_ = defAdminEndpointNwName
	adminEndpointNwName := ""
	if defAdminEndpointNwName != nil {
		if strings.EqualFold(defAdminEndpointNwName.RawString(), "private") {
			adminEndpointNwName = opts.cfg.Infrastructures[infrastructureName].GetString("private_network_name")
		} else if defAdminEndpointNwName.RawString() != "" {
			adminEndpointNwName = defAdminEndpointNwName.RawString()
		}
	}
	if adminEndpointNwName == "" {
		return nil, "", errors.Errorf(
			"You should either specify a default private network name using the "+
				`private_network_name" configuration parameter for the "openstack" `+
				`infrastructure or specify a "network_name" property in the "endpoint" `+
				"capability of node %q`", opts.nodeName)
	}

	nwReqKeys, err := deployments.GetRequirementsKeysByTypeForNode(opts.kv, opts.deploymentID, opts.nodeName, "network")
	if err != nil {
		return
	}
	for _, nwReqKey := range nwReqKeys {
		requirementIndex := deployments.GetRequirementIndexFromRequirementKey(nwReqKey)
		var capability string
		capability, err = deployments.GetCapabilityForRequirement(opts.kv, opts.deploymentID, opts.nodeName, requirementIndex)
		if err != nil {
			return
		}

		var networkNodeName string
		networkNodeName, err = deployments.GetTargetNodeForRequirement(opts.kv, opts.deploymentID, opts.nodeName, requirementIndex)
		if err != nil {
			return
		}

		var isFIP bool
		if capability != "" {
			isFIP, err = deployments.IsTypeDerivedFrom(opts.kv, opts.deploymentID, capability, "yorc.capabilities.openstack.FIPConnectivity")
			if err != nil {
				return
			}
		}

		if isFIP {
			fIPAssociateName := "FIP" + instance.Name
			err = computeFloatingIPAddress(ctx, opts, fIPAssociateName, networkNodeName,
				instanceKey, instance, outputs)
			if err != nil {
				return
			}
			if accessIP == "" && opts.cfg.Infrastructures[infrastructureName].GetBool("provisioning_over_fip_allowed") {
				accessIP = fmt.Sprintf("${%s.%s.floating_ip}", opts.resourceTypes[computeFloatingIPAssociate], fIPAssociateName)
			}
		} else {
			var cn ComputeNetwork
			networkIdx := len(networks)
			cn, err = computeNetworkAttributes(ctx, opts, networkNodeName, instanceKey, instance, networkIdx, outputs)
			if err != nil {
				return
			}
			if accessIP == "" && (cn.UUID == adminEndpointNwName || cn.Name == adminEndpointNwName) {
				cn.AccessNetwork = true
				accessIP = fmt.Sprintf("${%s.%s.network.%d.fixed_ip_v4}", opts.resourceTypes[computeInstance], instance.Name, networkIdx)
			}
			networks = append(networks, cn)
		}
	}

	// If the accessIP hasn't been set after all configured networks were added, we add the admin endpoint
	// as a network for convenience.
	if accessIP == "" {
		cn := ComputeNetwork{Name: adminEndpointNwName, AccessNetwork: true}
		accessIP = fmt.Sprintf("${%s.%s.network.%d.fixed_ip_v4}", opts.resourceTypes[computeInstance], instance.Name, len(instance.Networks))
		networks = append(networks, cn)
	}

	return
}

func addResources(ctx context.Context, opts osInstanceOptions, accessIP, instancesKey string, instance *ComputeInstance, outputs map[string]string,
	env *[]string) error {

	commons.AddResource(opts.infrastructure, opts.resourceTypes[computeInstance], instance.Name, instance)

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
	user, privateKey, err := commons.GetConnInfoFromEndpointCredentials(ctx, opts.kv, opts.deploymentID, opts.nodeName)
	if err != nil {
		return err
	}

	return commons.AddConnectionCheckResource(opts.infrastructure, user,
		privateKey, accessIP, instance.Name, env)

}

func computeFloatingIPAddress(ctx context.Context, opts osInstanceOptions,
	fipAssociateName, networkNodeName, instancesKey string,
	instance *ComputeInstance, outputs map[string]string) error {

	kv := opts.kv
	deploymentID := opts.deploymentID
	infrastructure := opts.infrastructure
	nodeName := opts.nodeName
	instanceName := opts.instanceName

	log.Debugf("Looking for Floating IP")
	var floatingIP string
	resultChan := make(chan string, 1)
	go func() {
		for {
			if fip, _ := deployments.GetInstanceCapabilityAttributeValue(kv, deploymentID, networkNodeName, instanceName, "endpoint", "floating_ip_address"); fip != nil && fip.RawString() != "" {
				resultChan <- fip.RawString()
				return
			}

			select {
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				// context cancelled, give up!
				return
			}
		}
	}()
	select {
	case floatingIP = <-resultChan:
	case <-ctx.Done():
		return ctx.Err()
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
	networkNodeName, instancesKey string, instance *ComputeInstance, networkIndex int,
	outputs map[string]string) (ComputeNetwork, error) {

	log.Debugf("Looking for Network id for %q", networkNodeName)
	var networkID string
	resultChan := make(chan string, 1)
	go func() {
		for {
			nID, err := deployments.GetInstanceAttributeValue(
				opts.kv, opts.deploymentID, networkNodeName, opts.instanceName, "network_id")
			if err != nil {
				log.Printf("[Warning] bypassing error while waiting for a network id: %v", err)
			}
			// As networkID is an optional property GetInstanceAttribute then GetProperty
			// may return an empty networkID so keep checking as long as we have it
			if nID != nil && nID.RawString() != "" {
				resultChan <- nID.RawString()
				return
			}
			select {
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				// context cancelled, give up!
				return
			}
		}
	}()
	select {
	case networkID = <-resultChan:
	case <-ctx.Done():
		return ComputeNetwork{}, ctx.Err()
	}
	cn := ComputeNetwork{UUID: networkID, AccessNetwork: false}

	// Provide output for network_name, network_id, addresses attributes
	networkIDKey := opts.nodeName + "-" + opts.instanceName + "-networkID"
	networkNameKey := opts.nodeName + "-" + opts.instanceName + "-networkName"
	networkAddressesKey := opts.nodeName + "-" + opts.instanceName + "-addresses"
	commons.AddOutput(opts.infrastructure, networkIDKey, &commons.Output{
		Value: fmt.Sprintf("${%s.%s.network.%d.uuid}",
			opts.resourceTypes[computeInstance], instance.Name, networkIndex)})
	commons.AddOutput(opts.infrastructure, networkNameKey, &commons.Output{
		Value: fmt.Sprintf("${%s.%s.network.%d.name}",
			opts.resourceTypes[computeInstance], instance.Name, networkIndex)})
	commons.AddOutput(opts.infrastructure, networkAddressesKey, &commons.Output{
		Value: fmt.Sprintf("[ ${%s.%s.network.%d.fixed_ip_v4} ]",
			opts.resourceTypes[computeInstance], instance.Name, networkIndex)})

	prefix := path.Join(instancesKey, opts.instanceName, "attributes/networks", strconv.Itoa(networkIndex))
	outputs[path.Join(prefix, "network_name")] = networkNameKey
	outputs[path.Join(prefix, "network_id")] = networkIDKey
	outputs[path.Join(prefix, "addresses")] = networkAddressesKey
	return cn, nil
}
