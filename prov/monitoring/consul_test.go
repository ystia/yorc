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

package monitoring

import (
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/testutil"
	"testing"
	"time"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulMonitoringPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	defer srv.Stop()

	srv.PopulateKV(t, map[string][]byte{
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/types/tosca.nodes.Root/name":                      []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/types/tosca.nodes.Compute/derived_from":           []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/types/yorc.nodes.Compute/derived_from":            []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/types/yorc.nodes.openstack.Compute/derived_from":  []byte("yorc.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/nodes/Compute1/type":                              []byte("yorc.nodes.openstack.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/nodes/Compute1/metadata/monitoring_time_interval": []byte("1"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/instances/Compute1/0/attributes/ip_address":       []byte("1.2.3.4"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/instances/Compute1/0/attributes/state":            []byte("started"),

		consulutil.DeploymentKVPrefix + "/monitoring2/topology/types/tosca.nodes.Root/name":                     []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring2/topology/types/tosca.nodes.Compute/derived_from":          []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring2/topology/types/yorc.nodes.Compute/derived_from":           []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring2/topology/types/yorc.nodes.openstack.Compute/derived_from": []byte("yorc.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring2/topology/nodes/Compute1/type":                             []byte("yorc.nodes.openstack.Compute"),

		consulutil.DeploymentKVPrefix + "/monitoring3/topology/types/tosca.nodes.Root/name":                      []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring3/topology/types/tosca.nodes.Compute/derived_from":           []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring3/topology/types/yorc.nodes.Compute/derived_from":            []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring3/topology/types/yorc.nodes.openstack.Compute/derived_from":  []byte("yorc.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring3/topology/nodes/Compute1/type":                              []byte("yorc.nodes.openstack.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring3/topology/nodes/Compute1/metadata/monitoring_time_interval": []byte("0"),

		consulutil.DeploymentKVPrefix + "/monitoring4/topology/types/tosca.nodes.Root/name":                      []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring4/topology/types/tosca.nodes.Compute/derived_from":           []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring4/topology/types/yorc.nodes.Compute/derived_from":            []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring4/topology/types/yorc.nodes.openstack.Compute/derived_from":  []byte("yorc.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring4/topology/nodes/Compute1/type":                              []byte("yorc.nodes.openstack.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring4/topology/nodes/Compute1/metadata/monitoring_time_interval": []byte("30"),
		consulutil.DeploymentKVPrefix + "/monitoring4/topology/instances/Compute1/0/attributes/state":            []byte("started"),

		consulutil.DeploymentKVPrefix + "/monitoring5/topology/types/tosca.nodes.Root/name":                      []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/types/tosca.nodes.Compute/derived_from":           []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/types/yorc.nodes.Compute/derived_from":            []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/types/yorc.nodes.openstack.Compute/derived_from":  []byte("yorc.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/nodes/Compute1/type":                              []byte("yorc.nodes.openstack.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/nodes/Compute1/metadata/monitoring_time_interval": []byte("1"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/instances/Compute1/0/attributes/ip_address":       []byte("1.2.3.4"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/instances/Compute1/0/attributes/state":            []byte("started"),
	})

	cfg := config.Configuration{
		Consul: config.Consul{
			HealthCheckPollingInterval: 1 * time.Second,
		},
	}

	t.Run("groupMonitoring", func(t *testing.T) {
		t.Run("testHandleMonitoringWithCheckCreated", func(t *testing.T) {
			testHandleMonitoringWithCheckCreated(t, client, cfg)
		})
		t.Run("testHandleMonitoringWithoutMonitoringRequiredWithNoTimeInterval", func(t *testing.T) {
			testHandleMonitoringWithoutMonitoringRequiredWithNoTimeInterval(t, client, cfg)
		})
		t.Run("testHandleMonitoringWithoutMonitoringRequiredWithZeroTimeInterval", func(t *testing.T) {
			testHandleMonitoringWithoutMonitoringRequiredWithZeroTimeInterval(t, client, cfg)
		})
		t.Run("testHandleMonitoringWithNoIP", func(t *testing.T) {
			testHandleMonitoringWithNoIP(t, client, cfg)
		})
		t.Run("testAddAndRemoveHealthCheck", func(t *testing.T) {
			testAddAndRemoveHealthCheck(t, client, cfg)
		})
	})
}
