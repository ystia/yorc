package openstack

import (
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"context"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/commons"
)

func TestGroupedNOsInstanceParallel(t *testing.T) {
	t.Run("groupVolume", func(t *testing.T) {
		t.Run("generateOSInstance", generateOSInstance)
		//t.Run("generateSubNetwork", generateSubNetwork)
	})
}

func generateOSInstance(t *testing.T) {
	t.Parallel()
	srv1, err := testutil.NewTestServer()
	require.Nil(t, err)
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	require.Nil(t, err)

	kv := client.KV()
	consulutil.InitConsulPublisher(config.DefaultConsulPubMaxRoutines, kv)
	deploymentID := "testOsInstance"
	err = deployments.StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/dep.yaml")
	assert.Nil(t, err)

	g := osGenerator{}
	err = g.generateOSInstance(context.Background(), kv, config.Configuration{}, deploymentID, "Compute", "0",&commons.Infrastructure{}, nil)
	assert.Nil(t, err)

}

