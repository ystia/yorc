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
	"io/ioutil"
	"path"
	"testing"

	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
)

var expectedKey []byte

func init() {
	var err error
	expectedKey, err = ioutil.ReadFile("./testdata/mykey.pem")
	if err != nil {
		panic(err)
	}
}

func loadTestYaml(t *testing.T) string {
	deploymentID := path.Base(t.Name())
	yamlName := "testdata/" + deploymentID + ".yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), deploymentID, yamlName)
	require.NoError(t, err, "Failed to parse "+yamlName+" definition")
	return deploymentID
}

func testSimpleComputeInstance(t *testing.T, cfg config.Configuration) {
	privateKey := expectedKey
	t.Parallel()
	deploymentID := loadTestYaml(t)
	resourcePrefix := getResourcesPrefix(cfg, deploymentID)
	infrastructure := commons.Infrastructure{}
	g := googleGenerator{}
	env := make([]string, 0)
	err := g.generateComputeInstance(context.Background(), cfg, deploymentID, "ComputeInstance", "0", 0, &infrastructure, make(map[string]string), &env)
	require.NoError(t, err, "Unexpected error attempting to generate compute instance for %s", deploymentID)

	instanceName := resourcePrefix + "computeinstance-0"
	require.Len(t, infrastructure.Resource["google_compute_instance"], 1, "Expected one compute instance")
	instancesMap := infrastructure.Resource["google_compute_instance"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, instanceName)

	compute, ok := instancesMap[instanceName].(*ComputeInstance)
	require.True(t, ok, "%s is not a ComputeInstance", instanceName)
	assert.Equal(t, "n1-standard-1", compute.MachineType)
	assert.Equal(t, "europe-west1-b", compute.Zone)
	require.NotNil(t, compute.BootDisk, 1, "Expected boot disk")
	assert.Equal(t, "centos-cloud/centos-7", compute.BootDisk.InitializeParams.Image, "Unexpected boot disk image")

	require.Len(t, compute.NetworkInterfaces, 1, "Expected one network interface for external access")
	assert.Equal(t, "", compute.NetworkInterfaces[0].AccessConfigs[0].NatIP, "Unexpected external IP address")

	require.Len(t, compute.ServiceAccounts, 1, "Expected one service account")
	assert.Equal(t, "yorc@yorc.net", compute.ServiceAccounts[0].Email, "Unexpected Service Account")

	assert.Equal(t, []string{"tag1", "tag2"}, compute.Tags)
	assert.Equal(t, map[string]string{"key1": "value1", "key2": "value2"}, compute.Labels)

	assert.Equal(t, map[string]string{"enable-oslogin": "false", "ssh-keys": "testuser:ecdsa-sha2-nistp256 AAAAE2VjZH/LlTXfXIr+N= test.user@name.org"}, compute.Metadata)

	require.Contains(t, infrastructure.Resource, "null_resource")
	require.Len(t, infrastructure.Resource["null_resource"], 1)
	nullResources := infrastructure.Resource["null_resource"].(map[string]interface{})

	connectionCheckName := instanceName + "-ConnectionCheck"
	require.Contains(t, nullResources, connectionCheckName)
	nullRes, ok := nullResources[connectionCheckName].(*commons.Resource)
	assert.True(t, ok)
	require.Len(t, nullRes.Provisioners, 1)
	mapProv := nullRes.Provisioners[0]
	require.Contains(t, mapProv, "remote-exec")
	rex, ok := mapProv["remote-exec"].(commons.RemoteExec)
	require.True(t, ok)
	assert.Equal(t, "centos", rex.Connection.User)
	assert.Equal(t, "${var.private_key}", rex.Connection.PrivateKey)
	require.Len(t, env, 1)
	assert.Equal(t, "TF_VAR_private_key="+string(privateKey), env[0], "env var for private key expected")

	require.Len(t, compute.ScratchDisks, 2, "Expected 2 scratch disks")
	assert.Equal(t, "SCSI", compute.ScratchDisks[0].Interface, "SCSI interface expected for 1st scratch")
	assert.Equal(t, "NVME", compute.ScratchDisks[1].Interface, "NVME interface expected for 2nd scratch")

	assert.Len(t, compute.NetworkInterfaces, 1, "1 NetworkInterface expected")
	assert.Equal(t, "default", compute.NetworkInterfaces[0].Network, "default network is not retrieved")
}

