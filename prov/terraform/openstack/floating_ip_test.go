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
	"path"
	"strings"
	"testing"

	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
)

func testGeneratePoolIP(t *testing.T, srv1 *testutil.TestServer) {
	t.Parallel()
	log.SetDebug(true)
	ctx := context.Background()
	depID := path.Base(t.Name())

	yamlName := "testdata/OSBaseImports.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), depID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")

	g := osGenerator{}
	nodeName := "NetworkFIP" + depID
	nodeNetwork := tosca.NodeTemplate{
		Type: "yorc.nodes.openstack.FloatingIP",
		Properties: map[string]*tosca.ValueAssignment{
			"floating_network_name": {
				Type:  0,
				Value: "Public_Network",
			},
		},
	}

	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, path.Join(consulutil.DeploymentKVPrefix, depID, "topology/nodes", nodeName), nodeNetwork)
	require.Nil(t, err)

	gia, err := g.generateFloatingIP(context.Background(), depID, nodeName, "0")
	assert.Nil(t, err)
	assert.Equal(t, "Public_Network", gia.Pool)
	assert.False(t, gia.IsIP)
}

func testGenerateSingleIP(t *testing.T, srv1 *testutil.TestServer) {
	t.Parallel()
	log.SetDebug(true)
	ctx := context.Background()
	depID := path.Base(t.Name())
	yamlName := "testdata/OSBaseImports.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), depID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")

	g := osGenerator{}
	nodeName := "NetworkFIP" + depID
	nodeNetwork := tosca.NodeTemplate{
		Type: "yorc.nodes.openstack.FloatingIP",
		Properties: map[string]*tosca.ValueAssignment{
			"ip": {
				Type:  0,
				Value: "10.0.0.2",
			},
		},
	}

	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, path.Join(consulutil.DeploymentKVPrefix, depID, "topology/nodes", nodeName), nodeNetwork)
	require.Nil(t, err)

	gia, err := g.generateFloatingIP(context.Background(), depID, nodeName, "0")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.2", gia.Pool)
	assert.True(t, gia.IsIP)
}

func testGenerateMultipleIP(t *testing.T, srv1 *testutil.TestServer) {
	t.Parallel()
	log.SetDebug(true)
	ctx := context.Background()
	depID := path.Base(t.Name())
	yamlName := "testdata/OSBaseImports.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), depID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")

	g := osGenerator{}
	nodeName := "NetworkFIP" + depID
	nodeNetwork := tosca.NodeTemplate{
		Type: "yorc.nodes.openstack.FloatingIP",
		Properties: map[string]*tosca.ValueAssignment{
			"ip": {
				Type:  0,
				Value: "10.0.0.2,10.0.0.4,10.0.0.5,10.0.0.6",
			},
		},
	}

	err = storage.GetStore(types.StoreTypeDeployment).Set(ctx, path.Join(consulutil.DeploymentKVPrefix, depID, "topology/nodes", nodeName), nodeNetwork)
	require.Nil(t, err)

	gia, err := g.generateFloatingIP(context.Background(), depID, nodeName, "0")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.2,10.0.0.4,10.0.0.5,10.0.0.6", gia.Pool)
	assert.True(t, gia.IsIP)
	ips := strings.Split(gia.Pool, ",")
	assert.Len(t, ips, 4)
}
