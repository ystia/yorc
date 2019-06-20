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

package deployments

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
)

func testArtifacts(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	log.SetDebug(true)

	deploymentID := strings.Replace(t.Name(), "/", "_", -1)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/artifacts.yaml")
	require.Nil(t, err)

	srv1.PopulateKV(t, map[string][]byte{
		consulutil.DeploymentKVPrefix + "/" + deploymentID + "/topology/types/yorc.types.A/importPath": []byte("path/to/typeA"),
	})

	t.Run("groupDeploymentArtifacts", func(t *testing.T) {
		t.Run("TestGetArtifactsForType", func(t *testing.T) {
			testGetArtifactsForType(t, kv, deploymentID)
		})
		t.Run("TestGetArtifactsForNode", func(t *testing.T) {
			testGetArtifactsForNode(t, kv, deploymentID)
		})
	})
}

func testGetArtifactsForType(t *testing.T, kv *api.KV, deploymentID string) {
	artifacts, err := GetArtifactsForType(kv, deploymentID, "yorc.types.A")
	require.Nil(t, err)
	require.NotNil(t, artifacts)
	require.Len(t, artifacts, 5)
	require.Contains(t, artifacts, "art1")
	require.Equal(t, "path/to/typeA/TypeA", artifacts["art1"])
	require.Contains(t, artifacts, "art2")
	require.Equal(t, "path/to/typeA/TypeA", artifacts["art2"])
	require.Contains(t, artifacts, "art6")
	require.Equal(t, "path/to/typeA/TypeA", artifacts["art6"])
	require.Contains(t, artifacts, "art3")
	require.Equal(t, "ParentA", artifacts["art3"])
	require.Contains(t, artifacts, "art5")
	require.Equal(t, "ParentA", artifacts["art5"])

	artifacts, err = GetArtifactsForType(kv, deploymentID, "yorc.types.ParentA")
	require.Nil(t, err)
	require.NotNil(t, artifacts)
	require.Len(t, artifacts, 3)
	require.Contains(t, artifacts, "art1")
	require.Equal(t, "ParentA", artifacts["art1"])
	require.Contains(t, artifacts, "art3")
	require.Equal(t, "ParentA", artifacts["art3"])
	require.Contains(t, artifacts, "art5")
	require.Equal(t, "ParentA", artifacts["art5"])

	artifacts, err = GetArtifactsForType(kv, deploymentID, "root")
	require.Nil(t, err)
	require.NotNil(t, artifacts)
	require.Len(t, artifacts, 0)

}
func testGetArtifactsForNode(t *testing.T, kv *api.KV, deploymentID string) {
	artifacts, err := GetArtifactsForNode(kv, deploymentID, "NodeA")
	require.Nil(t, err)
	require.NotNil(t, artifacts)
	require.Len(t, artifacts, 6)

	require.Contains(t, artifacts, "art1")
	require.Equal(t, "artifacts.yaml", artifacts["art1"])
	require.Contains(t, artifacts, "art2")
	require.Equal(t, "artifacts.yaml", artifacts["art2"])
	require.Contains(t, artifacts, "art3")
	require.Equal(t, "artifacts.yaml", artifacts["art3"])
	require.Contains(t, artifacts, "art4")
	require.Equal(t, "artifacts.yaml", artifacts["art4"])
	require.Contains(t, artifacts, "art5")
	require.Equal(t, "ParentA", artifacts["art5"])
	require.Contains(t, artifacts, "art6")
	require.Equal(t, "path/to/typeA/TypeA", artifacts["art6"])

	artifacts, err = GetArtifactsForNode(kv, deploymentID, "NodeB")
	require.Nil(t, err)
	require.NotNil(t, artifacts)
	require.Len(t, artifacts, 0)
}
