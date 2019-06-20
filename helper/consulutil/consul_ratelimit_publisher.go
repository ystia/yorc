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

package consulutil

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/ystia/yorc/v4/log"
)

// ConsulStoreTxnTimeoutEnvName is the name of the environment variable that allows to activate the feature that
// pack ConsulStore operations into transactions. If the variable is empty or not set then the process is the same
// as usual and operations are sent individually to Consul. Otherwise if set to a valid Go duration then operations
// are packed into transactions up to 64 ops and this timeout represent the time to wait for new operations before
// sending an incomplete (less than 64 ops) transaction to Consul.
const ConsulStoreTxnTimeoutEnvName = "YORC_CONSUL_STORE_TXN_TIMEOUT"

var envTimeoutDuration time.Duration

func init() {
	loadConsulStoreTxnEnv()
}

func loadConsulStoreTxnEnv() {
	envTimeoutDurationString := os.Getenv(ConsulStoreTxnTimeoutEnvName)
	if envTimeoutDurationString != "" {
		var err error
		envTimeoutDuration, err = time.ParseDuration(envTimeoutDurationString)
		if err != nil {
			log.Panicf("%v", errors.Wrapf(err, "invalid duration format for %q env var", ConsulStoreTxnTimeoutEnvName))
		}
	}
}

// Internal type used to uniquely identify the errorgroup in a context
type ctxErrGroupKey struct{}

// Internal variable used to uniquely identify the errorgroup in a context
var errGroupKey = ctxErrGroupKey{}

// rateLimitedConsulPublisher singleton
var consulPub *rateLimitedConsulPublisher

// ConsulStore allows to store keys in Consul
type ConsulStore interface {
	StoreConsulKey(key string, value []byte)
	StoreConsulKeyWithFlags(key string, value []byte, flags uint64)
	StoreConsulKeyAsString(key, value string)
	StoreConsulKeyAsStringWithFlags(key, value string, flags uint64)
}

// consulStore is the default implementation for ConsulStore.
// It allows to use a context and an errgroup.Group to store several keys in parallel in Consul but with a defined parallelism.
type consulStore struct {
	ctx              context.Context
	l                sync.Mutex
	tx               api.KVTxnOps
	close            chan struct{}
	txPackingTimeout time.Duration
}

// GetKV returns the KV associated to the consul publisher
func GetKV() *api.KV {
	return consulPub.kv
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
	return withContext(ctx)
}

func withContext(ctx context.Context) (context.Context, *errgroup.Group, *consulStore) {
	if ctx == nil {
		log.Panic(errors.New("Context can't be nil"))
	}
	errGroup, errCtx := errgroup.WithContext(ctx)
	errCtx = context.WithValue(errCtx, errGroupKey, errGroup)

	return errCtx, errGroup, &consulStore{ctx: errCtx, tx: make(api.KVTxnOps, 0, 64), close: make(chan struct{}), txPackingTimeout: envTimeoutDuration}
}

// StoreConsulKeyAsString is equivalent to StoreConsulKeyWithFlags(key, []byte(value),0)
func StoreConsulKeyAsString(key, value string) error {
	return StoreConsulKeyWithFlags(key, []byte(value), 0)
}

// StoreConsulKeyAsStringWithFlags is equivalent to StoreConsulKeyWithFlags(key, []byte(value),flags)
func StoreConsulKeyAsStringWithFlags(key, value string, flags uint64) error {
	return StoreConsulKeyWithFlags(key, []byte(value), flags)
}

// StoreConsulKey is equivalent to StoreConsulKeyWithFlags(key, []byte(value),0)
func StoreConsulKey(key string, value []byte) error {
	return StoreConsulKeyWithFlags(key, value, 0)
}

// StoreConsulKeyWithFlags stores a Consul key without the use of a ConsulStore you should avoid to use it when storing several keys that could be
// stored concurrently as this function has an important overhead of creating an execution context using the WithContext function and
// waiting for the key to be store in Consul using errGroup.Wait()
// The given flags mask is associated to the K/V (See Flags in api.KVPair).
func StoreConsulKeyWithFlags(key string, value []byte, flags uint64) error {
	_, errGroup, store := withContext(context.Background())
	// As we store only one key do not use a transaction
	store.txPackingTimeout = 0
	store.StoreConsulKeyWithFlags(key, value, flags)
	return errGroup.Wait()
}

// StoreConsulKeyAsString is equivalent to StoreConsulKeyWithFlags(key, []byte(value),0)
func (cs *consulStore) StoreConsulKeyAsString(key, value string) {
	cs.StoreConsulKeyWithFlags(key, []byte(value), 0)
}

