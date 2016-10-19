package tosca

import (
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"novaforge.bull.com/starlings-janus/janus/log"
	"testing"
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

	require.Equal(t, "get_property", i.ValueAssign.Expression.Value)
	require.Equal(t, "SELF", i.ValueAssign.Expression.Children()[0].Value)
	require.Equal(t, "component_version", i.ValueAssign.Expression.Children()[1].Value)

	i = inputs["ip_address"]
	require.Nil(t, i.PropDef)
	require.NotNil(t, i.ValueAssign)

	require.Equal(t, "get_attribute", i.ValueAssign.Expression.Value)
	require.Equal(t, "SELF", i.ValueAssign.Expression.Children()[0].Value)
	require.Equal(t, "ip_address", i.ValueAssign.Expression.Children()[1].Value)

	i = inputs["nb_replicas"]
	require.Nil(t, i.ValueAssign)
	require.NotNil(t, i.PropDef)

	require.Equal(t, "integer", i.PropDef.Type)
	require.Equal(t, "Number of replicas for indexes", i.PropDef.Description)
	require.Equal(t, true, i.PropDef.Required)

	i = inputs["index"]
	require.Nil(t, i.ValueAssign)
	require.NotNil(t, i.PropDef)

	require.Equal(t, "string", i.PropDef.Type)
	require.Equal(t, "The name of the index to be updated (specify no value for all indexes)", i.PropDef.Description)
	require.Equal(t, false, i.PropDef.Required)
}
