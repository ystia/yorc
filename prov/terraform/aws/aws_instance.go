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

package aws

import (
	"context"
	"fmt"
	"net"
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

func (g *awsGenerator) generateAWSInstance(ctx context.Context, cfg config.Configuration, deploymentID, nodeName, instanceName string, infrastructure *commons.Infrastructure, outputs map[string]string, env *[]string) error {
	nodeType, err := deployments.GetNodeType(ctx, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType != "yorc.nodes.aws.Compute" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}
	instance := ComputeInstance{}
	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances")
	instancesKey := path.Join(instancesPrefix, nodeName)

	instance.Tags.Name = cfg.ResourcesPrefix + nodeName + "-" + instanceName

	// image_id is mandatory
	image, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "image_id")
	if err != nil {
		return err
	}
	if image == nil || image.RawString() == "" {
		return errors.Errorf("Missing mandatory parameter 'image_id' node type for %s", nodeName)
	}
	instance.ImageID = image.RawString()

	// instance_type is mandatory
	instanceType, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "instance_type")
	if err != nil {
		return err
	}
	if instanceType == nil || instanceType.RawString() == "" {
		return errors.Errorf("Missing mandatory parameter 'instance_type' node type for %s", nodeName)
	}
	instance.InstanceType = instanceType.RawString()

	// key_name is mandatory
	keyName, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "key_name")
	if err != nil {
		return err
	}
	if keyName == nil || keyName.RawString() == "" {
		return errors.Errorf("Missing mandatory parameter 'key_name' node type for %s", nodeName)
	}
	instance.KeyName = keyName.RawString()

	secGroups, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "security_groups")
	if err != nil {
		return err
	}

	if secGroups.RawString() != "" {
		for _, secGroup := range strings.Split(strings.NewReplacer("\"", "", "'", "").Replace(secGroups.RawString()), ",") {
			secGroup = strings.TrimSpace(secGroup)
			instance.SecurityGroups = append(instance.SecurityGroups, secGroup)
		}
	}

	// Get connection info (user, private key)
	user, privateKey, err := commons.GetConnInfoFromEndpointCredentials(ctx, deploymentID, nodeName)
	if err != nil {
		return err
	}

	// Check optional provided Elastic IPs
	eips, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "elastic_ips")
	if err != nil {
		return err
	}
	if eips != nil && eips.RawString() != "" {
		for _, eip := range strings.Split(strings.NewReplacer("\"", "", "'", "").Replace(eips.RawString()), ",") {
			eip = strings.TrimSpace(eip)
			if net.ParseIP(eip) == nil {
				return errors.Errorf("Malformed provided Elastic IP: %s", eip)
			}
			instance.ElasticIps = append(instance.ElasticIps, eip)
		}
	}

	// Check if the root block device must be deleted on termination
	deleteVolumeOnTermination := true // Default is deleting root block device on compute termination
	s, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "delete_volume_on_termination")
	if err != nil {
		return err
	}
	if s != nil && s.RawString() != "" {
		deleteVolumeOnTermination, err = strconv.ParseBool(s.RawString())
		if err != nil {
			return err
		}
	}
	instance.RootBlockDevice = BlockDevice{DeleteOnTermination: deleteVolumeOnTermination}

	// Optional property to select availability zone of the compute instance
	availabilityZone, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "availability_zone")
	if err != nil {
		return err
	}
	if availabilityZone != nil {
		instance.AvailabilityZone = availabilityZone.RawString()
	}

	// Optional property to add the compute to a logical grouping of instances
	placementGroup, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "placement_group")
	if err != nil {
		return err
	}
	if placementGroup != nil {
		instance.PlacementGroup = placementGroup.RawString()
	}

	// Network
	subnetID, err := deployments.GetNodePropertyValue(ctx, deploymentID, nodeName, "subnet_id")
	if err != nil {
		return err
	}
	if subnetID != nil {
		instance.SubnetID = subnetID.RawString()
	}

	// VPC
	hasVPCNetwork, _, err := deployments.HasAnyRequirementFromNodeType(ctx, deploymentID, nodeName, "network", "yorc.nodes.aws.VPC")
	if err != nil {
		return err
	}
	if hasVPCNetwork {
		networkRequirements, err := deployments.GetRequirementsByTypeForNode(ctx, deploymentID, nodeName, "network")
		if err != nil {
			return err
		}

		i := 0
		for _, networkReq := range networkRequirements {
			networkInterface := &NetworkInterface{}
			networkInterface.SecurityGroups = make([]string, 1)

			if networkReq.Relationship == "tosca.relationships.Network" {
				defaultSubnetID, err := deployments.LookupInstanceAttributeValue(ctx, deploymentID, networkReq.RequirementAssignment.Node, instanceName, "default_subnet_id")
				if err != nil {
					return err
				}
				networkInterface.SubnetID = defaultSubnetID
				defaultSecurityGroup, err := deployments.LookupInstanceAttributeValue(ctx, deploymentID, networkReq.RequirementAssignment.Node, instanceName, "default_security_group")
				if err != nil {
					return err
				}
				networkInterface.SecurityGroups = append(networkInterface.SecurityGroups, defaultSecurityGroup)

			} else if networkReq.Relationship == "yorc.relationships.aws.Network" {
				networkInterface.SubnetID, err = deployments.LookupInstanceAttributeValue(ctx, deploymentID, networkReq.RequirementAssignment.Node, instanceName, "subnet_id")
				if err != nil {
					return err
				}
				securityGroup, err := deployments.LookupInstanceAttributeValue(ctx, deploymentID, networkReq.RequirementAssignment.Node, instanceName, "subnet_id")
				if err != nil {
					return err
				}
				networkInterface.SecurityGroups = append(networkInterface.SecurityGroups, securityGroup)
			}

			name := strings.ToLower("network-inteface-" + strconv.Itoa(i))
			name = strings.Replace(strings.ToLower(name), "_", "-", -1)

			// First interface will be considered the network interface of the Compute Instance
			if i == 0 {
				instance.NetworkInterface = map[string]string{
					"network_interface_id": "${aws_network_interface." + name + ".id}",
					"device_index":         strconv.Itoa(i),
				}
			} else {
				// Others interfaces are considered as attachment to the Compute Instance
				networkInterface.Attachment = map[string]string{
					"instance":     "${aws_instance." + instance.Tags.Name + ".id}",
					"device_index": strconv.Itoa(i),
				}
			}

			// No security groups on instance level when defining customs ENI, it will use a default one created by the VPC
			instance.SecurityGroups = nil
			commons.AddResource(infrastructure, "aws_network_interface", name, networkInterface)
			i++
		}
	}

	// Subnet
	hasSubnetNetwork, _, err := deployments.HasAnyRequirementFromNodeType(ctx, deploymentID, nodeName, "network", "yorc.nodes.aws.Subnet")
	if err != nil {
		return err
	}
	if hasSubnetNetwork && !hasVPCNetwork {
		networkRequirements, err := deployments.GetRequirementsByTypeForNode(ctx, deploymentID, nodeName, "network")
		if err != nil {
			return err
		}

		i := 0
		for _, networkReq := range networkRequirements {
			networkInterface := &NetworkInterface{}
			networkInterface.SecurityGroups = make([]string, 1)

			if networkReq.Relationship == "tosca.relationships.Network" {
				subnetID, err := deployments.LookupInstanceAttributeValue(ctx, deploymentID, networkReq.RequirementAssignment.Node, instanceName, "subnet_id")
				if err != nil {
					return err
				}
				networkInterface.SubnetID = subnetID

				networkInterface.SecurityGroups = instance.SecurityGroups

			}

			name := strings.ToLower("network-inteface-" + strconv.Itoa(i))
			name = strings.Replace(strings.ToLower(name), "_", "-", -1)

			// First interface will be considered the network interface of the Compute Instance
			if i == 0 {
				instance.NetworkInterface = map[string]string{
					"network_interface_id": "${aws_network_interface." + name + ".id}",
					"device_index":         strconv.Itoa(i),
				}
			} else {
				// Others interfaces are considered as attachment to the Compute Instance
				networkInterface.Attachment = map[string]string{
					"instance":     "${aws_instance." + instance.Tags.Name + ".id}",
					"device_index": strconv.Itoa(i),
				}
			}

			// No security groups on instance level when defining customs ENI, it will use a default one created by the VPC
			instance.SecurityGroups = nil
			commons.AddResource(infrastructure, "aws_network_interface", name, networkInterface)
			i++
		}
	}

	// Add the AWS instance
	commons.AddResource(infrastructure, "aws_instance", instance.Tags.Name, &instance)

	// Provide private_address attribute output
	privateIPKey := nodeName + "-" + instanceName + "-privateIP"
	commons.AddOutput(infrastructure, privateIPKey, &commons.Output{Value: fmt.Sprintf("${aws_instance.%s.private_ip}", instance.Tags.Name)})
	outputs[path.Join(instancesKey, instanceName, "/attributes/private_address")] = privateIPKey

	// Provide specific DNS attribute attribute
	publicDNSKey := nodeName + "-" + instanceName + "-publicDNS"
	commons.AddOutput(infrastructure, publicDNSKey, &commons.Output{Value: fmt.Sprintf("${aws_instance.%s.public_dns}", instance.Tags.Name)})
	outputs[path.Join(instancesKey, instanceName, "/attributes/public_dns")] = publicDNSKey

	// If any Elastic IP is provided without any network requirement, EIP is anyway associated to the compute instance
	// Check existing network requirement otherwise
	var isElasticIP = len(instance.ElasticIps) > 0
	if !isElasticIP {
		isElasticIP, _, err = deployments.HasAnyRequirementFromNodeType(ctx, deploymentID, nodeName, "network", "yorc.nodes.aws.PublicNetwork")
		if err != nil {
			return err
		}
	}

	// Add the EIP Association
	// The scalability is handled with a list of EIPs in the limit of EIPs provided
	var eipAssociationName string
	if isElasticIP {
		if len(instance.ElasticIps) > 0 {
			// Find the associated EIP in the list in function of the association index
			ind := getEIPAssociationIndex(infrastructure, cfg.ResourcesPrefix+nodeName)
			if ind < len(instance.ElasticIps) {
				eipAssociationName = associateEIP(infrastructure, &instance, instance.ElasticIps[ind])
			} else {
				// Not enough provided EIPs, rest of EIP is generated
				eipAssociationName = associateEIP(infrastructure, &instance, "")
			}
		} else {
			// No provided EIP
			eipAssociationName = associateEIP(infrastructure, &instance, "")
		}
	}

	// Define the public IP according to the EIP association if one exists
	var accessIP string
	if eipAssociationName != "" {
		accessIP = fmt.Sprintf("${aws_eip_association.%s.public_ip}", eipAssociationName)
	} else {
		accessIP = fmt.Sprintf("${aws_instance.%s.public_ip}", instance.Tags.Name)
	}

	// public_address/ ip_address output attributes
	accessIPKey := nodeName + "-" + instanceName + "-publicIP"
	commons.AddOutput(infrastructure, accessIPKey, &commons.Output{Value: accessIP})
	outputs[path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address")] = accessIPKey
	outputs[path.Join(instancesKey, instanceName, "/attributes/ip_address")] = accessIPKey

	outputs[path.Join(instancesKey, instanceName, "/attributes/public_address")] = accessIPKey
	outputs[path.Join(instancesKey, instanceName, "/attributes/public_ip_address")] = accessIPKey

	// Add Connection check
	if err = commons.AddConnectionCheckResource(ctx, deploymentID, nodeName, infrastructure, user, privateKey, accessIP, instance.Tags.Name, env); err != nil {
		return err
	}

	return addAttachedDisks(ctx, cfg, deploymentID, nodeName, instanceName, instance.Tags.Name, infrastructure, outputs)
}

