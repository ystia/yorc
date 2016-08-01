package tosca

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"log"
	"testing"
)

func TestValueAssignment_String(t *testing.T) {
	data := `{ concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port] ] }`
	va := ValueAssignment{}

	yaml.Unmarshal([]byte(data), &va)

	assert.Equal(t, `concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port]]`, va.String())
}

func TestValueAssignment_ReadWrite(t *testing.T) {
	data := `{ concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port] ] }`
	va := ValueAssignment{}

	yaml.Unmarshal([]byte(data), &va)

	assert.Equal(t, `concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port]]`, va.String())

	va2 := ValueAssignment{}
	yaml.Unmarshal([]byte(va.String()), &va2)

	assert.Equal(t, va.String(), va2.String())
}

func TestValueAssignment_Malformed1(t *testing.T) {
	data := `{ concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port] ], 2: 3 }`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	log.Printf("%v", err)
	assert.Error(t, err)
}
func TestValueAssignment_Malformed2(t *testing.T) {
	data := `{ concat: ["http://", [get_attribute, get_property]: [HOST, public_ip_address], ":", get_property: [SELF, port] ]}`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	log.Printf("%v", err)
	assert.Error(t, err)
}
