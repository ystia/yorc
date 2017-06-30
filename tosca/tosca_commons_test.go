package tosca

import (
	"log"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestGroupedCommonsParallel(t *testing.T) {
	t.Run("groupCommons", func(t *testing.T) {
		t.Run("TestvalueAssignmentString", valueAssignmentString)
		t.Run("TestvalueAssignmentReadWrite", valueAssignmentReadWrite)
		t.Run("TestvalueAssignmentMalformed1", valueAssignmentMalformed1)
		t.Run("TestvalueAssignmentMalformed2", valueAssignmentMalformed2)
		t.Run("TestvalueAssignmentGetInput", valueAssignmentGetInput)
		t.Run("TestvalueAssignmentSlurmResult", valueAssignmentSlurmResult)
	})
}

func valueAssignmentString(t *testing.T) {
	t.Parallel()
	data := `{ concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port] ] }`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	require.Nil(t, err)

	require.Equal(t, `concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port]]`, va.String())
}

func valueAssignmentReadWrite(t *testing.T) {
	t.Parallel()
	data := `{ concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port] ] }`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	require.Nil(t, err)
	require.Equal(t, `concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port]]`, va.String())

	va2 := ValueAssignment{}
	err = yaml.Unmarshal([]byte(va.String()), &va2)
	require.Nil(t, err)
	require.Equal(t, va.String(), va2.String())
}

func valueAssignmentMalformed1(t *testing.T) {
	t.Parallel()
	data := `{ concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port] ], 2: 3 }`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	log.Printf("%v", err)
	require.Error(t, err)
}
func valueAssignmentMalformed2(t *testing.T) {
	t.Parallel()
	data := `{ concat: ["http://", [get_attribute, get_property]: [HOST, public_ip_address], ":", get_property: [SELF, port] ]}`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	log.Printf("%v", err)
	require.Error(t, err)
}

func valueAssignmentGetInput(t *testing.T) {
	t.Parallel()
	data := `{ get_input: port }`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	require.Nil(t, err)
	require.NotNil(t, va.Expression)
	require.Len(t, va.Expression.children, 1)
	require.Equal(t, `get_input: [port]`, va.String())

	va2 := ValueAssignment{}
	err = yaml.Unmarshal([]byte(va.String()), &va2)
	require.Nil(t, err)
	require.NotNil(t, va2.Expression)
	require.Len(t, va2.Expression.children, 1)
}

func valueAssignmentSlurmResult(t *testing.T) {
	t.Parallel()
	data := `"Final Results: \"Minibatch[1-11]\": errs = 0.550%"`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	require.Nil(t, err)
	require.NotNil(t, va.Expression)
	require.Equal(t, `"Final Results: \"Minibatch[1-11]\": errs = 0.550%"`, va.String())
}

func TestValueAssignmentStringWithQuote(t *testing.T) {
	t.Parallel()
	data := `{ concat: ["Hello:", "\"World\"", "!", "!" ] }`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	require.Nil(t, err)

	require.Equal(t, `concat: ["Hello:", "\"World\"", "!", "!"]`, va.String())
}
