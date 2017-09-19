package aws

import (
	"context"
	"path"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
)

func loadTestYaml(t *testing.T, kv *api.KV) string {
	deploymentID := path.Base(t.Name())
	yamlName := "testdata/" + deploymentID + ".yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), kv, deploymentID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")
	return deploymentID
}

func testSimpleAWSInstanceFailed(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)

	cfg := config.Configuration{
		Infrastructures: map[string]config.InfrastructureConfig{
			infrastructureName: {
				"region":     "us-east-2",
				"access_key": "test",
				"secret_key": "test",
			}}}
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}

	err := g.generateAWSInstance(context.Background(), kv, cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string))
	require.Error(t, err, "Expecting missing mandatory parameter 'instance_type' error")
}

func testSimpleAWSInstance(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)

	cfg := config.Configuration{
		Infrastructures: map[string]config.InfrastructureConfig{
			infrastructureName: {
				"region":     "us-east-2",
				"access_key": "test",
				"secret_key": "test",
			}}}
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}

	err := g.generateAWSInstance(context.Background(), kv, cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string))
	require.Nil(t, err)

	require.Len(t, infrastructure.Resource["aws_instance"], 1)
	instancesMap := infrastructure.Resource["aws_instance"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, "ComputeAWS-0")

	compute, ok := instancesMap["ComputeAWS-0"].(*ComputeInstance)
	require.True(t, ok, "ComputeAWS-0 is not a ComputeInstance")
	require.Equal(t, "janus-keypair", compute.KeyName)
	require.Equal(t, "ami-16dffe73", compute.ImageID)
	require.Equal(t, "t2.micro", compute.InstanceType)
	require.Equal(t, "ComputeAWS-0", compute.Tags.Name)
	require.Len(t, compute.SecurityGroups, 1)
	require.Contains(t, compute.SecurityGroups, "janus-securityGroup")

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
	require.Equal(t, `${file("~/.ssh/janus.pem")}`, rex.Connection.PrivateKey)
}

func testSimpleAWSInstanceWithEIP(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)

	cfg := config.Configuration{
		Infrastructures: map[string]config.InfrastructureConfig{
			infrastructureName: {
				"region":     "us-east-2",
				"access_key": "test",
				"secret_key": "test",
			}}}
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}

	err := g.generateAWSInstance(context.Background(), kv, cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string))
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

func testSimpleAWSInstanceWithProvidedEIP(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)

	cfg := config.Configuration{
		Infrastructures: map[string]config.InfrastructureConfig{
			infrastructureName: {
				"region":     "us-east-2",
				"access_key": "test",
				"secret_key": "test",
			}}}
	g := awsGenerator{}
	infrastructure := commons.Infrastructure{}

	err := g.generateAWSInstance(context.Background(), kv, cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string))
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
