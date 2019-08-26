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

package google

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/helper/sshutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/prov/terraform/commons"
)

func (g *googleGenerator) generateComputeInstance(ctx context.Context, kv *api.KV,
	cfg config.Configuration, deploymentID, nodeName, instanceName string, instanceID int,
	infrastructure *commons.Infrastructure,
	outputs map[string]string, env *[]string) error {

	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType != "yorc.nodes.google.Compute" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}

	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID,
		"topology", "instances")
	instancesKey := path.Join(instancesPrefix, nodeName)

	instance := ComputeInstance{}

	// Must be a match of regex '(?:[a-z](?:[-a-z0-9]{0,61}[a-z0-9])?)'
	instance.Name = strings.ToLower(getResourcesPrefix(cfg, deploymentID) + nodeName + "-" + instanceName)
	instance.Name = strings.Replace(instance.Name, "_", "-", -1)

	// Getting string parameters
	var imageProject, imageFamily, image, serviceAccount string

	stringParams := []struct {
		pAttr        *string
		propertyName string
		mandatory    bool
	}{
		{&instance.MachineType, "machine_type", true},
		{&instance.Zone, "zone", true},
		{&imageProject, "image_project", false},
		{&imageFamily, "image_family", false},
		{&image, "image", false},
		{&instance.Description, "description", false},
		{&serviceAccount, "service_account", false},
	}

	for _, stringParam := range stringParams {
		if *stringParam.pAttr, err = deployments.GetStringNodeProperty(kv, deploymentID, nodeName,
			stringParam.propertyName, stringParam.mandatory); err != nil {
			return err
		}
	}

	// Define the boot disk from image settings
	var bootImage string
	if imageProject != "" {
		bootImage = imageProject
		if image != "" {
			bootImage = bootImage + "/" + image
		} else if imageFamily != "" {
			bootImage = bootImage + "/" + imageFamily
		} else {
			// Unexpected image project without a family or image
			return errors.Errorf("Exepected an image or family for image project %s on %s", imageProject, nodeName)
		}
	} else if image != "" {
		bootImage = image
	} else {
		bootImage = imageFamily
	}

	var bootDisk BootDisk
	if bootImage != "" {
		bootDisk.InitializeParams = InitializeParams{Image: bootImage}
	}
	instance.BootDisk = bootDisk

	// Network definition
	var noAddress bool
	if noAddress, err = deployments.GetBooleanNodeProperty(kv, deploymentID, nodeName, "no_address"); err != nil {
		return err
	}

	// Define if a private network access is required
	var netInterfaces []NetworkInterface
	reqPrivateNetwork, _, err := deployments.HasAnyRequirementFromNodeType(kv, deploymentID, nodeName, "network", "yorc.nodes.google.PrivateNetwork")
	if err != nil {
		return err
	}
	// Check for subnet otherwise
	if !reqPrivateNetwork {
		reqPrivateNetwork, _, err = deployments.HasAnyRequirementFromNodeType(kv, deploymentID, nodeName, "network", "yorc.nodes.google.Subnetwork")
		if err != nil {
			return err
		}
	}
	if reqPrivateNetwork {
		netInterfaces, err = addPrivateNetworkInterfaces(ctx, kv, deploymentID, nodeName)
		if err != nil {
			return err
		}
	} else {
		// Create a default private network interface
		netInterfaces = append(netInterfaces, NetworkInterface{Network: "default"})
	}

	// Define an external access if there will be an external IP address
	if !noAddress {
		hasStaticAddressReq, addressNode, err := deployments.HasAnyRequirementCapability(kv, deploymentID, nodeName, "assignment", "yorc.capabilities.Assignable")
		if err != nil {
			return err
		}
		var externalAddress string
		// External IP address can be static if required
		if hasStaticAddressReq {
			// Address Lookup
			externalAddress, err = deployments.LookupInstanceAttributeValue(ctx, kv, deploymentID, addressNode, instanceName, "ip_address")
			if err != nil {
				return err
			}
		}
		// else externalAddress is empty, which means an ephemeral external IP
		// address will be assigned to the instance
		accessConfig := AccessConfig{NatIP: externalAddress}
		netInterfaces[0].AccessConfigs = []AccessConfig{accessConfig}
	}
	instance.NetworkInterfaces = netInterfaces

	// Scheduling definition
	var preemptible bool
	if preemptible, err = deployments.GetBooleanNodeProperty(kv, deploymentID, nodeName, "preemptible"); err != nil {
		return err
	}

	if preemptible {
		instance.Scheduling = Scheduling{Preemptible: true}
	}

	// Get list of strings parameters
	var scopes []string
	if scopes, err = deployments.GetStringArrayNodeProperty(kv, deploymentID, nodeName, "scopes"); err != nil {
		return err
	}

	if serviceAccount != "" || len(scopes) > 0 {
		// Adding a service account section, where scopes can't be empty
		if len(scopes) == 0 {
			scopes = []string{"cloud-platform"}
		}
		configuredAccount := ServiceAccount{serviceAccount, scopes}
		instance.ServiceAccounts = []ServiceAccount{configuredAccount}
	}

	if instance.Tags, err = deployments.GetStringArrayNodeProperty(kv, deploymentID, nodeName, "tags"); err != nil {
		return err
	}

	// Get list of key/value pairs parameters
	if instance.Labels, err = deployments.GetKeyValuePairsNodeProperty(kv, deploymentID, nodeName, "labels"); err != nil {
		return err
	}

	if instance.Metadata, err = deployments.GetKeyValuePairsNodeProperty(kv, deploymentID, nodeName, "metadata"); err != nil {
		return err
	}

	// Get connection info (user, private key)
	user, privateKey, err := commons.GetConnInfoFromEndpointCredentials(ctx, kv, deploymentID, nodeName)
	if err != nil {
		return err
	}

	// Add additional Scratch disks
	scratchDisks, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "scratch_disks")
	if err != nil {
		return err
	}

	if scratchDisks != nil && scratchDisks.RawString() != "" {
		list, ok := scratchDisks.Value.([]interface{})
		if !ok {
			return errors.New("failed to retrieve scratch disk Tosca Value: not expected type")
		}
		instance.ScratchDisks = make([]ScratchDisk, 0)
		for _, n := range list {
			v, ok := n.(map[string]interface{})
			if !ok {
				return errors.New("failed to retrieve scratch disk map: not expected type")
			}
			for _, val := range v {
				i, ok := val.(string)
				if !ok {
					return errors.New("failed to retrieve scratch disk interface value: not expected type")
				}
				scratch := ScratchDisk{Interface: i}
				instance.ScratchDisks = append(instance.ScratchDisks, scratch)
			}
		}
	}

	// Add the compute instance
	commons.AddResource(infrastructure, "google_compute_instance", instance.Name, &instance)

	// Attach Persistent disks
	devices, err := addAttachedDisks(ctx, cfg, kv, deploymentID, nodeName, instanceName, instance.Name, infrastructure, outputs)
	if err != nil {
		return err
	}

	// Define the private IP address using the value exported by Terraform
	privateIP := fmt.Sprintf("${google_compute_instance.%s.network_interface.0.address}",
		instance.Name)

	privateIPKey := nodeName + "-" + instanceName + "-privateIP"
	commons.AddOutput(infrastructure, privateIPKey, &commons.Output{Value: privateIP})
	outputs[path.Join(instancesKey, instanceName, "/attributes/private_address")] = privateIPKey

	// Define the public IP using the value exported by Terraform
	// except if it was specified the instance shouldn't have a public address
	var accessIP, accessIPKey string
	if noAddress {
		accessIP = privateIP
		accessIPKey = privateIPKey
	} else {
		accessIP = fmt.Sprintf("${google_compute_instance.%s.network_interface.0.access_config.0.assigned_nat_ip}",
			instance.Name)

		publicIPKey := nodeName + "-" + instanceName + "-publicIP"
		accessIPKey = publicIPKey
		commons.AddOutput(infrastructure, publicIPKey, &commons.Output{Value: accessIP})

		outputs[path.Join(instancesKey, instanceName, "/attributes/public_address")] = publicIPKey
		outputs[path.Join(instancesKey, instanceName, "/attributes/public_ip_address")] = publicIPKey
	}

	// ip_adress attribute and endpoint capability
	outputs[path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address")] = accessIPKey
	outputs[path.Join(instancesKey, instanceName, "/attributes/ip_address")] = accessIPKey

	// Add Connection check
	if err = commons.AddConnectionCheckResource(infrastructure, user, privateKey, accessIP, instance.Name, env); err != nil {
		return err
	}

	// Retrieve devices
	if len(devices) > 0 {
		if err = handleDeviceAttributes(ctx, cfg, infrastructure, &instance, devices, user, privateKey, accessIP); err != nil {
			return err
		}
	}

	return nil
}

