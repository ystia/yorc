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
	"github.com/ystia/yorc/log"
	"testing"
	"time"
)

func testHandleMonitoringWithCheckCreated(t *testing.T, client *api.Client) {
	t.Parallel()
	log.SetDebug(true)
	ctx := context.Background()

	err := Start(client)
	require.Nil(t, err, "Unexpected error while starting monitoring")
	err = handleMonitoring(ctx, client, "", "monitoring1", "Compute1", "install")
	require.Nil(t, err, "Unexpected error during handleMonitoring function")
	time.Sleep(2 * time.Second)
	checks, err := client.Agent().Checks()
	require.Nil(t, err, "Unexpected error while getting consul agent checks")
	require.Len(t, checks, 1, "1 check is expected")
	require.Contains(t, checks, "monitoring1:Compute1:0")

	check := checks["monitoring1:Compute1:0"]
	require.Equal(t, "monitoring1:Compute1:0", check.Name, "Name is not equal to \"monitoring1:Compute1:0\"")
	require.Equal(t, "critical", check.Status, "Status is not equal to critical")

	err = client.Agent().CheckDeregister(check.Name)
	require.Nil(t, err, "Unexpected error while unregistering consul agent checks")
}

func testHandleMonitoringWithoutMonitoringRequiredWithNoTimeInterval(t *testing.T, client *api.Client) {
	t.Parallel()
	ctx := context.Background()
	err := handleMonitoring(ctx, client, "", "monitoring2", "Compute1", "install")
	require.Nil(t, err, "Unexpected error during handleMonitoring function")

	checks, err := client.Agent().Checks()
	require.Nil(t, err, "Unexpected error while getting consul agent checks")
	require.Len(t, checks, 0, "No check expected")
}

func testHandleMonitoringWithoutMonitoringRequiredWithZeroTimeInterval(t *testing.T, client *api.Client) {
	t.Parallel()
	ctx := context.Background()
	err := handleMonitoring(ctx, client, "", "monitoring3", "Compute1", "install")
	require.Nil(t, err, "Unexpected error during handleMonitoring function")

	checks, err := client.Agent().Checks()
	require.Nil(t, err, "Unexpected error while getting consul agent checks")
	require.Len(t, checks, 0, "No check expected")
}

func testHandleMonitoringWithNoIP(t *testing.T, client *api.Client) {
	t.Parallel()
	ctx := context.Background()
	err := handleMonitoring(ctx, client, "", "monitoring4", "Compute1", "install")
	require.NotNil(t, err, "Unexpected error during handleMonitoring function")
}

func testAddAndRemoveHealthCheck(t *testing.T, client *api.Client) {
	log.SetDebug(true)
	ctx := context.Background()
	err := Start(client)
	require.Nil(t, err, "Unexpected error while starting monitoring")

	err = defaultMonManager.addHealthCheck(ctx, "monitoring1:Compute1:0", "1.2.3.4", 22, 2)
	require.Nil(t, err, "Unexpected error while adding health check")

	checks, err := client.Agent().Checks()
	require.Nil(t, err, "Unexpected error while getting consul agent checks")
	require.Len(t, checks, 1, "1 check is expected")
	require.Contains(t, checks, "monitoring1:Compute1:0")

	err = defaultMonManager.removeHealthCheck("monitoring1:Compute1:0")
	require.Nil(t, err, "Unexpected error while removing health check")
}
