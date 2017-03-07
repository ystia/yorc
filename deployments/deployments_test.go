package deployments

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeploymentStatusFromString(t *testing.T) {
	t.Parallel()
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
