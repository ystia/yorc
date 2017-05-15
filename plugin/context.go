package plugin

import (
	"context"
	"net/rpc"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func clientMonitorContextCancellation(ctx context.Context, closeChan chan struct{}, id uint32, broker *plugin.MuxBroker) {
	select {
	case <-ctx.Done():
		conn, err := broker.Dial(id)
		if err != nil {
			log.Panicf("Failed to dial plugin %+v", errors.WithStack(err))
		}
		defer conn.Close()
		client := rpc.NewClient(conn)
		defer client.Close()
		client.Call("Plugin.CancelContext", new(interface{}), &CancelContextResponse{})

	case <-closeChan:
	}
}

//
type RPCContextCanceller struct {
	CancelFunc context.CancelFunc
}

type CancelContextResponse struct{}

func (r *RPCContextCanceller) CancelContext(nothing interface{}, resp *CancelContextResponse) error {
	r.CancelFunc()
	return nil
}
