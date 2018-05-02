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
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/config"
	"testing"
	"time"
)

func testHandleMonitoringWithCheckCreated(t *testing.T, client *api.Client, cfg config.Configuration) {
	t.Parallel()
	ctx := context.Background()

	err := Start(client, cfg)
	require.Nil(t, err, "Unexpected error while starting monitoring")
	err = handleMonitoring(ctx, client, "", "monitoring1", "Compute1", "install")
	require.Nil(t, err, "Unexpected error during handleMonitoring function")
	time.Sleep(2 * time.Second)
	checks, err := defaultMonManager.listCheckReports(func(cr CheckReport) bool {
		if cr.DeploymentID == "monitoring1" {
			return true
		}
		return false
	})
	require.Nil(t, err, "Unexpected error while getting check reports list")
	require.Len(t, checks, 1, "1 check is expected")
	require.Contains(t, checks, CheckReport{DeploymentID: "monitoring1", NodeName: "Compute1", Status: CheckStatusCRITICAL, Instance: "0"})

	err = defaultMonManager.removeHealthCheck(defaultMonManager.buildCheckID("monitoring1", "Compute1", "0"))
	require.Nil(t, err, "Unexpected error while removing health check")
	checks, err = defaultMonManager.listCheckReports(func(cr CheckReport) bool {
		if cr.DeploymentID == "monitoring1" {
			return true
		}
		return false
	})
	require.Nil(t, err, "Unexpected error while getting check reports list")
	require.Len(t, checks, 0, "0 check is expected")
}

func testHandleMonitoringWithoutMonitoringRequiredWithNoTimeInterval(t *testing.T, client *api.Client, cfg config.Configuration) {
	t.Parallel()
	ctx := context.Background()
	err := handleMonitoring(ctx, client, "", "monitoring2", "Compute1", "install")
	require.Nil(t, err, "Unexpected error during handleMonitoring function")

	checks, err := defaultMonManager.listCheckReports(func(cr CheckReport) bool {
		if cr.DeploymentID == "monitoring2" {
			return true
		}
		return false
	})
	require.Nil(t, err, "Unexpected error while getting consul agent checks")
	require.Len(t, checks, 0, "No check expected")
}

func testHandleMonitoringWithoutMonitoringRequiredWithZeroTimeInterval(t *testing.T, client *api.Client, cfg config.Configuration) {
	t.Parallel()
	ctx := context.Background()
	err := handleMonitoring(ctx, client, "", "monitoring3", "Compute1", "install")
	require.Nil(t, err, "Unexpected error during handleMonitoring function doesn't occur")

	checks, err := defaultMonManager.listCheckReports(func(cr CheckReport) bool {
		if cr.DeploymentID == "monitoring3" {
			return true
		}
		return false
	})
	require.Nil(t, err, "Unexpected error while getting consul agent checks")
	require.Len(t, checks, 0, "No check expected")
}

func testHandleMonitoringWithNoIP(t *testing.T, client *api.Client, cfg config.Configuration) {
	t.Parallel()
	ctx := context.Background()
	err := handleMonitoring(ctx, client, "", "monitoring4", "Compute1", "install")
	require.NotNil(t, err, "Expected error during handleMonitoring function")
}

func testAddAndRemoveHealthCheck(t *testing.T, client *api.Client, cfg config.Configuration) {
	t.Parallel()
	ctx := context.Background()
	err := Start(client, cfg)
	require.Nil(t, err, "Unexpected error while starting monitoring")

	checkID := defaultMonManager.buildCheckID("monitoring5", "Compute1", "0")

	err = defaultMonManager.addHealthCheck(ctx, checkID, "1.2.3.4", 22, 1)
	require.Nil(t, err, "Unexpected error while adding health check")

	time.Sleep(2 * time.Second)
	checks, err := defaultMonManager.listCheckReports(func(cr CheckReport) bool {
		if cr.DeploymentID == "monitoring5" {
			return true
		}
		return false
	})
	require.Nil(t, err, "Unexpected error while getting check reports list")
	require.Len(t, checks, 1, "1 check is expected")
	require.Contains(t, checks, CheckReport{DeploymentID: "monitoring5", NodeName: "Compute1", Status: CheckStatusCRITICAL, Instance: "0"})

	err = defaultMonManager.removeHealthCheck(checkID)
	require.Nil(t, err, "Unexpected error while removing health check")

	require.Nil(t, err, "Unexpected error while removing health check")
	checks, err = defaultMonManager.listCheckReports(func(cr CheckReport) bool {
		if cr.DeploymentID == "monitoring5" {
			return true
		}
		return false
	})
	require.Nil(t, err, "Unexpected error while getting check reports list")
	require.Len(t, checks, 0, "0 check is expected")
}
