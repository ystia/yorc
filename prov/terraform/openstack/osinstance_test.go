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
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/helper/sshutil"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
)

func loadTestYaml(t *testing.T, kv *api.KV) string {
	deploymentID := path.Base(t.Name())
	yamlName := "testdata/" + deploymentID + ".yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), kv, deploymentID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")
	return deploymentID
}

func testSimpleOSInstance(t *testing.T, kv *api.KV) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)

	cfg := config.Configuration{
		Infrastructures: map[string]config.DynamicMap{
			infrastructureName: config.DynamicMap{
				"region":               "RegionTwo",
				"private_network_name": "test",
			}}}
	g := osGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)
	outputs := make(map[string]string, 0)

	err := g.generateOSInstance(context.Background(), kv, cfg, deploymentID, "Compute", "0", &infrastructure, outputs, &env)
	require.Nil(t, err)

	require.Len(t, infrastructure.Resource["openstack_compute_instance_v2"], 1)
	instancesMap := infrastructure.Resource["openstack_compute_instance_v2"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, "Compute-0")

	compute, ok := instancesMap["Compute-0"].(*ComputeInstance)
	require.True(t, ok, "Compute-0 is not a ComputeInstance")
	require.Equal(t, "yorc", compute.KeyPair)
	require.Equal(t, "4bde6002-649d-4868-a5cb-fcd36d5ffa63", compute.ImageID)
	require.Equal(t, "nova", compute.AvailabilityZone)
	require.Equal(t, "2", compute.FlavorID)
	require.Equal(t, "RegionTwo", compute.Region)
	require.Len(t, compute.SecurityGroups, 2)
	require.Contains(t, compute.SecurityGroups, "default")
	require.Contains(t, compute.SecurityGroups, "openbar")

	require.Len(t, compute.Provisioners, 0)
	require.Contains(t, infrastructure.Resource, "null_resource")
	require.Len(t, infrastructure.Resource["null_resource"], 1)
	nullResources := infrastructure.Resource["null_resource"].(map[string]interface{})

	require.Contains(t, nullResources, "Compute-0-ConnectionCheck")
	nullRes, ok := nullResources["Compute-0-ConnectionCheck"].(*commons.Resource)
	require.True(t, ok)
	require.Len(t, nullRes.Provisioners, 1)
	mapProv := nullRes.Provisioners[0]
	require.Contains(t, mapProv, "remote-exec")
	rex, ok := mapProv["remote-exec"].(commons.RemoteExec)
	require.True(t, ok)
	require.Equal(t, "cloud-user", rex.Connection.User)
	yorcPem, err := sshutil.ToPrivateKeyContent("~/.ssh/yorc.pem")
	require.Nil(t, err)
	assert.Equal(t, "${var.private_key}", rex.Connection.PrivateKey)
	require.Len(t, env, 1)
	assert.Equal(t, "TF_VAR_private_key="+string(yorcPem), env[0], "env var for private key expected")
	require.Equal(t, `${openstack_compute_instance_v2.Compute-0.network.0.fixed_ip_v4}`, rex.Connection.Host)

	require.Len(t, outputs, 3, "3 outputs are expected")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "Compute", "0", "attributes/ip_address"), "expected ip_address instance attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "Compute", "0", "attributes/private_address"), "expected private_address instance attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "Compute", "0", "/capabilities/endpoint/attributes/ip_address"), "expected capability endpoint ip_address instance attribute output")
}

func testFipOSInstance(t *testing.T, kv *api.KV, srv *testutil.TestServer) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)

	srv.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/Network/0/capabilities/endpoint/attributes/floating_ip_address"): []byte("10.0.0.200"),
	})
	cfg := config.Configuration{
		Infrastructures: map[string]config.DynamicMap{
			infrastructureName: config.DynamicMap{
				"provisioning_over_fip_allowed": true,
				"private_network_name":          "test",
			}}}
	g := osGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)
	outputs := make(map[string]string, 0)
	err := g.generateOSInstance(context.Background(), kv, cfg, deploymentID, "Compute", "0", &infrastructure, outputs, &env)
	require.Nil(t, err)

	require.Len(t, infrastructure.Resource["openstack_compute_instance_v2"], 1)
	instancesMap := infrastructure.Resource["openstack_compute_instance_v2"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, "Compute-0")

	compute, ok := instancesMap["Compute-0"].(*ComputeInstance)
	require.True(t, ok, "Compute-0 is not a ComputeInstance")
	require.Equal(t, "yorc", compute.KeyPair)
	require.Equal(t, "4bde6002-649d-4868-a5cb-fcd36d5ffa63", compute.ImageID)
	require.Equal(t, "nova", compute.AvailabilityZone)
	require.Equal(t, "2", compute.FlavorID)
	require.Equal(t, "Region3", compute.Region)
	require.Len(t, compute.SecurityGroups, 2)
	require.Contains(t, compute.SecurityGroups, "default")
	require.Contains(t, compute.SecurityGroups, "openbar")

	require.Len(t, compute.Provisioners, 0)
	require.Contains(t, infrastructure.Resource, "null_resource")
	require.Len(t, infrastructure.Resource["null_resource"], 1)
	nullResources := infrastructure.Resource["null_resource"].(map[string]interface{})

	require.Contains(t, nullResources, "Compute-0-ConnectionCheck")
	require.Contains(t, nullResources, "Compute-0-ConnectionCheck")
	nullRes, ok := nullResources["Compute-0-ConnectionCheck"].(*commons.Resource)
	require.True(t, ok)
	require.Len(t, nullRes.Provisioners, 1)
	mapProv := nullRes.Provisioners[0]
	require.Contains(t, mapProv, "remote-exec")
	rex, ok := mapProv["remote-exec"].(commons.RemoteExec)
	require.True(t, ok)
	require.True(t, ok, "expecting remote-exec to be a RemoteExec")
	require.Equal(t, "cloud-user", rex.Connection.User)
	yorcPem, err := sshutil.ToPrivateKeyContent("~/.ssh/yorc.pem")
	require.Nil(t, err)
	assert.Equal(t, "${var.private_key}", rex.Connection.PrivateKey)
	require.Len(t, env, 1)
	assert.Equal(t, "TF_VAR_private_key="+string(yorcPem), env[0], "env var for private key expected")
	require.Equal(t, `${openstack_compute_floatingip_associate_v2.FIPCompute-0.floating_ip}`, rex.Connection.Host)

	require.Len(t, outputs, 5, "5 outputs are expected")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "Compute", "0", "attributes/public_address"), "expected public_address instance attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "Compute", "0", "attributes/public_ip_address"), "expected public_ip_address instance attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "Compute", "0", "attributes/ip_address"), "expected ip_address instance attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "Compute", "0", "attributes/private_address"), "expected private_address instance attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "Compute", "0", "/capabilities/endpoint/attributes/ip_address"), "expected capability endpoint ip_address instance attribute output")

}

