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

package hostspool

import (
	"reflect"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v3/helper/labelsutil"
)

func TestUpdateHostResourcesLabels(t *testing.T) {
	type args struct {
		origin    map[string]string
		diff      map[string]string
		operation func(a int64, b int64) int64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    map[string]string
	}{

		{"testSimpleAlloc", args{map[string]string{"host.num_cpus": "16", "host.disk_size": "150 GB", "host.mem_size": "20 GB"}, map[string]string{"host.num_cpus": "8", "host.disk_size": "50 GB", "host.mem_size": "20 GB"}, subtract}, false, map[string]string{"host.num_cpus": "8", "host.disk_size": "100 GB", "host.mem_size": "0 B"}},
		{"testSimpleAllocWithMiB", args{map[string]string{"host.num_cpus": "16", "host.disk_size": "150 GiB", "host.mem_size": "20 GiB"}, map[string]string{"host.num_cpus": "8", "host.disk_size": "50 GiB", "host.mem_size": "20 GiB"}, subtract}, false, map[string]string{"host.num_cpus": "8", "host.disk_size": "100 GiB", "host.mem_size": "0 B"}},
		{"testSimpleRelease", args{map[string]string{"host.num_cpus": "8", "host.disk_size": "100 GB", "host.mem_size": "0 GB"}, map[string]string{"host.num_cpus": "8", "host.disk_size": "50 GB", "host.mem_size": "20 GB"}, add}, false, map[string]string{"host.num_cpus": "16", "host.disk_size": "150 GB", "host.mem_size": "20 GB"}},
		{"testSimpleReleaseWithMiB", args{map[string]string{"host.num_cpus": "8", "host.disk_size": "100 GiB", "host.mem_size": "0 GiB"}, map[string]string{"host.num_cpus": "8", "host.disk_size": "50 GiB", "host.mem_size": "20 GiB"}, add}, false, map[string]string{"host.num_cpus": "16", "host.disk_size": "150 GiB", "host.mem_size": "20 GiB"}},
		{"testSimpleAllocWithoutHostResources", args{map[string]string{"host.num_cpus": "8", "host.disk_size": "100 GiB", "host.mem_size": "0 GiB"}, map[string]string{}, subtract}, false, map[string]string{}},
		{"testSimpleAllocWithoutHostResources", args{map[string]string{"host.num_cpus": "8", "host.disk_size": "100 GiB", "host.mem_size": "0 GiB"}, map[string]string{}, add}, false, map[string]string{}},
		{"testSimpleAllocWithoutLabels", args{map[string]string{}, map[string]string{"host.num_cpus": "8", "host.disk_size": "50 GiB", "host.mem_size": "20 GiB"}, subtract}, false, map[string]string{}},
		{"testSimpleReleaseWithoutLabels", args{map[string]string{}, map[string]string{"host.num_cpus": "8", "host.disk_size": "50 GiB", "host.mem_size": "20 GiB"}, add}, false, map[string]string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			var labels map[string]string
			labels, err = updateResourcesLabels(tt.args.origin, tt.args.diff, tt.args.operation)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetCapabilitiesOfType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(labels, tt.want) {
				t.Fatalf("GetCapabilitiesOfType() = %v, want %v", labels, tt.want)
			}
			require.Equal(t, tt.want, labels)
		})
	}
}

// testCreateFiltersFromComputeCapabilities expects a deployment stored in Consul
// so it is called in consul_test.go in this package
func testCreateFiltersFromComputeCapabilities(t *testing.T, kv *api.KV, deploymentID string) {

	// See corresponding deployment at testdata/topology_hp_compute.yaml
	// It defines one node Compute with :
	// - 1 CPU
	// - 20 GB of disk
	// - a linux OS
	filters, err := createFiltersFromComputeCapabilities(kv, deploymentID, "Compute")
	require.NoError(t, err, "Unexpected error creating filters for Compute node")

	// Creating Host labels declaring more resources than required,
	// so it is expected to match the filters defined in deployment
	labels := map[string]string{
		"host.num_cpus":  "2",
		"host.disk_size": "40 GB",
		"os.type":        "linux",
	}

	matches, matchErr := labelsutil.MatchesAll(labels, filters...)

	require.NoError(t, matchErr, "Unexpected error matching %+v", labels)
	assert.True(t, matches, "Filters not matching, unexpected error %+v", matchErr)

	// Declaring less disk resources than required
	labels["host.disk_size"] = "5 GB"

	matches, matchErr = labelsutil.MatchesAll(labels, filters...)
	require.NoError(t, matchErr, "Unexpected error matching %+v", labels)
	assert.False(t, matches, "Filters wrongly matching as host has not enough disk size")

	// Declaring less CPU resources than required
	labels["host.disk_size"] = "21 GB"
	labels["host.num_cpus"] = "0"

	matches, matchErr = labelsutil.MatchesAll(labels, filters...)
	require.NoError(t, matchErr, "Unexpected error matching %+v", labels)
	assert.False(t, matches, "Filters wrongly matching as host has not enough CPUs")

	//  Checking a wrong os type is detected
	labels["host.num_cpus"] = "2"
	labels["os.type"] = "Windows"

	matches, matchErr = labelsutil.MatchesAll(labels, filters...)
	require.NoError(t, matchErr, "Unexpected error matching %+v", labels)
	assert.False(t, matches, "Filters wrongly matching as host has not linux as os type ")
}
