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

// ActionOperator is an extension of prov.ActionOperator that expose its supported action types
type ActionOperator interface {
	prov.ActionOperator
	// Returns an array of action types types
	GetActionTypes() ([]string, error)
}

// ActionPlugin is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ActionPlugin struct {
	F           func() prov.ActionOperator
	ActionTypes []string
}

// Server is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (p *ActionPlugin) Server(b *plugin.MuxBroker) (interface{}, error) {
	aes := &ActionOperatorServer{Broker: b, ActionTypes: p.ActionTypes}
	if p.F != nil {
		aes.ActionOperator = p.F()
	} else if len(p.ActionTypes) > 0 {
		return nil, errors.New("If ActionTypes is defined then you have to defined an ActionFunc")
	}

	return aes, nil
}

// Client is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (p *ActionPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ActionOperatorClient{Broker: b, Client: c}, nil
}

// ActionOperatorClient is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ActionOperatorClient struct {
	Broker *plugin.MuxBroker
	Client *rpc.Client
}

// ExecAction is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (c *ActionOperatorClient) ExecAction(ctx context.Context, conf config.Configuration, taskID, deploymentID string, action *prov.Action) (bool, error) {
	lof, ok := events.FromContext(ctx)
	if !ok {
		return false, errors.New("Missing contextual log optional fields")
	}
	id := c.Broker.NextId()
	closeChan := make(chan struct{}, 0)
	defer close(closeChan)
	go clientMonitorContextCancellation(ctx, closeChan, id, c.Broker)

	var resp ActionOperatorExecOperationResponse
	args := &ActionOperatorExecOperationArgs{
		ChannelID:         id,
		Conf:              conf,
		TaskID:            taskID,
		DeploymentID:      deploymentID,
		Action:            action,
		LogOptionalFields: lof,
	}
	err := c.Client.Call("Plugin.ExecAction", args, &resp)
	if err != nil {
		return false, errors.Wrap(err, "Failed to call ExecOperation for plugin")
	}
	return resp.Deregister, toError(resp.Error)
}

// ActionOperatorServer is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ActionOperatorServer struct {
	Broker         *plugin.MuxBroker
	ActionOperator prov.ActionOperator
	ActionTypes    []string
}

// ActionOperatorExecOperationArgs is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ActionOperatorExecOperationArgs struct {
	ChannelID         uint32
	Conf              config.Configuration
	TaskID            string
	DeploymentID      string
	Action            *prov.Action
	LogOptionalFields events.LogOptionalFields
}

// ActionOperatorExecOperationResponse is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ActionOperatorExecOperationResponse struct {
	Deregister bool
	Error      *RPCError
}

// ExecAction is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (s *ActionOperatorServer) ExecAction(args *ActionOperatorExecOperationArgs, reply *ActionOperatorExecOperationResponse) error {

	ctx, cancelFunc := context.WithCancel(events.NewContext(context.Background(), args.LogOptionalFields))
	defer cancelFunc()

	go s.Broker.AcceptAndServe(args.ChannelID, &RPCContextCanceller{CancelFunc: cancelFunc})
	var resp ActionOperatorExecOperationResponse
	var err error
	resp.Deregister, err = s.ActionOperator.ExecAction(ctx, args.Conf, args.TaskID, args.DeploymentID, args.Action)
	if err != nil {
		resp.Error = NewRPCError(err)
	}
	*reply = resp
	return nil
}

// GetActionTypes is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (s *ActionOperatorServer) GetActionTypes(_ interface{}, reply *ActionOperatorGetTypesResponse) error {
	*reply = ActionOperatorGetTypesResponse{ActionTypes: s.ActionTypes}
	return nil
}

// ActionOperatorGetTypesResponse is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type ActionOperatorGetTypesResponse struct {
	ActionTypes []string
	Error       *RPCError
}

// GetActionTypes is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (c *ActionOperatorClient) GetActionTypes() ([]string, error) {
	var resp ActionOperatorGetTypesResponse
	err := c.Client.Call("Plugin.GetActionTypes", new(interface{}), &resp)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get supported types for plugin")
	}
	return resp.ActionTypes, toError(resp.Error)
}