func testFipOSInstanceNotAllowed(t *testing.T, kv *api.KV, srv *testutil.TestServer) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)

	srv.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/instances/Network/0/capabilities/endpoint/attributes/floating_ip_address"): []byte("10.0.0.200"),
	})
	cfg := config.Configuration{
		Infrastructures: map[string]config.DynamicMap{
			infrastructureName: config.DynamicMap{
				"provisioning_over_fip_allowed": false,
				"private_network_name":          "test",
			}}}
	g := osGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)

	err := g.generateOSInstance(context.Background(), kv, cfg, deploymentID, "Compute", "0", &infrastructure, make(map[string]string), &env)
	require.Nil(t, err)

	require.Len(t, infrastructure.Resource["openstack_compute_instance_v2"], 1)
	instancesMap := infrastructure.Resource["openstack_compute_instance_v2"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, "Compute-0")

	compute, ok := instancesMap["Compute-0"].(*ComputeInstance)
	require.True(t, ok, "Compute-0 is not a ComputeInstance")
	require.Equal(t, "yorc", compute.KeyPair)
	require.Equal(t, "4bde6002-649d-4868-a5cb-fcd36d5ffa63", compute.ImageID)
	require.Equal(t, "nova", compute.AvailabilityZone)
	require.Equal(t, "2", compute.FlavorID)
	require.Equal(t, "Region3", compute.Region)
	require.Len(t, compute.SecurityGroups, 2)
	require.Contains(t, compute.SecurityGroups, "default")
	require.Contains(t, compute.SecurityGroups, "openbar")

	require.Len(t, compute.Provisioners, 0)
	require.Contains(t, infrastructure.Resource, "null_resource")
	require.Len(t, infrastructure.Resource["null_resource"], 1)
	nullResources := infrastructure.Resource["null_resource"].(map[string]interface{})

	require.Contains(t, nullResources, "Compute-0-ConnectionCheck")
	require.Contains(t, nullResources, "Compute-0-ConnectionCheck")
	nullRes, ok := nullResources["Compute-0-ConnectionCheck"].(*commons.Resource)
	require.True(t, ok)
	require.Len(t, nullRes.Provisioners, 1)
	mapProv := nullRes.Provisioners[0]
	require.Contains(t, mapProv, "remote-exec")
	rex, ok := mapProv["remote-exec"].(commons.RemoteExec)
	require.True(t, ok)
	require.True(t, ok, "expecting remote-exec to be a RemoteExec")
	require.Equal(t, "cloud-user", rex.Connection.User)

	yorcPem, err := sshutil.ToPrivateKeyContent("~/.ssh/yorc.pem")
	require.Nil(t, err)
	assert.Equal(t, "${var.private_key}", rex.Connection.PrivateKey)
	require.Len(t, env, 1)
	assert.Equal(t, "TF_VAR_private_key="+string(yorcPem), env[0], "env var for private key expected")
	require.Equal(t, `${openstack_compute_instance_v2.Compute-0.network.0.fixed_ip_v4}`, rex.Connection.Host)
}

func testOSInstanceWithServerGroup(t *testing.T, kv *api.KV, srv *testutil.TestServer) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)

	cfg := config.Configuration{
		Infrastructures: map[string]config.DynamicMap{
			infrastructureName: config.DynamicMap{
				"provisioning_over_fip_allowed": false,
				"private_network_name":          "test",
			}}}
	g := osGenerator{}
	infrastructure := commons.Infrastructure{}
	env := make([]string, 0)
	outputs := make(map[string]string, 0)

	srv.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/nodes/ServerGroupPolicy_sg/type"):                []byte("yorc.nodes.openstack.ServerGroup"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/ServerGroupPolicy_sg/0/attributes/id"): []byte("my_sg_id"),
	})

	err := g.generateOSInstance(context.Background(), kv, cfg, deploymentID, "ComputeA", "0", &infrastructure, outputs, &env)
	require.Nil(t, err)

	require.Len(t, infrastructure.Resource["openstack_compute_instance_v2"], 1)
	instancesMap := infrastructure.Resource["openstack_compute_instance_v2"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, "ComputeA-0")

	compute, ok := instancesMap["ComputeA-0"].(*ComputeInstance)
	require.True(t, ok, "ComputeA-0 is not a ComputeInstance")
	require.Equal(t, "yorc", compute.KeyPair)
	require.Equal(t, "7d9bd308-d9c1-4952-a410-95b761672499", compute.ImageID)
	require.Equal(t, "4", compute.FlavorID)
	require.Equal(t, "my_sg_id", compute.SchedulerHints.Group)
}
