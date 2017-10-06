package plugin

import (
	"context"
	"net/rpc"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

// DelegateExecutor is an extention of prov.DelegateExecutor that expose its supported node types
type DelegateExecutor interface {
	prov.DelegateExecutor
	// Returns a list of regexp matches for node types
	GetSupportedTypes() ([]string, error)
}

// DelegatePlugin is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type DelegatePlugin struct {
	F              func() prov.DelegateExecutor
	SupportedTypes []string
}

// Server is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (p *DelegatePlugin) Server(b *plugin.MuxBroker) (interface{}, error) {
	des := &DelegateExecutorServer{Broker: b, SupportedTypes: p.SupportedTypes}
	if p.F != nil {
		des.Delegate = p.F()
	} else if len(p.SupportedTypes) > 0 {
		return nil, NewPluginErrorFromMessage("If DelegateSupportedTypes is defined then you have to defined a DelegateFunc")
	}
	return des, nil
}

// Client is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (p *DelegatePlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &DelegateExecutorClient{Broker: b, Client: c}, nil
}

// DelegateExecutorClient is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type DelegateExecutorClient struct {
	Broker *plugin.MuxBroker
	Client *rpc.Client
}

// ExecDelegate is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (c *DelegateExecutorClient) ExecDelegate(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	id := c.Broker.NextId()
	closeChan := make(chan struct{}, 0)
	defer close(closeChan)
	go clientMonitorContextCancellation(ctx, closeChan, id, c.Broker)

	var resp DelegateExecutorExecDelegateResponse
	args := &DelegateExecutorExecDelegateArgs{
		ChannelID:         id,
		Conf:              conf,
		TaskID:            taskID,
		DeploymentID:      deploymentID,
		NodeName:          nodeName,
		DelegateOperation: delegateOperation,
	}
	err := c.Client.Call("Plugin.ExecDelegate", args, &resp)
	if err != nil {
		return err
	}
	return toError(resp.Error)
}

// DelegateExecutorServer is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type DelegateExecutorServer struct {
	Broker         *plugin.MuxBroker
	Delegate       prov.DelegateExecutor
	SupportedTypes []string
}

// DelegateExecutorExecDelegateArgs is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type DelegateExecutorExecDelegateArgs struct {
	ChannelID         uint32
	Conf              config.Configuration
	TaskID            string
	DeploymentID      string
	NodeName          string
	DelegateOperation string
}

// DelegateExecutorExecDelegateResponse is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type DelegateExecutorExecDelegateResponse struct {
	Error *PluginError
}

// ExecDelegate is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (s *DelegateExecutorServer) ExecDelegate(args *DelegateExecutorExecDelegateArgs, reply *DelegateExecutorExecDelegateResponse) error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	go s.Broker.AcceptAndServe(args.ChannelID, &RPCContextCanceller{CancelFunc: cancelFunc})
	err := s.Delegate.ExecDelegate(ctx, args.Conf, args.TaskID, args.DeploymentID, args.NodeName, args.DelegateOperation)

	var resp DelegateExecutorExecDelegateResponse
	if err != nil {
		resp.Error = NewPluginError(err)
	}
	*reply = resp
	return nil
}

// GetSupportedTypes is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (s *DelegateExecutorServer) GetSupportedTypes(_ interface{}, reply *DelegateExecutorGetTypesResponse) error {
	*reply = DelegateExecutorGetTypesResponse{SupportedTypes: s.SupportedTypes}
	return nil
}

// DelegateExecutorGetTypesResponse is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type DelegateExecutorGetTypesResponse struct {
	SupportedTypes []string
	Error          *PluginError
}

// GetSupportedTypes is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (c *DelegateExecutorClient) GetSupportedTypes() ([]string, error) {
	var resp DelegateExecutorGetTypesResponse
	err := c.Client.Call("Plugin.GetSupportedTypes", new(interface{}), &resp)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get supported types for delegate plugin")
	}

	return resp.SupportedTypes, toError(resp.Error)
}
