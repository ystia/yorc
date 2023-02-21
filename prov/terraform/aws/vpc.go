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
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
)

func (g *awsGenerator) generateVPC(ctx context.Context, nodeParams nodeParams, instanceName string, outputs map[string]string) error {
	err := verifyThatNodeIsTypeOf(ctx, nodeParams, "yorc.nodes.aws.VPC")
	if err != nil {
		return err
	}

	vpc := &VPC{}
	err = g.getVPCProperties(ctx, nodeParams, vpc)
	if err != nil {
		return err
	}

	// Get tags map
	vpc.Tags, err = getTagsMap(ctx, nodeParams)
	if err != nil {
		return err
	}

	// Create the name for the resource
	var name = ""
	if vpc.Tags["Name"] != "" {
		name = vpc.Tags["Name"]
	} else {
		name = strings.ToLower(nodeParams.deploymentID + "-" + nodeParams.nodeName)
	}
	// Name must respect regular expression
	name = strings.Replace(strings.ToLower(name), "_", "-", -1)

	commons.AddResource(nodeParams.infrastructure, "aws_vpc", name, vpc)

	nodeKey := path.Join(consulutil.DeploymentKVPrefix, nodeParams.deploymentID, "topology", "instances", nodeParams.nodeName, instanceName)

	err = g.generateVPCSubnets(ctx, nodeParams, name, vpc, nodeKey, outputs)
	if err != nil {
		return err
	}
	err = g.generateVPCSecurityGroups(ctx, nodeParams, name, nodeKey, outputs)
	if err != nil {
		return err
	}
	g.generateIGandRT(ctx, nodeParams, name)

	// Terraform  output
	idKey := nodeParams.nodeName + "-" + instanceName + "-id"
	idValue := fmt.Sprintf("${aws_vpc.%s.id}", name)
	commons.AddOutput(nodeParams.infrastructure, idKey, &commons.Output{Value: idValue})
	outputs[path.Join(nodeKey, "/attributes/vpc_id")] = idKey
	return nil
}

func (g *awsGenerator) generateSubnet(ctx context.Context, nodeParams nodeParams, instanceName string, outputs map[string]string) error {
	err := verifyThatNodeIsTypeOf(ctx, nodeParams, "yorc.nodes.aws.Subnet")
	if err != nil {
		return err
	}

	subnet := &Subnet{}
	err = g.getSubnetProperties(ctx, nodeParams, subnet)
	if err != nil {
		return err
	}

	subnet.Tags, err = getTagsMap(ctx, nodeParams)
	if err != nil {
		return nil
	}

	// Create the name for the resource
	var name = ""
	if subnet.Tags["Name"] != "" {
		name = subnet.Tags["Name"]
	} else {
		name = strings.ToLower(nodeParams.deploymentID + "-" + nodeParams.nodeName)
	}
	// Name must respect regular expression
	name = strings.Replace(strings.ToLower(name), "_", "-", -1)

	commons.AddResource(nodeParams.infrastructure, "aws_subnet", name, subnet)

	// Terraform  output
	nodeKey := path.Join(consulutil.DeploymentKVPrefix, nodeParams.deploymentID, "topology", "instances", nodeParams.nodeName, instanceName)

	idKey := nodeParams.nodeName + "-" + instanceName + "-id"
	idValue := fmt.Sprintf("${aws_subnet.%s.id}", name)
	commons.AddOutput(nodeParams.infrastructure, idKey, &commons.Output{Value: idValue})
	outputs[path.Join(nodeKey, "/attributes/subnet_id")] = idKey

	return nil
}