func addAttachedDisks(ctx context.Context, cfg config.Configuration, deploymentID, nodeName, instanceName, instanceTagName string, infrastructure *commons.Infrastructure, outputs map[string]string) error {
	storageReqs, err := deployments.GetRequirementsByTypeForNode(ctx, deploymentID, nodeName, "local_storage")
	if err != nil {
		return err
	}

	for _, storageReq := range storageReqs {
		volumeNodeName := storageReq.Node

		log.Debugf("Volume attachment required form Volume named %s", volumeNodeName)

		volumeID, err := deployments.LookupInstanceAttributeValue(ctx, deploymentID, volumeNodeName, instanceName, "volume_id")
		if err != nil {
			return err
		}

		deviceNameVal, err := deployments.GetInstanceAttributeValue(ctx, deploymentID, volumeNodeName, instanceName, "device")
		if err != nil || deviceNameVal == nil {
			return errors.Wrapf(err, "Can't find the required device name for deploymentID:%q, volume name:%q, instance name:%q", deploymentID, volumeNodeName, instanceName)
		}
		deviceName := deviceNameVal.RawString()

		attachedDisk := &VolumeAttachment{
			DeviceName: deviceName,
			InstanceID: fmt.Sprintf("${aws_instance.%s.id}", instanceTagName),
			VolumeID:   volumeID,
		}

		attachName := strings.ToLower(volumeNodeName + "-" + instanceName + "-to-" + nodeName + "-" + instanceName)
		attachName = strings.Replace(attachName, "_", "-", -1)
		commons.AddResource(infrastructure, "aws_volume_attachment", attachName, attachedDisk)

		// Terraform device_name output
		deviceKey := attachName + ".device_name"
		deviceValue := fmt.Sprintf("${aws_volume_attachment.%s.device_name}", attachName)
		commons.AddOutput(infrastructure, deviceKey, &commons.Output{Value: deviceValue})
		instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances")
		outputs[path.Join(instancesPrefix, volumeNodeName, instanceName, "attributes/device")] = deviceKey
		outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "relationship_instances", nodeName, storageReq.Index, instanceName, "attributes/device")] = deviceKey
		outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "relationship_instances", volumeNodeName, storageReq.Index, instanceName, "attributes/device")] = deviceKey
	}
	return nil
}

