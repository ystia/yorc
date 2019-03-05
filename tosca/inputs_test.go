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

package tosca

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v3/log"
)

func TestInput_UnmarshalYAML(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	data := `
ES_VERSION: { get_property: [SELF, component_version] }
nb_replicas:
  type: integer
  description: Number of replicas for indexes
  required: true
ip_address: { get_attribute: [SELF, ip_address] }
index:
  type: string
  description: The name of the index to be updated (specify no value for all indexes)
  required: false
`
	inputs := make(map[string]Input)
	err := yaml.Unmarshal([]byte(data), inputs)
	require.Nil(t, err)

	require.Len(t, inputs, 4)

	require.Contains(t, inputs, "ES_VERSION")
	i := inputs["ES_VERSION"]
	require.Nil(t, i.PropDef)
	require.NotNil(t, i.ValueAssign)

	require.Equal(t, ValueAssignmentFunction, i.ValueAssign.Type)
	require.EqualValues(t, "get_property", i.ValueAssign.GetFunction().Operator)
	require.Equal(t, "SELF", i.ValueAssign.GetFunction().Operands[0].String())
	require.Equal(t, "component_version", i.ValueAssign.GetFunction().Operands[1].String())

	i = inputs["ip_address"]
	require.Nil(t, i.PropDef)
	require.NotNil(t, i.ValueAssign)

	require.Equal(t, ValueAssignmentFunction, i.ValueAssign.Type)
	require.EqualValues(t, "get_attribute", i.ValueAssign.GetFunction().Operator)
	require.Equal(t, "SELF", i.ValueAssign.GetFunction().Operands[0].String())
	require.Equal(t, "ip_address", i.ValueAssign.GetFunction().Operands[1].String())

	i = inputs["nb_replicas"]
	require.Nil(t, i.ValueAssign)
	require.NotNil(t, i.PropDef)

	require.Equal(t, "integer", i.PropDef.Type)
	require.Equal(t, "Number of replicas for indexes", i.PropDef.Description)
	require.Equal(t, true, *i.PropDef.Required)

	i = inputs["index"]
	require.Nil(t, i.ValueAssign)
	require.NotNil(t, i.PropDef)

	require.Equal(t, "string", i.PropDef.Type)
	require.Equal(t, "The name of the index to be updated (specify no value for all indexes)", i.PropDef.Description)
	require.Equal(t, false, *i.PropDef.Required)
}
