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
