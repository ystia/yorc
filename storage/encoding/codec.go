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

package encoding

// Codec encodes/decodes Go values to/from slices of bytes.
type Codec interface {
	// Marshal encodes a Go value to a slice of bytes.
	Marshal(v interface{}) ([]byte, error)
	// Unmarshal decodes a slice of bytes into a Go value.
	Unmarshal(data []byte, v interface{}) error
}

// Convenience variables
var (
	// JSON is a JSONcodec that encodes/decodes Go values to/from JSON.
	JSON = JSONcodec{}
	// Gob is a GobCodec that encodes/decodes Go values to/from gob.
	Gob = GobCodec{}
)
