package deployments

import (
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func TestArtifacts(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	require.Nil(t, err)

	kv := client.KV()

	srv1.PopulateKV(map[string][]byte{

		consulutil.DeploymentKVPrefix + "/t1/topology/types/janus.types.A/derived_from":        []byte("janus.types.ParentA"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/janus.types.A/artifacts/art1/name": []byte("art1"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/janus.types.A/artifacts/art1/file": []byte("TypeA"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/janus.types.A/artifacts/art2/name": []byte("art2"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/janus.types.A/artifacts/art2/file": []byte("TypeA"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/janus.types.A/artifacts/art6/name": []byte("art6"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/janus.types.A/artifacts/art6/file": []byte("TypeA"),

		consulutil.DeploymentKVPrefix + "/t1/topology/types/janus.types.ParentA/derived_from":        []byte("root"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/janus.types.ParentA/artifacts/art1/name": []byte("art1"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/janus.types.ParentA/artifacts/art1/file": []byte("ParentA"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/janus.types.ParentA/artifacts/art3/name": []byte("art3"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/janus.types.ParentA/artifacts/art3/file": []byte("ParentA"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/janus.types.ParentA/artifacts/art5/name": []byte("art5"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/janus.types.ParentA/artifacts/art5/file": []byte("ParentA"),

		consulutil.DeploymentKVPrefix + "/t1/topology/types/root/name": []byte("root"),

		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/NodeA/type":                []byte("janus.types.A"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/NodeA/artifacts/art1/name": []byte("art1"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/NodeA/artifacts/art1/file": []byte("NodeA"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/NodeA/artifacts/art2/name": []byte("art2"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/NodeA/artifacts/art2/file": []byte("NodeA"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/NodeA/artifacts/art3/name": []byte("art3"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/NodeA/artifacts/art3/file": []byte("NodeA"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/NodeA/artifacts/art4/name": []byte("art4"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/NodeA/artifacts/art4/file": []byte("NodeA"),

		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/NodeB/type": []byte("root"),
	})

	t.Run("deployment/artifacts", func(t *testing.T) {
		t.Run("GetArtifactsForType", func(t *testing.T) {
			testGetArtifactsForType(t, kv)
		})
		t.Run("GetArtifactsForNode", func(t *testing.T) {
			testGetArtifactsForNode(t, kv)
		})
	})
}

func testGetArtifactsForType(t *testing.T, kv *api.KV) {
	artifacts, err := GetArtifactsForType(kv, "t1", "janus.types.A")
	require.Nil(t, err)
	require.NotNil(t, artifacts)
	require.Len(t, artifacts, 5)
	require.Contains(t, artifacts, "art1")
	require.Equal(t, "TypeA", artifacts["art1"])
	require.Contains(t, artifacts, "art2")
	require.Equal(t, "TypeA", artifacts["art2"])
	require.Contains(t, artifacts, "art6")
	require.Equal(t, "TypeA", artifacts["art6"])
	require.Contains(t, artifacts, "art3")
	require.Equal(t, "ParentA", artifacts["art3"])
	require.Contains(t, artifacts, "art5")
	require.Equal(t, "ParentA", artifacts["art5"])

	artifacts, err = GetArtifactsForType(kv, "t1", "janus.types.ParentA")
	require.Nil(t, err)
	require.NotNil(t, artifacts)
	require.Len(t, artifacts, 3)
	require.Contains(t, artifacts, "art1")
	require.Equal(t, "ParentA", artifacts["art1"])
	require.Contains(t, artifacts, "art3")
	require.Equal(t, "ParentA", artifacts["art3"])
	require.Contains(t, artifacts, "art5")
	require.Equal(t, "ParentA", artifacts["art5"])

	artifacts, err = GetArtifactsForType(kv, "t1", "root")
	require.Nil(t, err)
	require.NotNil(t, artifacts)
	require.Len(t, artifacts, 0)

}
func testGetArtifactsForNode(t *testing.T, kv *api.KV) {
	artifacts, err := GetArtifactsForNode(kv, "t1", "NodeA")
	require.Nil(t, err)
	require.NotNil(t, artifacts)
	require.Len(t, artifacts, 6)

	require.Contains(t, artifacts, "art1")
	require.Equal(t, "NodeA", artifacts["art1"])
	require.Contains(t, artifacts, "art2")
	require.Equal(t, "NodeA", artifacts["art2"])
	require.Contains(t, artifacts, "art3")
	require.Equal(t, "NodeA", artifacts["art3"])
	require.Contains(t, artifacts, "art4")
	require.Equal(t, "NodeA", artifacts["art4"])
	require.Contains(t, artifacts, "art5")
	require.Equal(t, "ParentA", artifacts["art5"])
	require.Contains(t, artifacts, "art6")
	require.Equal(t, "TypeA", artifacts["art6"])

	artifacts, err = GetArtifactsForNode(kv, "t1", "NodeB")
	require.Nil(t, err)
	require.NotNil(t, artifacts)
	require.Len(t, artifacts, 0)
}
