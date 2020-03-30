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
		return nil
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

	// Terraform  output
	nodeKey := path.Join(consulutil.DeploymentKVPrefix, nodeParams.deploymentID, "topology", "instances", nodeParams.nodeName, instanceName)

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

	// If not VPCId defined, it should be a dependancy, find it
	if subnet.VPCId == "" {
		hasDep, vpcNodeName, err := deployments.HasAnyRequirementFromNodeType(ctx, nodeParams.deploymentID, nodeParams.nodeName, "dependency", "yorc.nodes.aws.VPC")
		if err != nil {
			return err
		}
		if !hasDep {
			return errors.Errorf("failed to retrieve dependency btw any network and the subnet with name:%q", name)
		}

		subnet.VPCId, err = deployments.LookupInstanceAttributeValue(ctx, nodeParams.deploymentID, vpcNodeName, instanceName, "vpc_id")
		if err != nil {
			return err
		}
	}

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
		{&subnet.VPCId, "vpc_id", false},
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
