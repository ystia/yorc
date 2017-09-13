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

func (g *osGenerator) generateOSInstance(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName, instanceName string, infrastructure *commons.Infrastructure, outputs map[string]string) error {
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

	_, image, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "image_id")
	if err != nil {
		return err
	}
	instance.ImageID = image
	_, instanceType, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "instance_type")
	if err != nil {
		return err
	}
	instance.InstanceType = instanceType
	_, keyName, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "key_name")
	if err != nil {
		return err
	}
	instance.KeyName = keyName

	instance.SecurityGroups = cfg.OSDefaultSecurityGroups
	_, secGroups, err := deployments.GetNodeProperty(kv, deploymentID, nodeName, "security_groups")
	if err != nil {
		return err
	} else if secGroups != "" {
		for _, secGroup := range strings.Split(strings.NewReplacer("\"", "", "'", "").Replace(secGroups), ",") {
			secGroup = strings.TrimSpace(secGroup)
			instance.SecurityGroups = append(instance.SecurityGroups, secGroup)
		}
	}

	if instance.ImageID == "" {
		return errors.Errorf("Missing mandatory parameter 'image_id' node type for %s", nodeName)
	}
	if instance.InstanceType == "" {
		return errors.Errorf("Missing mandatory parameter 'instance_type' node type for %s", nodeName)
	}

	var user string
	if _, user, err = deployments.GetNodeProperty(kv, deploymentID, nodeName, "user"); err != nil {
		return err
	} else if user == "" {
		return errors.Errorf("Missing mandatory parameter 'user' node type for %s", nodeName)
	}

	consulKey := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/ip_address"), Value: fmt.Sprintf("${aws_instance.%s.public_ip}", instance.Tags.Name)} // Use Public ip here
	consulKeys := commons.ConsulKeys{Keys: []commons.ConsulKey{consulKey}}

	// Do this in order to be sure that ansible will be able to log on the instance
	// TODO private key should not be hard-coded
	re := commons.RemoteExec{Inline: []string{`echo "connected"`}, Connection: commons.Connection{User: user, PrivateKey: `${file("~/.ssh/janus.pem")}`}}
	instance.Provisioners = make(map[string]interface{})
	instance.Provisioners["remote-exec"] = re

	addResource(infrastructure, "aws_instance", instance.Tags.Name, &instance)

	consulKeyPubAddr := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/capabilities/endpoint/attributes/public_address"), Value: fmt.Sprintf("${aws_instance.%s.public_ip}", instance.Tags.Name)}
	consulKeyDNS := commons.ConsulKey{Path: path.Join(instancesKey, instanceName, "/attributes/public_dns"), Value: fmt.Sprintf("${aws_instance.%s.public_dns}", instance.Tags.Name)}
	consulKeys.Keys = append(consulKeys.Keys, consulKeyPubAddr, consulKeyDNS)

	addResource(infrastructure, "consul_keys", instance.Tags.Name, &consulKeys)

	return nil
}