// StoreConsulKeyAsString is equivalent to StoreConsulKeyWithFlags(key, []byte(value), flags)
func (cs *consulStore) StoreConsulKeyAsStringWithFlags(key, value string, flags uint64) {
	cs.StoreConsulKeyWithFlags(key, []byte(value), flags)
}

// StoreConsulKey is equivalent to StoreConsulKeyWithFlags(key, []byte(value), 0)
func (cs *consulStore) StoreConsulKey(key string, value []byte) {
	cs.StoreConsulKeyWithFlags(key, value, 0)
}

// StoreConsulKeyWithFlags sends a Consul key/value down to the rate limited Consul Publisher.
//
// The given flags mask is associated to the K/V (See Flags in api.KVPair).
// This function may be blocking if the rate limited queue is full. The number of maximum parallel execution is controlled with an option.
func (cs *consulStore) StoreConsulKeyWithFlags(key string, value []byte, flags uint64) {
	// PUT a new KV pair
	select {
	case <-cs.ctx.Done():
		return
	default:
	}
	if cs.txPackingTimeout == 0 {
		cs.publishWithoutTx(key, value, flags)
	} else {
		cs.publishWithinTx(key, value, flags)
	}
}

func (cs *consulStore) publishWithoutTx(key string, value []byte, flags uint64) {
	p := &api.KVPair{Key: key, Value: value, Flags: flags}
	// Will block if the rateLimitedConsulPublisher is itself blocked by its semaphore
	consulPub.publish(cs.ctx, func(kv *api.KV) error {
		_, err := kv.Put(p, nil)
		return errors.Wrap(err, ConsulGenericErrMsg)
	})
}

func executeKVTxn(kv *api.KV, ops api.KVTxnOps) error {
	_, response, _, err := kv.Txn(ops, nil)

	if err != nil {
		return err
	}
	if len(response.Errors) > 0 {
		errs := ""
		for i, e := range response.Errors {
			if i != 0 {
				errs = errs + ", "
			}
			errs = errs + e.What
		}
		return errors.New(errs)
	}
	return nil
}

func (cs *consulStore) publishTxn() {
	// Copy slice to reuse it
	ops := make(api.KVTxnOps, len(cs.tx))
	copy(ops, cs.tx)
	consulPub.publish(cs.ctx, func(kv *api.KV) error {
		return executeKVTxn(kv, ops)
	})
	// reset slice
	cs.tx = make(api.KVTxnOps, 0, 64)
	//  Release others
	close(cs.close)
	cs.close = make(chan struct{})
}

func (cs *consulStore) publishWithinTx(key string, value []byte, flags uint64) {
	cs.l.Lock()
	defer cs.l.Unlock()
	cs.tx = append(cs.tx, &api.KVTxnOp{
		Verb:  api.KVSet,
		Key:   key,
		Value: value,
		Flags: flags,
	})

	closeCh := cs.close
	errGroup := cs.ctx.Value(errGroupKey).(*errgroup.Group)
	if len(cs.tx) == 1 {
		errGroup.Go(func() error {
			select {
			case <-time.After(cs.txPackingTimeout):
				cs.l.Lock()
				defer cs.l.Unlock()
				select {
				case <-closeCh:
					// chanel was close while waiting for the lock so its published
					return nil
				default:
				}
				cs.publishTxn()
			case <-closeCh:
			}
			return nil
		})
	} else if len(cs.tx) == 64 {
		cs.publishTxn()
	} else {
		closeCh := cs.close
		errGroup.Go(func() error {
			<-closeCh
			return nil
		})
	}
}

// rateLimitedConsulPublisher is used to store Consul keys.
//
// It uses a buffered channel as semaphore to limit the number of parallel call to the Consul API
type rateLimitedConsulPublisher struct {
	sem chan struct{}
	kv  *api.KV
}

// InitConsulPublisher starts a Consul Publisher and limit the number of parallel call to the Consul API to store keys with the given maxItems.
//
// It is safe but discouraged to call several times this function. Only the last call will be taken into account. The only valid reason to
// do so is for reconfiguration and changing the limit.
func InitConsulPublisher(maxItems int, kv *api.KV) {
	log.Debugf("Consul Publisher created with a maximum of %d parallel routines.", maxItems)

	consulPub = &rateLimitedConsulPublisher{sem: make(chan struct{}, maxItems), kv: kv}

}

func (c *rateLimitedConsulPublisher) publish(ctx context.Context, f func(kv *api.KV) error) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	// will block if channel is full
	c.sem <- struct{}{}
	errGroup := ctx.Value(errGroupKey).(*errgroup.Group)
	errGroup.Go(func() error {
		defer func() {
			// Release semaphore
			<-c.sem
		}()
		return f(c.kv)
	})
}
