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

package openstack

import (
	"context"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
	"path"
	"testing"
)

func testSimpleServerGroup(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	infrastructure := commons.Infrastructure{}
	g := osGenerator{}
	cfg := config.Configuration{}
	outputs := make(map[string]string, 0)
	resourceTypes := getOpenstackResourceTypes(cfg, infrastructureName)
	err := g.generateServerGroup(
		context.Background(),
		serverGroupOptions{
			kv:            kv,
			deploymentID:  deploymentID,
			nodeName:      "ServerGroupA",
			resourceTypes: resourceTypes,
		},
		&infrastructure, outputs, nil)
	require.NoError(t, err, "Unexpected error attempting to generate server group for %s", deploymentID)

	require.Len(t, infrastructure.Resource["openstack_compute_servergroup_v2"], 1, "Expected one server group")
	infraResource := infrastructure.Resource["openstack_compute_servergroup_v2"].(map[string]interface{})
	require.Len(t, infraResource, 1)
	sgName := "sg-SimpleApp-ServerGroupPolicy"
	serverGroup, ok := infraResource[sgName].(*ServerGroup)
	require.True(t, ok, "%s is not a ServerGroup", sgName)
	assert.Equal(t, sgName, serverGroup.Name)
	assert.Len(t, serverGroup.Policies, 1, "serverGroup.Policies has not length equal to 1")
	assert.Equal(t, "anti-affinity", serverGroup.Policies[0])

	require.Len(t, outputs, 1, "1 output is expected")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "ServerGroupA", "0", "attributes/id"), "expected id instance attribute output")
}
