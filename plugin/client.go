package plugin

import (
	"os/exec"

	gplugin "github.com/hashicorp/go-plugin"
)

func NewClient(pluginPath string) *gplugin.Client {
	return gplugin.NewClient(&gplugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         getPlugins(nil),
		Cmd:             exec.Command(pluginPath),
	})
}
