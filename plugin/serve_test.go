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
	"testing"

	"github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/require"
)

func TestServeDefaultOpts(t *testing.T) {
	t.Parallel()
	client, _ := plugin.TestPluginRPCConn(t, getPlugins(nil), nil)
	defer client.Close()

	raw, err := client.Dispense(OperationPluginName)
	require.Nil(t, err)

	opPlugin := raw.(OperationExecutor)
	arts, err := opPlugin.GetSupportedArtifactTypes()
	require.Nil(t, err)
	require.Len(t, arts, 0)

	raw, err = client.Dispense(DelegatePluginName)
	require.Nil(t, err)

	delPlugin := raw.(DelegateExecutor)
	types, err := delPlugin.GetSupportedTypes()
	require.Nil(t, err)
	require.Len(t, types, 0)

	raw, err = client.Dispense(DefinitionsPluginName)
	require.Nil(t, err)

	defPlugin := raw.(Definitions)
	defs, err := defPlugin.GetDefinitions()
	require.Nil(t, err)
	require.Len(t, defs, 0)

}
