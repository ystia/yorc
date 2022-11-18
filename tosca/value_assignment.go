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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ystia/yorc/v4/log"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// ValueAssignmentType defines the type of value for an assignment
type ValueAssignmentType uint64

const (
	// ValueAssignmentLiteral defines an assignment of a literal
	ValueAssignmentLiteral ValueAssignmentType = iota
	// ValueAssignmentFunction defines an assignment of a TOSCA function
	ValueAssignmentFunction
	// ValueAssignmentList defines an assignment of a list
	ValueAssignmentList
	// ValueAssignmentMap defines an assignment of a map or a complex type
	ValueAssignmentMap
)

func (vat ValueAssignmentType) String() string {
	switch vat {
	case ValueAssignmentLiteral:
		return "literal"
	case ValueAssignmentList:
		return "list"
	case ValueAssignmentFunction:
		return "function"
	case ValueAssignmentMap:
		return "map"
	}
	return "unsupported"
}

// UnmarshalJSON unmarshals json into a ValueAssignmentType
func (vat *ValueAssignmentType) UnmarshalJSON(b []byte) error {

	var i int
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}

	var v ValueAssignmentType
	switch i {
	case 0:
		v = ValueAssignmentLiteral
	case 1:
		v = ValueAssignmentFunction
	case 2:
		v = ValueAssignmentList
	case 3:
		v = ValueAssignmentMap
	default:
		return errors.Errorf("unknown value of ValueAssignmentType:%d", i)
	}
	*vat = v
	return nil
}

// ValueAssignmentTypeFromString converts a textual representation of a ValueAssignmentType into its value
func ValueAssignmentTypeFromString(s string) (ValueAssignmentType, error) {
	s = strings.ToLower(s)
	switch s {
	case "literal":
		return ValueAssignmentLiteral, nil
	case "list":
		return ValueAssignmentList, nil
	case "function":
		return ValueAssignmentFunction, nil
	case "map":
		return ValueAssignmentMap, nil
	default:
		return ValueAssignmentLiteral, errors.Errorf("Unknown type for value assignment: %q", s)
	}
}

// An ValueAssignment is the representation of a TOSCA Value Assignment
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_PROPERTY_VALUE_ASSIGNMENT and
// http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_ATTRIBUTE_VALUE_ASSIGNMENT for more details
type ValueAssignment struct {
	Type  ValueAssignmentType `json:"type"`
	Value interface{}         `json:"value,omitempty"`
}

// GetLiteral retruns the string representation of a literal value
//
// If ValueAssignment.Type is not ValueAssignmentLiteral then an empty string is returned
func (p ValueAssignment) GetLiteral() string {
	if p.Type == ValueAssignmentLiteral && p.Value != nil {
		return fmt.Sprint(p.Value)
	}
	return ""
}

// GetFunction returns the TOSCA Function of a this ValueAssignment
//
// If ValueAssignment.Type is not ValueAssignmentFunction then nil is returned
func (p ValueAssignment) GetFunction() *Function {
	if p.Type == ValueAssignmentFunction && p.Value != nil {
		f, err := ParseFunction(p.String())
		if err != nil {
			log.Printf("err:%+v", err)
			return nil
		}
		return f
	}
	return nil

}

// GetList retruns the list associated with this ValueAssignment
//
// If ValueAssignment.Type is not ValueAssignmentList then nil is returned
func (p ValueAssignment) GetList() []interface{} {
	if p.Type == ValueAssignmentList && p.Value != nil {
		return p.Value.([]interface{})
	}
	return nil
}

// GetMap retruns the map associated with this ValueAssignment
//
// If ValueAssignment.Type is not ValueAssignmentMap then nil is returned
func (p ValueAssignment) GetMap() map[string]interface{} {
	if p.Type == ValueAssignmentMap && p.Value != nil {
		return p.Value.(map[string]interface{})
	}
	return nil
}

