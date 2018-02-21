package plugin

import (
	"errors"
	"testing"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/config"
)

type mockConfigManager struct {
	setupConfigCalled bool
	conf              config.Configuration
	deploymentID      string
}

func (m *mockConfigManager) SetupConfig(cfg config.Configuration) error {
	m.setupConfigCalled = true
	m.conf = cfg

	if m.conf.ConsulAddress == "testFailure" {
		return NewRPCError(errors.New("a failure occurred during plugin exec operation"))
	}
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

func TestConfigManagerSetupConfigWithFailure(t *testing.T) {
	t.Parallel()
	mock := new(mockConfigManager)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		ConfigManagerPluginName: &ConfigManagerPlugin{mock},
	})
	defer client.Close()

	raw, err := client.Dispense(ConfigManagerPluginName)
	require.Nil(t, err)

	cm := raw.(ConfigManager)

	err = cm.SetupConfig(config.Configuration{ConsulAddress: "testFailure", ConsulDatacenter: "testdc"})
	require.Error(t, err, "An error was expected during executing plugin operation")
}
