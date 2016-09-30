package deployments

import (
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/log"
	"testing"
)

func TestRequirements(t *testing.T) {
	log.SetDebug(true)
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	require.Nil(t, err)

	kv := client.KV()

	srv1.PopulateKV(map[string][]byte{
		DeploymentKVPrefix + "/t1/topology/nodes/Compute1/type":                []byte("tosca.nodes.Compute"),
		DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/0/name": []byte("network"),
		DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/1/name": []byte("host"),
		DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/2/name": []byte("network"),
		DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/3/name": []byte("host"),
		DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/4/name": []byte("network"),
		DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/5/name": []byte("storage"),
	})

	t.Run("deployment/requirements", func(t *testing.T) {
		t.Run("GetRequirementsKeysByNameForNode", func(t *testing.T) {
			testGetRequirementsKeysByNameForNode(t, kv)
		})
	})
}
func testGetRequirementsKeysByNameForNode(t *testing.T, kv *api.KV) {
	t.Parallel()
	keys, err := GetRequirementsKeysByNameForNode(kv, "t1", "Compute1", "network")
	require.Nil(t, err)
	require.Len(t, keys, 3)
	require.Contains(t, keys, DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/0")
	require.Contains(t, keys, DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/2")
	require.Contains(t, keys, DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/4")

	keys, err = GetRequirementsKeysByNameForNode(kv, "t1", "Compute1", "host")
	require.Nil(t, err)
	require.Len(t, keys, 2)
	require.Contains(t, keys, DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/1")
	require.Contains(t, keys, DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/3")

	keys, err = GetRequirementsKeysByNameForNode(kv, "t1", "Compute1", "storage")
	require.Nil(t, err)
	require.Len(t, keys, 1)
	require.Contains(t, keys, DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/5")

	keys, err = GetRequirementsKeysByNameForNode(kv, "t1", "Compute1", "dns")
	require.Nil(t, err)
	require.Len(t, keys, 0)

}
