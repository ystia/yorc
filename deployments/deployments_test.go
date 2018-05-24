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
	"testing"

	"github.com/stretchr/testify/require"
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

	_, err = DeploymentStatusFromString("startOfDepStatusConst", false)
	require.NotNil(t, err)

	_, err = DeploymentStatusFromString("endOfDepStatusConst", false)
	require.NotNil(t, err)

	_, err = DeploymentStatusFromString("does_not_exist", false)
	require.NotNil(t, err)

}