func testSimpleComputeInstanceMissingMandatoryParameter(t *testing.T, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	g := googleGenerator{}
	env := make([]string, 0)
	infrastructure := commons.Infrastructure{}

	err := g.generateComputeInstance(context.Background(), cfg, deploymentID, "ComputeInstance", "0", 0, &infrastructure, make(map[string]string), &env)
	require.Error(t, err, "Expected missing mandatory parameter error, but had no error")
	assert.Contains(t, err.Error(), "mandatory parameter zone", "Expected an error on missing parameter zone")
}

func testSimpleComputeInstanceWithAddress(t *testing.T, srv1 *testutil.TestServer, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	ctx := context.Background()
	nodeAddress := tosca.NodeTemplate{
		Type: "yorc.nodes.google.Address",
	}
	err := storage.GetStore(types.StoreTypeDeployment).Set(ctx, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/nodes/address_Compute"), nodeAddress)
	require.Nil(t, err)

	// Simulate the google address "ip_address" attribute registration
	srv1.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/address_Compute/0/attributes/ip_address"): []byte("1.2.3.4"),
	})

	infrastructure := commons.Infrastructure{}
	g := googleGenerator{}
	env := make([]string, 0)
	err = g.generateComputeInstance(context.Background(), cfg, deploymentID, "Compute", "0", 0, &infrastructure, make(map[string]string), &env)
	require.NoError(t, err, "Unexpected error attempting to generate compute instance for %s", deploymentID)
	resourcePrefix := getResourcesPrefix(cfg, deploymentID)
	instanceName := resourcePrefix + "compute-0"
	require.Len(t, infrastructure.Resource["google_compute_instance"], 1, "Expected one compute instance")
	instancesMap := infrastructure.Resource["google_compute_instance"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, instanceName)

	compute, ok := instancesMap[instanceName].(*ComputeInstance)
	require.True(t, ok, "%s is not a ComputeInstance", instanceName)
	assert.Equal(t, "n1-standard-1", compute.MachineType)
	assert.Equal(t, "europe-west1-b", compute.Zone)
	require.NotNil(t, compute.BootDisk, 1, "Expected boot disk")
	assert.Equal(t, "centos-cloud/centos-7", compute.BootDisk.InitializeParams.Image, "Unexpected boot disk image")

	require.Len(t, compute.NetworkInterfaces, 1, "Expected one network interface for external access")
	assert.Equal(t, "1.2.3.4", compute.NetworkInterfaces[0].AccessConfigs[0].NatIP, "Unexpected external IP address")
}

