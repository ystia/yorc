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

package registry

import "github.com/pkg/errors"

// Definition represents a TOSCA definition with its Name, Origin and Data content
//
// Deprecated: use store.Definition instead
//             will be removed in Yorc 4.0
type Definition struct {
	Name   string `json:"name"`
	Origin string `json:"origin"`
	Data   []byte `json:"-"`
}

// Register a TOSCA definition file. Origin is the origin of the executor (builtin for builtin executors or the plugin name in case of a plugin)
//
// Deprecated: use store.CommonDefinition instead
//             will be removed in Yorc 4.0
func (r *defaultRegistry) AddToscaDefinition(name, origin string, data []byte) {
	r.definitionsLock.Lock()
	defer r.definitionsLock.Unlock()
	// Insert as first
	r.definitions = append([]Definition{Definition{name, origin, data}}, r.definitions...)
}

// GetToscaDefinition retruns the definitions for the given name.
//
// If the given definition name can't match any definition an error is returned
//
// Deprecated: use store.GetCommonsTypesPaths instead to know supported definitions
//             will be removed in Yorc 4.0
func (r *defaultRegistry) GetToscaDefinition(name string) ([]byte, error) {
	r.definitionsLock.RLock()
	defer r.definitionsLock.RUnlock()
	for _, def := range r.definitions {
		if def.Name == name {
			return def.Data, nil
		}
	}
	return nil, errors.Errorf("Unknown definition: %q", name)
}

// ListToscaDefinitions returns a map of definitions names to their origin
//
// Deprecated: use store.GetCommonsTypesPaths instead to know supported definitions
//             will be removed in Yorc 4.0
func (r *defaultRegistry) ListToscaDefinitions() []Definition {
	r.definitionsLock.RLock()
	defer r.definitionsLock.RUnlock()
	result := make([]Definition, len(r.definitions))
	copy(result, r.definitions)
	return result
}
