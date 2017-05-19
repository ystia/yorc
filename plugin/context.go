package plugin

import (
	"context"
	"net/rpc"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func clientMonitorContextCancellation(ctx context.Context, closeChan chan struct{}, id uint32, broker *plugin.MuxBroker) {
	conn, err := broker.Dial(id)
	if err != nil {
		log.Panicf("Failed to dial plugin %+v", errors.WithStack(err))
	}
	client := rpc.NewClient(conn)
	defer client.Close()

	select {
	case <-ctx.Done():
		client.Call("Plugin.CancelContext", new(interface{}), &CancelContextResponse{})
	case <-closeChan:
	}
}

// RPCContextCanceller is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type RPCContextCanceller struct {
	CancelFunc context.CancelFunc
}

// CancelContextResponse is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
type CancelContextResponse struct{}

// CancelContext is public for use by reflexion and should be considered as private to this package.
// Please do not use it directly.
func (r *RPCContextCanceller) CancelContext(nothing interface{}, resp *CancelContextResponse) error {
	r.CancelFunc()
	return nil
}
