package plugin

import (
	"io/ioutil"
	stdlog "log"
	"os/exec"

	gplugin "github.com/hashicorp/go-plugin"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func NewClient(pluginPath string) *gplugin.Client {
	if !log.IsDebug() {
		stdlog.SetOutput(ioutil.Discard)
	}
	return gplugin.NewClient(&gplugin.ClientConfig{
		Managed:         true,
		HandshakeConfig: HandshakeConfig,
		Plugins:         getPlugins(nil),
		Cmd:             exec.Command(pluginPath),
	})
}
