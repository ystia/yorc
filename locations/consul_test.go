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

package locations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulLocationsPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	defer srv.Stop()

	t.Run("groupLocations", func(t *testing.T) {
		deploymentID := testutil.BuildDeploymentID(t)
		err := deployments.StoreDeploymentDefinition(context.Background(), deploymentID, "testdata/test_topology.yml")
		require.NoError(t, err)
		t.Run("testLocationsFromConfig", func(t *testing.T) {
			testLocationsFromConfig(t, srv, client, deploymentID)
		})
	})
}
