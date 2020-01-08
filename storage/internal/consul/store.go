// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package consul

import (
	"context"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/storage/encoding"
	"github.com/ystia/yorc/v4/storage/store"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/storage/utils"
	"reflect"
	"time"
)

type consulStore struct {
	codec encoding.Codec
}

// NewStore returns a new Consul store
func NewStore() store.Store {
	return &consulStore{encoding.JSON}
}

func (c *consulStore) Set(ctx context.Context, k string, v interface{}) error {
	if err := utils.CheckKeyAndValue(k, v); err != nil {
		return err
	}

	data, err := c.codec.Marshal(v)
	if err != nil {
		return err
	}

	return consulutil.StoreConsulKey(k, data)
}

func (c *consulStore) SetCollection(ctx context.Context, keyValues []*types.KeyValue) error {
	if keyValues == nil || len(keyValues) == 0 {
		return nil
	}
	ctx, errGroup, consulStore := consulutil.WithContext(ctx)
	for _, kv := range keyValues {
		if err := utils.CheckKeyAndValue(kv.Key, kv.Value); err != nil {
			return err
		}

		data, err := c.codec.Marshal(kv.Value)
		if err != nil {
			return err
		}

		consulStore.StoreConsulKey(kv.Key, data)
	}
	return errGroup.Wait()
}

func (c *consulStore) Get(k string, v interface{}) (bool, error) {
	if err := utils.CheckKeyAndValue(k, v); err != nil {
		return false, err
	}

	found, value, err := consulutil.GetValue(k)
	if err != nil || !found {
		return found, err
	}

	return true, c.codec.Unmarshal(value, v)
}

func (c *consulStore) Exist(k string) (bool, error) {
	if err := utils.CheckKey(k); err != nil {
		return false, err
	}

	found, _, err := consulutil.GetValue(k)
	if err != nil {
		return false, err
	}
	return found, nil
}

func (c *consulStore) Keys(k string) ([]string, error) {
	return consulutil.GetKeys(k)
}

func (c *consulStore) Delete(ctx context.Context, k string, recursive bool) error {
	return consulutil.Delete(k, recursive)
}

func (c *consulStore) GetLastIndex(k string) (uint64, error) {
	kvp, qm, err := consulutil.GetKV().Get(k, nil)
	if err != nil || qm == nil {
		return 0, errors.Errorf("Failed to retrieve last index for key:%q", k)
	}
	// must be useful that lastIndex is 0 when key doesn't exist
	if kvp == nil {
		return 0, nil
	}
	return qm.LastIndex, nil
}

func (c *consulStore) List(k string, v interface{}, waitIndex uint64, timeout time.Duration) ([]types.KeyValue, uint64, error) {
	if err := utils.CheckKey(k); err != nil {
		return nil, 0, err
	}

	kvps, qm, err := consulutil.GetKV().List(k, &api.QueryOptions{WaitIndex: waitIndex, WaitTime: timeout})
	if err != nil || qm == nil {
		return nil, 0, err
	}

	values := make([]types.KeyValue, 0)
	for _, kvp := range kvps {
		value := reflect.New(reflect.TypeOf(v)).Interface()
		if err := c.codec.Unmarshal(kvp.Value, &value); err != nil {
			return nil, 0, errors.Wrapf(err, "failed to unmarshal stored value into %q instance", reflect.TypeOf(v))
		}

		values = append(values, types.KeyValue{
			Key:       kvp.Key,
			LastIndex: kvp.ModifyIndex,
			Value:     value,
		})
	}
	return values, qm.LastIndex, nil
}
