package operations

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsTargetOperation(t *testing.T) {
	t.Parallel()

	res := IsTargetOperation("tosca.interfaces.node.lifecycle.Configure.pre_configure_target")
	require.True(t, res)

	res = IsTargetOperation("tosca.interfaces.node.lifecycle.Configure.add_source")
	require.True(t, res)

	res = IsTargetOperation("tosca.interfaces.node.lifecycle.Configure.pre_configure_source")
	require.False(t, res)
}
