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

func testSimpleOSInstanceFailed(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)

	cfg := config.Configuration{
		Infrastructures: map[string]config.InfrastructureConfig{
			infrastructureName: {
				"region":     "us-east-2",
				"access_key": "test",
				"secret_key": "test",
			}}}
	g := osGenerator{}
	infrastructure := commons.Infrastructure{}

	err := g.generateOSInstance(context.Background(), kv, cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string))
	require.Error(t, err, "Expecting missing mandatory parameter 'instance_type' error")
}

func testSimpleOSInstance(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)

	cfg := config.Configuration{
		Infrastructures: map[string]config.InfrastructureConfig{
			infrastructureName: {
				"region":     "us-east-2",
				"access_key": "test",
				"secret_key": "test",
			}}}
	g := osGenerator{}
	infrastructure := commons.Infrastructure{}

	err := g.generateOSInstance(context.Background(), kv, cfg, deploymentID, "ComputeAWS", "0", &infrastructure, make(map[string]string))
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
