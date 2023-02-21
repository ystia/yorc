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
	"path"
	"strconv"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
)

var expectedPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAuEl5Wjgdvlqbz0x2vcllSQrDiRd+bWdA2MgpOl726ovxw9uE
QJSlXYBJbSCQg+q++OEtXmvfahN5Y9aemuPey/o/S9RWyQ/X+uVeXdNV4Xkgar6b
uYr1n1Ju7ltmdVJME7fr+Ti+2d+EMBs7V+jGXyZzBTdr6wCJuBHHXV/ZKDzw1cHd
bRF8obBmMcxyzNbXnhSUvBgXT+GQ0/CgkNdrTwGOgtckqNYTuw1Rd6wAsF5xgN23
uss5WJOg3/eMW2JMjyxNqaJhBUtA2CKcdnLjwyDxWdmC1NMHKL1umPOjuCyNczpl
axMKW//UZT3WyfVt/gcHGGNIuI0izwFJ6QjlrQIDAQABAoIBAAet8COlUP/8sJ97
1TrlaJYZn7pXw0n10or2FFm9WVa+zC1YOXOjfhyeWvD0OXF1181xPL3BiwbVlupl
KCjWNBOV8wtK5u7r/RkUc9E/HEYQERzBoqWht8iS29KM9oEPE+KCeI/jIHjdypli
mR95sMKITKS8AYBCfnqwKvmmI9t8VIXsrZWsg1dUD9TCa8QxoA66raSpXegDgjox
T8IjZW90BwD6oG/5+HfbuwtjKR1Lca5tMzqxDMvqBf3KdCuee1x2Uuzla9/MsK/4
Nuqv88gpoI7bDJOJnF/KrJqEH1ihF5zNVOs5c7XKmnAdry05tA7CjbiILOeFq3yn
elkdR5UCgYEA3RC0bAR/TjSKGBEzvzfJFNp2ipdlZ1svobHl5sqDJzQp7P9aIogU
qUhw2vr/nHg4dfmLnMHJYh6GCIjN1H7NZzaBiQSUcT+s2GRxYJqRV4geFHvKNzt3
a50Hi5rSsbKm0LvlUA3vGkMABICyzkETPDl2WSFtKWUYrTMZSKixCtsCgYEA1Wjj
fn+3HOmAv3lX4PzhHiBBfBj60BKPCnBbWG6TTF4ya7UEU+e5aAbLD10QdQyx77rL
V3G3Xda1BWA2wGKRDfC9ksFUuxH2egNPGadOVZH2U/a/87YGOFUmbf03jJ6mbeRV
BBBVcB8oGSD+NemiDPqYUi/G1lT+oRLFIkkYhBcCgYEApjKj4j2zVCFt3NA5/j27
gGEKFAHka8MDWWY8uLlxxuyRxKrpoeJ63hYnOorP11wO3qshClYqyAi4rfvj+yjl
1f4FfvShgU7k7L7++ijaslsUekPi8IlVq+MfxBY+5vewMGfC69+97hmHDtuPEj+c
bX+p+TKHNkLaPYSYMqcYi1cCgYEAxf6JSfyt48oT5BFtYdTb+zpL5xm54T/GrBWv
+eylBm5Cc0E/YaUUlBnxXTCnqyD7GQKB04AycoJX8kPgqD8KexeGmlh6BxFUTsEx
KwjZGXTRR/cfAbo4LR17CQKr/e/XUw9LfPi2e868QgwlLdmzujzpAx9GZ+X1U3V5
piSQ9UMCgYBdegnYh2fqU/oGH+d9WahuO1LW9vz8sFEIhRgJyLfA/ypAg6WCgJF2
GtepEYBXL+QZnhudVxi0YPTmNN3+gtHdr+B4dKZ8z7m9NO2nk5AKdf0sYGWHEzhy
PAgZzG5OTZiu+YohUPnC66eFiyS6anLBj0DGNa9VA8j352ecgeNO4A==
-----END RSA PRIVATE KEY-----
`

func loadTestYaml(t *testing.T) string {
	deploymentID := path.Base(t.Name())
	yamlName := "testdata/" + deploymentID + ".yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), deploymentID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")
	return deploymentID
}

func testSimpleAWSInstanceFailed(t *testing.T, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	err := g.generateAWSInstance(context.Background(), cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string), &env)
	require.Error(t, err, "Expecting missing mandatory parameter 'instance_type' error")
}

func testSimpleAWSInstance(t *testing.T, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)
	outputs := make(map[string]string, 0)
	err := g.generateAWSInstance(context.Background(), cfg, deploymentID, "ComputeAWS", "0", &infrastructure, outputs, &env)
	require.Nil(t, err)

	require.Len(t, infrastructure.Resource["aws_instance"], 1)
	instancesMap := infrastructure.Resource["aws_instance"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, "ComputeAWS-0")

	compute, ok := instancesMap["ComputeAWS-0"].(*ComputeInstance)
	require.True(t, ok, "ComputeAWS-0 is not a ComputeInstance")
	require.Equal(t, "yorc-keypair", compute.KeyName)
	require.Equal(t, "ami-16dffe73", compute.ImageID)
	require.Equal(t, "t2.micro", compute.InstanceType)
	require.Equal(t, "ComputeAWS-0", compute.Tags.Name)
	require.Equal(t, "us-east-2c", compute.AvailabilityZone)
	require.Equal(t, "myPlacement", compute.PlacementGroup)
	require.Equal(t, "subnet-f578918e", compute.SubnetID)
	require.Equal(t, true, compute.RootBlockDevice.DeleteOnTermination)
	require.Len(t, compute.SecurityGroups, 1)
	require.Contains(t, compute.SecurityGroups, "yorc-securityGroup")

	require.Len(t, compute.Provisioners, 0)
	require.Contains(t, infrastructure.Resource, "null_resource")
	require.Len(t, infrastructure.Resource["null_resource"], 1)
	nullResources := infrastructure.Resource["null_resource"].(map[string]interface{})

	require.Contains(t, nullResources, "ComputeAWS-0-ConnectionCheck")
	nullRes, ok := nullResources["ComputeAWS-0-ConnectionCheck"].(*commons.Resource)
	require.True(t, ok)
	require.Len(t, nullRes.Provisioners, 1)
	mapProv := nullRes.Provisioners[0]
	require.Contains(t, mapProv, "remote-exec")
	rex, ok := mapProv["remote-exec"].(commons.RemoteExec)
	require.True(t, ok)
	require.Equal(t, "centos", rex.Connection.User)

	require.Nil(t, err)
	assert.Equal(t, "${var.private_key}", rex.Connection.PrivateKey)
	require.Len(t, env, 1)
	assert.Equal(t, "TF_VAR_private_key="+expectedPrivateKey, env[0], "env var for private key expected")

	require.NotContains(t, infrastructure.Resource, "aws_eip_association")

	require.Len(t, outputs, 6, "six outputs are expected")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "ComputeAWS", "0", "attributes/public_address"), "expected public_address instance attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "ComputeAWS", "0", "attributes/public_dns"), "expected public_dns instance attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "ComputeAWS", "0", "attributes/public_ip_address"), "expected public_ip_address instance attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "ComputeAWS", "0", "attributes/ip_address"), "expected ip_address instance attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "ComputeAWS", "0", "attributes/private_address"), "expected private_address instance attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "ComputeAWS", "0", "/capabilities/endpoint/attributes/ip_address"), "expected capability endpoint ip_address instance attribute output")
}

func testSimpleAWSInstanceWithPrivateKey(t *testing.T, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	err := g.generateAWSInstance(context.Background(), cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string), &env)
	require.Nil(t, err)

	require.Len(t, infrastructure.Resource["aws_instance"], 1)
	instancesMap := infrastructure.Resource["aws_instance"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, "ComputeAWS-0")
	require.Contains(t, infrastructure.Resource, "null_resource")
	require.Len(t, infrastructure.Resource["null_resource"], 1)
	nullResources := infrastructure.Resource["null_resource"].(map[string]interface{})

	require.Contains(t, nullResources, "ComputeAWS-0-ConnectionCheck")
	nullRes, ok := nullResources["ComputeAWS-0-ConnectionCheck"].(*commons.Resource)
	require.True(t, ok)
	require.Len(t, nullRes.Provisioners, 1)
	mapProv := nullRes.Provisioners[0]
	require.Contains(t, mapProv, "remote-exec")
	rex, ok := mapProv["remote-exec"].(commons.RemoteExec)
	require.True(t, ok)
	require.Equal(t, "centos", rex.Connection.User)
	assert.Equal(t, "${var.private_key}", rex.Connection.PrivateKey)
	require.Len(t, env, 1)
	assert.Equal(t, "TF_VAR_private_key="+expectedPrivateKey, env[0], "env var for private key expected")

	require.NotContains(t, infrastructure.Resource, "aws_eip_association")
}

func testSimpleAWSInstanceWithNoDeleteVolumeOnTermination(t *testing.T, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	err := g.generateAWSInstance(context.Background(), cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string), &env)
	require.Nil(t, err)

	require.Len(t, infrastructure.Resource["aws_instance"], 1)
	instancesMap := infrastructure.Resource["aws_instance"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, "ComputeAWS-0")

	compute, ok := instancesMap["ComputeAWS-0"].(*ComputeInstance)
	require.True(t, ok, "ComputeAWS-0 is not a ComputeInstance")
	require.Equal(t, false, compute.RootBlockDevice.DeleteOnTermination)
}

func testSimpleAWSInstanceWithEIP(t *testing.T, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	err := g.generateAWSInstance(context.Background(), cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string), &env)
	require.Nil(t, err)

	require.Contains(t, infrastructure.Resource, "aws_eip")
	require.Len(t, infrastructure.Resource["aws_eip"], 1)
	eip := infrastructure.Resource["aws_eip"].(map[string]interface{})
	require.Contains(t, eip, "EIP-ComputeAWS-0")
	_, ok := eip["EIP-ComputeAWS-0"].(*ElasticIP)
	require.True(t, ok, "EIP-ComputeAWS-0 is not an ElasticIP")

	require.Contains(t, infrastructure.Resource, "aws_eip_association")
	require.Len(t, infrastructure.Resource["aws_eip_association"], 1)
	eipAssoc := infrastructure.Resource["aws_eip_association"].(map[string]interface{})
	require.Contains(t, eipAssoc, "EIPAssoc-ComputeAWS-0")
	assoc, ok := eipAssoc["EIPAssoc-ComputeAWS-0"].(*ElasticIPAssociation)
	require.True(t, ok, "EIPAssoc-ComputeAWS-0 is not an ElasticIPAssociation")
	require.Equal(t, "${aws_instance.ComputeAWS-0.id}", assoc.InstanceID)
	require.Equal(t, "${aws_eip.EIP-ComputeAWS-0.id}", assoc.AllocationID)

}

func testSimpleAWSInstanceWithProvidedEIP(t *testing.T, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	err := g.generateAWSInstance(context.Background(), cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string), &env)
	require.Nil(t, err)

	require.NotContains(t, infrastructure.Resource, "aws_eip")

	require.Contains(t, infrastructure.Resource, "aws_eip_association")
	require.Len(t, infrastructure.Resource["aws_eip_association"], 1)
	eipAssoc := infrastructure.Resource["aws_eip_association"].(map[string]interface{})
	require.Contains(t, eipAssoc, "EIPAssoc-ComputeAWS-0")
	assoc, ok := eipAssoc["EIPAssoc-ComputeAWS-0"].(*ElasticIPAssociation)
	require.True(t, ok, "EIPAssoc-ComputeAWS-0 is not an ElasticIPAssociation")
	require.Equal(t, "${aws_instance.ComputeAWS-0.id}", assoc.InstanceID)
	require.Equal(t, "10.10.10.10", assoc.PublicIP)

}

func testSimpleAWSInstanceWithListOfProvidedEIP(t *testing.T, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)
	ctx := context.Background()
	nb, err := deployments.GetDefaultNbInstancesForNode(ctx, deploymentID, "ComputeAWS")
	require.Nil(t, err)
	require.Equal(t, uint32(4), nb)

	for i := 0; i < int(nb); i++ {
		istr := strconv.Itoa(i)
		err = g.generateAWSInstance(context.Background(), cfg, deploymentID, "ComputeAWS", istr, &infrastructure, make(map[string]string), &env)
		require.Nil(t, err)
		require.NotContains(t, infrastructure.Resource, "aws_eip")
		require.Contains(t, infrastructure.Resource, "aws_eip_association")
		require.Len(t, infrastructure.Resource["aws_eip_association"], 1+i)
		eipAssoc := infrastructure.Resource["aws_eip_association"].(map[string]interface{})
		require.Contains(t, eipAssoc, "EIPAssoc-ComputeAWS-"+istr)
		assoc, ok := eipAssoc["EIPAssoc-ComputeAWS-"+istr].(*ElasticIPAssociation)
		require.True(t, ok, "EIPAssoc-ComputeAWS-"+istr+" is not an ElasticIPAssociation")
		require.Equal(t, "${aws_instance.ComputeAWS-"+istr+".id}", assoc.InstanceID)
		switch i {
		case 0:
			require.Equal(t, "10.10.10.10", assoc.PublicIP)
		case 1:
			require.Equal(t, "11.11.11.11", assoc.PublicIP)
		case 2:
			require.Equal(t, "12.12.12.12", assoc.PublicIP)
		case 3:
			require.Equal(t, "13.13.13.13", assoc.PublicIP)
		}
	}
}

func testSimpleAWSInstanceWithMalformedEIP(t *testing.T, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	err := g.generateAWSInstance(context.Background(), cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string), &env)
	require.Error(t, err, "An error was expected due to malformed provided Elastic IP: 12.12.oups.12")
}

func testSimpleAWSInstanceWithNotEnoughProvidedEIPS(t *testing.T, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	ctx := context.Background()
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	nb, err := deployments.GetDefaultNbInstancesForNode(ctx, deploymentID, "ComputeAWS")
	require.Nil(t, err)
	require.Equal(t, uint32(5), nb)

	for i := 0; i < int(nb); i++ {
		istr := strconv.Itoa(i)
		err = g.generateAWSInstance(context.Background(), cfg, deploymentID, "ComputeAWS", istr, &infrastructure, make(map[string]string), &env)
		require.Nil(t, err)
		// EIP is only created for the last instance (only 4 EIPs are provided for 5 default instances
		if i < 4 {
			require.NotContains(t, infrastructure.Resource, "aws_eip")
		} else {
			require.Contains(t, infrastructure.Resource, "aws_eip")
		}
		require.Contains(t, infrastructure.Resource, "aws_eip_association")
		require.Len(t, infrastructure.Resource["aws_eip_association"], 1+i)
		eipAssoc := infrastructure.Resource["aws_eip_association"].(map[string]interface{})
		require.Contains(t, eipAssoc, "EIPAssoc-ComputeAWS-"+istr)
		assoc, ok := eipAssoc["EIPAssoc-ComputeAWS-"+istr].(*ElasticIPAssociation)
		require.True(t, ok, "EIPAssoc-ComputeAWS-"+istr+" is not an ElasticIPAssociation")
		require.Equal(t, "${aws_instance.ComputeAWS-"+istr+".id}", assoc.InstanceID)
		switch i {
		case 0:
			require.Equal(t, "10.10.10.10", assoc.PublicIP)
		case 1:
			require.Equal(t, "11.11.11.11", assoc.PublicIP)
		case 2:
			require.Equal(t, "12.12.12.12", assoc.PublicIP)
		case 3:
			require.Equal(t, "13.13.13.13", assoc.PublicIP)
		}
	}
}

func testSimpleAWSInstanceWithPersistentDisk(t *testing.T, cfg config.Configuration, srv *testutil.TestServer) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	ctx := context.Background()
	env := make([]string, 0)
	outputs := make(map[string]string)

	// Simulate the aws ebs "volume_id" attribute registration
	srv.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/nodes/BlockStorage/type"):                       []byte("yorc.nodes.aws.EBSVolume"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/BlockStorage/0/attributes/volume_id"): []byte("my_vol_id"),
	})

	err := g.generateAWSInstance(ctx, cfg, deploymentID, "ComputeAWS", "0", &infrastructure, outputs, &env)
	require.Nil(t, err)
	require.Len(t, infrastructure.Resource["aws_instance"], 1)

	instancesMap := infrastructure.Resource["aws_instance"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, "ComputeAWS-0")

	compute, ok := instancesMap["ComputeAWS-0"].(*ComputeInstance)
	require.True(t, ok, "ComputeAWS-0 is not a ComputeInstance")
	require.Equal(t, "yorc-keypair", compute.KeyName)
	require.Equal(t, "ami-16dffe73", compute.ImageID)
	require.Equal(t, "t2.micro", compute.InstanceType)
	require.Equal(t, "ComputeAWS-0", compute.Tags.Name)
	require.Len(t, compute.SecurityGroups, 1)
	require.Contains(t, compute.SecurityGroups, "yorc-securityGroup")

	require.Len(t, infrastructure.Resource["aws_volume_attachment"], 1, "Expected one attached disk")
	instancesMap = infrastructure.Resource["aws_volume_attachment"].(map[string]interface{})
	require.Len(t, instancesMap, 1)

	attachmentResourceName := "blockstorage-0-to-computeaws-0"
	require.Contains(t, instancesMap, attachmentResourceName)
	attachedDisk, ok := instancesMap[attachmentResourceName].(*VolumeAttachment)
	require.True(t, ok, "%s is not a VolumeAttachment", attachmentResourceName)
	assert.Equal(t, "/dev/sdf", attachedDisk.DeviceName)

}

func testSimpleAWSInstanceWitthVPC(t *testing.T, cfg config.Configuration, srv *testutil.TestServer) {
	t.Parallel()

	deploymentID := loadTestYaml(t)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	ctx := context.Background()
	env := make([]string, 0)
	outputs := make(map[string]string)

	srv.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/nodes/Network/type"):                                    []byte("yorc.nodes.aws.VPC"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/Network/0/attributes/default_subnet_id"):      []byte("default_subnet_id"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/Network/0/attributes/default_security_group"): []byte("default_security_group"),
	})

	err := g.generateAWSInstance(ctx, cfg, deploymentID, "AWS_Compute", "0", &infrastructure, outputs, &env)
	require.Nil(t, err)
	require.Len(t, infrastructure.Resource["aws_instance"], 1)

	instancesMap := infrastructure.Resource["aws_instance"].(map[string]interface{})
	require.Contains(t, instancesMap, "AWS_Compute-0")

	compute, ok := instancesMap["AWS_Compute-0"].(*ComputeInstance)
	require.True(t, ok, "%s is not a ComputeInstance", "AWS_Compute-0")

	assert.Equal(t, "${aws_network_interface.network-inteface-0.id}", compute.NetworkInterface["network_interface_id"], "Subnetwork is not retrieved")
	assert.Equal(t, "0", compute.NetworkInterface["device_index"], "Subnetwork is not retrieved")

	instancesMap = infrastructure.Resource["aws_network_interface"].(map[string]interface{})
	require.Contains(t, instancesMap, "network-inteface-0")
	eni, ok := instancesMap["network-inteface-0"].(*NetworkInterface)
	assert.Equal(t, "default_subnet_id", eni.SubnetID)
}
