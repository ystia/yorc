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
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/log"
)

func testArtifacts(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	t.Parallel()
	log.SetDebug(true)

	srv1.PopulateKV(t, map[string][]byte{

		consulutil.DeploymentKVPrefix + "/t1/topology/types/yorc.types.A/derived_from":        []byte("yorc.types.ParentA"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/yorc.types.A/artifacts/art1/name": []byte("art1"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/yorc.types.A/artifacts/art1/file": []byte("TypeA"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/yorc.types.A/artifacts/art2/name": []byte("art2"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/yorc.types.A/artifacts/art2/file": []byte("TypeA"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/yorc.types.A/artifacts/art6/name": []byte("art6"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/yorc.types.A/artifacts/art6/file": []byte("TypeA"),

		consulutil.DeploymentKVPrefix + "/t1/topology/types/yorc.types.ParentA/derived_from":        []byte("root"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/yorc.types.ParentA/artifacts/art1/name": []byte("art1"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/yorc.types.ParentA/artifacts/art1/file": []byte("ParentA"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/yorc.types.ParentA/artifacts/art3/name": []byte("art3"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/yorc.types.ParentA/artifacts/art3/file": []byte("ParentA"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/yorc.types.ParentA/artifacts/art5/name": []byte("art5"),
		consulutil.DeploymentKVPrefix + "/t1/topology/types/yorc.types.ParentA/artifacts/art5/file": []byte("ParentA"),

		consulutil.DeploymentKVPrefix + "/t1/topology/types/root/name": []byte("root"),

		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/NodeA/type":                []byte("yorc.types.A"),
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

	t.Run("groupDeploymentArtifacts", func(t *testing.T) {
		t.Run("TestGetArtifactsForType", func(t *testing.T) {
			testGetArtifactsForType(t, kv)
		})
		t.Run("TestGetArtifactsForNode", func(t *testing.T) {
			testGetArtifactsForNode(t, kv)
		})
	})
}

func testGetArtifactsForType(t *testing.T, kv *api.KV) {
	artifacts, err := GetArtifactsForType(kv, "t1", "yorc.types.A")
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

	artifacts, err = GetArtifactsForType(kv, "t1", "yorc.types.ParentA")
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
