package plugin

import (
	"github.com/hashicorp/go-plugin"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

const (
	DelegatePluginName    = "delegate"
	DefinitionsPluginName = "definitions"
)

// HandshakeConfig are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "JANUS_PLUG_API",
	MagicCookieValue: "a3292e718f7c96578aae47e92b7475394e72e6da3de3455554462ba15dde56d1b3187ad0e5f809f50767e0d10ca6944fdf4c6c412380d3aa083b9e8951f7101e",
}

type DelegateFunc func() prov.DelegateExecutor

// ServeOpts are the configurations to serve a plugin.
type ServeOpts struct {
	DelegateFunc           DelegateFunc
	DelegateSupportedTypes []string
	Definitions            map[string][]byte
}

// Serve serves a plugin. This function never returns and should be the final
// function called in the main function of the plugin.
func Serve(opts *ServeOpts) {
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
		DelegatePluginName:    &DelegatePlugin{F: opts.DelegateFunc, SupportedTypes: opts.DelegateSupportedTypes},
		DefinitionsPluginName: &DefinitionsPlugin{Definitions: opts.Definitions},
	}
}
