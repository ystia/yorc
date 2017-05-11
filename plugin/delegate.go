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
	return &DelegateProviderServer{Broker: b, Delegate: p.F()}, nil
}

func (p *DelegatePlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &DelegateProvider{Broker: b, Client: c}, nil
}

// DelegateProvider is an implementation of terraform.DelegateProvider
// that communicates over RPC.
type DelegateProvider struct {
	Broker *plugin.MuxBroker
	Client *rpc.Client
}

func (p *DelegateProvider) ExecDelegate(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	var resp DelegateProviderExecDelegateResponse
	// api.KV is transient and rebuild based on infos from conf on server side
	args := &DelegateProviderExecDelegateArgs{
		//	Ctx:               ctx,
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

// DelegateProviderServer is a net/rpc compatible structure for serving
// a DelegateProvider. This should not be used directly.
type DelegateProviderServer struct {
	Broker   *plugin.MuxBroker
	Delegate prov.DelegateExecutor
}

type DelegateProviderExecDelegateArgs struct {
	//Ctx               context.Context
	Conf              config.Configuration
	TaskID            string
	DeploymentID      string
	NodeName          string
	DelegateOperation string
}

type DelegateProviderExecDelegateResponse struct {
	Error error
}

func (s *DelegateProviderServer) ExecDelegate(
	args *DelegateProviderExecDelegateArgs,
	reply *DelegateProviderExecDelegateResponse) error {

	err := s.Delegate.ExecDelegate(context.Background(), args.Conf, args.TaskID, args.DeploymentID, args.NodeName, args.DelegateOperation)
	*reply = DelegateProviderExecDelegateResponse{
		Error: err,
	}
	return nil
}
