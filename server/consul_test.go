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

package server

import (
	"testing"

	"github.com/blang/semver"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/testutil"
)

func TestConsulServerPackage(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	defer srv.Stop()

	t.Run("groupSchema", func(t *testing.T) {
		t.Run("testSetupVersion", func(t *testing.T) {
			testSetupVersion(t, client)
		})
	})
}

func setSchemaVersion(t *testing.T, kv *api.KV, version string) {
	t.Helper()
	_, err := kv.Put(&api.KVPair{Key: consulutil.YorcSchemaVersionPath, Value: []byte(version)}, nil)
	require.NoError(t, err)
}

func checkSchemaVersion(t *testing.T, kv *api.KV, version string) {
	t.Helper()
	kvp, _, err := kv.Get(consulutil.YorcSchemaVersionPath, nil)
	require.NoError(t, err)
	require.NotNil(t, kvp, "Consul schema version not set at the end of the process")
	assert.Equal(t, version, string(kvp.Value), "Consul schema version not set at the end of the process")

}

func checkCurrentSchemaVersion(t *testing.T, kv *api.KV) {
	checkSchemaVersion(t, kv, consulutil.YorcSchemaVersion)
}

func testSetupVersion(t *testing.T, client *api.Client) {
	// Test without any version set
	kv := client.KV()
	_, err := kv.Put(&api.KVPair{Key: consulutil.DeploymentKVPrefix + "/something", Value: []byte{1}}, nil)
	require.NoError(t, err)

	err = setupConsulDBSchema(client)
	assert.NoError(t, err)
	checkCurrentSchemaVersion(t, kv)

	// Now set a pre-3.1 schema version
	setSchemaVersion(t, kv, "0.5.0")
	err = setupConsulDBSchema(client)
	assert.NoError(t, err)
	checkCurrentSchemaVersion(t, kv)

	// Now check current version
	setSchemaVersion(t, kv, consulutil.YorcSchemaVersion)
	err = setupConsulDBSchema(client)
	assert.NoError(t, err)
	checkCurrentSchemaVersion(t, kv)

	// Now check a newer version
	v := semver.MustParse(consulutil.YorcSchemaVersion)
	v.Major++
	setSchemaVersion(t, kv, v.String())
	err = setupConsulDBSchema(client)

	assert.Error(t, err)
	checkSchemaVersion(t, kv, v.String())

}
