package plugin

import (
	"context"
	"net/rpc"

	plugin "github.com/hashicorp/go-plugin"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

type DelegatePlugin struct {
	F func() prov.DelegateExecutor
}

func (p *DelegatePlugin) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &DelegateExecutorServer{Broker: b, Delegate: p.F()}, nil
}

func (p *DelegatePlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &DelegateExecutor{Broker: b, Client: c}, nil
}

// DelegateExecutor is an implementation of prov.DelegateExecutor
// that communicates over RPC.
type DelegateExecutor struct {
	Broker *plugin.MuxBroker
	Client *rpc.Client
}

// ExecDelegate implements prov.DelegateExecutor
func (p *DelegateExecutor) ExecDelegate(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	id := p.Broker.NextId()
	closeChan := make(chan struct{}, 0)
	defer close(closeChan)
	go clientMonitorContextCancellation(ctx, closeChan, id, p.Broker)

	var resp DelegateProviderExecDelegateResponse
	args := &DelegateProviderExecDelegateArgs{
		//	Ctx:               ctx,
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
	Broker   *plugin.MuxBroker
	Delegate prov.DelegateExecutor
}

type DelegateProviderExecDelegateArgs struct {
	//Ctx               context.Context
	ChannelID         uint32
	Conf              config.Configuration
	TaskID            string
	DeploymentID      string
	NodeName          string
	DelegateOperation string
}

type DelegateProviderExecDelegateResponse struct {
	Error error
}

func (s *DelegateExecutorServer) ExecDelegate(args *DelegateProviderExecDelegateArgs, reply *DelegateProviderExecDelegateResponse) error {

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	go s.Broker.AcceptAndServe(args.ChannelID, &RPCContextCanceller{CancelFunc: cancelFunc})
	err := s.Delegate.ExecDelegate(ctx, args.Conf, args.TaskID, args.DeploymentID, args.NodeName, args.DelegateOperation)
	*reply = DelegateProviderExecDelegateResponse{
		Error: err,
	}
	return nil
}
