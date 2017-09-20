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

	publicIP := fmt.Sprintf("${aws_instance.%s.public_ip}", instance.Tags.Name)
	consulKey := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address"), Value: publicIP} // Use Public ip here
	consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}

	commons.AddResource(infrastructure, "aws_instance", instance.Tags.Name, &instance)

	nullResource := commons.Resource{}
	// Do this in order to be sure that ansible will be able to log on the instance
	// TODO private key should not be hard-coded
	re := commons.RemoteExec{Inline: []string{`echo "connected"`}, Connection: &commons.Connection{User: user, Host: publicIP, PrivateKey: `${file("~/.ssh/janus.pem")}`}}
	nullResource.Provisioners = make([]map[string]interface{}, 0)
	provMap := make(map[string]interface{})
	provMap["remote-exec"] = re
	nullResource.Provisioners = append(nullResource.Provisioners, provMap)

	commons.AddResource(infrastructure, "null_resource", instance.Tags.Name+"-ConnectionCheck", &nullResource)

	// Default TOSCA Attributes
	consulKeyIPAddr := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/ip_address"), Value: publicIP}
	consulKeyPublicAddr := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/public_address"), Value: publicIP}
	// For backward compatibility...
	consulKeyPublicIPAddr := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/public_ip_address"), Value: publicIP}
	consulKeyPrivateAddr := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/private_address"), Value: fmt.Sprintf("${aws_instance.%s.private_ip}", instance.Tags.Name)}

	// Specific DNS attribute
	consulKeyPublicDNS := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/public_dns"), Value: fmt.Sprintf("${aws_instance.%s.public_dns}", instance.Tags.Name)}
	consulKeys.Keys = append(consulKeys.Keys, consulKeyIPAddr, consulKeyPrivateAddr, consulKeyPublicAddr, consulKeyPublicIPAddr, consulKeyPublicDNS)

	commons.AddResource(infrastructure, "consul_keys", instance.Tags.Name, &consulKeys)

	return nil
}
