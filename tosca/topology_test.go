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
)

func TestTopologyTemplate_Inputs(t *testing.T) {
	data := `
name: topo test
topology_template:
  inputs:
    repository_default:
      type: string
      required: false
      default: "http://10.197.132.16/sla"
    repository_value:
      type: string
      required: false
      value: http://10.197.132.16/sla
`
	topo := Topology{}

	err := yaml.Unmarshal([]byte(data), &topo)
	require.Nil(t, err)
	require.NotNil(t, topo.TopologyTemplate)
	require.Contains(t, topo.TopologyTemplate.Inputs, "repository_default")
	require.Contains(t, topo.TopologyTemplate.Inputs, "repository_value")

	input := topo.TopologyTemplate.Inputs["repository_default"]

	require.Equal(t, "string", input.Type)
	require.Equal(t, false, *input.Required)
	require.Equal(t, `http://10.197.132.16/sla`, input.Default.GetLiteral())

	input = topo.TopologyTemplate.Inputs["repository_value"]

	require.Equal(t, "string", input.Type)
	require.Equal(t, false, *input.Required)
	require.Equal(t, `http://10.197.132.16/sla`, input.Value.GetLiteral())
}