func associateEIP(infrastructure *commons.Infrastructure, instance *ComputeInstance, providedEIP string) string {
	// Add the association EIP/Instance
	eipAssociation := ElasticIPAssociation{InstanceID: fmt.Sprintf("${aws_instance.%s.id}", instance.Tags.Name)}
	log.Printf("Adding ElasticIP association for instance name:%s", instance.Tags.Name)
	if providedEIP != "" {
		// Use existing EIP
		log.Printf("The following Elastic IP has been provided:%s", providedEIP)
		eipAssociation.PublicIP = providedEIP
	} else {
		// Add the EIP
		log.Printf("Adding ElasticIP for instance name:%s", instance.Tags.Name)
		elasticIPName := "EIP-" + instance.Tags.Name
		elasticIP := ElasticIP{}
		commons.AddResource(infrastructure, "aws_eip", elasticIPName, &elasticIP)

		eipAssociation.AllocationID = fmt.Sprintf("${aws_eip.%s.id}", elasticIPName)
	}

	eipAssociationName := "EIPAssoc-" + instance.Tags.Name
	commons.AddResource(infrastructure, "aws_eip_association", eipAssociationName, &eipAssociation)
	return eipAssociationName
}

func getEIPAssociationIndex(infrastructure *commons.Infrastructure, nodeNamePrefix string) int {
	var ind int
	if infrastructure.Resource["aws_eip_association"] != nil {
		instances := infrastructure.Resource["aws_eip_association"].(map[string]interface{})
		for key := range instances {
			if strings.HasPrefix(key, "EIPAssoc-"+nodeNamePrefix) {
				ind++
			}

		}
	}
	return ind

}
