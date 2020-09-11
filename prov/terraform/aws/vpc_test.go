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
	"strings"
	"testing"

	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
)

func testVPC(t *testing.T, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	ctx := context.Background()
	infrastructure := commons.Infrastructure{}
	g := awsGenerator{}
	networkName := "vpc-network"

	nodeParams := nodeParams{
		deploymentID:   deploymentID,
		nodeName:       "Network",
		infrastructure: &infrastructure,
	}

	err := g.generateVPC(ctx, nodeParams, "instance0", make(map[string]string))

	require.NoError(t, err, "Unexpected error attempting to generate vpc for %s", deploymentID)
	require.Len(t, infrastructure.Resource["aws_vpc"], 1, "Expected one vpc")

	// VPC resource test
	instancesMap := infrastructure.Resource["aws_vpc"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, networkName)

	vpc, ok := instancesMap[networkName].(*VPC)
	require.True(t, ok, "%s is not a VPC", networkName)
	assert.Equal(t, "10.0.0.0/16", vpc.CidrBlock)
	assert.Equal(t, true, vpc.AssignGeneratedIpv6CidrBlock)
	assert.Equal(t, "foo", vpc.Tags["tag1"])

	// Default subnet test
	defaultSubnetName := strings.ToLower(nodeParams.deploymentID + "-" + nodeParams.nodeName + "-" + "defaultsubnet")
	instancesMap = infrastructure.Resource["aws_subnet"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, defaultSubnetName)

	subnet, ok := instancesMap[defaultSubnetName].(*Subnet)
	require.True(t, ok, "%s is not a Subnet", defaultSubnetName)
	assert.Equal(t, "10.0.0.0/16", subnet.CidrBlock)
	assert.Equal(t, true, subnet.MapPublicIPOnLaunch)

	defaultSubnetKey := nodeParams.nodeName + "-defaultSubnet"
	require.NotNil(t, infrastructure.Output[defaultSubnetKey], "Expected related output to default subnet ID")
	output := infrastructure.Output[defaultSubnetKey]
	defaultSubnetName = strings.Replace(strings.ToLower(defaultSubnetName), "_", "-", -1)
	require.Equal(t, output.Value, fmt.Sprintf("${aws_subnet.%s.id}", defaultSubnetName))

	// Default Security Group test
	defaultSecurityGroupName := strings.ToLower(nodeParams.deploymentID + "-" + nodeParams.nodeName + "-" + "defaultSecurityGroup")
	instancesMap = infrastructure.Resource["aws_security_group"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, defaultSecurityGroupName)

	securityGroup, ok := instancesMap[defaultSecurityGroupName].(*SecurityGroups)
	require.True(t, ok, "%s is not a Subnet", defaultSubnetName)
	require.NoError(t, err, "Can't read IP on ipify response", deploymentID)
	assert.Equal(t, SecurityRule{"22", "22", "TCP", []string{"0.0.0.0/0"}}, securityGroup.Egress)
}

func testVPCWithNestedSubnetAndSG(t *testing.T, srv1 *testutil.TestServer, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	ctx := context.Background()
	infrastructure := commons.Infrastructure{}
	g := awsGenerator{}
	networkName := "vpcwithnestedsubnetandsg-network"

	nodeParams := nodeParams{
		deploymentID:   deploymentID,
		nodeName:       "Network",
		infrastructure: &infrastructure,
	}

	err := g.generateVPC(ctx, nodeParams, "instance0", make(map[string]string))

	require.NoError(t, err, "Unexpected error attempting to generate vpc for %s", deploymentID)
	require.Len(t, infrastructure.Resource["aws_vpc"], 1, "Expected one vpc")

	// VPC resource test
	instancesMap := infrastructure.Resource["aws_vpc"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, networkName)

	vpc, ok := instancesMap[networkName].(*VPC)
	require.True(t, ok, "%s is not a VPC", networkName)
	assert.Equal(t, "10.0.0.0/16", vpc.CidrBlock)
	assert.Equal(t, true, vpc.AssignGeneratedIpv6CidrBlock)
	assert.Equal(t, "foo", vpc.Tags["tag1"])

	// Generated SubnetTest
	subnetName := strings.ToLower(nodeParams.deploymentID + "-" + nodeParams.nodeName + "-" + networkName)
	subnetName = strings.Replace(strings.ToLower(subnetName), "_", "-", -1)
	instancesMap = infrastructure.Resource["aws_subnet"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, subnetName)

	subnet, ok := instancesMap[subnetName].(*Subnet)
	require.True(t, ok, "%s is not a subnet", subnet)
	assert.Equal(t, "10.0.0.0/24", subnet.CidrBlock)
	assert.Equal(t, "us-east-2a", subnet.AvailabilityZone)
	assert.Equal(t, true, subnet.MapPublicIPOnLaunch)

	// Generated SecurityGroups
	securityGroupName := "groupOpen"
	instancesMap = infrastructure.Resource["aws_security_group"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, securityGroupName)

	securityGroup, ok := instancesMap[securityGroupName].(*SecurityGroups)
	require.True(t, ok, "%s is not a securityGroup", securityGroup)
	assert.Equal(t, SecurityRule{FromPort: "0", ToPort: "0", Protocol: "-1"}, securityGroup.Egress)
	assert.Equal(t, SecurityRule{FromPort: "1", ToPort: "15", Protocol: "TCP"}, securityGroup.Ingress)

}

func testSubnet(t *testing.T, srv1 *testutil.TestServer, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	ctx := context.Background()
	infrastructure := commons.Infrastructure{}
	g := awsGenerator{}
	subnetName := "subnet-network-subnet" // deploymentID + "-" + nodeName (with _ changed to -)

	nodeParams := nodeParams{
		deploymentID:   deploymentID,
		nodeName:       "Network_Subnet",
		infrastructure: &infrastructure,
	}

	err := g.generateSubnet(ctx, nodeParams, "0", make(map[string]string))
	require.NoError(t, err, "Unexpected error attempting to generate sub-network for %s", deploymentID)

	instancesMap := infrastructure.Resource["aws_subnet"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, subnetName)

	subnet, ok := instancesMap[subnetName].(*Subnet)
	require.True(t, ok, "%s is not a Subnet", subnetName)
	assert.Equal(t, "10.0.0.0/24", subnet.CidrBlock)
	assert.Equal(t, "vpc_id", subnet.VPCId)
}
