package tosca

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestGroupedCommonsParallel(t *testing.T) {
	t.Run("groupCommons", func(t *testing.T) {
		t.Run("TestValueAssignment_String", valueAssignment_String)
		t.Run("TestValueAssignment_ReadWrite", valueAssignment_ReadWrite)
		t.Run("TestValueAssignment_Malformed1", valueAssignment_Malformed1)
		t.Run("TestValueAssignment_Malformed2", valueAssignment_Malformed2)
		t.Run("TestValueAssignment_GetInput", valueAssignment_GetInput)
		t.Run("TestValueAssignment_SlurmResult", valueAssignment_SlurmResult)
	})
}

func valueAssignment_String(t *testing.T) {
	t.Parallel()
	data := `{ concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port] ] }`
	va := ValueAssignment{}

	yaml.Unmarshal([]byte(data), &va)

	assert.Equal(t, `concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port]]`, va.String())
}

func valueAssignment_ReadWrite(t *testing.T) {
	t.Parallel()
	data := `{ concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port] ] }`
	va := ValueAssignment{}

	yaml.Unmarshal([]byte(data), &va)

	assert.Equal(t, `concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port]]`, va.String())

	va2 := ValueAssignment{}
	yaml.Unmarshal([]byte(va.String()), &va2)

	assert.Equal(t, va.String(), va2.String())
}

func valueAssignment_Malformed1(t *testing.T) {
	t.Parallel()
	data := `{ concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port] ], 2: 3 }`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	log.Printf("%v", err)
	assert.Error(t, err)
}
func valueAssignment_Malformed2(t *testing.T) {
	t.Parallel()
	data := `{ concat: ["http://", [get_attribute, get_property]: [HOST, public_ip_address], ":", get_property: [SELF, port] ]}`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	log.Printf("%v", err)
	assert.Error(t, err)
}

func valueAssignment_GetInput(t *testing.T) {
	t.Parallel()
	data := `{ get_input: port }`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	require.Nil(t, err)
	require.NotNil(t, va.Expression)
	require.Len(t, va.Expression.children, 1)
	assert.Equal(t, `get_input: [port]`, va.String())

	va2 := ValueAssignment{}
	err = yaml.Unmarshal([]byte(va.String()), &va2)
	require.Nil(t, err)
	require.NotNil(t, va2.Expression)
	require.Len(t, va2.Expression.children, 1)
}

func valueAssignment_SlurmResult(t *testing.T) {
	t.Parallel()
	data := `"Final Results: Minibatch[1-11]: errs = 0.550%"`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	require.Nil(t, err)
	require.NotNil(t, va.Expression)
	require.Equal(t, "Final Results: Minibatch[1-11]: errs = 0.550%", va.String())
}
