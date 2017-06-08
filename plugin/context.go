package plugin

import (
	"context"
	"net/rpc"

	"time"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func clientMonitorContextCancellation(ctx context.Context, closeChan chan struct{}, id uint32, broker *plugin.MuxBroker) {
	var dialFailed bool
RETRY:
	conn, err := broker.Dial(id)
	if err != nil {
		if !dialFailed {
			log.Printf("[ERROR] Failed to dial plugin %+v \nLet's retry until we succeed or the execution terminate. This that cancellation may not work.", errors.WithStack(err))
		}
		dialFailed = true
		select {
		case <-closeChan:
			return
		case <-time.After(500 * time.Millisecond):
			goto RETRY
		}
	}
	if dialFailed {
		log.Printf("Plugin remote context cancellation dial finnally succeeded after failures. Things are back to normal.")
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
