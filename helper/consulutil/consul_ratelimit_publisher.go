package consulutil

import (
	"context"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"novaforge.bull.com/starlings-janus/janus/log"
)

// Internal type used to uniquely identify the errorgroup in a context
type ctxErrGroupKey struct{}

// Internal variable used to uniquely identify the errorgroup in a context
var errGroupKey = ctxErrGroupKey{}

// rateLimitedConsulPublisher singleton
var consulPub *rateLimitedConsulPublisher

// ConsulStore allows to store keys in Consul
type ConsulStore interface {
	StoreConsulKey(key string, value []byte)
	StoreConsulKeyAsString(key, value string)
}

// consulStore is the default implementation for ConsulStore.
// It allows to use a context and an errgroup.Group to store several keys in parallel in Consul but with a defined parallelism.
type consulStore struct {
	ctx context.Context
}

// WithContext uses a given context to create an errgroup.Group that will be use to store keys in Consul in parallel.
//
// If the given context is cancelled then keys storing are not executed. If an error occurs when storing a key then the context is cancelled
// and others keys are not stored. The error could be retrieved with the Wait() function of the errgroup.Group.
//
// Here is a typical use case for this function:
//
//	errCtx, errGroup, consulStore := consulutil.WithContext(ctx)
//	consulStore.StoreConsulKey(key1, val1)
//	// several concurrent keys storing
//	consulStore.StoreConsulKey(keyN, valN)
//	err := errGroup.Wait()
func WithContext(ctx context.Context) (context.Context, *errgroup.Group, ConsulStore) {
	if ctx == nil {
		log.Panic(errors.New("Context can't be nil."))
	}
	errGroup, errCtx := errgroup.WithContext(ctx)
	errCtx = context.WithValue(errCtx, errGroupKey, errGroup)
	return errCtx, errGroup, &consulStore{ctx: errCtx}
}

// StoreConsulKeyAsString is equivalent to StoreConsulKey(key, []byte(value))
func StoreConsulKeyAsString(key, value string) error {
	return StoreConsulKey(key, []byte(value))
}

// StoreConsulKey stores a Consul key without the use of a ConsulStore you should avoid to use it when storing several keys that could be
// stored concurrently as this function has an important overhead of creating an execution context using the WithContext function and
// waiting for the key to be store in Consul using errGroup.Wait()
func StoreConsulKey(key string, value []byte) error {
	_, errGroup, store := WithContext(context.Background())
	store.StoreConsulKey(key, value)
	return errGroup.Wait()
}

// StoreConsulKeyAsString is equivalent to StoreConsulKey(key, []byte(value))
func (cs *consulStore) StoreConsulKeyAsString(key, value string) {
	cs.StoreConsulKey(key, []byte(value))
}

// StoreConsul sends a Consul key/value down to the rate limited Consul Publisher.
//
// This function may be blocking if the rate limited queue is full. The number of maximum parallel execution is controlled with an option.
func (cs *consulStore) StoreConsulKey(key string, value []byte) {
	// PUT a new KV pair
	select {
	case <-cs.ctx.Done():
		return
	default:
	}
	p := &api.KVPair{Key: key, Value: value}
	// Will block if the rateLimitedConsulPublisher is itself blocked by its semaphore
	consulPub.pubChannel <- consulPublishRequest{kvp: p, ctx: cs.ctx}
}

// consulPublishRequest holds the Key/Value pair for Consul and the context used to store it in Consul
type consulPublishRequest struct {
	kvp *api.KVPair
	ctx context.Context
}

// rateLimitedConsulPublisher is used to store Consul keys.
//
// It uses a buffered channel as semaphore to limit the number of parallel call to the Consul API
type rateLimitedConsulPublisher struct {
	pubChannel chan consulPublishRequest
	sem        chan struct{}
	kv         *api.KV
	shutdownCh chan struct{}
	quitCh     chan struct{}
}

// RunConsulPublisher starts a Consul Publisher and limit the number of parallel call to the Consul API to store keys with the given maxItems.
//
// It is safe but discouraged to call several times this function. Only the last call will be taken into account. The only valid reason to
// do so is for reconfiguration and changing the limit.
func RunConsulPublisher(maxItems int, kv *api.KV, shutdownCh chan struct{}) {
	log.Debugf("Consul Publisher created with a maximum of %d parallel routines.", maxItems)

	oldPub := consulPub

	consulPub = &rateLimitedConsulPublisher{pubChannel: make(chan consulPublishRequest), sem: make(chan struct{}, maxItems), kv: kv, shutdownCh: shutdownCh, quitCh: make(chan struct{})}
	go consulPub.run()

	if oldPub != nil {
		close(oldPub.quitCh)
	}
}

// run starts listening on consulPublishRequest and publish it.
func (c *rateLimitedConsulPublisher) run() {
	for {
		select {
		case cpr := <-c.pubChannel:
			// will block if channel is full
			c.sem <- struct{}{}
			cpr = cpr
			select {
			case <-cpr.ctx.Done():
				// Release semaphore
				<-c.sem
				continue
			default:
			}
			errGroup := cpr.ctx.Value(errGroupKey).(*errgroup.Group)
			errGroup.Go(func() error {
				defer func() {
					// Release semaphore
					<-c.sem
				}()
				if _, err := c.kv.Put(cpr.kvp, nil); err != nil {
					return errors.Wrapf(err, "Failed to store consul key %q", cpr.kvp.Key)
				}
				return nil
			})

		case <-c.shutdownCh:
			log.Debugf("consul publisher recieved shutdown signal, exiting.")
			return
		case <-c.quitCh:
			log.Debugf("consul publisher recieved quit signal, exiting.")
			return
		}
	}
}
