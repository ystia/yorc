package plugin

import (
	"io/ioutil"
	stdlog "log"
	"os/exec"

	gplugin "github.com/hashicorp/go-plugin"
	"novaforge.bull.com/starlings-janus/janus/log"
)

// NewClient returns a properly configured plugin client for a given plugin path
func NewClient(pluginPath string) *gplugin.Client {
	if !log.IsDebug() {
		stdlog.SetOutput(ioutil.Discard)
	}
	// Setup RPC communication
	SetupPluginCommunication()

	return gplugin.NewClient(&gplugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         getPlugins(nil),
		Cmd:             exec.Command(pluginPath),
	})
}
