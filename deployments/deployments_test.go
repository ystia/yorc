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

package deployments

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/testutil"
)

func TestDeploymentStatusFromString(t *testing.T) {
	// t.Parallel()
	status, err := DeploymentStatusFromString("initial", true)
	require.Nil(t, err)
	require.Equal(t, INITIAL, status)

	status, err = DeploymentStatusFromString("InItIal", true)
	require.Nil(t, err)
	require.Equal(t, INITIAL, status)

	status, err = DeploymentStatusFromString("INITIAL", true)
	require.Nil(t, err)
	require.Equal(t, INITIAL, status)

	status, err = DeploymentStatusFromString("INITIAL", false)
	require.Nil(t, err)
	require.Equal(t, INITIAL, status)

	_, err = DeploymentStatusFromString("initial", false)
	require.NotNil(t, err)

	_, err = DeploymentStatusFromString("iNiTiAL", false)
	require.NotNil(t, err)

	_, err = DeploymentStatusFromString("iNiTiAL", false)
	require.NotNil(t, err)

	status, err = DeploymentStatusFromString("UNDEPLOYMENT_FAILED", true)
	require.Nil(t, err)
	require.Equal(t, UNDEPLOYMENT_FAILED, status)

	status, err = DeploymentStatusFromString("undeployment_failed", true)
	require.Nil(t, err)
	require.Equal(t, UNDEPLOYMENT_FAILED, status)

	_, err = DeploymentStatusFromString("does_not_exist", false)
	require.NotNil(t, err)

}

func testPurgedDeployments(t *testing.T, cc *api.Client) {
	t.Parallel()
	deploymentID := testutil.BuildDeploymentID(t)
	ctx := context.Background()
	initiallyPurgedNb := 15
	generatePurgedDeployments(ctx, t, cc, deploymentID, initiallyPurgedNb)

	require.Equal(t, initiallyPurgedNb, getPurgedDeploymentsNb(t, cc.KV()))

	// Too long timeout no deployments specified
	err := CleanupPurgedDeployments(ctx, cc, 30*time.Minute)
	require.NoError(t, err)

	require.Equal(t, initiallyPurgedNb, getPurgedDeploymentsNb(t, cc.KV()))

	// Specify some deployments but still use a too long timeout
	err = CleanupPurgedDeployments(ctx, cc, 30*time.Minute, deploymentID+"-2", deploymentID+"-10", deploymentID+"-8")
	require.NoError(t, err)

	require.Equal(t, initiallyPurgedNb-3, getPurgedDeploymentsNb(t, cc.KV()))

	time.Sleep(2 * time.Second)
	err = TagDeploymentAsPurged(ctx, cc, deploymentID+"-16")
	require.NoError(t, err)
	err = TagDeploymentAsPurged(ctx, cc, deploymentID+"-17")
	require.NoError(t, err)

	// Should remove everything except deploymentID-16
	err = CleanupPurgedDeployments(ctx, cc, 2*time.Second, deploymentID+"-17")
	require.NoError(t, err)

	require.Equal(t, 1, getPurgedDeploymentsNb(t, cc.KV()))

	// Specify ridiculously short timeout
	err = CleanupPurgedDeployments(ctx, cc, 30*time.Nanosecond)
	require.NoError(t, err)

	require.Equal(t, 0, getPurgedDeploymentsNb(t, cc.KV()))

}

func generatePurgedDeployments(ctx context.Context, t *testing.T, cc *api.Client, deploymentID string, nb int) {
	for i := 0; i < nb; i++ {
		err := TagDeploymentAsPurged(ctx, cc, fmt.Sprintf("%s-%d", deploymentID, i+1))
		require.NoError(t, err)
	}
}

func getPurgedDeploymentsNb(t *testing.T, kv *api.KV) int {
	k, _, err := kv.Keys(consulutil.PurgedDeploymentKVPrefix+"/", "/", nil)
	require.NoError(t, err)
	return len(k)
}
