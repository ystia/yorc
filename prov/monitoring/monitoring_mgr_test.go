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
	"context"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tasks/workflow/builder"
	"github.com/ystia/yorc/v4/tosca"
)

type mockActivity struct {
	t builder.ActivityType
	v string
}

func (m *mockActivity) Type() builder.ActivityType {
	return m.t
}

func (m *mockActivity) Value() string {
	return m.v
}

func testComputeMonitoringHook(t *testing.T, client *api.Client, cfg config.Configuration) {
	log.SetDebug(true)

	ctx := context.Background()

	type args struct {
		deploymentID string
		nodeName     string
		instanceName string
	}
	tests := []struct {
		name string
		args args
	}{
		{"TCPMonitoring", args{"monitoring1", "Compute1", "0"}},
		{"HTTPMonitoring", args{"monitoring1", "Compute2", "0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			expectedCheck := NewCheck(tt.args.deploymentID, tt.args.nodeName, tt.args.instanceName)

			activity := &mockActivity{t: builder.ActivityTypeDelegate, v: "install"}

			addMonitoringHook(ctx, cfg, "", tt.args.deploymentID, tt.args.nodeName, activity)
			time.Sleep(2 * time.Second)

			checkReports, err := defaultMonManager.listCheckReports(func(cr CheckReport) bool {
				if cr.DeploymentID == tt.args.deploymentID {
					return true
				}
				return false
			})
			require.Nil(t, err, "Unexpected error while getting check reports list")
			require.Len(t, checkReports, 1, "1 check is expected")
			require.Equal(t, expectedCheck.Report.DeploymentID, checkReports[0].DeploymentID, "unexpected deploymentID")
			require.Equal(t, expectedCheck.Report.NodeName, checkReports[0].NodeName, "unexpected node name")
			require.Equal(t, expectedCheck.Report.Instance, checkReports[0].Instance, "unexpected instance")
			require.Equal(t, CheckStatusCRITICAL, checkReports[0].Status, "unexpected status")

			// Check the instance state has been updated
			state, err := deployments.GetInstanceState(context.Background(), tt.args.deploymentID, tt.args.nodeName, "0")
			require.Nil(t, err, "Unexpected error while node state")
			require.Equal(t, tosca.NodeStateError, state)

			activity = &mockActivity{t: builder.ActivityTypeDelegate, v: "uninstall"}
			removeMonitoringHook(ctx, cfg, "", tt.args.deploymentID, tt.args.nodeName, activity)

			time.Sleep(1 * time.Second)
			require.Nil(t, err, "Unexpected error while removing check")
			checkReports, err = defaultMonManager.listCheckReports(func(cr CheckReport) bool {
				if cr.DeploymentID == tt.args.deploymentID {
					return true
				}
				return false
			})
			require.Nil(t, err, "Unexpected error while getting check reports list")
			require.Len(t, checkReports, 0, "0 check is expected")
			require.Len(t, defaultMonManager.checks, 0, "0 check is expected in work map")
		})
	}

}

func testIsMonitoringRequiredWithNoPolicy(t *testing.T, client *api.Client) {
	t.Parallel()
	is, _, err := checkExistingMonitoringPolicy(context.Background(), "monitoring2", "Compute1")
	require.Nil(t, err, "Unexpected error during isMonitoringRequired function")
	require.Equal(t, false, is, "unexpected monitoring required")
}

func testIsMonitoringRequiredWithNoPolicyForTarget(t *testing.T, client *api.Client) {
	t.Parallel()
	is, _, err := checkExistingMonitoringPolicy(context.Background(), "monitoring3", "Compute1")
	require.Nil(t, err, "Unexpected error during isMonitoringRequired function")
	require.Equal(t, false, is, "unexpected monitoring required")
}

func testAddAndRemoveCheck(t *testing.T, client *api.Client) {
	log.SetDebug(true)

	dep := "monitoring5"
	node := "Compute1"
	instance1 := "0"
	instance11 := "02"
	expectedCheck := NewCheck(dep, node, instance1)
	NewCheck(dep, node, instance11)

	err := defaultMonManager.registerTCPCheck(dep, node, instance1, "1.2.3.4", 22, 1*time.Second)
	require.Nil(t, err, "Unexpected error while adding check")
	err = defaultMonManager.registerTCPCheck(dep, node, instance11, "1.2.3.4", 22, 1*time.Second)
	require.Nil(t, err, "Unexpected error while adding check")

	time.Sleep(2 * time.Second)
	checkReports, err := defaultMonManager.listCheckReports(func(cr CheckReport) bool {
		if cr.DeploymentID == dep {
			return true
		}
		return false
	})
	require.Nil(t, err, "Unexpected error while getting check reports list")
	require.Len(t, checkReports, 2, "2 checks are expected")
	require.Equal(t, expectedCheck.Report.DeploymentID, checkReports[0].DeploymentID, "unexpected deploymentID")
	require.Equal(t, expectedCheck.Report.NodeName, checkReports[0].NodeName, "unexpected node name")
	require.Equal(t, expectedCheck.Report.Instance, checkReports[0].Instance, "unexpected instance")
	require.Equal(t, CheckStatusCRITICAL, checkReports[0].Status, "unexpected status")

	// Check the instance state has been updated
	state, err := deployments.GetInstanceState(context.Background(), "monitoring5", "Compute1", "0")
	require.Nil(t, err, "Unexpected error while node state")
	require.Equal(t, tosca.NodeStateError, state)

	err = defaultMonManager.flagCheckForRemoval(dep, node, instance1)
	time.Sleep(1 * time.Second)
	require.Nil(t, err, "Unexpected error while removing check")

	require.Nil(t, err, "Unexpected error while removing check")
	checkReports, err = defaultMonManager.listCheckReports(func(cr CheckReport) bool {
		if cr.DeploymentID == dep {
			return true
		}
		return false
	})
	require.Nil(t, err, "Unexpected error while getting check reports list")
	require.Len(t, checkReports, 1, "1 check is expected")
	require.Len(t, defaultMonManager.checks, 1, "0 check is expected in work map")
}
