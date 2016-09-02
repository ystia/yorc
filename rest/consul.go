package rest

import (
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/log"
	"sync"
)

func (s *Server) storeConsulKey(errCh chan error, wg *sync.WaitGroup, key, value string) {
	// PUT a new KV pair
	p := &api.KVPair{Key: key, Value: []byte(value)}
	wg.Add(1)
	// will block if channel is full
	s.consulPublisher.pubChannel <- consulPublishRequest{kvp: p, wg: wg, errChan: errCh}
}

type consulPublishRequest struct {
	kvp     *api.KVPair
	wg      *sync.WaitGroup
	errChan chan error
}

type consulPublisher struct {
	pubChannel chan consulPublishRequest
	kv         *api.KV
	shutdownCh chan struct{}
}

func newConsulPublisher(maxItems int, kv *api.KV, shutdownCh chan struct{}) *consulPublisher {
	log.Debugf("Consul Publisher created for the REST API with a maximum of %d parallel routines.", maxItems)
	return &consulPublisher{pubChannel: make(chan consulPublishRequest, maxItems), kv: kv, shutdownCh: shutdownCh}
}

func (c *consulPublisher) run() {
	for {
		select {
		case cpr := <-c.pubChannel:
			go func(p *api.KVPair, wg *sync.WaitGroup, errCh chan error) {
				defer wg.Done()
				if _, err := c.kv.Put(p, nil); err != nil {
					errCh <- err
				}
			}(cpr.kvp, cpr.wg, cpr.errChan)
		case <-c.shutdownCh:
			log.Debugf("consul publisher recieved shutdown signal, exiting.")
			return
		}
	}
}