func (g *awsGenerator) getVPCProperties(ctx context.Context, nodeParams nodeParams, vpc *VPC) error {
	stringParams := []struct {
		pAttr        *string
		propertyName string
		mandatory    bool
	}{
		{&vpc.CidrBlock, "cidr_block", true},
		{&vpc.InstanceTenancy, "instance_tenancy", false},
	}

	for _, stringParam := range stringParams {
		val, err := deployments.GetStringNodeProperty(ctx, nodeParams.deploymentID, nodeParams.nodeName, stringParam.propertyName, stringParam.mandatory)
		if err != nil {
			return err
		}
		*stringParam.pAttr = val
	}

	// Get bool properties
	boolProps := []struct {
		pAttr        *bool
		propertyName string
	}{
		{&vpc.EnableDNSSupport, "enable_dns_support"},
		{&vpc.EnableDNSHostnames, "enable_dns_hostnames"},
		{&vpc.EnableClassiclink, "enable_classiclink"},
		{&vpc.EnableClassiclinkDNSSupport, "enable_classiclink_dns_support"},
		{&vpc.AssignGeneratedIpv6CidrBlock, "assign_generated_ipv6_cidr_block"},
	}
	for _, boolProps := range boolProps {
		val, err := deployments.GetBooleanNodeProperty(ctx, nodeParams.deploymentID, nodeParams.nodeName, boolProps.propertyName)
		if err != nil {
			return err
		}
		*boolProps.pAttr = val
	}

	return nil
}

func (g *awsGenerator) getSubnetProperties(ctx context.Context, nodeParams nodeParams, subnet *Subnet) error {
	stringParams := []struct {
		pAttr        *string
		propertyName string
		mandatory    bool
	}{
		{&subnet.AvailabilityZone, "availability_zone", false},
		{&subnet.AvailabilityZoneID, "availability_zone_id", false},
		{&subnet.CidrBlock, "cidr_block", true},
		{&subnet.Ipv6CidrBlock, "ipv6_cidr_block", true},
		{&subnet.VPCId, "vpc_id", true},
	}

	for _, stringParam := range stringParams {
		val, err := deployments.GetStringNodeProperty(ctx, nodeParams.deploymentID, nodeParams.nodeName, stringParam.propertyName, stringParam.mandatory)
		if err != nil {
			return err
		}
		*stringParam.pAttr = val
	}

	// Get bool properties
	boolProps := []struct {
		pAttr        *bool
		propertyName string
	}{
		{&subnet.AssignIpv6AddressOnCreation, "assign_ipv6_address_on_creation"},
		{&subnet.MapPublicIPOnLaunch, "map_public_ip_on_launch"},
	}
	for _, boolProps := range boolProps {
		val, err := deployments.GetBooleanNodeProperty(ctx, nodeParams.deploymentID, nodeParams.nodeName, boolProps.propertyName)
		if err != nil {
			return err
		}
		*boolProps.pAttr = val
	}

	return nil
}

func (g *awsGenerator) generateVPCSubnets(ctx context.Context, nodeParams nodeParams, vpcName string, vpc *VPC, nodeKey string, outputs map[string]string) error {
	subNetsRaw, err := deployments.GetNodePropertyValue(ctx, nodeParams.deploymentID, nodeParams.nodeName, "subnets")
	if err != nil {
		return err
	}

	if subNetsRaw == nil || subNetsRaw.RawString() == "" {
		// Generate default subnet
		g.generateDefaultSubnet(ctx, nodeParams, vpcName, vpc, nodeKey, outputs)
	} else {
		list, ok := subNetsRaw.Value.([]interface{})
		if !ok {
			return errors.New("failed to retrieve yorc.datatypes.aws.SubnetType Tosca Value: not expected type")
		}

		for i := range list {
			g.generatedVPCSubnet(ctx, nodeParams, vpcName, i)
		}
	}

	return nil
}

func (g *awsGenerator) generateDefaultSubnet(ctx context.Context, nodeParams nodeParams, vpcName string, vpc *VPC, nodeKey string, outputs map[string]string) {
	subnet := &Subnet{}
	subnet.MapPublicIPOnLaunch = true
	subnet.CidrBlock = vpc.CidrBlock
	subnet.VPCId = fmt.Sprintf("${aws_vpc.%s.id}", vpcName)
	subnet.Tags = map[string]string{
		"name": "DefaultSubnet",
	}

	name := strings.ToLower(nodeParams.deploymentID + "-" + nodeParams.nodeName + "-" + "defaultsubnet")
	name = strings.Replace(strings.ToLower(name), "_", "-", -1)

	commons.AddResource(nodeParams.infrastructure, "aws_subnet", name, subnet)

	idKey := nodeParams.nodeName + "-defaultSubnet"
	idValue := fmt.Sprintf("${aws_subnet.%s.id}", name)
	commons.AddOutput(nodeParams.infrastructure, idKey, &commons.Output{Value: idValue})
	outputs[path.Join(nodeKey, "/attributes/default_subnet_id")] = idKey
}

