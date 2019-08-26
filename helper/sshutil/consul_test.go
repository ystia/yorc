// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package sshutil

import (
	"context"
	"path"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/testutil"
)

func TestRunConsulSSHUtilPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()

	t.Run("KeysTests", func(t *testing.T) {
		testGetKeysFromCredentialsAttribute(t, kv)
	})
}

func loadTestYaml(t *testing.T, kv *api.KV, yamlName string) string {
	deploymentID := path.Base(t.Name())
	yamlName = "testdata/" + yamlName
	err := deployments.StoreDeploymentDefinition(context.Background(), kv, deploymentID, yamlName)
	require.NoError(t, err, "Failed to parse "+yamlName+" definition")
	return deploymentID
}
