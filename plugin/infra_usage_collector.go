package plugin

import (
	"net/rpc"
	"novaforge.bull.com/starlings-janus/janus/prov"

	"context"
	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/config"
)

// InfraUsageCollector is an extension of prov.InfraStructureUsageCollector
type InfraUsageCollector interface {
	prov.InfraUsageCollector
	GetSupportedInfras() ([]string, error)
}

// InfraUsageCollectorPlugin is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type InfraUsageCollectorPlugin struct {
	F               func() prov.InfraUsageCollector
	SupportedInfras []string
}

// Server is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (p *InfraUsageCollectorPlugin) Server(b *plugin.MuxBroker) (interface{}, error) {
	des := &InfraUsageCollectorServer{Broker: b, SupportedInfras: p.SupportedInfras}
	if p.F != nil {
		des.InfraUsageCollector = p.F()
	}
	return des, nil
}

// Client is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (p *InfraUsageCollectorPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &InfraUsageCollectorClient{Broker: b, Client: c}, nil
}

// InfraUsageCollectorClient is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type InfraUsageCollectorClient struct {
	Broker *plugin.MuxBroker
	Client *rpc.Client
}

// GetUsageInfo is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (c *InfraUsageCollectorClient) GetUsageInfo(ctx context.Context, cfg config.Configuration, taskID, infraName string) (map[string]interface{}, error) {
	id := c.Broker.NextId()
	closeChan := make(chan struct{}, 0)
	defer close(closeChan)
	go clientMonitorContextCancellation(ctx, closeChan, id, c.Broker)

	var resp InfraUsageCollectorGetUsageInfoResponse
	args := &InfraUsageCollectorGetUsageInfoArgs{
		ChannelID: id,
		Conf:      cfg,
		TaskID:    taskID,
		InfraName: infraName,
	}
	err := c.Client.Call("Plugin.GetUsageInfo", args, &resp)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get usage info for infra collector plugin")
	}
	return resp.UsageInfo, toError(resp.Error)
}

// GetSupportedInfras is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (c *InfraUsageCollectorClient) GetSupportedInfras() ([]string, error) {
	var resp InfraUsageCollectorGetSupportedInfrasResponse
	err := c.Client.Call("Plugin.GetSupportedInfras", new(interface{}), &resp)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get supported infra for infra collector plugin")
	}

	return resp.Infras, toError(resp.Error)
}

// InfraUsageCollectorServer is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type InfraUsageCollectorServer struct {
	Broker              *plugin.MuxBroker
	InfraUsageCollector prov.InfraUsageCollector
	SupportedInfras     []string
}

// InfraUsageCollectorGetUsageInfoArgs is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type InfraUsageCollectorGetUsageInfoArgs struct {
	ChannelID uint32
	Conf      config.Configuration
	TaskID    string
	InfraName string
}

// InfraUsageCollectorGetUsageInfoResponse is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type InfraUsageCollectorGetUsageInfoResponse struct {
	UsageInfo map[string]interface{}
	Error     *RPCError
}

// InfraUsageCollectorGetSupportedInfrasResponse is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type InfraUsageCollectorGetSupportedInfrasResponse struct {
	Infras []string
	Error  *RPCError
}

// GetUsageInfo is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (s *InfraUsageCollectorServer) GetUsageInfo(args *InfraUsageCollectorGetUsageInfoArgs, reply *InfraUsageCollectorGetUsageInfoResponse) error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	go s.Broker.AcceptAndServe(args.ChannelID, &RPCContextCanceller{CancelFunc: cancelFunc})
	usageInfo, err := s.InfraUsageCollector.GetUsageInfo(ctx, args.Conf, args.TaskID, args.InfraName)

	var resp InfraUsageCollectorGetUsageInfoResponse
	resp.UsageInfo = usageInfo

	if err != nil {
		resp.Error = NewRPCError(err)
	}
	*reply = resp
	return nil
}

// GetSupportedInfras is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (s *InfraUsageCollectorServer) GetSupportedInfras(_ interface{}, reply *InfraUsageCollectorGetSupportedInfrasResponse) error {
	*reply = InfraUsageCollectorGetSupportedInfrasResponse{Infras: s.SupportedInfras}
	return nil
}
