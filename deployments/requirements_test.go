package deployments

import (
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func TestRequirements(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	srv1, err := testutil.NewTestServer()
	if err != nil {
		t.Fatalf("Failed to create consul server: %v", err)
	}
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	require.Nil(t, err)

	kv := client.KV()

	srv1.PopulateKV(t, map[string][]byte{
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/type":                []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/0/name": []byte("network"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/1/name": []byte("host"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/2/name": []byte("network"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/3/name": []byte("host"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/4/name": []byte("network"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/5/name": []byte("storage"),

		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/0/node": []byte("TNode1"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/1/node": []byte("TNode1"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/2/node": []byte("TNode2"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/3/node": []byte("TNode2"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/4/node": []byte("TNode3"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/5/node": []byte("TNode4"),
	})

	t.Run("deployment/requirements", func(t *testing.T) {
		t.Run("GetRequirementsKeysByNameForNode", func(t *testing.T) {
			testGetRequirementsKeysByNameForNode(t, kv)
		})
		t.Run("GetRequirementByNameAndTargetForNode", func(t *testing.T) {
			testGetRequirementByNameAndTargetForNode(t, kv)
		})
		t.Run("GetNbRequirementsForNode", func(t *testing.T) {
			testGetNbRequirementsForNode(t, kv)
		})
	})
}

func testGetNbRequirementsForNode(t *testing.T, kv *api.KV) {
	t.Parallel()
	reqNb, err := GetNbRequirementsForNode(kv, "t1", "Compute1")
	require.Nil(t, err)
	require.Equal(t, 6, reqNb)

	reqNb, err = GetNbRequirementsForNode(kv, "t1", "do_not_exits")
	require.Nil(t, err)
	require.Equal(t, 0, reqNb)

}

func testGetRequirementsKeysByNameForNode(t *testing.T, kv *api.KV) {
	t.Parallel()
	keys, err := GetRequirementsKeysByNameForNode(kv, "t1", "Compute1", "network")
	require.Nil(t, err)
	require.Len(t, keys, 3)
	require.Contains(t, keys, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/0")
	require.Contains(t, keys, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/2")
	require.Contains(t, keys, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/4")

	keys, err = GetRequirementsKeysByNameForNode(kv, "t1", "Compute1", "host")
	require.Nil(t, err)
	require.Len(t, keys, 2)
	require.Contains(t, keys, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/1")
	require.Contains(t, keys, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/3")

	keys, err = GetRequirementsKeysByNameForNode(kv, "t1", "Compute1", "storage")
	require.Nil(t, err)
	require.Len(t, keys, 1)
	require.Contains(t, keys, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/5")

	keys, err = GetRequirementsKeysByNameForNode(kv, "t1", "Compute1", "dns")
	require.Nil(t, err)
	require.Len(t, keys, 0)

}

func testGetRequirementByNameAndTargetForNode(t *testing.T, kv *api.KV) {
	t.Parallel()
	reqKey, err := GetRequirementByNameAndTargetForNode(kv, "t1", "Compute1", "network", "TNode1")
	require.Nil(t, err)
	require.Equal(t, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/0", reqKey)

	reqKey, err = GetRequirementByNameAndTargetForNode(kv, "t1", "Compute1", "host", "TNode1")
	require.Nil(t, err)
	require.Equal(t, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/1", reqKey)

	reqKey, err = GetRequirementByNameAndTargetForNode(kv, "t1", "Compute1", "network", "TNode2")
	require.Nil(t, err)
	require.Equal(t, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/2", reqKey)

	reqKey, err = GetRequirementByNameAndTargetForNode(kv, "t1", "Compute1", "host", "TNode2")
	require.Nil(t, err)
	require.Equal(t, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/3", reqKey)

	reqKey, err = GetRequirementByNameAndTargetForNode(kv, "t1", "Compute1", "network", "TNode3")
	require.Nil(t, err)
	require.Equal(t, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/4", reqKey)

	reqKey, err = GetRequirementByNameAndTargetForNode(kv, "t1", "Compute1", "storage", "TNode4")
	require.Nil(t, err)
	require.Equal(t, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/5", reqKey)

	reqKey, err = GetRequirementByNameAndTargetForNode(kv, "t1", "Compute1", "do_not_exits", "TNode2")
	require.Nil(t, err)
	require.Equal(t, "", reqKey)

	reqKey, err = GetRequirementByNameAndTargetForNode(kv, "t1", "Compute1", "storage", "TNode2")
	require.Nil(t, err)
	require.Equal(t, "", reqKey)

	reqKey, err = GetRequirementByNameAndTargetForNode(kv, "t1", "Compute1", "do_not_exits", "do_not_exits")
	require.Nil(t, err)
	require.Equal(t, "", reqKey)

	reqKey, err = GetRequirementByNameAndTargetForNode(kv, "t1", "do_not_exits", "storage", "TNode2")
	require.Nil(t, err)
	require.Equal(t, "", reqKey)
}
