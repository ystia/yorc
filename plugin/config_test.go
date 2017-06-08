package plugin

import (
	"testing"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/config"
)

type mockConfigManager struct {
	setupConfigCalled bool
	conf              config.Configuration
}

func (m *mockConfigManager) SetupConfig(cfg config.Configuration) error {
	m.setupConfigCalled = true
	m.conf = cfg
	return nil
}

func TestConfigManagerSetupConfig(t *testing.T) {
	t.Parallel()
	mock := new(mockConfigManager)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		ConfigManagerPluginName: &ConfigManagerPlugin{mock},
	})
	defer client.Close()

	raw, err := client.Dispense(ConfigManagerPluginName)
	require.Nil(t, err)

	cm := raw.(ConfigManager)

	err = cm.SetupConfig(config.Configuration{ConsulAddress: "test", ConsulDatacenter: "testdc"})
	require.Nil(t, err)
	require.True(t, mock.setupConfigCalled)
	require.Equal(t, "test", mock.conf.ConsulAddress)
	require.Equal(t, "testdc", mock.conf.ConsulDatacenter)
}
