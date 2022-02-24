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
	"fmt"
	"path"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
)

func testSimplePrivateNetwork(t *testing.T, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	resourcePrefix := getResourcesPrefix(cfg, deploymentID)
	networkName := resourcePrefix + "network"
	infrastructure := commons.Infrastructure{}
	g := googleGenerator{}
	err := g.generatePrivateNetwork(context.Background(), cfg, deploymentID, "Network", &infrastructure, make(map[string]string))
	require.NoError(t, err, "Unexpected error attempting to generate private network for %s", deploymentID)

	require.Len(t, infrastructure.Resource["google_compute_network"], 1, "Expected one private network")
	instancesMap := infrastructure.Resource["google_compute_network"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, networkName)

	privateNetwork, ok := instancesMap[networkName].(*PrivateNetwork)
	require.True(t, ok, "%s is not a PrivateNetwork", networkName)
	assert.Equal(t, networkName, privateNetwork.Name)
	assert.Equal(t, "myproj", privateNetwork.Project)
	assert.Equal(t, "mydesc", privateNetwork.Description)
	assert.Equal(t, false, privateNetwork.AutoCreateSubNetworks)
	assert.Equal(t, "REGIONAL", privateNetwork.RoutingMode)

	fwName := fmt.Sprintf("%s-default-external-fw", networkName)
	firewallsMap := infrastructure.Resource["google_compute_firewall"].(map[string]interface{})
	require.Len(t, firewallsMap, 1)
	require.Contains(t, firewallsMap, fwName)

	defaultFirewall, ok := firewallsMap[fwName].(*Firewall)
	require.True(t, ok, "defaultFirewall is not a Firewall")
	assert.Equal(t, fwName, defaultFirewall.Name)
	assert.Equal(t, fmt.Sprintf("${google_compute_network.%s.name}", networkName), defaultFirewall.Network)
	assert.Equal(t, []string{"0.0.0.0/0"}, defaultFirewall.SourceRanges)
	assert.Equal(t, []AllowRule{
		{Protocol: "icmp"},
		{Protocol: "TCP", Ports: []string{"3389"}},
		{Protocol: "TCP", Ports: []string{"22"}},
	}, defaultFirewall.Allow)
}

func testSimpleSubnet(t *testing.T, srv1 *testutil.TestServer, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	subnetName := "network-custom-subnet"
	srv1.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/Network/0/attributes/network_name"): []byte("network"),
	})

	infrastructure := commons.Infrastructure{}
	g := googleGenerator{}
	err := g.generateSubNetwork(context.Background(), cfg, deploymentID, "Network_custom_subnet", &infrastructure, make(map[string]string))
	require.NoError(t, err, "Unexpected error attempting to generate sub-network for %s", deploymentID)

	require.Len(t, infrastructure.Resource["google_compute_subnetwork"], 1, "Expected one sub-network")
	instancesMap := infrastructure.Resource["google_compute_subnetwork"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, subnetName)

	subnet, ok := instancesMap[subnetName].(*SubNetwork)
	require.True(t, ok, "%s is not a SubNetwork", subnetName)
	assert.Equal(t, subnetName, subnet.Name)
	assert.Equal(t, "network", subnet.Network)
	assert.Equal(t, "myproj", subnet.Project)
	assert.Equal(t, "mydesc", subnet.Description)
	assert.Equal(t, "europe-west1", subnet.Region)
	assert.Equal(t, true, subnet.EnableFlowLogs)
	assert.Equal(t, false, subnet.PrivateIPGoogleAccess)

	fwName := fmt.Sprintf("%s-default-internal-fw", subnetName)
	firewallsMap := infrastructure.Resource["google_compute_firewall"].(map[string]interface{})
	require.Len(t, firewallsMap, 1)
	require.Contains(t, firewallsMap, fwName)

	defaultFirewall, ok := firewallsMap[fwName].(*Firewall)
	require.True(t, ok, "%s is not a Firewall", fwName)
	assert.Equal(t, fwName, defaultFirewall.Name)
	assert.Equal(t, subnet.Network, defaultFirewall.Network)
	assert.Equal(t, []string{"10.1.0.0/24", "10.2.0.0/24", "10.10.0.0/24"}, defaultFirewall.SourceRanges)
	assert.Equal(t, []AllowRule{
		{Protocol: "icmp"},
		{Protocol: "TCP", Ports: []string{"0-65535"}},
		{Protocol: "UDP", Ports: []string{"0-65535"}},
	}, defaultFirewall.Allow)

	assert.Equal(t, []IPRange{
		{Name: "secondaryip", IPCIDRRange: "10.1.0.0/24"},
		{Name: "secondaryip2", IPCIDRRange: "10.2.0.0/24"},
	}, subnet.SecondaryIPRanges)
}