func handleDeviceAttributes(ctx context.Context, cfg config.Configuration, infrastructure *commons.Infrastructure, instance *ComputeInstance, devices []string, user string, privateKey *sshutil.PrivateKey, accessIP string) error {
	var env map[string]interface{}
	// Retrieve devices {
	for _, dev := range devices {
		devResource := commons.Resource{}

		// Remote exec to retrieve the logical device for google device ID and to redirect stdout to file
		re := commons.RemoteExec{Inline: []string{fmt.Sprintf("readlink -f /dev/disk/by-id/%s > %s", dev, dev)},
			Connection: &commons.Connection{User: user, Host: accessIP, PrivateKey: "${var.private_key}"}}
		devResource.Provisioners = make([]map[string]interface{}, 0)
		provMap := make(map[string]interface{})
		provMap["remote-exec"] = re
		devResource.Provisioners = append(devResource.Provisioners, provMap)
		devResource.DependsOn = []string{
			fmt.Sprintf("null_resource.%s", instance.Name+"-ConnectionCheck"),
			fmt.Sprintf("google_compute_attached_disk.%s", dev),
		}
		commons.AddResource(infrastructure, "null_resource", fmt.Sprintf("%s-GetDevice-%s", instance.Name, dev), &devResource)

		// local exec to scp the stdout file locally (use ssh-agent to make it if allowed by config)
		var scpCommand string
		sshAgent, ok := commons.SSHAgentFromContext(ctx)
		if ok {
			scpCommand = fmt.Sprintf("scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null %s@%s:~/%s %s", user, accessIP, dev, dev)
			env = make(map[string]interface{})
			env["SSH_AUTH_SOCK"] = sshAgent.Socket
		} else {
			if privateKey.Path == "" {
				return errors.New("trying to get GCP volumes devices with an ssh private key that is not stored on disk, this is possible only with an ssh agent, unfortunately ssh-agent is currently disabled by configuration")
			}
			scpCommand = fmt.Sprintf("scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i %s %s@%s:~/%s %s", privateKey.Path, user, accessIP, dev, dev)
		}
		loc := commons.LocalExec{
			Command:     scpCommand,
			Environment: env,
		}
		locMap := make(map[string]interface{})
		locMap["local-exec"] = loc
		locResource := commons.Resource{}
		locResource.Provisioners = append(locResource.Provisioners, locMap)
		locResource.DependsOn = []string{fmt.Sprintf("null_resource.%s", fmt.Sprintf("%s-GetDevice-%s", instance.Name, dev))}
		commons.AddResource(infrastructure, "null_resource", fmt.Sprintf("%s-CopyOut-%s", instance.Name, dev), &locResource)

		// Remote exec to cleanup  created file
		cleanResource := commons.Resource{}
		re = commons.RemoteExec{Inline: []string{fmt.Sprintf("rm -f %s", dev)},
			Connection: &commons.Connection{User: user, Host: accessIP, PrivateKey: "${var.private_key}"}}
		cleanResource.Provisioners = make([]map[string]interface{}, 0)
		m := make(map[string]interface{})
		m["remote-exec"] = re
		cleanResource.Provisioners = append(devResource.Provisioners, m)
		cleanResource.DependsOn = []string{fmt.Sprintf("null_resource.%s", fmt.Sprintf("%s-CopyOut-%s", instance.Name, dev))}
		commons.AddResource(infrastructure, "null_resource", fmt.Sprintf("%s-cleanup-%s", instance.Name, dev), &cleanResource)

		consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{}}
		consulKeys.DependsOn = []string{fmt.Sprintf("null_resource.%s", fmt.Sprintf("%s-CopyOut-%s", instance.Name, dev))}
	}
	return nil
}

