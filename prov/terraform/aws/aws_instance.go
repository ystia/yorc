package aws

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
)

func (g *awsGenerator) generateAWSInstance(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName, instanceName string, infrastructure *commons.Infrastructure, outputs map[string]string) error {
	nodeType, err := deployments.GetNodeType(kv, deploymentID, nodeName)
	if err != nil {
		return err
	}
	if nodeType != "janus.nodes.aws.Compute" {
		return errors.Errorf("Unsupported node type for %q: %s", nodeName, nodeType)
	}
	instance := ComputeInstance{}
	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "instances")
	instancesKey := path.Join(instancesPrefix, nodeName)

	instance.Tags.Name = cfg.ResourcesPrefix + nodeName + "-" + instanceName

	// image_id is mandatory
	var image string
	if _, image, err = deployments.GetNodeProperty(kv, deploymentID, nodeName, "image_id"); err != nil {
		return err
	} else if image == "" {
		return errors.Errorf("Missing mandatory parameter 'image_id' node type for %s", nodeName)
	}
	instance.ImageID = image

	// instance_type is mandatory
	var instanceType string
	if _, instanceType, err = deployments.GetNodeProperty(kv, deploymentID, nodeName, "instance_type"); err != nil {
		return err
	} else if instanceType == "" {
		return errors.Errorf("Missing mandatory parameter 'instance_type' node type for %s", nodeName)
	}
	instance.InstanceType = instanceType

	// key_name is mandatory
	var keyName string
	if _, keyName, err = deployments.GetNodeProperty(kv, deploymentID, nodeName, "key_name"); err != nil {
		return err
	} else if keyName == "" {
		return errors.Errorf("Missing mandatory parameter 'key_name' node type for %s", nodeName)
	}
	instance.KeyName = keyName

	// security_groups needs to contain a least one occurrence
	var secGroups string
	if _, secGroups, err = deployments.GetNodeProperty(kv, deploymentID, nodeName, "security_groups"); err != nil {
		return err
	} else if secGroups == "" {
		return errors.Errorf("Missing mandatory parameter 'security_groups' node type for %s", nodeName)
	} else {
		for _, secGroup := range strings.Split(strings.NewReplacer("\"", "", "'", "").Replace(secGroups), ",") {
			secGroup = strings.TrimSpace(secGroup)
			instance.SecurityGroups = append(instance.SecurityGroups, secGroup)
		}
	}

	// user is mandatory
	var user string
	if _, user, err = deployments.GetNodeProperty(kv, deploymentID, nodeName, "user"); err != nil {
		return err
	} else if user == "" {
		return errors.Errorf("Missing mandatory parameter 'user' node type for %s", nodeName)
	}

	// Check optional provided Elastic IPs
	var eips string
	if _, eips, err = deployments.GetNodeProperty(kv, deploymentID, nodeName, "elastic_ips"); err != nil {
		return err
	} else if eips != "" {
		for _, eips := range strings.Split(strings.NewReplacer("\"", "", "'", "").Replace(eips), ",") {
			eips = strings.TrimSpace(eips)
			instance.ElasticIps = append(instance.ElasticIps, eips)
		}
	}

	// Add the AWS instance
	commons.AddResource(infrastructure, "aws_instance", instance.Tags.Name, &instance)

	// Provide Consul Keys
	consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{}}

	//Private IP Address
	consulKeyPrivateAddr := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/private_address"), Value: fmt.Sprintf("${aws_instance.%s.private_ip}", instance.Tags.Name)}

	// Specific DNS attribute
	consulKeyPublicDNS := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/public_dns"), Value: fmt.Sprintf("${aws_instance.%s.public_dns}", instance.Tags.Name)}
	consulKeys.Keys = append(consulKeys.Keys, consulKeyPrivateAddr, consulKeyPublicDNS)

	commons.AddResource(infrastructure, "consul_keys", instance.Tags.Name, &consulKeys)

	// If any Elastic IP is provided without any network requirement, EIP is anyway associated to the compute instance
	// Check existing network requirement otherwise
	var isElasticIP = len(instance.ElasticIps) == 0
	if !isElasticIP {
		isElasticIP, err = isElasticIPPRequired(kv, deploymentID, nodeName)
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

	// IP Address capability
	capabilityIPAddr := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address"), Value: accessIP}
	// Default TOSCA Attributes
	consulKeyIPAddr := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/ip_address"), Value: accessIP}
	consulKeyPublicAddr := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/public_address"), Value: accessIP}
	// For backward compatibility...
	consulKeyPublicIPAddr := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/public_ip_address"), Value: accessIP}
	consulKeys.Keys = append(consulKeys.Keys, consulKeyIPAddr, consulKeyPublicAddr, consulKeyPublicIPAddr, capabilityIPAddr)

	// Check the connection in order to be sure that ansible will be able to log on the instance
	nullResource := commons.Resource{}
	// TODO private key should not be hard-coded
	re := commons.RemoteExec{Inline: []string{`echo "connected"`}, Connection: &commons.Connection{User: user, Host: accessIP, PrivateKey: `${file("~/.ssh/janus.pem")}`}}
	nullResource.Provisioners = make([]map[string]interface{}, 0)
	provMap := make(map[string]interface{})
	provMap["remote-exec"] = re
	nullResource.Provisioners = append(nullResource.Provisioners, provMap)

	commons.AddResource(infrastructure, "null_resource", instance.Tags.Name+"-ConnectionCheck", &nullResource)

	return nil
}

func isElasticIPPRequired(kv *api.KV, deploymentID, nodeName string) (bool, error) {
	networkKeys, err := deployments.GetRequirementsKeysByNameForNode(kv, deploymentID, nodeName, "network")
	if err != nil {
		return false, err
	}
	for _, networkReqPrefix := range networkKeys {
		requirementIndex := deployments.GetRequirementIndexFromRequirementKey(networkReqPrefix)
		capability, err := deployments.GetCapabilityForRequirement(kv, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return false, err
		}
		networkNodeName, err := deployments.GetTargetNodeForRequirement(kv, deploymentID, nodeName, requirementIndex)
		if err != nil {
			return false, err
		}

		if capability != "" {
			is, err := deployments.IsNodeDerivedFrom(kv, deploymentID, networkNodeName, "janus.nodes.aws.PublicNetwork")
			if err != nil {
				return false, err
			} else if is {
				return is, nil
			}
		}
	}

	return false, nil
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