func testSimpleComputeInstanceWithPersistentDisk(t *testing.T, srv1 *testutil.TestServer, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)

	// Simulate the google persistent disk "volume_id" attribute registration
	srv1.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/nodes/BS1/type"):                       []byte("yorc.nodes.google.PersistentDisk"),
		path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/BS1/0/attributes/volume_id"): []byte("my_vol_id"),
	})

	infrastructure := commons.Infrastructure{}
	g := googleGenerator{}
	env := make([]string, 0)
	outputs := make(map[string]string, 0)
	err := g.generateComputeInstance(context.Background(), cfg, deploymentID, "Compute", "0", 0, &infrastructure, outputs, &env)
	require.NoError(t, err, "Unexpected error attempting to generate compute instance for %s", deploymentID)

	resourcePrefix := getResourcesPrefix(cfg, deploymentID)
	instanceName := resourcePrefix + "compute-0"
	require.Len(t, infrastructure.Resource["google_compute_instance"], 1, "Expected one compute instance")
	instancesMap := infrastructure.Resource["google_compute_instance"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, instanceName)

	compute, ok := instancesMap[instanceName].(*ComputeInstance)
	require.True(t, ok, "%s is not a ComputeInstance", instanceName)
	assert.Equal(t, "n1-standard-1", compute.MachineType)
	assert.Equal(t, "europe-west1-b", compute.Zone)
	require.NotNil(t, compute.BootDisk, 1, "Expected boot disk")
	assert.Equal(t, "centos-cloud/centos-7", compute.BootDisk.InitializeParams.Image, "Unexpected boot disk image")

	require.Len(t, infrastructure.Resource["google_compute_attached_disk"], 1, "Expected one attached disk")
	instancesMap = infrastructure.Resource["google_compute_attached_disk"].(map[string]interface{})
	require.Len(t, instancesMap, 1)

	attachmentName := fmt.Sprintf("%sbs1-0-to-compute-0", resourcePrefix)
	attachmentResourceName := fmt.Sprintf("google-%s", attachmentName)
	require.Contains(t, instancesMap, attachmentResourceName)
	attachedDisk, ok := instancesMap[attachmentResourceName].(*ComputeAttachedDisk)
	require.True(t, ok, "%s is not a ComputeAttachedDisk", attachmentResourceName)
	assert.Equal(t, "my_vol_id", attachedDisk.Disk)
	assert.Equal(t, fmt.Sprintf("${google_compute_instance.%s.name}", instanceName), attachedDisk.Instance)
	assert.Equal(t, "europe-west1-b", attachedDisk.Zone)
	assert.Equal(t, attachmentName, attachedDisk.DeviceName)
	assert.Equal(t, "READ_ONLY", attachedDisk.Mode)

	require.Contains(t, infrastructure.Resource, "null_resource")
	require.Len(t, infrastructure.Resource["null_resource"], 4)

	deviceAttribute := "file:" + attachmentResourceName

	require.Len(t, outputs, 8, "eight outputs are expected")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "BS1", "0", "attributes/device"), "expected instances attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/relationship_instances/", "Compute", "0", "0", "attributes/device"), "expected relationship attribute output for Compute")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/relationship_instances/", "BS1", "0", "0", "attributes/device"), "expected relationship attribute output for Block storage")

	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "Compute", "0", "attributes/public_address"), "expected public_address instance attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "Compute", "0", "attributes/public_ip_address"), "expected public_ip_address instance attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "Compute", "0", "attributes/ip_address"), "expected ip_address instance attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "Compute", "0", "attributes/private_address"), "expected private_address instance attribute output")
	require.Contains(t, outputs, path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "Compute", "0", "/capabilities/endpoint/attributes/ip_address"), "expected capability endpoint ip_address instance attribute output")

	require.Equal(t, deviceAttribute, outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/", "BS1", "0", "attributes/device")], "output file value expected")
	require.Equal(t, deviceAttribute, outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/relationship_instances/", "Compute", "0", "0", "attributes/device")], "output file value expected")
	require.Equal(t, deviceAttribute, outputs[path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/relationship_instances/", "BS1", "0", "0", "attributes/device")], "output file value expected")

}

func testSimpleComputeInstanceWithAutoCreationModeNetwork(t *testing.T, srv1 *testutil.TestServer, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	resourcePrefix := getResourcesPrefix(cfg, deploymentID)
	instanceName := resourcePrefix + "compute-0"

	// Simulate the google persistent disk "volume_id" attribute registration
	srv1.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/Network/0/attributes/network_name"): []byte("mynet"),
	})

	infrastructure := commons.Infrastructure{}
	g := googleGenerator{}
	env := make([]string, 0)
	outputs := make(map[string]string, 0)
	err := g.generateComputeInstance(context.Background(), cfg, deploymentID, "Compute", "0", 0, &infrastructure, outputs, &env)
	require.NoError(t, err, "Unexpected error attempting to generate compute instance for %s", deploymentID)

	require.Len(t, infrastructure.Resource["google_compute_instance"], 1, "Expected one compute instance")
	instancesMap := infrastructure.Resource["google_compute_instance"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, instanceName)

	compute, ok := instancesMap[instanceName].(*ComputeInstance)
	require.True(t, ok, "%s is not a ComputeInstance", instanceName)

	assert.Len(t, compute.NetworkInterfaces, 1, "1 NetworkInterface expected")
	assert.Equal(t, "mynet", compute.NetworkInterfaces[0].Network, "Network is not retrieved")
}

func testSimpleComputeInstanceWithSimpleNetwork(t *testing.T, srv1 *testutil.TestServer, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	resourcePrefix := getResourcesPrefix(cfg, deploymentID)
	instanceName := resourcePrefix + "comp1-0"

	// Simulate the google persistent disk "volume_id" attribute registration
	srv1.PopulateKV(t, map[string][]byte{
		path.Join(consulutil.DeploymentKVPrefix, deploymentID+"/topology/instances/Network_custom_subnet/0/attributes/subnetwork_name"): []byte("mysubnet"),
	})

	infrastructure := commons.Infrastructure{}
	g := googleGenerator{}
	env := make([]string, 0)
	outputs := make(map[string]string, 0)
	err := g.generateComputeInstance(context.Background(), cfg, deploymentID, "Comp1", "0", 0, &infrastructure, outputs, &env)
	require.NoError(t, err, "Unexpected error attempting to generate compute instance for %s", deploymentID)

	require.Len(t, infrastructure.Resource["google_compute_instance"], 1, "Expected one compute instance")
	instancesMap := infrastructure.Resource["google_compute_instance"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, instanceName)

	compute, ok := instancesMap[instanceName].(*ComputeInstance)
	require.True(t, ok, "%s is not a ComputeInstance", instanceName)

	assert.Len(t, compute.NetworkInterfaces, 1, "1 NetworkInterface expected")
	assert.Equal(t, "mysubnet", compute.NetworkInterfaces[0].Subnetwork, "Subnetwork is not retrieved")
}
