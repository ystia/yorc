package plugin

import (
	"testing"

	"github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/require"
)

func TestServeDefaultOpts(t *testing.T) {
	t.Parallel()
	client, _ := plugin.TestPluginRPCConn(t, getPlugins(nil))
	defer client.Close()

	raw, err := client.Dispense(OperationPluginName)
	require.Nil(t, err)

	opPlugin := raw.(OperationExecutor)
	arts, err := opPlugin.GetSupportedArtifactTypes()
	require.Nil(t, err)
	require.Len(t, arts, 0)

	raw, err = client.Dispense(DelegatePluginName)
	require.Nil(t, err)

	delPlugin := raw.(DelegateExecutor)
	types, err := delPlugin.GetSupportedTypes()
	require.Nil(t, err)
	require.Len(t, types, 0)

	raw, err = client.Dispense(DefinitionsPluginName)
	require.Nil(t, err)

	defPlugin := raw.(Definitions)
	defs, err := defPlugin.GetDefinitions()
	require.Nil(t, err)
	require.Len(t, defs, 0)

}
