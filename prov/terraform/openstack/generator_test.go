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
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
)

func Test_addOutput(t *testing.T) {
	type args struct {
		infrastructure *commons.Infrastructure
		outputName     string
		output         *commons.Output
	}
	tests := []struct {
		name       string
		args       args
		jsonResult string
	}{
		{"OneOutput", args{&commons.Infrastructure{}, "O1", &commons.Output{Value: "V1"}}, `{"output":{"O1":{"value":"V1"}}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commons.AddOutput(tt.args.infrastructure, tt.args.outputName, tt.args.output)
			res, err := json.Marshal(tt.args.infrastructure)
			require.Nil(t, err)
			require.Equal(t, tt.jsonResult, string(res))
		})
	}
}

func testGenerateTerraformInfo(t *testing.T, srv1 *testutil.TestServer, kv *api.KV,
	locationMgr locations.Manager) {
	t.Parallel()
	log.SetDebug(true)

	depID := path.Base(t.Name())
	yamlName := "testdata/topology_test.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), depID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")

	// Simulate the persistent disk "volume_id" attribute registration
	instancesPrefix := path.Join(consulutil.DeploymentKVPrefix,
		depID, "topology/instances")
	srv1.PopulateKV(t, map[string][]byte{
		path.Join(instancesPrefix, "BlockStorage/0/attributes/volume_id"): []byte("my_vol_id"),
		path.Join(instancesPrefix, "/FIPCompute/0/capabilities/endpoint",
			"attributes/floating_ip_address"): []byte("1.2.3.4"),
		path.Join(instancesPrefix, "/Network_2/0/attributes/network_id"): []byte("netID"),
	})

	locationProps := config.DynamicMap{
		"auth_url":                "http://1.2.3.4:5000/v2.0",
		"default_security_groups": []string{"default", "sec2"},
		"password":                "test",
		"private_network_name":    "private-net",
		"region":                  "RegionOne",
		"tenant_name":             "test",
		"user_name":               "test",
	}
	err = locationMgr.CreateLocation(
		locations.LocationConfiguration{
			Name:       t.Name(),
			Type:       infrastructureType,
			Properties: locationProps,
		})
	require.NoError(t, err, "Failed to create a location")
	defer func() {
		locationMgr.RemoveLocation(t.Name(), infrastructureType)
	}()

	var cfg config.Configuration
	g := osGenerator{}

	expectedComputeOutputs := map[string]string{
		path.Join(instancesPrefix, "BlockStorage/0/attributes/device"):                      "VolBlockStoragetoCompute-0-device",
		path.Join(instancesPrefix, "Compute/0/attributes/ip_address"):                       "Compute-0-IPAddress",
		path.Join(instancesPrefix, "Compute/0/attributes/private_address"):                  "Compute-0-privateIP",
		path.Join(instancesPrefix, "Compute/0/attributes/public_address"):                   "Compute-0-publicIP",
		path.Join(instancesPrefix, "Compute/0/attributes/public_ip_address"):                "Compute-0-publicIP",
		path.Join(instancesPrefix, "Compute/0/capabilities/endpoint/attributes/ip_address"): "Compute-0-IPAddress",
		path.Join(consulutil.DeploymentKVPrefix, depID, "topology/relationship_instances",
			"BlockStorage/2/0/attributes/device"): "VolBlockStoragetoCompute-0-device",
		path.Join(consulutil.DeploymentKVPrefix, depID, "topology/relationship_instances",
			"Compute/2/0/attributes/device"): "VolBlockStoragetoCompute-0-device",
	}

	tempdir, err := ioutil.TempDir("", depID)
	require.NoError(t, err, "Failed to to create temporary directory")
	defer os.RemoveAll(tempdir)

	var testData = []struct {
		nodeName        string
		expectedOutputs map[string]string
	}{
		{"Compute", expectedComputeOutputs},
		{"BlockStorage", map[string]string{}},
		{"FIPCompute", map[string]string{}},
		{"Network_2", map[string]string{}},
	}
	for _, tt := range testData {
		res, outputs, _, _, err := g.generateTerraformInfraForNode(
			context.Background(), kv, cfg, depID, tt.nodeName, tempdir)
		require.NoError(t, err, "Unexpected error generating %s terraform info", tt.nodeName)
		assert.Equal(t, true, res, "Unexpected result for node name %s", tt.nodeName)

		for k, v := range tt.expectedOutputs {
			assert.Equal(t, v, outputs[k], "Unexpected output")
		}
	}

	// Error case
	infra := commons.Infrastructure{}
	infraOpts := generateInfraOptions{
		cfg:            cfg,
		infrastructure: &infra,
		locationProps:  locationProps,
		instancesKey:   "instancesKey",
		deploymentID:   depID,
		nodeName:       "Compute",
		nodeType:       "yorc.nodes.openstack.ServerGroup",
		instanceName:   "0",
		instanceIndex:  0,
		resourceTypes:  getOpenstackResourceTypes(locationProps),
	}
	outputs := make(map[string]string)
	cmdEnv := make([]string, 0)
	err = g.generateInstanceInfra(context.Background(), infraOpts, outputs, &cmdEnv)
	require.Error(t, err, "Expected to get an error on wrong node type")

	// Case where the floating IP is available as a property
	nodePrefix := path.Join(consulutil.DeploymentKVPrefix, depID, "topology/nodes")

	srv1.PopulateKV(t, map[string][]byte{
		path.Join(nodePrefix, "FIPCompute/properties/ip"): []byte("1.2.3.4"),
	})
	_, outputs, _, _, err = g.generateTerraformInfraForNode(
		context.Background(), kv, cfg, depID, "FIPCompute", tempdir)
	require.NoError(t, err, "Unexpected error generating FIPCompute terraform info")

}
