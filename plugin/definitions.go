package plugin

import (
	"net/rpc"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
)

// Definitions is the interface that allows a plugin to export TOSCA definitions
type Definitions interface {
	// GetDefinitions returns a map of definition names / definition content
	GetDefinitions() (map[string][]byte, error)
}

// DefinitionsPlugin is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type DefinitionsPlugin struct {
	Definitions map[string][]byte
}

// DefinitionsServer is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type DefinitionsServer struct {
	Definitions map[string][]byte
}

// Server is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (p *DefinitionsPlugin) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &DefinitionsServer{Definitions: p.Definitions}, nil
}

// DefinitionsClient is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type DefinitionsClient struct {
	Client *rpc.Client
}

// Client is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (p *DefinitionsPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &DefinitionsClient{Client: c}, nil
}

// GetDefinitions is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (s *DefinitionsServer) GetDefinitions(_ interface{}, reply *DelegateExecutorGetDefinitionsResponse) error {
	*reply = DelegateExecutorGetDefinitionsResponse{
		Definitions: s.Definitions,
	}
	return nil
}

// DelegateExecutorGetDefinitionsResponse is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type DelegateExecutorGetDefinitionsResponse struct {
	Definitions map[string][]byte
	Error       *RPCError
}

// GetDefinitions is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (c *DefinitionsClient) GetDefinitions() (map[string][]byte, error) {
	var resp DelegateExecutorGetDefinitionsResponse
	err := c.Client.Call("Plugin.GetDefinitions", new(interface{}), &resp)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get tosca definitions for delegate plugin")
	}
	return resp.Definitions, toError(resp.Error)
}
