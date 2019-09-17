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
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/log"
)

func testLocationsFromConfig(t *testing.T, srv1 *testutil.TestServer, cc *api.Client) {
	log.SetDebug(true)

	openStackLocation1 := config.Location{
		Type: "openstack",
		Configuration: config.DynamicMap{
			"auth_url":                "http://1.2.3.1:5000/v2.0",
			"default_security_groups": []string{"sec11", "sec12"},
			"password":                "test1",
			"private_network_name":    "private-net1",
			"region":                  "RegionOne",
			"tenant_name":             "test1",
			"user_name":               "test1",
		},
	}

	openStackLocation2 := config.Location{
		Type: "openstack",
		Configuration: config.DynamicMap{
			"auth_url":                "http://1.2.3.2:5000/v2.0",
			"default_security_groups": []string{"sec21", "sec22"},
			"password":                "test2",
			"private_network_name":    "private-net2",
			"region":                  "RegionOne",
			"tenant_name":             "test2",
			"user_name":               "test2",
		},
	}

	slurmLocation := config.Location{
		Type: "slurm",
		Configuration: config.DynamicMap{
			"user_name":   "slurmuser1",
			"private_key": "/path/to/key",
			"url":         "10.1.2.3",
			"port":        22,
		},
	}

	testConfig := config.Configuration{
		Locations: map[string]config.Location{
			"myLocation1": openStackLocation1,
			"myLocation2": openStackLocation2,
			"myLocation3": slurmLocation,
		},
	}

	err := Initialize(testConfig, cc)
	require.NoError(t, err, "Failed to initialize locations")

	location, err := GetLocation("myLocation2")
	require.NoError(t, err, "Unexpected error attempting to get location myLocation2")
	assert.Equal(t, "openstack", location.Type)
	assert.Equal(t, "test2", location.Configuration["user_name"])

	location, err = GetFirstLocationOfType("slurm")
	require.NoError(t, err, "Unexpected error attempting to get slurm location")
	assert.Equal(t, "slurm", location.Type, "Wrong location type in %+v", location)
	assert.Equal(t, "slurmuser1", location.Configuration["user_name"], "Wrong user name in %+v", location)

	location, err = GetFirstLocationOfType("UnknownType")
	require.Error(t, err, "Expected to have an error attempting to get location of unknown type, got %+v", location)

	locations, err := GetLocations()
	require.NoError(t, err, "Unexpected error attempting to get all locations")
	assert.Equal(t, 3, len(locations), "Unexpected number of locations returned by GetLocations():%+v", locations)
}
