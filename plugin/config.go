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
	"net/rpc"
	"text/template"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/helper/consulutil"
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

	// Configuring standard log to use Hashicorp hclog within the plugin so that
	// these standard logs can be parsed and filtered in the Yorc server according
	// to their log level
	initPluginStdLog()

	// Currently we only use this plugin part to initialize the Consul publisher
	cClient, err := cfg.GetNewConsulClient()
	if err != nil {
		return err
	}
	kv := cClient.KV()
	maxPubSub := cfg.Consul.PubMaxRoutines
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
	Broker              *plugin.MuxBroker
}

// Server is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (p *ConfigManagerPlugin) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &ConfigManagerServer{PluginConfigManager: p.PluginConfigManager, Broker: b}, nil
}

// ConfigManagerClient is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ConfigManagerClient struct {
	Client *rpc.Client
	Broker *plugin.MuxBroker
}

// Client is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (p *ConfigManagerPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ConfigManagerClient{Client: c, Broker: b}, nil
}

// ConfigManagerSetupConfigArgs is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ConfigManagerSetupConfigArgs struct {
	Cfg config.Configuration
	ID  uint32
}

// ConfigManagerSetupConfigResponse is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ConfigManagerSetupConfigResponse struct {
	Error *RPCError
}

// SetupConfig is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (s *ConfigManagerServer) SetupConfig(args *ConfigManagerSetupConfigArgs, reply *ConfigManagerSetupConfigResponse) error {
	var resp ConfigManagerSetupConfigResponse
	conn, err := s.Broker.Dial(args.ID)
	if err != nil {
		resp.Error = NewRPCError(err)
		*reply = resp
		return nil
	}

	client := rpc.NewClient(conn)
	// TODO check how and when to close the client connection: when plugin stops ? using a custom operation ?
	// client.Close()
	ctrc := &ConfigTemplateResolverClient{Client: client}
	config.DefaultConfigTemplateResolver = ctrc

	err = s.PluginConfigManager.SetupConfig(args.Cfg)

	if err != nil {
		resp.Error = NewRPCError(err)
	}
	*reply = resp
	return nil
}

// SetupConfig is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (c *ConfigManagerClient) SetupConfig(cfg config.Configuration) error {
	var resp ConfigManagerSetupConfigResponse
	id := c.Broker.NextId()

	// TODO: Check how and when to stop this routine when plugin stops ? using a custom operation ?
	go c.Broker.AcceptAndServe(id, &ConfigTemplateResolverServer{})
	args := ConfigManagerSetupConfigArgs{Cfg: cfg, ID: id}

	err := c.Client.Call("Plugin.SetupConfig", &args, &resp)
	if err != nil {
		return errors.Wrap(err, "Failed call ConfigManager setup for plugin")
	}
	return toError(resp.Error)
}

// ConfigTemplateResolverClient is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ConfigTemplateResolverClient struct {
	Client *rpc.Client
}

// SetTemplatesFunctions is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (c *ConfigTemplateResolverClient) SetTemplatesFunctions(fm template.FuncMap) {
	// Not Implemented
}

// ConfigTemplateResolverResolveValueWithTemplatesArgs is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ConfigTemplateResolverResolveValueWithTemplatesArgs struct {
	Key   string
	Value interface{}
}

// ConfigTemplateResolverResolveValueWithTemplatesResponse is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ConfigTemplateResolverResolveValueWithTemplatesResponse struct {
	Value interface{}
}

// ResolveValueWithTemplates is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (c *ConfigTemplateResolverClient) ResolveValueWithTemplates(key string, value interface{}) interface{} {
	var resp ConfigTemplateResolverResolveValueWithTemplatesResponse
	args := ConfigTemplateResolverResolveValueWithTemplatesArgs{Key: key, Value: value}

	err := c.Client.Call("Plugin.ResolveValueWithTemplates", &args, &resp)
	if err != nil {
		return nil
	}
	return resp.Value

}

// ConfigTemplateResolverServer is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ConfigTemplateResolverServer struct{}

// ResolveValueWithTemplates is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (s *ConfigTemplateResolverServer) ResolveValueWithTemplates(args *ConfigTemplateResolverResolveValueWithTemplatesArgs, reply *ConfigTemplateResolverResolveValueWithTemplatesResponse) error {

	res := config.DefaultConfigTemplateResolver.ResolveValueWithTemplates(args.Key, args.Value)

	*reply = ConfigTemplateResolverResolveValueWithTemplatesResponse{res}
	return nil
}
