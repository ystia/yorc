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

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/prov"
)

// OperationExecutor is an extension of prov.OperationExecutor that expose its supported node types
type OperationExecutor interface {
	prov.OperationExecutor
	// Returns a list of regexp matches for node types
	GetSupportedArtifactTypes() ([]string, error)
}

// OperationPlugin is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type OperationPlugin struct {
	F              func() prov.OperationExecutor
	SupportedTypes []string
}

// Server is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (p *OperationPlugin) Server(b *plugin.MuxBroker) (interface{}, error) {
	oes := &OperationExecutorServer{Broker: b, SupportedTypes: p.SupportedTypes}
	if p.F != nil {
		oes.OpExecutor = p.F()
	} else if len(p.SupportedTypes) > 0 {
		return nil, errors.New("If OperationSupportedArtifactTypes is defined then you have to defined an OperationFunc")
	}

	return oes, nil
}

// Client is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (p *OperationPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &OperationExecutorClient{Broker: b, Client: c}, nil
}

// OperationExecutorClient is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type OperationExecutorClient struct {
	Broker *plugin.MuxBroker
	Client *rpc.Client
}

// ExecAsyncOperation is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (c *OperationExecutorClient) ExecAsyncOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation, stepName string) error {
	return errors.New("Asynchronous operation is not yet handled by this executor")
}

// ExecOperation is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (c *OperationExecutorClient) ExecOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
	lof, ok := events.FromContext(ctx)
	if !ok {
		return errors.New("Missing contextual log optionnal fields")
	}
	id := c.Broker.NextId()
	closeChan := make(chan struct{}, 0)
	defer close(closeChan)
	go clientMonitorContextCancellation(ctx, closeChan, id, c.Broker)

	var resp OperationExecutorExecOperationResponse
	args := &OperationExecutorExecOperationArgs{
		ChannelID:         id,
		Conf:              conf,
		TaskID:            taskID,
		DeploymentID:      deploymentID,
		NodeName:          nodeName,
		Operation:         operation,
		LogOptionalFields: lof,
	}
	err := c.Client.Call("Plugin.ExecOperation", args, &resp)
	if err != nil {
		return errors.Wrap(err, "Failed to call ExecOperation for plugin")
	}
	return toError(resp.Error)
}

// OperationExecutorServer is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type OperationExecutorServer struct {
	Broker         *plugin.MuxBroker
	OpExecutor     prov.OperationExecutor
	SupportedTypes []string
}

// OperationExecutorExecOperationArgs is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type OperationExecutorExecOperationArgs struct {
	ChannelID         uint32
	Conf              config.Configuration
	TaskID            string
	DeploymentID      string
	NodeName          string
	Operation         prov.Operation
	LogOptionalFields events.LogOptionalFields
}

// OperationExecutorExecOperationResponse is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type OperationExecutorExecOperationResponse struct {
	Error *RPCError
}

// ExecOperation is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (s *OperationExecutorServer) ExecOperation(args *OperationExecutorExecOperationArgs, reply *OperationExecutorExecOperationResponse) error {

	ctx, cancelFunc := context.WithCancel(events.NewContext(context.Background(), args.LogOptionalFields))
	defer cancelFunc()

	go s.Broker.AcceptAndServe(args.ChannelID, &RPCContextCanceller{CancelFunc: cancelFunc})
	err := s.OpExecutor.ExecOperation(ctx, args.Conf, args.TaskID, args.DeploymentID, args.NodeName, args.Operation)
	var resp OperationExecutorExecOperationResponse
	if err != nil {
		resp.Error = NewRPCError(err)
	}
	*reply = resp
	return nil
}

// GetSupportedArtifactTypes is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (s *OperationExecutorServer) GetSupportedArtifactTypes(_ interface{}, reply *OperationExecutorGetTypesResponse) error {
	*reply = OperationExecutorGetTypesResponse{SupportedTypes: s.SupportedTypes}
	return nil
}

// OperationExecutorGetTypesResponse is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type OperationExecutorGetTypesResponse struct {
	SupportedTypes []string
	Error          *RPCError
}

// GetSupportedArtifactTypes is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (c *OperationExecutorClient) GetSupportedArtifactTypes() ([]string, error) {
	var resp OperationExecutorGetTypesResponse
	err := c.Client.Call("Plugin.GetSupportedArtifactTypes", new(interface{}), &resp)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get supported types for plugin")
	}
	return resp.SupportedTypes, toError(resp.Error)
}
