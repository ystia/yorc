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

package locations

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/helper/sshutil"
	"github.com/ystia/yorc/v4/locations/adapter"
	"github.com/ystia/yorc/v4/prov/hostspool"
	"golang.org/x/crypto/ssh"

	"gopkg.in/yaml.v2"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/log"
)

// GetManagerWithSSHFactory creates a Location Manager with a given ssh factory
//
// Currently this is used for testing purpose to mock the ssh connection.
func GetManagerWithSSHFactory(cfg config.Configuration, sshClientFactory hostspool.SSHClientFactory) (Manager, Manager, error) {
	mgr, err := GetManager(cfg)
	if err != nil {
		return nil, nil, err
	}
	var locationMgr *locationManager
	if locationMgr == nil {
		client, err := cfg.GetConsulClient()
		if err != nil {
			return nil, nil, err
		}
		locationMgr = &locationManager{cc: client}
		locationMgr.hpAdapter = adapter.NewHostsPoolLocationAdapterWithSSHFactory(client, cfg, sshClientFactory)
	}

	return locationMgr, mgr, nil
}

var mockSSHClientFactory = func(config *ssh.ClientConfig, conn hostspool.Connection) sshutil.Client {
	return &sshutil.MockSSHClient{
		MockRunCommand: func(string) (string, error) {
			if config != nil && config.User == "fail" {
				return "", errors.Errorf("Failed to connect")
			}

			return "ok", nil
		},
	}
}

