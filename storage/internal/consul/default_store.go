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
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/storage/encoding"
	"github.com/ystia/yorc/v4/storage/internal"
	"github.com/ystia/yorc/v4/storage/types"
)

type consulStore struct {
	codec encoding.Codec
}

// NewStore returns a new Consul store
func NewStore() *consulStore {
	return &consulStore{encoding.JSON}
}

func (c *consulStore) Set(ctx context.Context, k string, v interface{}) error {
	if err := internal.CheckKeyAndValue(k, v); err != nil {
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
		if err := internal.CheckKeyAndValue(kv.Key, kv.Value); err != nil {
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
	if err := internal.CheckKeyAndValue(k, v); err != nil {
		return false, err
	}

	found, value, err := consulutil.GetValue(k)
	if err != nil || !found {
		return found, err
	}

	return true, c.codec.Unmarshal(value, v)
}

func (c *consulStore) Exist(k string) (bool, error) {
	if err := internal.CheckKey(k); err != nil {
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

func (c *consulStore) Types() []types.StoreType {
	return nil
}
