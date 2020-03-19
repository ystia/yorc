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
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/labelsutil"
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
func testCreateFiltersFromComputeCapabilities(t *testing.T, deploymentID string) {

	// See corresponding deployment at testdata/topology_hp_compute.yaml
	// It defines one node Compute with :
	// - 1 CPU
	// - 20 GB of disk
	// - a linux OS
	filters, err := createFiltersFromComputeCapabilities(context.Background(), deploymentID, "Compute")
	require.NoError(t, err, "Unexpected error creating filters for Compute node")

	// Creating Host labels declaring more resources than required,
	// so it is expected to match the filters defined in deployment
	labels := map[string]string{
		"host.num_cpus":  "2",
		"host.disk_size": "40 GB",
		"host.mem_size":  "8 GB",
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

// testConcurrentExecDelegateShareableHost tests concurrent attempts to allocate
// shareable hosts in parallel, and verifies it won't lead to over-allocate a
// shareable host
func testConcurrentExecDelegateShareableHost(t *testing.T, srv *testutil.TestServer, cc *api.Client, cfg config.Configuration, deploymentID, location string) {

	ctx, cfg, hpManager, nodeNames, initialLabels, testExecutor := prepareTestEnv(t, srv, cc, cfg, location, deploymentID, 2)

	// Testing the delegate operation install which will allocate resources
	// Expected Hosts Pool allocations after this operation:
	expectedResources := map[string]map[string]string{
		"host0": map[string]string{
			"host.num_cpus":     "0",
			"host.mem_size":     "1 GB",
			"host.disk_size":    "10 GB",
			"host.resource.gpu": "gpu2",
			"host.resource.cpu": "cpu1,cpu2",
		},
		"host1": map[string]string{
			"host.num_cpus":     "2",
			"host.mem_size":     "3 GB",
			"host.disk_size":    "50 GB",
			"host.resource.gpu": "gpu2",
			"host.resource.cpu": "cpu1,cpu2",
		},
	}
	testExecDelegateForNodes(ctx, t, testExecutor, cc, hpManager, cfg, location, "taskTest", deploymentID,
		"install", nodeNames, expectedResources)

	// Testing the delegate operation uninstall which will free resources
	// Expected Hosts Pool resources after this operation:
	expectedResources = map[string]map[string]string{
		"host0": initialLabels,
		"host1": initialLabels,
	}
	testExecDelegateForNodes(ctx, t, testExecutor, cc, hpManager, cfg, location, "taskTest", deploymentID,
		"uninstall", nodeNames, expectedResources)
}

func testExecDelegateForNodes(ctx context.Context, t *testing.T, testExecutor *defaultExecutor,
	cc *api.Client,
	hpManager Manager,
	cfg config.Configuration,
	location,
	taskID, deploymentID, delegateOperation string,
	nodeNames []string,
	expectedResources map[string]map[string]string) {

	// Calling the delegate operation on all nodes in parallel
	errors := make(chan error, len(nodeNames))
	var waitGroup sync.WaitGroup
	for _, nodeName := range nodeNames {
		waitGroup.Add(1)
		go routineExecDelegate(ctx, testExecutor, cc, hpManager, cfg, location, "taskTest", deploymentID,
			nodeName, delegateOperation, &waitGroup, errors)
	}
	waitGroup.Wait()

	// Check no error occurred attempting to execute the delegate operation
	select {
	case err := <-errors:
		require.NoError(t, err, "Unexpected error executing operation %s in parallel", delegateOperation)
	default:
	}

	close(errors)

	for k, v := range expectedResources {
		host, err := hpManager.GetHost(location, k)
		require.NoError(t, err, "Could not get host %s", k)

		for label, val := range v {
			// Rounding sizes for comparison
			result := strings.Replace(host.Labels[label], ".0 GB", " GB", 1)
			assert.Equal(t, val, result, "Unexpected value for %s after delegate operation %s on host %s",
				label, delegateOperation, k)
		}
	}

	// Check compute ip_address attributes
	expected := map[string]string{
		"public_address":    "1.2.3.4", // to cover some code using this resource
		"public_ip_address": "1.2.3.4",
		"private_address":   "5.6.7.8",
	}

	for attribute, expectedValue := range expected {
		for _, nodeName := range nodeNames {
			val, err := deployments.GetInstanceAttributeValue(ctx, deploymentID, nodeName, "0", attribute)
			require.NoError(t, err, "Could not get instance attribute value for deploymentID:%s, node name:%s, attribute:%s", deploymentID, nodeName, attribute)
			require.NotNil(t, val, "Unexpected nil value for deploymentID:%s, node name:%s, attribute:%s", deploymentID, nodeName, attribute)
			require.Equal(t, expectedValue, val.RawString(), "unexpected value %s for attribute %q instead of %s", val, expectedValue)
		}
	}

	// Check compute cpu and gpu attributes for compute1 and compute2
	expected = map[string]string{
		"cpu": "cpu0",
		"gpu": "gpu0,gpu1",
	}

	for attribute, expectedValue := range expected {
		for _, nodeName := range []string{"Compute", "Compute2"} {
			val, err := deployments.GetInstanceAttributeValue(ctx, deploymentID, nodeName, "0", attribute)
			require.NoError(t, err, "Could not get instance attribute value for deploymentID:%s, node name:%s, attribute:%s", deploymentID, nodeName, attribute)
			require.NotNil(t, val, "Unexpected nil value for deploymentID:%s, node name:%s, attribute:%s", deploymentID, nodeName, attribute)
			require.Equal(t, expectedValue, val.RawString(), "unexpected value %s for attribute %q instead of %s", val, expectedValue)
		}
	}
}

func routineExecDelegate(ctx context.Context, e *defaultExecutor, cc *api.Client,
	hpManager Manager,
	cfg config.Configuration,
	location, taskID, deploymentID, nodeName, delegateOperation string,
	waitGroup *sync.WaitGroup,
	errors chan<- error) {

	defer waitGroup.Done()
	fmt.Printf("Executing operation %s on node %s\n", delegateOperation, nodeName)
	operationParams := operationParameters{
		location:          location,
		taskID:            taskID,
		deploymentID:      deploymentID,
		nodeName:          nodeName,
		delegateOperation: delegateOperation,
		hpManager:         hpManager,
	}

	err := e.execDelegateHostsPool(ctx, cc, cfg, operationParams)
	if err != nil {
		fmt.Printf("Error executing operation %s on node %s: %s\n", delegateOperation, nodeName, err.Error())
		errors <- err
	} else {
		fmt.Printf("Executed operation %s on node %s\n", delegateOperation, nodeName)
	}
}

// testFailureExecDelegateShareableHost tests the failure to allocate a host
// due to missing resources
func testFailureExecDelegateShareableHost(t *testing.T, srv *testutil.TestServer, cc *api.Client, cfg config.Configuration, deploymentID, location string) {

	ctx, cfg, hpManager, nodeNames, _, testExecutor := prepareTestEnv(t, srv, cc, cfg, location, deploymentID, 1)

	lastIndex := 1
	operationParams := operationParameters{
		location:          location,
		taskID:            "taskTest",
		deploymentID:      deploymentID,
		delegateOperation: "install",
		hpManager:         hpManager,
	}
	for _, nodeName := range nodeNames[:lastIndex] {
		operationParams.nodeName = nodeName
		err := testExecutor.execDelegateHostsPool(ctx, cc, cfg, operationParams)
		require.NoError(t, err, "Error executing operation %s on node %s\n", operationParams.delegateOperation, nodeName)
	}

	operationParams.nodeName = nodeNames[lastIndex]
	err := testExecutor.execDelegateHostsPool(ctx, cc, cfg, operationParams)
	require.Error(t, err, "Expected a not enough resources error executing operation %s on node %s\n",
		operationParams.delegateOperation, operationParams.nodeName)

}

func prepareTestEnv(t *testing.T, srv *testutil.TestServer, cc *api.Client, cfg config.Configuration, location, deploymentID string, hostsNumber int) (context.Context, config.Configuration, Manager, []string, map[string]string, *defaultExecutor) {

	// The topology in testdata/topology_hp_compute.yaml defines 4 compute node
	// instances, each asking for 1CPU, 1GB of RAM, and 20GB of disk on a
	// shareable host.
	// Building a Hosts Pool without enough resources for the last compute node
	cleanupHostsPool(t, cc)
	ctx := context.Background()
	hpManager := NewManagerWithSSHFactory(cc, cfg, mockSSHClientFactory)
	initialLabels := map[string]string{
		"host.num_cpus":           "3",
		"host.mem_size":           "4 GB",
		"host.disk_size":          "70 GB",
		"os.type":                 "linux",
		"label1":                  "stringvalue1",
		"public_address":          "1.2.3.4", // to cover some code using this resource
		"public_ip_address":       "1.2.3.4",
		"private_address":         "5.6.7.8",
		"networks.0.network_name": "mynetwork",
		"networks.0.network_id":   "123",
		"networks.0.addresses":    "1.2.3.4,1.2.3.5",
		"host.resource.gpu":       "gpu0,gpu1,gpu2",
		"host.resource.cpu":       "cpu0,cpu1,cpu2",
	}

	var hostpool = createHostsWithLabels(hostsNumber, initialLabels)

	// Apply this definition
	var checkpoint uint64
	err := hpManager.Apply(location, hostpool, &checkpoint)
	require.NoError(t, err, "Unexpected failure applying host pool configuration")

	// Getting now nodes for which to call the install delegate operation install
	// which will allocate resources in the Hosts Pool

	allNodes, err := deployments.GetNodes(ctx, deploymentID)
	require.NoError(t, err, "Unexpected error getting nodes for deployment %s",
		deploymentID)

	var nodeNames []string
	for _, nodeName := range allNodes {
		nodeType, err := deployments.GetNodeType(ctx, deploymentID, nodeName)
		require.NoError(t, err, "Unexpected error getting type for node %s in deployment %s",
			nodeName, deploymentID)
		if nodeType == "yorc.nodes.hostspool.Compute" {
			nodeNames = append(nodeNames, nodeName)
		}
	}

	testExecutor := &defaultExecutor{}
	cfg.Consul.Address = srv.HTTPAddr
	cfg.Consul.PubMaxRoutines = config.DefaultConsulPubMaxRoutines

	return ctx, cfg, hpManager, nodeNames, initialLabels, testExecutor
}

// testExecFDelegateFailure tests a failure to execute an install operation
// due to a host connection issue (as not using the mockSSHClientFactory)
func testExecDelegateFailure(t *testing.T, srv *testutil.TestServer, cc *api.Client, cfg config.Configuration, deploymentID, location string) {

	ctx, cfg, _, nodeNames, _, testExecutor := prepareTestEnv(t, srv, cc, cfg, location, deploymentID, 1)

	err := testExecutor.ExecDelegate(ctx, cfg, "taskTest", deploymentID, nodeNames[0], "install")
	require.Error(t, err, "Should have failed to allocate a host, on connection error")
}
