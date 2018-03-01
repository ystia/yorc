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
	"reflect"
	"testing"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/require"
)

func createClientServer(t *testing.T, opts *ServeOpts) (*plugin.RPCClient, *plugin.RPCServer) {
	return plugin.TestPluginRPCConn(t, getPlugins(opts))
}

func TestDefinitionsClient_GetDefinitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		want    map[string][]byte
		wantErr bool
	}{
		{"TestOneDefinition", map[string][]byte{"TestOne": []byte{0}}, false},
		{"TestThreeDefinitions", map[string][]byte{"TestOne": []byte{0}, "TestTwo": []byte{0, 1}, "TestThree": []byte{0, 1, 2}}, false},
		{"TestNoDefinitions", map[string][]byte{}, false},
		{"TestNilDefinitions", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, _ := createClientServer(t, &ServeOpts{Definitions: tt.want})
			defer c.Close()
			raw, err := c.Dispense(DefinitionsPluginName)
			require.Nil(t, err)
			def := raw.(Definitions)
			got, err := def.GetDefinitions()
			if (err != nil) != tt.wantErr {
				t.Errorf("DefinitionsClient.GetDefinitions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefinitionsClient.GetDefinitions() = %v, want %v", got, tt.want)
			}
		})
	}
}
