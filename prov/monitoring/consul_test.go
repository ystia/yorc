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
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
	"testing"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulMonitoringPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)

	cfg := config.Configuration{
		HTTPAddress: "localhost",
		ServerID:    "0",
		Consul: config.Consul{
			Address:        srv.HTTPAddr,
			PubMaxRoutines: config.DefaultConsulPubMaxRoutines,
		},
	}

	// Register the consul service
	chStop := make(chan struct{})
	err := consulutil.RegisterServerAsConsulService(cfg, client, chStop)
	if err != nil {
		t.Fatalf("failed to setup Yorc Consul service %v", err)
	}
	// Start/Stop the monitoring manager
	Start(cfg, client)
	defer func() {
		Stop()
		srv.Stop()
	}()

	policy1 := tosca.Policy{
		Type:    "yorc.policies.monitoring.TCPMonitoring",
		Targets: []string{"Compute1"},
		Properties: map[string]*tosca.ValueAssignment{
			"port":          &tosca.ValueAssignment{Type: 0, Value: 22},
			"time_interval": &tosca.ValueAssignment{Type: 0, Value: "1s"},
		},
	}
	err = storage.GetStore(types.StoreTypeDeployment).Set(consulutil.DeploymentKVPrefix+"/monitoring1/topology/policies/TCPMonitoring", policy1)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(consulutil.DeploymentKVPrefix+"/monitoring5/topology/policies/TCPMonitoring", policy1)
	require.Nil(t, err)

	policy2 := tosca.Policy{
		Type:    "yorc.policies.monitoring.HTTPMonitoring",
		Targets: []string{"Compute2"},
		Properties: map[string]*tosca.ValueAssignment{
			"port":          &tosca.ValueAssignment{Type: 0, Value: 22},
			"time_interval": &tosca.ValueAssignment{Type: 0, Value: "1s"},
		},
	}
	err = storage.GetStore(types.StoreTypeDeployment).Set(consulutil.DeploymentKVPrefix+"/monitoring1/topology/policies/HTTPMonitoring", policy2)
	require.Nil(t, err)

	policy3 := tosca.Policy{
		Type:    "yorc.policies.monitoring.TCPMonitoring",
		Targets: []string{"Compute"},
		Properties: map[string]*tosca.ValueAssignment{
			"port":          &tosca.ValueAssignment{Type: 0, Value: 22},
			"time_interval": &tosca.ValueAssignment{Type: 0, Value: "1s"},
		},
	}
	err = storage.GetStore(types.StoreTypeDeployment).Set(consulutil.DeploymentKVPrefix+"/monitoring3/topology/policies/HTTPMonitoring", policy3)
	require.Nil(t, err)

	nodeCompute := tosca.NodeTemplate{
		Type: "yorc.nodes.openstack.Compute",
	}
	err = storage.GetStore(types.StoreTypeDeployment).Set(consulutil.DeploymentKVPrefix+"/monitoring1/topology/nodes/Compute1", nodeCompute)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(consulutil.DeploymentKVPrefix+"/monitoring1/topology/nodes/Compute2", nodeCompute)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(consulutil.DeploymentKVPrefix+"/monitoring2/topology/nodes/Compute1", nodeCompute)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(consulutil.DeploymentKVPrefix+"/monitoring3/topology/nodes/Compute1", nodeCompute)
	require.Nil(t, err)
	err = storage.GetStore(types.StoreTypeDeployment).Set(consulutil.DeploymentKVPrefix+"/monitoring5/topology/nodes/Compute1", nodeCompute)
	require.Nil(t, err)

	srv.PopulateKV(t, map[string][]byte{
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/instances/Compute1/0/attributes/ip_address": []byte("1.2.3.4"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/instances/Compute1/0/attributes/state":      []byte("started"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/instances/Compute2/0/attributes/ip_address": []byte("10.20.30.40"),
		consulutil.DeploymentKVPrefix + "/monitoring1/topology/instances/Compute2/0/attributes/state":      []byte("started"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/instances/Compute1/0/attributes/ip_address": []byte("1.2.3.4"),
		consulutil.DeploymentKVPrefix + "/monitoring5/topology/instances/Compute1/0/attributes/state":      []byte("started"),
	})

	t.Run("groupMonitoring", func(t *testing.T) {
		t.Run("testComputeMonitoringHook", func(t *testing.T) {
			testComputeMonitoringHook(t, client, cfg)
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
