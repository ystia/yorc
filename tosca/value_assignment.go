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
	"fmt"
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
	Type  ValueAssignmentType
	Value interface{}
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

// GetFunction retruns the TOSCA Function of a this ValueAssignment
//
// If ValueAssignment.Type is not ValueAssignmentFunction then nil is returned
func (p ValueAssignment) GetFunction() *Function {
	if p.Type == ValueAssignmentFunction && p.Value != nil {
		return p.Value.(*Function)
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
func (p ValueAssignment) GetMap() map[interface{}]interface{} {
	if p.Type == ValueAssignmentMap && p.Value != nil {
		return p.Value.(map[interface{}]interface{})
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

// UnmarshalYAML unmarshals a yaml into a ValueAssignment
func (p *ValueAssignment) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Value assignment could be:
	//   - a List
	//   - a TOSCA function
	//   - a map or complex type
	//   - a literal value

	// First try with a List
	var l []interface{}
	if err := unmarshal(&l); err == nil {
		p.Value = l
		p.Type = ValueAssignmentList
		return nil
	}
	// Not a List try a TOSCA function
	f := &Function{}
	if err := unmarshal(f); err == nil {
		p.Value = f
		p.Type = ValueAssignmentFunction
		return nil
	}

	// Not a List nor a TOSCA function, let's try a map or complex type
	var m map[interface{}]interface{}
	if err := unmarshal(&m); err == nil {
		p.Value = m
		p.Type = ValueAssignmentMap
		return nil
	}

	// Not a List nor a TOSCA function, nor a map or complex type, let's try literal
	var s interface{}
	if err := unmarshal(&s); err != nil {
		return err
	}

	p.Value = s
	p.Type = ValueAssignmentLiteral
	return nil
}
