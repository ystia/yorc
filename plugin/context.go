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
	"time"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/log"
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
		log.Printf("Plugin remote context cancellation dial finally succeeded after failures. Things are back to normal.")
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
