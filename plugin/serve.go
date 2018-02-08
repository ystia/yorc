package plugin

import (
	"encoding/gob"
	"os"
	"text/template"

	"github.com/hashicorp/go-plugin"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/vault"
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
	ProtocolVersion:  2,
	MagicCookieKey:   "JANUS_PLUG_API",
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
	DelegateFunc                      DelegateFunc
	DelegateSupportedTypes            []string
	Definitions                       map[string][]byte
	OperationFunc                     OperationFunc
	OperationSupportedArtifactTypes   []string
	InfraUsageCollectorFunc           InfraUsageCollectorFunc
	InfraUsageCollectorSupportedInfra string
}

// Serve serves a plugin. This function never returns and should be the final
// function called in the main function of the plugin.
func Serve(opts *ServeOpts) {
	SetupPluginCommunication()

	// As a plugin configure janus logs to go to stderr in order to be show in the parent process
	log.SetOutput(os.Stderr)
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         getPlugins(opts),
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
		InfraUsageCollectorPluginName: &InfraUsageCollectorPlugin{F: opts.InfraUsageCollectorFunc, SupportedInfra: opts.InfraUsageCollectorSupportedInfra},
	}
}

// SetupPluginCommunication makes mandatory actions to allow RPC calls btw server and plugins
// This must be called both by serve and each plugin
func SetupPluginCommunication() {
	// As we have type []interface{} in the config.Configuration structure, we need to register it before sending config from janus server to plugins
	gob.Register(make([]interface{}, 0))
	gob.Register(make([]string, 0))
	gob.RegisterName("DynamicMap", &config.DynamicMap{})
	gob.Register(template.FuncMap{})
	gob.Register(new(vault.Client))
	gob.RegisterName("RPCError", RPCError{})
}