func testLocationsFromConfig(t *testing.T, srv1 *testutil.TestServer, cc *api.Client,
	deploymentID string) {

	log.SetDebug(true)
	ctx := context.Background()
	openStackLocation1 := LocationConfiguration{
		Name: "myLocation1",
		Type: "openstack",
		Properties: config.DynamicMap{
			"auth_url":                "http://1.2.3.1:5000/v2.0",
			"default_security_groups": []string{"sec11", "sec12"},
			"password":                "test1",
			"private_network_name":    "private-net1",
			"region":                  "RegionOne",
			"tenant_name":             "test1",
			"user_name":               "test1",
		},
	}

	openStackLocation2 := LocationConfiguration{
		Name: "myLocation2",
		Type: "openstack",
		Properties: config.DynamicMap{
			"auth_url":                "http://1.2.3.2:5000/v2.0",
			"default_security_groups": []string{"sec21", "sec22"},
			"password":                "test2",
			"private_network_name":    "private-net2",
			"region":                  "RegionOne",
			"tenant_name":             "test2",
			"user_name":               "test2",
		},
	}

	slurmLocation := LocationConfiguration{
		Name: "myLocation3",
		Type: "slurm",
		Properties: config.DynamicMap{
			"user_name":   "slurmuser1",
			"private_key": "/path/to/key",
			"url":         "10.1.2.3",
			"port":        22,
		},
	}

	hostsPoolLocation := LocationConfiguration{
		Name: "myHostsPoolLocation",
		Type: "hostspool",
		Properties: config.DynamicMap{
			"hosts": []interface{}{
				map[string]interface{}{
					"name": "host1",
					"connection": map[string]interface{}{
						"user":     "test",
						"port":     22,
						"password": "pass",
						"host":     "host1.example.com",
					},
					"labels": map[string]interface{}{
						"environment":        "dev",
						"testlabel":          "hello",
						"host.cpu_frequency": "3 GHz",
						"host.disk_size":     "50 GB",
						"host.mem_size":      "4GB",
						"host.num_cpus":      "4",
						"os.architecture":    "x86_64",
						"os.distribution":    "ubuntu",
						"os.type":            "linux",
						"os.version":         "17.1",
					},
				},
				map[string]interface{}{
					"name": "host2",
					"connection": map[string]interface{}{
						"user":     "test",
						"port":     22,
						"password": "pass",
						"host":     "another.example.com",
					},
				},
			},
		},
	}

	myHostsPoolLocationPropertiesForUpdate := config.DynamicMap{
		"hosts": []interface{}{
			map[string]interface{}{
				"name": "host1",
				"connection": map[string]interface{}{
					"user":     "test1",
					"port":     22,
					"password": "pass",
					"host":     "host1.example.com",
				},
				"labels": map[string]interface{}{
					"environment":        "dev",
					"testlabel":          "hello",
					"host.cpu_frequency": "3 GHz",
					"host.disk_size":     "50 GB",
					"host.mem_size":      "4GB",
					"host.num_cpus":      "4",
					"os.architecture":    "x86_64",
					"os.distribution":    "ubuntu",
					"os.type":            "linux",
					"os.version":         "17.1",
				},
			},
			map[string]interface{}{
				"name": "host2",
				"connection": map[string]interface{}{
					"user":     "test",
					"port":     22,
					"password": "pass",
					"host":     "another.example.com",
				},
			},
		},
	}
	myHostsPoolLocationPropertiesForUpdateHosts := config.DynamicMap{
		"hostssssss": []interface{}{},
	}

	hostsPoolLocationModified := LocationConfiguration{
		Name: "myHostsPoolLocation",
		Type: "hostspool",
		Properties: config.DynamicMap{
			"hosts": []interface{}{
				map[string]interface{}{
					"name": "host1",
					"connection": map[string]interface{}{
						"user":     "test",
						"port":     22,
						"password": "pass",
						"host":     "host1.example.com",
					},
					"labels": map[string]interface{}{
						"environment":        "dev",
						"testlabel":          "hello",
						"host.cpu_frequency": "3 GHz",
						"host.disk_size":     "50 GB",
						"host.mem_size":      "4GB",
						"host.num_cpus":      "4",
						"os.architecture":    "x86_64",
						"os.distribution":    "ubuntu",
						"os.type":            "linux",
						"os.version":         "17.1",
					},
				},
				map[string]interface{}{
					"name": "host3",
					"connection": map[string]interface{}{
						"user":     "test",
						"port":     22,
						"password": "pass",
						"host":     "another.test.com",
					},
					"labels": map[string]interface{}{
						"environment":        "dev",
						"testlabel":          "hello",
						"host.cpu_frequency": "3 GHz",
						"host.disk_size":     "50 GB",
						"host.mem_size":      "4GB",
						"host.num_cpus":      "4",
						"os.architecture":    "x86_64",
						"os.distribution":    "ubuntu",
						"os.type":            "linux",
						"os.version":         "17.1",
					},
				},
				map[string]interface{}{
					"name": "host2",
					"connection": map[string]interface{}{
						"user":     "test",
						"port":     22,
						"password": "pass",
						"host":     "another.example.com",
					},
					"labels": map[string]interface{}{
						"environment":        "dev",
						"testlabel":          "hello",
						"host.cpu_frequency": "3 GHz",
						"host.disk_size":     "50 GB",
						"host.mem_size":      "4GB",
						"host.num_cpus":      "4",
						"os.architecture":    "x86_64",
						"os.distribution":    "ubuntu",
						"os.type":            "linux",
						"os.version":         "17.1",
					},
				},
			},
		},
	}

	testLocations := LocationsDefinition{
		Locations: []LocationConfiguration{
			openStackLocation1,
			openStackLocation2,
		},
	}

	// Create a location file
	f, err := ioutil.TempFile("", "yorctestlocation*.yaml")
	require.NoError(t, err, "Error creating a temp file")
	locationFilePath := f.Name()
	defer os.Remove(locationFilePath)

	bSlice, err := yaml.Marshal(testLocations)
	require.NoError(t, err, "Error marshaling %+v", testLocations)
	_, err = f.Write(bSlice)
	require.NoError(t, err, "Error writing to %s", locationFilePath)
	f.Sync()
	f.Close()

	testConfig := config.Configuration{
		Consul: config.Consul{
			Address:        srv1.HTTPAddr,
			PubMaxRoutines: config.DefaultConsulPubMaxRoutines,
		},
	}

	mgr, mgr1, err := GetManagerWithSSHFactory(testConfig, mockSSHClientFactory)
	require.NoError(t, err, "Failed to create a location manager")

	// Check no locations exist and use mgr1
	nolocations, err := mgr1.GetLocations()
	assert.Equal(t, 0, len(nolocations))

	done, err := mgr.InitializeLocations(locationFilePath)
	require.NoError(t, err, "Failed to initialize locations")
	require.Equal(t, true, done, "Initialization of locations form file was not done")

	// Attempt to create a location with an already existing name
	err = mgr.CreateLocation(openStackLocation2)
	require.Error(t, err, "Expected to have an error attempting to create an already existing location")

	props, err := mgr.GetLocationProperties("myLocation1", "openstack")
	require.NoError(t, err, "Unexpected error attempting to get location myLocation1")
	assert.Equal(t, "test1", props["user_name"])

	props, err = mgr.GetLocationProperties("myLocation2", "openstack")
	require.NoError(t, err, "Unexpected error attempting to get location myLocation2")
	assert.Equal(t, "test2", props["user_name"])

	props, err = mgr.GetLocationProperties("myLocation3", "openstack")
	require.Error(t, err, "Expected to have an error attempting to get a non existing location, got %+v", props)

	err = mgr.CreateLocation(slurmLocation)
	require.NoError(t, err, "Unexpected error attempting to create location myLocation3")

	props, err = mgr.GetLocationProperties("myLocation3", "slurm")
	require.NoError(t, err, "Unexpected error attempting to get location myLocation3")
	assert.Equal(t, "slurmuser1", props["user_name"])

	slurmLocation.Properties["user_name"] = "slurmuser2"
	err = mgr.UpdateLocation(slurmLocation)
	require.NoError(t, err, "Unexpected error attempting to update location myLocation3")

	props, err = mgr.GetLocationProperties("myLocation3", "slurm")
	require.NoError(t, err, "Unexpected error attempting to get location myLocation3")
	assert.Equal(t, "slurmuser2", props["user_name"])

	slurmLocation.Properties["user_name"] = "slurmuser3"
	err = mgr.SetLocationConfiguration(slurmLocation)
	require.NoError(t, err, "Unexpected error attempting to set location configuration for myLocation3")

	props, err = mgr.GetLocationProperties("myLocation3", "slurm")
	require.NoError(t, err, "Unexpected error attempting to get location myLocation3")
	assert.Equal(t, "slurmuser3", props["user_name"])

	// provide unexisting location name in update
	slurmLocation.Name = "myLocationUpdated"
	slurmLocation.Properties["user_name"] = "slurmuser3"
	err = mgr.UpdateLocation(slurmLocation)
	require.Error(t, err, "Expected to have an error attempting to update a non existing location")

	// testdata/test_topology.yaml defines a location in Compute1 metadata
	props, err = mgr.GetLocationPropertiesForNode(ctx, deploymentID, "Compute1", "openstack")
	require.NoError(t, err, "Unexpected error attempting to get location for Compute1")
	assert.Equal(t, "test2", props["user_name"])

	// testdata/test_topology.yaml defines no location in Compute2 metadata
	props, err = mgr.GetLocationPropertiesForNode(ctx, deploymentID, "Compute2", "openstack")
	require.NoError(t, err, "Unexpected error attempting to get location for Compute2")
	// Check an openstack-specific confiugration value is provided in result
	assert.Equal(t, "RegionOne", props["region"])

	props, err = mgr.GetPropertiesForFirstLocationOfType("slurm")
	require.NoError(t, err, "Unexpected error attempting to get slurm location")
	assert.Equal(t, "slurmuser3", props["user_name"], "Wrong user name in %+v", props)

	props, err = mgr.GetPropertiesForFirstLocationOfType("UnknownType")
	require.Error(t, err, "Expected to have an error attempting to get location of unknown type, got %+v", props)

	// Calling again initialize should not remove known locations
	done, err = mgr.InitializeLocations(locationFilePath)
	require.NoError(t, err, "Failed to reinitialize locations")
	require.Equal(t, false, done, "Initialization should have already been done")

	locations, err := mgr.GetLocations()
	require.NoError(t, err, "Unexpected error attempting to get all locations")
	assert.Equal(t, 3, len(locations), "Unexpected number of locations returned by GetLocations():%+v", locations)

	err = mgr.RemoveLocation("myLocation2")
	require.NoError(t, err, "Unexpected error attempting to remove location myLocation2")

	props, err = mgr.GetLocationProperties("myLocation2", "slurm")
	require.Error(t, err, "Expected to have an error attempting to get a non existing location, got %+v", props)

	locations, err = mgr.GetLocations()
	require.NoError(t, err, "Unexpected error attempting to get all locations")
	assert.Equal(t, 2, len(locations), "Unexpected number of locations returned by GetLocations():%+v", locations)

	err = mgr.Cleanup()
	require.NoError(t, err, "Unexpected error attempting to cleanup locations")

	locations, err = mgr.GetLocations()
	require.NoError(t, err, "Unexpected error attempting to get all locations after cleanup")
	assert.Equal(t, 0, len(locations), "Unexpected number of locations returned by GetLocations():%+v", locations)

	err = mgr.CreateLocation(hostsPoolLocation)
	require.NoError(t, err, "Unexpected error attempting to create hosts pool location")
	props, err = mgr.GetLocationProperties("myHostsPoolLocation", "hostspool")
	require.NoError(t, err, "Unexpected error attempting to get hostspool location")
	//assert.Equal(t, hostsPoolLocation.Properties, props, "Wrong props in %+v", props)

	hostsPoolLocation.Properties = myHostsPoolLocationPropertiesForUpdate
	err = mgr.UpdateLocation(hostsPoolLocation)
	require.NoError(t, err, "Unexpected error attempting to update hostsPoolLocation")

	hostsPoolLocation.Properties = myHostsPoolLocationPropertiesForUpdateHosts
	err = mgr.UpdateLocation(hostsPoolLocation)
	require.Error(t, err, "Expected error attempting to update hostsPoolLocation with no hosts in defeinition")

	err = mgr.SetLocationConfiguration(hostsPoolLocationModified)
	require.NoError(t, err, "Unexpected error attempting to update hosts pool location")
	props, err = mgr.GetLocationProperties("myHostsPoolLocation", "hostspool")
	require.NoError(t, err, "Unexpected error attempting to get hostspool location")
	locations, err = mgr.GetLocations()
	require.NoError(t, err, "Unexpected error attempting to get all locations with hosts pool locations")
	assert.Equal(t, 1, len(locations), "Unexpected number of locations returned by GetLocations():%+v", locations)

	mgr.RemoveLocation(hostsPoolLocation.Name)
	require.NoError(t, err, "Unexpected error attempting to remove hosts pool location")
	locations, err = mgr.GetLocations()
	require.NoError(t, err, "Unexpected error attempting to get all locations after cleanup")
	assert.Equal(t, 0, len(locations), "Unexpected number of locations returned by GetLocations():%+v", locations)
}
