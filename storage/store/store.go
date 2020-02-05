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

package store

import (
	"context"
	"time"
)

// Store is an abstraction for different key-value store implementations.
// A store must be able to store, retrieve and delete key-value pairs,
// with the key being a string and the value being any Go interface{}.
// inspired by https://github.com/philippgille/gokv
type Store interface {
	// Set stores the given value for the given key.
	// The implementation automatically marshalls the value.
	// The marshalling format depends on the implementation. It can be JSON, gob etc.
	// The key must not be "" and the value must not be nil.
	Set(ctx context.Context, k string, v interface{}) error
	// set a collection of key-values
	// The implementation automatically marshalls the value.
	// The marshalling format depends on the implementation. It can be JSON, gob etc.
	// It's the implementation concern to define storage mode (ie concurrently or serial)
	// ctx is the context provided for eventually cancelling a storage action
	SetCollection(ctx context.Context, keyValues []KeyValueIn) error
	// Get retrieves the value for the given key.
	// The implementation automatically unmarshalls the value.
	// The unmarshalling source depends on the implementation. It can be JSON, gob etc.
	// The automatic unmarshalling requires a pointer to an object of the correct type
	// being passed as parameter.
	// If no value is found it returns (false, nil).
	// The key must not be "" and the pointer must not be nil.
	Get(k string, v interface{}) (bool, error)
	// Exist returns true if the key exists in the store
	// If no value is found it returns (false, nil).
	// The key must not be "" and the pointer must not be nil.
	Exist(k string) (bool, error)
	// Keys returns all the sub-keys of a specified one.
	// The key must not be "".
	// If no sub-key is found, it returns an empty slice.
	// This is not recursive
	Keys(k string) ([]string, error)
	// Delete deletes the stored value for the given key.
	// Deleting a non-existing key-value pair does NOT lead to an error.
	// The key must not be "".
	// If recursive is true, all sub-keys are deleted too.
	Delete(ctx context.Context, k string, recursive bool) error
	// GetLastModifyIndex returns the last index that modified the key k
	GetLastModifyIndex(k string) (uint64, error)
	// List allows to lookup all sub-keys recursively under the defined key k and provided associated values.
	// The key must not be "" and v must not be nil.
	// waitIndex is used to enable a blocking query (waitIndex != 0) which wait until the timeout or a new index > waitIndex
	// if waitIndex is > 0, default timeout is 5 minutes. Max timeout allowed is 10 minutes
	// ctx can be useful to cancel a blocking query
	// The values are retrieved in a KeyValueOut collection.
	// It allows to return values in raw format without decode and in decoded generic format (map[string]interface{}
	// The lastIndex is returned to perform new blocking query.
	List(ctx context.Context, k string, waitIndex uint64, timeout time.Duration) ([]KeyValueOut, uint64, error)
}
