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
	"io/ioutil"
	"os"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/log"
)

func testLocationsFromConfig(t *testing.T, srv1 *testutil.TestServer, cc *api.Client,
	deploymentID string) {

	log.SetDebug(true)

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

	mgr, err := NewManager(testConfig)
	require.NoError(t, err, "Failed to create a location manager")

	done, err := mgr.InitializeLocations(locationFilePath)
	require.NoError(t, err, "Failed to initialize locations")
	require.Equal(t, true, done, "Initialization of locations form file was not done")

	// Attempt to create a location with an already existing name
	err = mgr.CreateLocation(openStackLocation2)
	require.Error(t, err, "Expected to have an error attempting to create an already existing location")

	props, err := mgr.GetLocationProperties("myLocation1")
	require.NoError(t, err, "Unexpected error attempting to get location myLocation1")
	assert.Equal(t, "test1", props["user_name"])

	props, err = mgr.GetLocationProperties("myLocation2")
	require.NoError(t, err, "Unexpected error attempting to get location myLocation2")
	assert.Equal(t, "test2", props["user_name"])

	props, err = mgr.GetLocationProperties("myLocation3")
	require.Error(t, err, "Expected to have an error attempting to get a non existing location, got %+v", props)

	err = mgr.CreateLocation(slurmLocation)
	require.NoError(t, err, "Unexpected error attempting to create location myLocation3")

	props, err = mgr.GetLocationProperties("myLocation3")
	require.NoError(t, err, "Unexpected error attempting to get location myLocation3")
	assert.Equal(t, "slurmuser1", props["user_name"])

	slurmLocation.Properties["user_name"] = "slurmuser2"
	err = mgr.SetLocationConfiguration(slurmLocation)
	require.NoError(t, err, "Unexpected error attempting to update location myLocation3")

	props, err = mgr.GetLocationProperties("myLocation3")
	require.NoError(t, err, "Unexpected error attempting to get location myLocation3")
	assert.Equal(t, "slurmuser2", props["user_name"])

	// testdata/test_topology.yaml defines a location in Compute1 metadata
	props, err = mgr.GetLocationPropertiesForNode(deploymentID, "Compute1", "openstack")
	require.NoError(t, err, "Unexpected error attempting to get location for Compute1")
	assert.Equal(t, "test2", props["user_name"])

	// testdata/test_topology.yaml defines no location in Compute2 metadata
	props, err = mgr.GetLocationPropertiesForNode(deploymentID, "Compute2", "openstack")
	require.NoError(t, err, "Unexpected error attempting to get location for Compute2")
	// Check an openstack-specific confiugration value is provided in result
	assert.Equal(t, "RegionOne", props["region"])

	props, err = mgr.GetPropertiesForFirstLocationOfType("slurm")
	require.NoError(t, err, "Unexpected error attempting to get slurm location")
	assert.Equal(t, "slurmuser2", props["user_name"], "Wrong user name in %+v", props)

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

	props, err = mgr.GetLocationProperties("myLocation2")
	require.Error(t, err, "Expected to have an error attempting to get a non existing location, got %+v", props)

	locations, err = mgr.GetLocations()
	require.NoError(t, err, "Unexpected error attempting to get all locations")
	assert.Equal(t, 2, len(locations), "Unexpected number of locations returned by GetLocations():%+v", locations)

	err = mgr.Cleanup()
	require.NoError(t, err, "Unexpected error attempting to cleanup locations")

	locations, err = mgr.GetLocations()
	require.NoError(t, err, "Unexpected error attempting to get all locations after cleanup")
	assert.Equal(t, 0, len(locations), "Unexpected number of locations returned by GetLocations():%+v", locations)

}
