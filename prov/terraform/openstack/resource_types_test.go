// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ystia/yorc/v4/config"
)

func Test_resourceTypes(t *testing.T) {

	cfg := config.Configuration{
		Locations: map[string]config.LocationConfiguration{
			"oldOpenStack": config.LocationConfiguration{
				Type: infrastructureType,
				Properties: config.DynamicMap{
					"auth_url":                "http://1.2.3.4:5000/v2.0",
					"default_security_groups": []string{"sec1", "sec2"},
					"user_name":               "user1",
					"password":                "password1",
					"tenant_name":             "tenant1",
					"private_network_name":    "private-net",
					"region":                  "RegionOne",
				},
			},
			"newOpenStack": config.LocationConfiguration{
				Type: infrastructureType,
				Properties: config.DynamicMap{
					"auth_url":                "http://1.2.3.4:5000/v3",
					"default_security_groups": []string{"sec1", "sec2"},
					"user_name":               "user1",
					"password":                "password1",
					"user_domain_name":        "domain1",
					"project_name":            "project1",
					"private_network_name":    "private-net",
					"region":                  "RegionOne",
				},
			},
		},
	}

	tests := []struct {
		name             string
		location         string
		volumeTypeResult string
	}{
		{"testOldOpenStack", "oldOpenStack", "openstack_blockstorage_volume_v1"},
		{"testNewOpenStack", "newOpenStack", "openstack_blockstorage_volume_v3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			resourceTypes := getOpenstackResourceTypes(cfg.Locations[tt.location].Properties)
			assert.Equal(t, tt.volumeTypeResult, resourceTypes[blockStorageVolume],
				"Unexpected resource type on location %s", tt.location)
		})
	}
}
