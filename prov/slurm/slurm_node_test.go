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

package slurm

import (
	"context"
	"path"
	"strconv"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/deployments"
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
	infrastructure := infrastructure{}

	err := generateNodeAllocation(context.Background(), kv, cfg, deploymentID, "Compute", "0", &infrastructure)
	require.Nil(t, err)

	require.Len(t, infrastructure.nodes, 1)
	require.Equal(t, "0", infrastructure.nodes[0].instanceName)
	require.Equal(t, "gpu:1", infrastructure.nodes[0].gres)
	require.Equal(t, "[rack1|rack2|rack3|rack4]", infrastructure.nodes[0].constraint)
	require.Equal(t, "debug", infrastructure.nodes[0].partition)
	require.Equal(t, "2G", infrastructure.nodes[0].memory)
	require.Equal(t, "4", infrastructure.nodes[0].cpu)
	require.Equal(t, "xyz", infrastructure.nodes[0].jobName)
	require.Equal(t, "johndoe", infrastructure.nodes[0].credentials.User)
	require.Equal(t, "passpass", infrastructure.nodes[0].credentials.Token)
	require.Equal(t, "resa_123", infrastructure.nodes[0].reservation)
	require.Equal(t, "account_test", infrastructure.nodes[0].account)
}

func testSimpleSlurmNodeAllocationWithoutProps(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	infrastructure := infrastructure{}

	err := generateNodeAllocation(context.Background(), kv, cfg, deploymentID, "Compute", "0", &infrastructure)
	require.Nil(t, err)

	require.Len(t, infrastructure.nodes, 1)
	require.Equal(t, "0", infrastructure.nodes[0].instanceName)
	require.Equal(t, "", infrastructure.nodes[0].gres)
	require.Equal(t, "", infrastructure.nodes[0].constraint)
	require.Equal(t, "", infrastructure.nodes[0].partition)
	require.Equal(t, "", infrastructure.nodes[0].memory)
	require.Equal(t, "", infrastructure.nodes[0].cpu)
	require.Equal(t, "root", infrastructure.nodes[0].credentials.User)
	require.Equal(t, "pwd", infrastructure.nodes[0].credentials.Token)
	require.Equal(t, "simpleSlurmNodeAllocationWithoutProps", infrastructure.nodes[0].jobName)
}

func testMultipleSlurmNodeAllocation(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	infrastructure := infrastructure{}

	nb, err := deployments.GetDefaultNbInstancesForNode(kv, deploymentID, "Compute")
	require.Nil(t, err)
	require.Equal(t, uint32(4), nb)

	for i := 0; i < int(nb); i++ {
		istr := strconv.Itoa(i)
		err := generateNodeAllocation(context.Background(), kv, cfg, deploymentID, "Compute", istr, &infrastructure)
		require.Nil(t, err)

		require.Len(t, infrastructure.nodes, i+1)
		require.Equal(t, istr, infrastructure.nodes[i].instanceName)
		require.Equal(t, "gpu:1", infrastructure.nodes[i].gres)
		require.Equal(t, "[rack1*2&rack2*4]", infrastructure.nodes[i].constraint)
		require.Equal(t, "debug", infrastructure.nodes[i].partition)
		require.Equal(t, "2G", infrastructure.nodes[i].memory)
		require.Equal(t, "4", infrastructure.nodes[i].cpu)
		require.Equal(t, "xyz", infrastructure.nodes[i].jobName)
	}
}
