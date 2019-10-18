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
	"context"
	"net/rpc"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/prov"
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
func (c *InfraUsageCollectorClient) GetUsageInfo(ctx context.Context, cfg config.Configuration, taskID, infraName, locationName string) (map[string]interface{}, error) {
	lof, ok := events.FromContext(ctx)
	if !ok {
		return nil, errors.New("Missing contextual log optionnal fields")
	}

	id := c.Broker.NextId()
	closeChan := make(chan struct{}, 0)
	defer close(closeChan)
	go clientMonitorContextCancellation(ctx, closeChan, id, c.Broker)

	var resp InfraUsageCollectorGetUsageInfoResponse
	args := &InfraUsageCollectorGetUsageInfoArgs{
		ChannelID:         id,
		Conf:              cfg,
		TaskID:            taskID,
		InfraName:         infraName,
		LocationName:      locationName,
		LogOptionalFields: lof,
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
	ChannelID         uint32
	Conf              config.Configuration
	TaskID            string
	InfraName         string
	LocationName      string
	LogOptionalFields events.LogOptionalFields
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
	ctx, cancelFunc := context.WithCancel(events.NewContext(context.Background(), args.LogOptionalFields))
	defer cancelFunc()

	go s.Broker.AcceptAndServe(args.ChannelID, &RPCContextCanceller{CancelFunc: cancelFunc})
	usageInfo, err := s.InfraUsageCollector.GetUsageInfo(ctx, args.Conf, args.TaskID, args.InfraName, args.LocationName)

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
