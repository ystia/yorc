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

package deployments

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/testutil"
)

func testDeleteInstance(t *testing.T) {
	ctx := context.Background()
	deploymentID := testutil.BuildDeploymentID(t)
	nodeName := "SomeNode"
	instanceName1 := "1"
	instanceName11 := "11"
	state := "created"
	err := SetInstanceStateStringWithContextualLogs(ctx, deploymentID, nodeName, instanceName1, state)
	require.NoError(t, err)
	err = SetInstanceStateStringWithContextualLogs(ctx, deploymentID, nodeName, instanceName11, state)
	require.NoError(t, err)

	actualState, err := GetInstanceStateString(deploymentID, nodeName, instanceName1)
	require.NoError(t, err)
	require.Equal(t, state, actualState)
	actualState, err = GetInstanceStateString(deploymentID, nodeName, instanceName11)
	require.NoError(t, err)
	require.Equal(t, state, actualState)

	err = DeleteInstance(deploymentID, nodeName, instanceName1)

	actualState, err = GetInstanceStateString(deploymentID, nodeName, instanceName1)
	require.Error(t, err)
	actualState, err = GetInstanceStateString(deploymentID, nodeName, instanceName11)
	require.NoError(t, err)
	require.Equal(t, state, actualState)

}

func testDeleteAllInstances(t *testing.T) {
	ctx := context.Background()
	deploymentID := testutil.BuildDeploymentID(t)
	nodeName1 := "SomeNode1"
	nodeName11 := "SomeNode11"
	instanceName1 := "1"
	instanceName11 := "11"
	state := "created"
	err := SetInstanceStateStringWithContextualLogs(ctx, deploymentID, nodeName1, instanceName1, state)
	require.NoError(t, err)
	err = SetInstanceStateStringWithContextualLogs(ctx, deploymentID, nodeName1, instanceName11, state)
	require.NoError(t, err)
	err = SetInstanceStateStringWithContextualLogs(ctx, deploymentID, nodeName11, instanceName1, state)
	require.NoError(t, err)
	err = SetInstanceStateStringWithContextualLogs(ctx, deploymentID, nodeName11, instanceName11, state)
	require.NoError(t, err)

	actualState, err := GetInstanceStateString(deploymentID, nodeName1, instanceName1)
	require.NoError(t, err)
	require.Equal(t, state, actualState)
	actualState, err = GetInstanceStateString(deploymentID, nodeName1, instanceName11)
	require.NoError(t, err)
	require.Equal(t, state, actualState)
	actualState, err = GetInstanceStateString(deploymentID, nodeName11, instanceName1)
	require.NoError(t, err)
	require.Equal(t, state, actualState)
	actualState, err = GetInstanceStateString(deploymentID, nodeName11, instanceName11)
	require.NoError(t, err)
	require.Equal(t, state, actualState)

	err = DeleteAllInstances(deploymentID, nodeName1)

	actualState, err = GetInstanceStateString(deploymentID, nodeName1, instanceName1)
	require.Error(t, err)
	actualState, err = GetInstanceStateString(deploymentID, nodeName1, instanceName11)
	require.Error(t, err)
	actualState, err = GetInstanceStateString(deploymentID, nodeName11, instanceName1)
	require.NoError(t, err)
	require.Equal(t, state, actualState)
	actualState, err = GetInstanceStateString(deploymentID, nodeName11, instanceName11)
	require.NoError(t, err)
	require.Equal(t, state, actualState)

}
