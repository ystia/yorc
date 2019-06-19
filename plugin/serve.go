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
	"encoding/gob"
	"text/template"

	"github.com/hashicorp/go-plugin"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/vault"
)

const (
	// DelegatePluginName is the name of Delegates Plugins it could be used as a lookup key in Client.Dispense
	DelegatePluginName = "delegate"
	// DefinitionsPluginName is the name of Delegates Plugins it could be used as a lookup key in Client.Dispense
	DefinitionsPluginName = "definitions"
	// ConfigManagerPluginName is the name of ConfigManager plugin it could be used as a lookup key in Client.Dispense
	ConfigManagerPluginName = "cfgManager"
	// OperationPluginName is the name of Operation Plugins it could be used as a lookup key in Client.Dispense
	OperationPluginName = "operation"
	// InfraUsageCollectorPluginName is the name of InfraUsageCollector Plugins it could be used as a lookup key in Client.Dispense
	InfraUsageCollectorPluginName = "infraUsageCollector"
)

// HandshakeConfig are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  3,
	MagicCookieKey:   "YORC_PLUG_API",
	MagicCookieValue: "a3292e718f7c96578aae47e92b7475394e72e6da3de3455554462ba15dde56d1b3187ad0e5f809f50767e0d10ca6944fdf4c6c412380d3aa083b9e8951f7101e",
}

// DelegateFunc is a function that is called when creating a plugin server
type DelegateFunc func() prov.DelegateExecutor

// OperationFunc is a function that is called when creating a plugin server
type OperationFunc func() prov.OperationExecutor

// InfraUsageCollectorFunc is a function that is called when creating a plugin server
type InfraUsageCollectorFunc func() prov.InfraUsageCollector

// ServeOpts are the configurations to serve a plugin.
type ServeOpts struct {
	DelegateFunc                       DelegateFunc
	DelegateSupportedTypes             []string
	Definitions                        map[string][]byte
	OperationFunc                      OperationFunc
	OperationSupportedArtifactTypes    []string
	InfraUsageCollectorFunc            InfraUsageCollectorFunc
	InfraUsageCollectorSupportedInfras []string
}

// Serve serves a plugin. This function never returns and should be the final
// function called in the main function of the plugin.
func Serve(opts *ServeOpts) {
	SetupPluginCommunication()

	// Configuring Yorc logs to use Hashicorp hclog in the plugin so that
	// these logs can be parsed and filtered in the Yorc server  according to their
	// log level
	initPluginYorcLog()

	// Cannot configure here standard log to use Hashicorp hclog, as next call
	// plugin.Serve() is setting standard log output to os.Stderr.
	// This will be done in config.go by the defaultConfigManager

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         getPlugins(opts),
		Logger:          hclogger,
	})
}

func getPlugins(opts *ServeOpts) map[string]plugin.Plugin {
	if opts == nil {
		opts = new(ServeOpts)
	}
	return map[string]plugin.Plugin{
		DelegatePluginName:            &DelegatePlugin{F: opts.DelegateFunc, SupportedTypes: opts.DelegateSupportedTypes},
		OperationPluginName:           &OperationPlugin{F: opts.OperationFunc, SupportedTypes: opts.OperationSupportedArtifactTypes},
		DefinitionsPluginName:         &DefinitionsPlugin{Definitions: opts.Definitions},
		ConfigManagerPluginName:       &ConfigManagerPlugin{&defaultConfigManager{}},
		InfraUsageCollectorPluginName: &InfraUsageCollectorPlugin{F: opts.InfraUsageCollectorFunc, SupportedInfras: opts.InfraUsageCollectorSupportedInfras},
	}
}

// SetupPluginCommunication makes mandatory actions to allow RPC calls btw server and plugins
// This must be called both by serve and each plugin
func SetupPluginCommunication() {
	// As we have type []interface{} in the config.Configuration structure, we need to register it before sending config from yorc server to plugins
	gob.Register(make(map[string]interface{}, 0))
	gob.Register(make(events.LogOptionalFields, 0))
	gob.Register(make([]interface{}, 0))
	gob.Register(make([]string, 0))
	gob.RegisterName("DynamicMap", &config.DynamicMap{})
	gob.Register(template.FuncMap{})
	gob.Register(new(vault.Client))
	gob.RegisterName("RPCError", RPCError{})
}
