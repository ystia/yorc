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

package google

import (
	"context"
	"path"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/prov/terraform/commons"
)

func loadTestYaml(t *testing.T, kv *api.KV) string {
	deploymentID := path.Base(t.Name())
	yamlName := "testdata/" + deploymentID + ".yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), kv, deploymentID, yamlName)
	require.NoError(t, err, "Failed to parse "+yamlName+" definition")
	return deploymentID
}

func testSimpleComputeInstance(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	infrastructure := commons.Infrastructure{}
	g := googleGenerator{}
	err := g.generateComputeInstance(context.Background(), kv, cfg, deploymentID, "ComputeInstance", "0", &infrastructure, make(map[string]string))
	require.NoError(t, err, "Unexpected error attempting to generate compute instance for %s", deploymentID)

	require.Len(t, infrastructure.Resource["google_compute_instance"], 1, "Expected one compute instance")
	instancesMap := infrastructure.Resource["google_compute_instance"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, "computeinstance-0")

	compute, ok := instancesMap["computeinstance-0"].(*ComputeInstance)
	require.True(t, ok, "computeinstance-0 is not a ComputeInstance")
	assert.Equal(t, "n1-standard-1", compute.MachineType)
	assert.Equal(t, "europe-west1-b", compute.Zone)
	require.Len(t, compute.Disks, 1, "Expected one boot disk")
	assert.Equal(t, "centos-cloud/centos-7", compute.Disks[0].Image, "Unexpected boot disk image")
	require.Len(t, compute.NetworkInterfaces, 1, "Expected one network interface for external access")
	assert.Equal(t, []string{"tag1", "tag2"}, compute.Tags)
	assert.Equal(t, map[string]string{"key1": "value1", "key2": "value2"}, compute.Labels)

	require.Len(t, compute.ServiceAccounts, 0, "Expected no service account")

	require.Contains(t, infrastructure.Resource, "null_resource")
	require.Len(t, infrastructure.Resource["null_resource"], 1)
	nullResources := infrastructure.Resource["null_resource"].(map[string]interface{})

	require.Contains(t, nullResources, "computeinstance-0-ConnectionCheck")
	nullRes, ok := nullResources["computeinstance-0-ConnectionCheck"].(*commons.Resource)
	assert.True(t, ok)
	require.Len(t, nullRes.Provisioners, 1)
	mapProv := nullRes.Provisioners[0]
	require.Contains(t, mapProv, "remote-exec")
	rex, ok := mapProv["remote-exec"].(commons.RemoteExec)
	require.True(t, ok)
	assert.Equal(t, "centos", rex.Connection.User)
	assert.Equal(t, `${file("~/.ssh/yorc.pem")}`, rex.Connection.PrivateKey)
}

func testSimpleComputeInstanceMissingMandatoryParameter(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := googleGenerator{}
	infrastructure := commons.Infrastructure{}

	err := g.generateComputeInstance(context.Background(), kv, cfg, deploymentID, "ComputeInstance", "0", &infrastructure, make(map[string]string))
	require.Error(t, err, "Expected missing mandatory parameter error, but had no error")
	assert.Contains(t, err.Error(), "mandatory parameter zone", "Expected an error on missing parameter zone")
}
