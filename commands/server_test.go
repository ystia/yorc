package commands

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// Test the following args:
// ./janus server --infrastructure_infra1_auth_url http://localhost:5000/v2.0 --infrastructure_infra1_tenant_name validation
func TestServerInitInfraExtraFlagsWithSpaceDelimiter(t *testing.T) {
	args := []string{"./janus server", "--infrastructure_infra1_auth_url", "http://localhost:5000/v2.0", "--infrastructure_infra1_tenant_name", "validation"}
	serverInitInfraExtraFlags(args)

	require.Len(t, args, 5)
	require.Len(t, serverExtraInfraParams, 2)
	require.Equal(t, serverExtraInfraParams[0], "infrastructures.infra1.auth_url")
	require.Equal(t, serverExtraInfraParams[1], "infrastructures.infra1.tenant_name")
}

// Test the following args:
// ./janus server --infrastructure_infra2_private_network_name=mag3-janus-network --infrastructure_infra2_region=regionOne
func TestServerInitInfraExtraFlagsWithEqualDelimiter(t *testing.T) {
	args := []string{"./janus server", "--infrastructure_infra2_private_network_name=mag3-janus-network", "--infrastructure_infra2_region=regionOne"}
	serverInitInfraExtraFlags(args)

	require.Len(t, args, 3)
	require.Len(t, serverExtraInfraParams, 2)
	require.Equal(t, serverExtraInfraParams[0], "infrastructures.infra2.private_network_name")
	require.Equal(t, serverExtraInfraParams[1], "infrastructures.infra2.region")
}

// Test the following args:
// ./janus server --infrastructure_infra3_private_network_name=mag3-janus-network --infrastructure_infra3_region regionOne
func TestServerInitInfraExtraFlagsWithSpaceAndEqualDelimiters(t *testing.T) {
	args := []string{"./janus server", "--infrastructure_infra3_private_network_name=mag3-janus-network", "--infrastructure_infra3_region", "regionOne"}
	serverInitInfraExtraFlags(args)

	require.Len(t, args, 4)
	require.Len(t, serverExtraInfraParams, 2)
	require.Equal(t, serverExtraInfraParams[0], "infrastructures.infra3.private_network_name")
	require.Equal(t, serverExtraInfraParams[1], "infrastructures.infra3.region")
}

// Test the following args:
// ./janus server --infrastructure_infra4_auth_url http://localhost:5000/v2.0 --infrastructure_infra4_secured --infrastructure_infra4_tenant_name validation
func TestServerInitInfraExtraFlagsWithSpaceDelimiterAndBool(t *testing.T) {
	args := []string{"./janus server", "--infrastructure_infra4_auth_url", "http://localhost:5000/v2.0", "--infrastructure_infra4_secured", "--infrastructure_infra4_tenant_name", "validation"}
	serverInitInfraExtraFlags(args)

	require.Len(t, args, 6)
	require.Len(t, serverExtraInfraParams, 3)
	require.Equal(t, serverExtraInfraParams[0], "infrastructures.infra4.auth_url")
	require.Equal(t, serverExtraInfraParams[1], "infrastructures.infra4.secured")
	require.Equal(t, serverExtraInfraParams[2], "infrastructures.infra4.tenant_name")
}

// Test the following args:
// ./janus server --infrastructure_infra4_auth_url http://localhost:5000/v2.0 --infrastructure_infra4_tenant_name validation --infrastructure_infra4_secured
func TestServerInitInfraExtraFlagsWithSpaceDelimiterAndBoolAtEnd(t *testing.T) {
	args := []string{"./janus server", "--infrastructure_infra5_auth_url", "http://localhost:5000/v2.0", "--infrastructure_infra5_tenant_name", "validation", "--infrastructure_infra5_secured"}
	serverInitInfraExtraFlags(args)

	require.Len(t, args, 6)
	require.Len(t, serverExtraInfraParams, 3)
	require.Equal(t, serverExtraInfraParams[0], "infrastructures.infra5.auth_url")
	require.Equal(t, serverExtraInfraParams[1], "infrastructures.infra5.tenant_name")
	require.Equal(t, serverExtraInfraParams[2], "infrastructures.infra5.secured")
}