func (g *awsGenerator) generatedVPCSubnet(ctx context.Context, nodeParams nodeParams, vpcName string, i int) error {
	ind := strconv.Itoa(i)
	subnet := &Subnet{}

	params := []struct {
		pStringAtt   *string
		pBoolAtt     *bool
		propertyName string
		mandatory    bool
	}{
		{&subnet.AvailabilityZone, nil, "availability_zone", false},
		{&subnet.AvailabilityZoneID, nil, "availability_zone_id", false},
		{&subnet.CidrBlock, nil, "cidr_block", true},
		{&subnet.Ipv6CidrBlock, nil, "ipv6_cidr_block", false},
		{nil, &subnet.AssignIpv6AddressOnCreation, "assign_ipv6_address_on_creation", false},
		{nil, &subnet.MapPublicIPOnLaunch, "map_public_ip_on_launch", false},
	}

	for _, params := range params {
		val, err := deployments.GetNodePropertyValue(ctx, nodeParams.deploymentID, nodeParams.nodeName, "subnets", ind, params.propertyName)
		if err != nil {
			return err
		}
		if val == nil && params.mandatory {
			return errors.New("Missing mandatory params " + params.propertyName)
		}
		if params.pStringAtt != nil {
			*params.pStringAtt = val.RawString()
		} else if params.pBoolAtt != nil {
			val, err := deployments.GetBooleanNodeProperty(ctx, nodeParams.deploymentID, nodeParams.nodeName, "subnets", ind, params.propertyName)
			if err != nil {
				return err
			}
			*params.pBoolAtt = val
		}

		subnet.Tags, err = getTagsMap(ctx, nodeParams, ind, "tags")
		if err != nil {
			return err
		}

		subnet.VPCId = fmt.Sprintf("${aws_vpc.%s.id}", vpcName)

		// Create the name for the resource
		var name = ""
		if subnet.Tags["Name"] != "" {
			name = subnet.Tags["Name"]
		} else {
			name = strings.ToLower(nodeParams.deploymentID + "-" + nodeParams.nodeName + "-" + vpcName)
		}
		// Name must respect regular expression
		name = strings.Replace(strings.ToLower(name), "_", "-", -1)

		commons.AddResource(nodeParams.infrastructure, "aws_subnet", name, subnet)
	}

	return nil
}

func (g *awsGenerator) generateVPCSecurityGroups(ctx context.Context, nodeParams nodeParams, vpcName string, nodeKey string, outputs map[string]string) error {
	securityGroupsRaw, err := deployments.GetNodePropertyValue(ctx, nodeParams.deploymentID, nodeParams.nodeName, "security_groups")
	if err != nil {
		return err
	}

	if securityGroupsRaw == nil || securityGroupsRaw.RawString() == "" {
		err := g.generateDefaultSecurityGroup(ctx, nodeParams, vpcName, nodeKey, outputs)
		if err != nil {
			return err
		}
	} else {
		list, ok := securityGroupsRaw.Value.([]interface{})
		if !ok {
			return errors.New("failed to retrieve yorc.datatypes.aws.SecurityGroupType Tosca Value: not expected type")
		}

		for i := range list {
			g.generateVPCSecurityGroup(ctx, nodeParams, vpcName, i)
		}
	}

	return nil
}

func (g *awsGenerator) generateDefaultSecurityGroup(ctx context.Context, nodeParams nodeParams, vpcName string, nodeKey string, outputs map[string]string) error {
	// Only allow connection SSH connection (port 22 and TCP)
	securityGroup := &SecurityGroups{}
	securityGroup.Egress = SecurityRule{"22", "22", "TCP", []string{"0.0.0.0/0"}}
	securityGroup.Ingress = SecurityRule{"22", "22", "TCP", []string{"0.0.0.0/0"}}
	securityGroup.Name = "Default " + vpcName
	securityGroup.VPCId = fmt.Sprintf("${aws_vpc.%s.id}", vpcName)

	name := strings.ToLower(nodeParams.deploymentID + "-" + nodeParams.nodeName + "-" + "defaultSecurityGroup")
	name = strings.Replace(strings.ToLower(name), "_", "-", -1)

	commons.AddResource(nodeParams.infrastructure, "aws_security_group", name, securityGroup)

	idKey := nodeParams.nodeName + "-defaultSecurityGroup"
	idValue := fmt.Sprintf("${aws_security_group.%s.id}", name)
	commons.AddOutput(nodeParams.infrastructure, idKey, &commons.Output{Value: idValue})
	outputs[path.Join(nodeKey, "/attributes/default_security_group")] = idKey

	return nil
}

