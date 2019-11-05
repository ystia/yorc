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

package internal

import (
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/storage/types"

	"github.com/ystia/yorc/v4/storage/encoding"
)

type consulStore struct {
	codec encoding.Codec
}

func NewStore() *consulStore {
	return &consulStore{encoding.JSON}
}

func (c *consulStore) Set(k string, v interface{}) error {
	if err := CheckKeyAndValue(k, v); err != nil {
		return err
	}

	data, err := c.codec.Marshal(v)
	if err != nil {
		return err
	}

	return consulutil.StoreConsulKey(k, data)
}

func (c *consulStore) Get(k string, v interface{}) (bool, error) {
	if err := CheckKeyAndValue(k, v); err != nil {
		return false, err
	}

	found, value, err := consulutil.GetValue(k)
	if err != nil || !found {
		return found, err
	}

	return true, c.codec.Unmarshal(value, v)
}

func (c *consulStore) List(k string) ([]string, error) {
	m, err := consulutil.List(k)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0)
	for k := range m {
		keys = append(keys, k)
	}
	return keys, nil
}

func (c *consulStore) Delete(k string, recursive bool) error {
	return consulutil.Delete(k, recursive)
}

func (c *consulStore) Types() []types.StoreType {
	return nil
}
