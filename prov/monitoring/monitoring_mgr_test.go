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
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"testing"
)

func testHandleMonitoringWithCheckCreated(t *testing.T, client *api.Client) {
	t.Parallel()

	err := handleMonitoring(client, "", "monitoring1", "Compute1", "install")
	require.Nil(t, err, "Unexpected error during handleMonitoring function")

	checks, err := client.Agent().Checks()
	require.Nil(t, err, "Unexpected error while getting consul agent checks")
	require.Len(t, checks, 1, "1 check is expected")
	require.Contains(t, checks, "monitoring1_Compute1-0")

	check := checks["monitoring1_Compute1-0"]
	require.Equal(t, "monitoring1_Compute1-0", check.Name, "Name is not equal to \"monitoring1_Compute1-0\"")
	require.Equal(t, "critical", check.Status, "Status is not equal to critical")
}

func testHandleMonitoringWithoutMonitoringRequiredWithNoTimeInterval(t *testing.T, client *api.Client) {
	t.Parallel()

	err := handleMonitoring(client, "", "monitoring2", "Compute1", "install")
	require.Nil(t, err, "Unexpected error during handleMonitoring function")

	checks, err := client.Agent().Checks()
	require.Nil(t, err, "Unexpected error while getting consul agent checks")
	require.Len(t, checks, 0, "No check expected")
}

func testHandleMonitoringWithoutMonitoringRequiredWithZeroTimeInterval(t *testing.T, client *api.Client) {
	t.Parallel()

	err := handleMonitoring(client, "", "monitoring3", "Compute1", "install")
	require.Nil(t, err, "Unexpected error during handleMonitoring function")

	checks, err := client.Agent().Checks()
	require.Nil(t, err, "Unexpected error while getting consul agent checks")
	require.Len(t, checks, 0, "No check expected")
}

func testHandleMonitoringWithNoIP(t *testing.T, client *api.Client) {
	t.Parallel()

	err := handleMonitoring(client, "", "monitoring4", "Compute1", "install")
	require.NotNil(t, err, "Unexpected error during handleMonitoring function")
}
