package plugin

import (
	"context"
	"testing"

	plugin "github.com/hashicorp/go-plugin"
)

func Test_clientMonitorContextCancellation(t *testing.T) {
	type args struct {
		ctx       context.Context
		closeChan chan struct{}
		id        uint32
		broker    *plugin.MuxBroker
	}
	tests := []struct {
		name string
		args args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientMonitorContextCancellation(tt.args.ctx, tt.args.closeChan, tt.args.id, tt.args.broker)
		})
	}
}

func TestRPCContextCanceller_CancelContext(t *testing.T) {
	type args struct {
		nothing interface{}
		resp    *CancelContextResponse
	}
	tests := []struct {
		name    string
		r       *RPCContextCanceller
		args    args
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.r.CancelContext(tt.args.nothing, tt.args.resp); (err != nil) != tt.wantErr {
				t.Errorf("RPCContextCanceller.CancelContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