func addAttachedDisks(ctx context.Context, cfg config.Configuration, kv *api.KV, deploymentID, nodeName, instanceName, computeName string, infrastructure *commons.Infrastructure, outputs map[string]string) ([]string, error) {
	devices := make([]string, 0)

	storageKeys, err := deployments.GetRequirementsKeysByTypeForNode(kv, deploymentID, nodeName, "local_storage")
	if err != nil {
		return nil, err
	}
	for _, storagePrefix := range storageKeys {
		requirementIndex := deployments.GetRequirementIndexFromRequirementKey(storagePrefix)
		volumeNodeName, err := deployments.GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return nil, err
		}

		log.Debugf("Volume attachment required form Volume named %s", volumeNodeName)

		zone, err := deployments.GetStringNodeProperty(kv, deploymentID, volumeNodeName, "zone", true)
		if err != nil {
			return nil, err
		}

		modeValue, err := deployments.GetRelationshipPropertyValueFromRequirement(kv, deploymentID, nodeName, requirementIndex, "mode")
		if err != nil {
			return nil, err
		}

		volumeIDValue, err := deployments.GetNodePropertyValue(kv, deploymentID, volumeNodeName, "volume_id")
		if err != nil {
			return nil, err
		}
		var volumeID string
		if volumeIDValue == nil || volumeIDValue.RawString() == "" {
			// Lookup for attribute volume_id
			volumeID, err = deployments.LookupInstanceAttributeValue(ctx, kv, deploymentID, volumeNodeName, instanceName, "volume_id")
			if err != nil {
				return nil, err
			}

		} else {
			volumeID = volumeIDValue.RawString()
		}

		attachedDisk := &ComputeAttachedDisk{
			Disk:     volumeID,
			Instance: fmt.Sprintf("${google_compute_instance.%s.name}", computeName),
			Zone:     zone,
		}
		if modeValue != nil && modeValue.RawString() != "" {
			attachedDisk.Mode = modeValue.RawString()
		}

		attachName := strings.ToLower(getResourcesPrefix(cfg, deploymentID) + volumeNodeName + "-" + instanceName + "-to-" + nodeName + "-" + instanceName)
		attachName = strings.Replace(attachName, "_", "-", -1)
		// attachName is used as device name to retrieve device attribute as logical volume name
		attachedDisk.DeviceName = attachName

		// Provide file outputs for device attributes which can't be resolved with Terraform
		device := fmt.Sprintf("google-%s", attachName)
		commons.AddResource(infrastructure, "google_compute_attached_disk", device, attachedDisk)
		outputDeviceVal := commons.FileOutputPrefix + device
		instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances")
		outputs[path.Join(instancesPrefix, volumeNodeName, instanceName, "attributes/device")] = outputDeviceVal
		outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "relationship_instances", nodeName, requirementIndex, instanceName, "attributes/device")] = outputDeviceVal
		outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "relationship_instances", volumeNodeName, requirementIndex, instanceName, "attributes/device")] = outputDeviceVal
		// Add device
		devices = append(devices, device)
	}
	return devices, nil
}

