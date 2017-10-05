package plugin

import (
	"net/rpc"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

// ConfigManager allows to send configuration to the plugin
//
// This is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ConfigManager interface {
	SetupConfig(cfg config.Configuration) error
}

type defaultConfigManager struct {
}

func (cm *defaultConfigManager) SetupConfig(cfg config.Configuration) error {
	// Currently we only use this plugin part to initialize the Consul publisher
	cClient, err := cfg.GetConsulClient()
	if err != nil {
		return err
	}
	kv := cClient.KV()
	maxPubSub := cfg.ConsulPubMaxRoutines
	if maxPubSub == 0 {
		maxPubSub = config.DefaultConsulPubMaxRoutines
	}
	consulutil.InitConsulPublisher(maxPubSub, kv)
	return nil
}

// ConfigManagerPlugin is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ConfigManagerPlugin struct {
	PluginConfigManager ConfigManager
}

// ConfigManagerServer is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ConfigManagerServer struct {
	PluginConfigManager ConfigManager
}

// Server is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (p *ConfigManagerPlugin) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &ConfigManagerServer{p.PluginConfigManager}, nil
}

// ConfigManagerClient is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ConfigManagerClient struct {
	Client *rpc.Client
}

// Client is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (p *ConfigManagerPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ConfigManagerClient{Client: c}, nil
}

// ConfigManagerSetupConfigArgs is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ConfigManagerSetupConfigArgs struct {
	Cfg config.Configuration
}

// ConfigManagerSetupConfigResponse is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ConfigManagerSetupConfigResponse struct {
	Error error
}

// SetupConfig is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (s *ConfigManagerServer) SetupConfig(args *ConfigManagerSetupConfigArgs, resp *ConfigManagerSetupConfigResponse) error {

	err := s.PluginConfigManager.SetupConfig(args.Cfg)
	*resp = ConfigManagerSetupConfigResponse{err}
	return nil
}

// SetupConfig is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (c *ConfigManagerClient) SetupConfig(cfg config.Configuration) error {
	var resp ConfigManagerSetupConfigResponse
	args := ConfigManagerSetupConfigArgs{cfg}

	err := c.Client.Call("Plugin.SetupConfig", &args, &resp)
	if err != nil {
		return errors.Wrap(err, "Failed call ConfigManager setup for plugin")
	}
	return resp.Error
}
