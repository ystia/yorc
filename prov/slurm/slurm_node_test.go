package slurm

import (
	"context"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"path"
	"strconv"
	"testing"
)

func loadTestYaml(t *testing.T, kv *api.KV) string {
	deploymentID := path.Base(t.Name())
	yamlName := "testdata/" + deploymentID + ".yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), kv, deploymentID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")
	return deploymentID
}

func testSimpleSlurmNodeAllocation(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := slurmGenerator{}
	infrastructure := infrastructure{}

	err := g.generateNodeAllocation(context.Background(), kv, cfg, deploymentID, "Compute", "0", &infrastructure)
	require.Nil(t, err)

	require.Len(t, infrastructure.nodes, 1)
	require.Equal(t, "0", infrastructure.nodes[0].instanceName)
	require.Equal(t, "gpu:1", infrastructure.nodes[0].gres)
	require.Equal(t, "debug", infrastructure.nodes[0].partition)
	require.Equal(t, "2G", infrastructure.nodes[0].memory)
	require.Equal(t, "4", infrastructure.nodes[0].cpu)
	require.Equal(t, "xyz", infrastructure.nodes[0].name)
}

func testSimpleSlurmNodeAllocationWithoutProps(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := slurmGenerator{}
	infrastructure := infrastructure{}

	err := g.generateNodeAllocation(context.Background(), kv, cfg, deploymentID, "Compute", "0", &infrastructure)
	require.Nil(t, err)

	require.Len(t, infrastructure.nodes, 1)
	require.Equal(t, "0", infrastructure.nodes[0].instanceName)
	require.Equal(t, "", infrastructure.nodes[0].gres)
	require.Equal(t, "", infrastructure.nodes[0].partition)
	require.Equal(t, "", infrastructure.nodes[0].memory)
	require.Equal(t, "", infrastructure.nodes[0].cpu)
	require.Equal(t, "", infrastructure.nodes[0].name)
}

func testMultipleSlurmNodeAllocation(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := slurmGenerator{}
	infrastructure := infrastructure{}

	nb, err := deployments.GetDefaultNbInstancesForNode(kv, deploymentID, "Compute")
	require.Nil(t, err)
	require.Equal(t, uint32(4), nb)

	for i := 0; i < int(nb); i++ {
		istr := strconv.Itoa(i)
		err := g.generateNodeAllocation(context.Background(), kv, cfg, deploymentID, "Compute", istr, &infrastructure)
		require.Nil(t, err)

		require.Len(t, infrastructure.nodes, i+1)
		require.Equal(t, istr, infrastructure.nodes[i].instanceName)
		require.Equal(t, "gpu:1", infrastructure.nodes[i].gres)
		require.Equal(t, "debug", infrastructure.nodes[i].partition)
		require.Equal(t, "2G", infrastructure.nodes[i].memory)
		require.Equal(t, "4", infrastructure.nodes[i].cpu)
		require.Equal(t, "xyz", infrastructure.nodes[i].name)
	}
}