// String retruns the textual representation of a ValueAssignment
func (p ValueAssignment) String() string {

	switch p.Type {
	case ValueAssignmentLiteral, ValueAssignmentFunction:
		if p.Value == nil {
			return ""
		}
		r := fmt.Sprint(p.Value)
		if p.Type == ValueAssignmentLiteral && shouldQuoteYamlString(r) {
			return strconv.Quote(r)
		}
		return r
	case ValueAssignmentList:
		var b bytes.Buffer
		b.WriteString("[")
		for i, v := range p.GetList() {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(strconv.Quote(fmt.Sprint(v)))
		}
		b.WriteString("]")
		return b.String()
	case ValueAssignmentMap:
		var b bytes.Buffer
		b.WriteString("{")
		i := 0
		for k, v := range p.GetMap() {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(strconv.Quote(fmt.Sprint(k)))
			b.WriteString(": ")
			b.WriteString(strconv.Quote(fmt.Sprint(v)))
			i++
		}
		b.WriteString("}")
		return b.String()
	default:
		return ""
	}
}

// ToValueAssignment builds a ValueAssignment from a value by deducing its type
func ToValueAssignment(value interface{}) (*ValueAssignment, error) {
	va := ValueAssignment{}
	switch v := value.(type) {
	case string:
		// Check if it's a function
		f, err := ParseFunction(v)
		if err == nil {
			va.Value = f
			va.Type = ValueAssignmentFunction
			break
		}
		va.Value = v
		va.Type = ValueAssignmentLiteral
	case []interface{}:
		va.Value = v
		va.Type = ValueAssignmentList
	case map[string]interface{}:
		va.Value = v
		va.Type = ValueAssignmentMap
	}

	return &va, nil
}

// UnmarshalYAML unmarshals a yaml into a ValueAssignment
func (p *ValueAssignment) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := p.unmarshalYAML(unmarshal); err == nil {
		return nil
	}

	// Not a List nor a TOSCA function, nor a map or complex type, let's try literal
	// For YAML parsing literals are always considered as string
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	p.Value = s
	p.Type = ValueAssignmentLiteral
	return nil

}

// unmarshalYAML unmarshals yaml into a ValueAssignment
func (p *ValueAssignment) unmarshalYAML(unmarshal func(interface{}) error) error {
	// Value assignment could be:
	//   - a List
	//   - a TOSCA function
	//   - a map or complex type
	//   - a literal value

	// First try with a List
	var l []interface{}
	if err := unmarshal(&l); err == nil {
		l := cleanUpInterfaceArray(l)
		p.Value = l
		p.Type = ValueAssignmentList
		return nil
	}
	// Not a List try a TOSCA function
	f := &Function{}
	if err := unmarshal(f); err == nil && IsOperator(string(f.Operator)) {
		// function are stored in string representation
		p.Value = f.String()
		p.Type = ValueAssignmentFunction
		return nil
	}

	// Not a List nor a TOSCA function, let's try a map or complex type
	var tmp map[interface{}]interface{}
	if err := unmarshal(&tmp); err == nil {
		// for nested map, we need to cast map[interface{}]interface{} to map[string]interface{} as JSON doesn't accept map[interface{}]interface{}
		m := cleanUpInterfaceMap(tmp)
		p.Value = m
		p.Type = ValueAssignmentMap
		return nil
	}

	// JSON map unmarshaling expects a map[string]interface{}
	var strMap map[string]interface{}
	if err := unmarshal(&strMap); err != nil {
		return err
	}

	p.Value = strMap
	p.Type = ValueAssignmentMap
	return nil
}

func cleanUpInterfaceArray(in []interface{}) []interface{} {
	result := make([]interface{}, len(in))
	for i, v := range in {
		result[i] = cleanUpMapValue(v)
	}
	return result
}

func cleanUpInterfaceMap(in map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range in {
		result[fmt.Sprintf("%v", k)] = cleanUpMapValue(v)
	}
	return result
}

func cleanUpMapValue(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		return cleanUpInterfaceArray(v)
	case map[interface{}]interface{}:
		return cleanUpInterfaceMap(v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