func (g *awsGenerator) generateVPCSecurityGroup(ctx context.Context, nodeParams nodeParams, vpcName string, i int) error {
	ind := strconv.Itoa(i)
	securityGroup := &SecurityGroups{}

	val, err := deployments.GetNodePropertyValue(ctx, nodeParams.deploymentID, nodeParams.nodeName, "security_groups", ind, "name")
	if err != nil {
		return err
	}
	securityGroup.Name = val.RawString()

	protocol, err := deployments.GetStringNodeProperty(ctx, nodeParams.deploymentID, nodeParams.nodeName, "security_groups", true, ind, "ingress", "protocol")
	if err != nil {
		return err
	}

	fromPort, err := deployments.GetStringNodeProperty(ctx, nodeParams.deploymentID, nodeParams.nodeName, "security_groups", true, ind, "ingress", "from_port")
	if err != nil {
		return err
	}

	toPort, err := deployments.GetStringNodeProperty(ctx, nodeParams.deploymentID, nodeParams.nodeName, "security_groups", true, ind, "ingress", "to_port")
	if err != nil {
		return err
	}

	securityGroup.Ingress = SecurityRule{FromPort: fromPort, ToPort: toPort, Protocol: protocol}

	protocol, err = deployments.GetStringNodeProperty(ctx, nodeParams.deploymentID, nodeParams.nodeName, "security_groups", true, ind, "egress", "protocol")
	if err != nil {
		return err
	}

	fromPort, err = deployments.GetStringNodeProperty(ctx, nodeParams.deploymentID, nodeParams.nodeName, "security_groups", true, ind, "egress", "from_port")
	if err != nil {
		return err
	}

	toPort, err = deployments.GetStringNodeProperty(ctx, nodeParams.deploymentID, nodeParams.nodeName, "security_groups", true, ind, "egress", "to_port")
	if err != nil {
		return err
	}

	securityGroup.Egress = SecurityRule{FromPort: fromPort, ToPort: toPort, Protocol: protocol}

	securityGroup.VPCId = fmt.Sprintf("${aws_vpc.%s.id}", vpcName)
	commons.AddResource(nodeParams.infrastructure, "aws_security_group", securityGroup.Name, securityGroup)

	return nil
}

// Generate an InternetGateway and Routable so the VPC is accesible through internet
func (g *awsGenerator) generateIGandRT(ctx context.Context, nodeParams nodeParams, vpcName string) {
	internetGateway := &InternetGateway{}
	internetGateway.VPCId = fmt.Sprintf("${aws_vpc.%s.id}", vpcName)
	internetGatewayName := nodeParams.nodeName + "_defaultInternetGateway"
	commons.AddResource(nodeParams.infrastructure, "aws_internet_gateway", internetGatewayName, internetGateway)

	routeTable := DefaultRouteTable{}
	// routeTable.VPCId = fmt.Sprintf("${aws_vpc.%s.id}", name)
	routeTable.DefaultRouteTableID = fmt.Sprintf("${aws_vpc.%s.default_route_table_id}", vpcName)
	routeTable.Route = map[string]string{
		"cidr_block": "0.0.0.0/0",
		"gateway_id": fmt.Sprintf("${aws_internet_gateway.%s.id}", internetGatewayName),
	}
	routeTable.DependsOn = []string{
		fmt.Sprintf("aws_internet_gateway.%s", internetGatewayName),
	}
	commons.AddResource(nodeParams.infrastructure, "aws_default_route_table", nodeParams.nodeName+"_defaultRouteTable", routeTable)
}
