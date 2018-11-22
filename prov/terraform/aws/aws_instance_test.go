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
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/helper/sshutil"
	"github.com/ystia/yorc/prov/terraform/commons"
	"strconv"
)

func loadTestYaml(t *testing.T, kv *api.KV) string {
	deploymentID := path.Base(t.Name())
	yamlName := "testdata/" + deploymentID + ".yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), kv, deploymentID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")
	return deploymentID
}

func testSimpleAWSInstanceFailed(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	err := g.generateAWSInstance(context.Background(), kv, cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string), &env)
	require.Error(t, err, "Expecting missing mandatory parameter 'instance_type' error")
}

func testSimpleAWSInstance(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	err := g.generateAWSInstance(context.Background(), kv, cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string), &env)
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

	yorcPem, err := sshutil.ToPrivateKeyContent("~/.ssh/yorc.pem")
	require.Nil(t, err)
	assert.Equal(t, "${var.private_key}", rex.Connection.PrivateKey)
	require.Len(t, env, 1)
	assert.Equal(t, "TF_VAR_private_key="+string(yorcPem), env[0], "env var for private key expected")

	require.NotContains(t, infrastructure.Resource, "aws_eip_association")
}

func testSimpleAWSInstanceWithPrivateKey(t *testing.T, kv *api.KV, cfg config.Configuration) {
	privateKey := []byte(`-----BEGIN RSA PRIVATE KEY----- my secure private key -----END RSA PRIVATE KEY-----`)
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	err := g.generateAWSInstance(context.Background(), kv, cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string), &env)
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
	assert.Equal(t, "TF_VAR_private_key="+string(privateKey), env[0], "env var for private key expected")

	require.NotContains(t, infrastructure.Resource, "aws_eip_association")
}

func testSimpleAWSInstanceWithNoDeleteVolumeOnTermination(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	err := g.generateAWSInstance(context.Background(), kv, cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string), &env)
	require.Nil(t, err)

	require.Len(t, infrastructure.Resource["aws_instance"], 1)
	instancesMap := infrastructure.Resource["aws_instance"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, "ComputeAWS-0")

	compute, ok := instancesMap["ComputeAWS-0"].(*ComputeInstance)
	require.True(t, ok, "ComputeAWS-0 is not a ComputeInstance")
	require.Equal(t, false, compute.RootBlockDevice.DeleteOnTermination)
}

func testSimpleAWSInstanceWithEIP(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	err := g.generateAWSInstance(context.Background(), kv, cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string), &env)
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

func testSimpleAWSInstanceWithProvidedEIP(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	err := g.generateAWSInstance(context.Background(), kv, cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string), &env)
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

func testSimpleAWSInstanceWithListOfProvidedEIP(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	nb, err := deployments.GetDefaultNbInstancesForNode(kv, deploymentID, "ComputeAWS")
	require.Nil(t, err)
	require.Equal(t, uint32(4), nb)

	for i := 0; i < int(nb); i++ {
		istr := strconv.Itoa(i)
		err = g.generateAWSInstance(context.Background(), kv, cfg, deploymentID, "ComputeAWS", istr, &infrastructure, make(map[string]string), &env)
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

func testSimpleAWSInstanceWithMalformedEIP(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	err := g.generateAWSInstance(context.Background(), kv, cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string), &env)
	require.Error(t, err, "An error was expected due to malformed provided Elastic IP: 12.12.oups.12")
}

func testSimpleAWSInstanceWithNotEnoughProvidedEIPS(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	nb, err := deployments.GetDefaultNbInstancesForNode(kv, deploymentID, "ComputeAWS")
	require.Nil(t, err)
	require.Equal(t, uint32(5), nb)

	for i := 0; i < int(nb); i++ {
		istr := strconv.Itoa(i)
		err = g.generateAWSInstance(context.Background(), kv, cfg, deploymentID, "ComputeAWS", istr, &infrastructure, make(map[string]string), &env)
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
