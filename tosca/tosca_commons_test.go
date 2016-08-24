package tosca

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"log"
	"testing"
)

func TestGroupedCommonsParallel(t *testing.T)  {
	t.Run("groupCommons", func(t *testing.T) {
		t.Run("TestValueAssignment_String", valueAssignment_String)
		t.Run("TestValueAssignment_ReadWrite", valueAssignment_ReadWrite)
		t.Run("TestValueAssignment_Malformed1", valueAssignment_Malformed1)
		t.Run("TestValueAssignment_Malformed2", valueAssignment_Malformed2)
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
