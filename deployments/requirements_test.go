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

func testRequirements(t *testing.T, srv1 *testutil.TestServer, kv *api.KV) {
	log.SetDebug(true)

	srv1.PopulateKV(t, map[string][]byte{
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/type":                            []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/0/name":             []byte("network"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/1/name":             []byte("host"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/2/name":             []byte("network_1"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/2/type_requirement": []byte("network"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/3/name":             []byte("host_1"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/3/type_requirement": []byte("host"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/4/name":             []byte("network_2"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/4/type_requirement": []byte("network"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/5/name":             []byte("storage"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/6/name":             []byte("storage_other"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/6/type_requirement": []byte("storage"),

		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/0/node": []byte("TNode1"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/1/node": []byte("TNode1"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/2/node": []byte("TNode2"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/3/node": []byte("TNode2"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/4/node": []byte("TNode3"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/5/node": []byte("TNode4"),
		consulutil.DeploymentKVPrefix + "/t1/topology/nodes/Compute1/requirements/6/node": []byte("TNode5"),
	})

	t.Run("groupDeploymentsRequirements", func(t *testing.T) {
		t.Run("TestGetRequirementsKeysByTypeForNode", func(t *testing.T) {
			testGetRequirementsKeysByTypeForNode(t, kv)
		})
		t.Run("TestGetRequirementKeyByNameForNode", func(t *testing.T) {
			testGetRequirementKeyByNameForNode(t, kv)
		})
		t.Run("TestGetNbRequirementsForNode", func(t *testing.T) {
			testGetNbRequirementsForNode(t, kv)
		})
	})
}

func testGetRequirementKeyByNameForNode(t *testing.T, kv *api.KV) {
	// t.Parallel()
	key, err := GetRequirementKeyByNameForNode(kv, "t1", "Compute1", "host_1")
	require.Nil(t, err)
	require.Equal(t, key, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/3")

	key, err = GetRequirementKeyByNameForNode(kv, "t1", "Compute1", "do_not_exits")
	require.Nil(t, err)
	require.Equal(t, "", key)
}

func testGetNbRequirementsForNode(t *testing.T, kv *api.KV) {
	// t.Parallel()
	reqNb, err := GetNbRequirementsForNode(kv, "t1", "Compute1")
	require.Nil(t, err)
	require.Equal(t, 7, reqNb)

	reqNb, err = GetNbRequirementsForNode(kv, "t1", "do_not_exits")
	require.Nil(t, err)
	require.Equal(t, 0, reqNb)

}

func testGetRequirementsKeysByTypeForNode(t *testing.T, kv *api.KV) {
	// t.Parallel()
	keys, err := GetRequirementsKeysByTypeForNode(kv, "t1", "Compute1", "network")
	require.Nil(t, err)
	require.Len(t, keys, 3)
	require.Contains(t, keys, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/0")
	require.Contains(t, keys, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/2")
	require.Contains(t, keys, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/4")

	keys, err = GetRequirementsKeysByTypeForNode(kv, "t1", "Compute1", "host")
	require.Nil(t, err)
	require.Len(t, keys, 2)
	require.Contains(t, keys, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/1")
	require.Contains(t, keys, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/3")

	keys, err = GetRequirementsKeysByTypeForNode(kv, "t1", "Compute1", "storage")
	require.Nil(t, err)
	require.Len(t, keys, 2)
	require.Contains(t, keys, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/5")
	require.Contains(t, keys, consulutil.DeploymentKVPrefix+"/t1/topology/nodes/Compute1/requirements/6")

	keys, err = GetRequirementsKeysByTypeForNode(kv, "t1", "Compute1", "dns")
	require.Nil(t, err)
	require.Len(t, keys, 0)

}
