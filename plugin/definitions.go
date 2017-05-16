package plugin

import (
	"net/rpc"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
)

type Definitions interface {
	GetDefinitions() (map[string][]byte, error)
}

type DefinitionsPlugin struct {
	Definitions map[string][]byte
}

type DefinitionsServer struct {
	Definitions map[string][]byte
}

func (p *DefinitionsPlugin) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &DefinitionsServer{Definitions: p.Definitions}, nil
}

type DefinitionsClient struct {
	Client *rpc.Client
}

func (p *DefinitionsPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &DefinitionsClient{Client: c}, nil
}

func (s *DefinitionsServer) GetDefinitions(_ interface{}, reply *DelegateExecutorGetDefinitionsResponse) error {
	*reply = DelegateExecutorGetDefinitionsResponse{
		Definitions: s.Definitions,
	}
	return nil
}

type DelegateExecutorGetDefinitionsResponse struct {
	Definitions map[string][]byte
	Error       error
}

func (c *DefinitionsClient) GetDefinitions() (map[string][]byte, error) {
	var resp DelegateExecutorGetDefinitionsResponse
	err := c.Client.Call("Plugin.GetDefinitions", new(interface{}), &resp)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get tosca definitions for delegate plugin")
	}
	return resp.Definitions, resp.Error
}
