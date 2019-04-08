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
	"testing"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulMonitoringPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)

	cfg := config.Configuration{
		HTTPAddress: "localhost",
		ServerID:    "0",
	}

	// Register the consul service
	chStop := make(chan struct{})
	consulutil.RegisterServerAsConsulService(cfg, client, chStop)

	// Start/Stop the monitoring manager
	Start(cfg, client)
	defer func() {
		Stop()
		srv.Stop()
	}()

	srv.PopulateKV(t, map[string][]byte{
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/types/yorc.policies.Monitoring/properties/time_interval/default":  []byte("5s"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/types/yorc.policies.Monitoring/properties/time_interval/type":     []byte("string"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/types/yorc.policies.Monitoring/properties/time_interval/required": []byte("true"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/types/yorc.policies.Monitoring/properties/time_interval/name":     []byte("time_interval"),

		consulutil.DeploymentKVPrefix + "/monitoring1/topology/types/yorc.policies.monitoring.TCPMonitoring/derived_from": []byte("yorc.policies.Monitoring"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/types/yorc.policies.monitoring.TCPMonitoring/targets": []byte("		tosca.nodes.Compute,tosca.nodes.SoftwareComponent"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/policies/TCPMonitoring/properties/port":          []byte("22"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/policies/TCPMonitoring/properties/time_interval": []byte("1s"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/policies/TCPMonitoring/targets":                  []byte("Compute1"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/policies/TCPMonitoring/type":                     []byte("yorc.policies.monitoring.TCPMonitoring"),

		consulutil.DeploymentKVPrefix + "/monitoring3/topology/types/yorc.policies.monitoring.TCPMonitoring/derived_from": []byte("yorc.policies.Monitoring"),
		consulutil.DeploymentKVPrefix + "/monitoring3/topology/types/yorc.policies.monitoring.TCPMonitoring/targets": []byte("		tosca.nodes.Compute,tosca.nodes.SoftwareComponent"),
		consulutil.DeploymentKVPrefix + "/monitoring3/topology/policies/TCPMonitoring/properties/port":          []byte("22"),
		consulutil.DeploymentKVPrefix + "/monitoring3/topology/policies/TCPMonitoring/properties/time_interval": []byte("1s"),
		consulutil.DeploymentKVPrefix + "/monitoring3/topology/policies/TCPMonitoring/targets":                  []byte("Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring3/topology/policies/TCPMonitoring/type":                     []byte("yorc.policies.monitoring.TCPMonitoring"),

		consulutil.DeploymentKVPrefix + "/monitoring5/topology/types/yorc.policies.monitoring.TCPMonitoring/derived_from": []byte("yorc.policies.Monitoring"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/types/yorc.policies.monitoring.TCPMonitoring/targets": []byte("		tosca.nodes.Compute,tosca.nodes.SoftwareComponent"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/policies/TCPMonitoring/properties/port":          []byte("22"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/policies/TCPMonitoring/properties/time_interval": []byte("1s"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/policies/TCPMonitoring/targets":                  []byte("Compute1"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/policies/TCPMonitoring/type":                     []byte("yorc.policies.monitoring.TCPMonitoring"),

		consulutil.DeploymentKVPrefix + "/monitoring1/topology/types/tosca.nodes.Root/name":                     []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/types/tosca.nodes.Compute/derived_from":          []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/types/yorc.nodes.Compute/derived_from":           []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/types/yorc.nodes.openstack.Compute/derived_from": []byte("yorc.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/nodes/Compute1/type":                             []byte("yorc.nodes.openstack.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/instances/Compute1/0/attributes/ip_address":      []byte("1.2.3.4"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/instances/Compute1/0/attributes/state":           []byte("started"),

		consulutil.DeploymentKVPrefix + "/monitoring2/topology/types/tosca.nodes.Root/name":                     []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring2/topology/types/tosca.nodes.Compute/derived_from":          []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring2/topology/types/yorc.nodes.Compute/derived_from":           []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring2/topology/types/yorc.nodes.openstack.Compute/derived_from": []byte("yorc.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring2/topology/nodes/Compute1/type":                             []byte("yorc.nodes.openstack.Compute"),

		consulutil.DeploymentKVPrefix + "/monitoring3/topology/types/tosca.nodes.Root/name":                     []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring3/topology/types/tosca.nodes.Compute/derived_from":          []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring3/topology/types/yorc.nodes.Compute/derived_from":           []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring3/topology/types/yorc.nodes.openstack.Compute/derived_from": []byte("yorc.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring3/topology/nodes/Compute1/type":                             []byte("yorc.nodes.openstack.Compute"),

		consulutil.DeploymentKVPrefix + "/monitoring5/topology/types/tosca.nodes.Root/name":                     []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/types/tosca.nodes.Compute/derived_from":          []byte("tosca.nodes.Root"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/types/yorc.nodes.Compute/derived_from":           []byte("tosca.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/types/yorc.nodes.openstack.Compute/derived_from": []byte("yorc.nodes.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/nodes/Compute1/type":                             []byte("yorc.nodes.openstack.Compute"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/instances/Compute1/0/attributes/ip_address":      []byte("1.2.3.4"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/instances/Compute1/0/attributes/state":           []byte("started"),
	})

	t.Run("groupMonitoring", func(t *testing.T) {
		t.Run("testComputeMonitoringHook", func(t *testing.T) {
			testComputeMonitoringHook(t, client, config.Configuration{})
		})
		t.Run("testIsMonitoringRequiredWithNoPolicy", func(t *testing.T) {
			testIsMonitoringRequiredWithNoPolicy(t, client)
		})
		t.Run("testIsMonitoringRequiredWithNoPolicyForTarget", func(t *testing.T) {
			testIsMonitoringRequiredWithNoPolicyForTarget(t, client)
		})
		t.Run("testAddAndRemoveCheck", func(t *testing.T) {
			testAddAndRemoveCheck(t, client)
		})
	})
}
