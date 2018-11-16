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

func testSimplePersistentDisk(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	resourcePrefix := getResourcesPrefix(cfg, deploymentID)
	infrastructure := commons.Infrastructure{}
	g := googleGenerator{}
	err := g.generatePersistentDisk(context.Background(), kv, cfg, deploymentID, "PersistentDisk", "0", 0, &infrastructure, make(map[string]string))
	require.NoError(t, err, "Unexpected error attempting to generate persistent disk for %s", deploymentID)

	diskName := resourcePrefix + "persistentdisk-0"
	require.Len(t, infrastructure.Resource["google_compute_disk"], 1, "Expected one persistent disk")
	instancesMap := infrastructure.Resource["google_compute_disk"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, diskName)

	persistentDisk, ok := instancesMap[diskName].(*PersistentDisk)
	require.True(t, ok, "%s is not a PersistentDisk", diskName)
	assert.Equal(t, diskName, persistentDisk.Name)
	assert.Equal(t, "europe-west1-b", persistentDisk.Zone)
	assert.Equal(t, 32, persistentDisk.Size)
	assert.Equal(t, "pd-ssd", persistentDisk.Type)
	assert.Equal(t, "1234", persistentDisk.DiskEncryptionKey.Raw)
	assert.Equal(t, "5678", persistentDisk.DiskEncryptionKey.SHA256)
	assert.Equal(t, "my description for persistent disk", persistentDisk.Description)
	assert.Equal(t, map[string]string{"key1": "value1", "key2": "value2"}, persistentDisk.Labels)
}