func addPrivateNetworkInterfaces(ctx context.Context, kv *api.KV, deploymentID, nodeName string) ([]NetworkInterface, error) {
	var netInterfaces []NetworkInterface

	// Check if subnets have been specified by user into network relationship
	storageKeys, err := deployments.GetRequirementsKeysByTypeForNode(kv, deploymentID, nodeName, "network")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to add network interfaces for deploymentID:%q, nodeName:%q", deploymentID, nodeName)
	}
	for _, storagePrefix := range storageKeys {
		requirementIndex := deployments.GetRequirementIndexFromRequirementKey(storagePrefix)

		networkNodeName, err := deployments.GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add network interfaces for deploymentID:%q, nodeName:%q", deploymentID, nodeName)
		}

		// Check if node is network or subnet
		netType, err := deployments.GetNodeType(kv, deploymentID, networkNodeName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add network interfaces for deploymentID:%q, nodeName:%q", deploymentID, nodeName)
		}
		switch netType {
		case "yorc.nodes.google.Subnetwork":
			subnet, err := deployments.LookupInstanceAttributeValue(ctx, kv, deploymentID, networkNodeName, "0", "subnetwork_name")
			if err != nil {
				return nil, errors.Wrapf(err, "failed to add network interfaces for deploymentID:%q, nodeName:%q, networkName:%q", deploymentID, nodeName, networkNodeName)
			}
			log.Debugf("add network interface with sub-network property:%s", subnet)
			netInterfaces = append(netInterfaces, NetworkInterface{Subnetwork: subnet})
		case "yorc.nodes.google.PrivateNetwork":
			// We mention subnet if provided by network relationship property
			subRaw, err := deployments.GetRelationshipPropertyValueFromRequirement(kv, deploymentID, nodeName, requirementIndex, "subnet")
			if err != nil {
				return nil, err
			}
			if subRaw != nil && subRaw.RawString() != "" {
				log.Debugf("add network interface with user-specified sub-network property:%s", subRaw.RawString())
				netInterfaces = append(netInterfaces, NetworkInterface{Subnetwork: subRaw.RawString()})
			} else { // we mention the network
				network, err := deployments.LookupInstanceAttributeValue(ctx, kv, deploymentID, networkNodeName, "0", "network_name")
				if err != nil {
					return nil, errors.Wrapf(err, "failed to add network interfaces for deploymentID:%q, nodeName:%q, networkName:%q", deploymentID, nodeName, networkNodeName)
				}
				log.Debugf("add network interface with network property:%s", network)
				netInterfaces = append(netInterfaces, NetworkInterface{Network: network})
			}
		default:
			return nil, errors.Errorf("type:%q is not handled for compute network interface addition", netType)
		}
	}
	return netInterfaces, nil
}
