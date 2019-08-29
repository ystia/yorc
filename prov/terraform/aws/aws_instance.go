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

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/prov/terraform/commons"
)

func (g *awsGenerator) generateAWSInstance(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName, instanceName string, infrastructure *commons.Infrastructure, outputs map[string]string, env *[]string) error {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
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
	image, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "image_id")
	if err != nil {
		return err
	}
	if image == nil || image.RawString() == "" {
		return errors.Errorf("Missing mandatory parameter 'image_id' node type for %s", nodeName)
	}
	instance.ImageID = image.RawString()

	// instance_type is mandatory
	instanceType, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "instance_type")
	if err != nil {
		return err
	}
	if instanceType == nil || instanceType.RawString() == "" {
		return errors.Errorf("Missing mandatory parameter 'instance_type' node type for %s", nodeName)
	}
	instance.InstanceType = instanceType.RawString()

	// key_name is mandatory
	keyName, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "key_name")
	if err != nil {
		return err
	}
	if keyName == nil || keyName.RawString() == "" {
		return errors.Errorf("Missing mandatory parameter 'key_name' node type for %s", nodeName)
	}
	instance.KeyName = keyName.RawString()

	// security_groups needs to contain a least one occurrence
	secGroups, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "security_groups")
	if err != nil {
		return err
	}
	if secGroups == nil || secGroups.RawString() == "" {
		return errors.Errorf("Missing mandatory parameter 'security_groups' node type for %s", nodeName)
	}

	for _, secGroup := range strings.Split(strings.NewReplacer("\"", "", "'", "").Replace(secGroups.RawString()), ",") {
		secGroup = strings.TrimSpace(secGroup)
		instance.SecurityGroups = append(instance.SecurityGroups, secGroup)
	}

	// Get connection info (user, private key)
	user, privateKey, err := commons.GetConnInfoFromEndpointCredentials(ctx, kv, deploymentID, nodeName)
	if err != nil {
		return err
	}

	// Check optional provided Elastic IPs
	eips, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "elastic_ips")
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
	s, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "delete_volume_on_termination")
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
	availabilityZone, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "availability_zone")
	if err != nil {
		return err
	}
	if availabilityZone != nil {
		instance.AvailabilityZone = availabilityZone.RawString()
	}

	// Optional property to add the compute to a logical grouping of instances
	placementGroup, err := deployments.GetNodePropertyValue(kv, deploymentID, nodeName, "placement_group")
	if err != nil {
		return err
	}
	if placementGroup != nil {
		instance.PlacementGroup = placementGroup.RawString()
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
		isElasticIP, _, err = deployments.HasAnyRequirementFromNodeType(kv, deploymentID, nodeName, "network", "yorc.nodes.aws.PublicNetwork")
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
	if err = commons.AddConnectionCheckResource(infrastructure, user, privateKey, accessIP, instance.Tags.Name, env); err != nil {
		return err
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
