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
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/prov/terraform/commons"
)

func testSimpleComputeAddress(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	infrastructure := commons.Infrastructure{}
	g := googleGenerator{}
	err := g.generateComputeAddress(context.Background(), kv, cfg, deploymentID, "ComputeAddress", "0", 0, &infrastructure, make(map[string]string))
	require.NoError(t, err, "Unexpected error attempting to generate compute address for %s", deploymentID)

	resourcePrefix := getResourcesPrefix(cfg, deploymentID)
	addressName := resourcePrefix + "computeaddress-0"
	require.Len(t, infrastructure.Resource["google_compute_address"], 1, "Expected one compute address")
	instancesMap := infrastructure.Resource["google_compute_address"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, addressName)

	computeAddress, ok := instancesMap[addressName].(*ComputeAddress)
	require.True(t, ok, "%s is not a ComputeAddress", addressName)
	assert.Equal(t, addressName, computeAddress.Name)
	assert.Equal(t, "europe-west1", computeAddress.Region)
	assert.Equal(t, "EXTERNAL", computeAddress.AddressType)
	assert.Equal(t, "PREMIUM", computeAddress.NetworkTier)
	assert.Equal(t, "the description", computeAddress.Description)
	assert.Equal(t, map[string]string{"key1": "value1", "key2": "value2"}, computeAddress.Labels)
}
