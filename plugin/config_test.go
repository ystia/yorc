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

package plugin

import (
	"errors"
	"testing"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v3/config"
)

type mockConfigManager struct {
	setupConfigCalled bool
	conf              config.Configuration
	deploymentID      string
}

func (m *mockConfigManager) SetupConfig(cfg config.Configuration) error {
	m.setupConfigCalled = true
	m.conf = cfg

	if m.conf.Consul.Address == "testFailure" {
		return NewRPCError(errors.New("a failure occurred during plugin exec operation"))
	}
	return nil
}

func setupConfigManagerTestEnv(t *testing.T) (*mockConfigManager, *plugin.RPCClient,
	ConfigManager) {

	mock := new(mockConfigManager)
	client, _ := plugin.TestPluginRPCConn(
		t,
		map[string]plugin.Plugin{
			ConfigManagerPluginName: &ConfigManagerPlugin{mock},
		},
		nil)

	raw, err := client.Dispense(ConfigManagerPluginName)
	require.Nil(t, err)

	cm := raw.(ConfigManager)

	return mock, client, cm

}
func TestConfigManagerSetupConfig(t *testing.T) {
	t.Parallel()
	mock, client, cm := setupConfigManagerTestEnv(t)
	defer client.Close()
	err := cm.SetupConfig(config.Configuration{Consul: config.Consul{Address: "test", Datacenter: "testdc"}})
	require.Nil(t, err)
	require.True(t, mock.setupConfigCalled)
	require.Equal(t, "test", mock.conf.Consul.Address)
	require.Equal(t, "testdc", mock.conf.Consul.Datacenter)
}

func TestConfigManagerSetupConfigWithFailure(t *testing.T) {
	t.Parallel()
	_, client, cm := setupConfigManagerTestEnv(t)
	defer client.Close()
	err := cm.SetupConfig(config.Configuration{Consul: config.Consul{Address: "testFailure", Datacenter: "testdc"}})
	require.Error(t, err, "An error was expected during executing plugin operation")
}
