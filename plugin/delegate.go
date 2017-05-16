package plugin

import (
	"context"
	"net/rpc"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

type DelegateExecutor interface {
	prov.DelegateExecutor
	GetSupportedTypes() ([]string, error)
}

type DelegatePlugin struct {
	F              func() prov.DelegateExecutor
	SupportedTypes []string
}

func (p *DelegatePlugin) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &DelegateExecutorServer{Broker: b, Delegate: p.F(), SupportedTypes: p.SupportedTypes}, nil
}

func (p *DelegatePlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &DelegateExecutorClient{Broker: b, Client: c}, nil
}

// DelegateExecutorClient is an implementation of prov.DelegateExecutorClient
// that communicates over RPC.
type DelegateExecutorClient struct {
	Broker *plugin.MuxBroker
	Client *rpc.Client
}

// ExecDelegate implements prov.DelegateExecutor
func (p *DelegateExecutorClient) ExecDelegate(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	id := p.Broker.NextId()
	closeChan := make(chan struct{}, 0)
	defer close(closeChan)
	go clientMonitorContextCancellation(ctx, closeChan, id, p.Broker)

	var resp DelegateExecutorExecDelegateResponse
	args := &DelegateExecutorExecDelegateArgs{
		ChannelID:         id,
		Conf:              conf,
		TaskID:            taskID,
		DeploymentID:      deploymentID,
		NodeName:          nodeName,
		DelegateOperation: delegateOperation,
	}
	err := p.Client.Call("Plugin.ExecDelegate", args, &resp)
	if err != nil {
		return err
	}
	return resp.Error
}

// DelegateExecutorServer is a net/rpc compatible structure for serving
// a DelegateProvider. This should not be used directly.
type DelegateExecutorServer struct {
	Broker         *plugin.MuxBroker
	Delegate       prov.DelegateExecutor
	SupportedTypes []string
}

type DelegateExecutorExecDelegateArgs struct {
	ChannelID         uint32
	Conf              config.Configuration
	TaskID            string
	DeploymentID      string
	NodeName          string
	DelegateOperation string
}

type DelegateExecutorExecDelegateResponse struct {
	Error error
}

func (s *DelegateExecutorServer) ExecDelegate(args *DelegateExecutorExecDelegateArgs, reply *DelegateExecutorExecDelegateResponse) error {

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	go s.Broker.AcceptAndServe(args.ChannelID, &RPCContextCanceller{CancelFunc: cancelFunc})
	err := s.Delegate.ExecDelegate(ctx, args.Conf, args.TaskID, args.DeploymentID, args.NodeName, args.DelegateOperation)
	*reply = DelegateExecutorExecDelegateResponse{
		Error: err,
	}
	return nil
}

func (s *DelegateExecutorServer) GetSupportedTypes(_ interface{}, reply *DelegateExecutorGetTypesResponse) error {
	*reply = DelegateExecutorGetTypesResponse{SupportedTypes: s.SupportedTypes}
	return nil
}

type DelegateExecutorGetTypesResponse struct {
	SupportedTypes []string
	Error          error
}

func (c *DelegateExecutorClient) GetSupportedTypes() ([]string, error) {
	var resp DelegateExecutorGetTypesResponse
	err := c.Client.Call("Plugin.GetSupportedTypes", new(interface{}), &resp)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get supported types for delegate plugin")
	}
	return resp.SupportedTypes, resp.Error
}
