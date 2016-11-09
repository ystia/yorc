package tosca

import (
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"testing"
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
	require.Equal(t, false, input.Required)
	require.Equal(t, "http://10.197.132.16/sla", input.Default)

	input = topo.TopologyTemplate.Inputs["repository_value"]

	require.Equal(t, "string", input.Type)
	require.Equal(t, false, input.Required)
	require.Equal(t, "http://10.197.132.16/sla", input.Value.String())
}
