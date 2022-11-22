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

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v4/log"
)

// Operator is the keyword of a given TOSCA operation
type Operator string

const (
	// GetPropertyOperator is the Operator of the get_property function
	GetPropertyOperator Operator = "get_property"
	// GetAttributeOperator is the Operator of the get_attribute function
	GetAttributeOperator Operator = "get_attribute"
	// GetInputOperator is the Operator of the get_input function
	GetInputOperator Operator = "get_input"
	// GetOperationOutputOperator is the Operator of the get_operation_output function
	GetOperationOutputOperator Operator = "get_operation_output"
	// ConcatOperator is the Operator of the concat function
	ConcatOperator Operator = "concat"
	// TokenOperator     Operator = "token"
	// GetNodesOfTypeOperator     Operator = "get_nodes_of_type"
	// GetArtifactOperator        Operator = "get_artifact"

	// GetSecretOperator is the Operator of the get_secret function (non-normative)
	GetSecretOperator Operator = "get_secret"
)

const getVaultSecretOperator = "get_vault_secret"
const getInputWithNestedFunctionsOperator = "get_input_nf"

// IsOperator checks if a given token is a known TOSCA function keyword
func IsOperator(op string) bool {
	return op == string(GetPropertyOperator) ||
		op == string(GetAttributeOperator) ||
		op == string(GetInputOperator) ||
		op == string(GetOperationOutputOperator) ||
		op == string(ConcatOperator) ||
		op == string(GetSecretOperator) ||
		op == getVaultSecretOperator ||
		op == getInputWithNestedFunctionsOperator
}

func parseOperator(op string) (Operator, error) {
	switch {
	case op == string(GetPropertyOperator):
		return GetPropertyOperator, nil
	case op == string(GetAttributeOperator):
		return GetAttributeOperator, nil
	case op == string(GetInputOperator), op == getInputWithNestedFunctionsOperator:
		return GetInputOperator, nil
	case op == string(GetOperationOutputOperator):
		return GetOperationOutputOperator, nil
	case op == string(ConcatOperator):
		return ConcatOperator, nil
	case op == string(GetSecretOperator), op == getVaultSecretOperator:
		return GetSecretOperator, nil
	default:
		return GetPropertyOperator, errors.Errorf("%q is not a known or supported TOSCA operator", op)

	}
}

// Operand represents the parameters part of a TOSCA function it could be a LiteralOperand or a Function
type Operand interface {
	fmt.Stringer
	// IsLiteral allows to know if an Operand is a LiteralOperand (true) or a TOSCA Function (false)
	IsLiteral() bool
}

// LiteralOperand represents a literal in a TOSCA function
type LiteralOperand string

// IsLiteral allows to know if an Operand is a LiteralOperand (true) or a TOSCA Function (false)
func (l LiteralOperand) IsLiteral() bool {
	return true
}

func (l LiteralOperand) String() string {
	s := string(l)
	if shouldQuoteYamlString(s) {
		// Quote String if it contains YAML special chars
		s = strconv.Quote(s)
	}
	return s
}

// Function models a TOSCA Function
//
// A Function is composed by an Operator and a list of Operand
type Function struct {
	Operator Operator
	Operands []Operand
}

// IsLiteral allows to know if an Operand is a LiteralOperand (true) or a TOSCA Function (false)
func (f Function) IsLiteral() bool {
	return false
}

func (f Function) String() string {
	var b bytes.Buffer
	b.WriteString(string(f.Operator))
	b.WriteString(": ")
	if len(f.Operands) == 1 && f.Operands[0].IsLiteral() {
		// Shortcut
		b.WriteString(f.Operands[0].String())
		return b.String()
	}
	b.WriteString("[")
	for i := range f.Operands {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(f.Operands[i].String())
	}
	b.WriteString("]")
	return b.String()
}

// GetFunctionsByOperator returns the list of functions with a given Operator type contained in this Function
//
// Note that Functions can be nested like in a concat for instance
func (f *Function) GetFunctionsByOperator(o Operator) []*Function {
	result := make([]*Function, 0)
	if f.Operator == o {
		result = append(result, f)
	}
	for _, op := range f.Operands {
		if !op.IsLiteral() {
			result = append(result, op.(*Function).GetFunctionsByOperator(o)...)
		}
	}
	return result
}

// UnmarshalYAML unmarshal a yaml into a Function
func (f *Function) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var m map[interface{}]interface{}
	if err := unmarshal(&m); err != nil {
		return err
	}
	if len(m) != 1 {
		return errors.Errorf("Not a TOSCA function")
	}

	fn, err := parseFunction(m)
	if err != nil {
		return err
	}
	f.Operator = fn.Operator
	f.Operands = fn.Operands
	return nil
}

func parseFunctionOperands(value interface{}) ([]Operand, error) {
	var ops []Operand
	switch v := value.(type) {
	case []interface{}:
		log.Debugf("Found array value %v %T", v, v)
		ops = make([]Operand, len(v))
		for i, op := range v {
			o, err := parseFunctionOperands(op)
			if err != nil {
				return nil, err
			}
			ops[i] = o[0]
		}
	case map[interface{}]interface{}:
		log.Debugf("Found map value %v %T", v, v)
		f, err := parseFunction(v)
		if err != nil {
			return nil, err
		}
		ops = make([]Operand, 1)
		ops[0] = f
	default:
		log.Debugf("Found a literal value %v %T", v, v)
		ops = make([]Operand, 1)
		ops[0] = LiteralOperand(fmt.Sprint(v))
	}
	return ops, nil
}

func parseFunction(f map[interface{}]interface{}) (*Function, error) {
	log.Debugf("parsing TOSCA function %+v", f)
	for k, v := range f {
		operator, err := parseOperator(fmt.Sprint(k))
		if err != nil {
			return nil, errors.Wrap(err, "Not a TOSCA function")
		}
		ops, err := parseFunctionOperands(v)
		if err != nil {
			return nil, err
		}

		return &Function{Operator: operator, Operands: ops}, nil
	}
	return nil, errors.Errorf("Not a TOSCA function")
}

// ParseFunction allows to cast a string function representation in Function struct
func ParseFunction(rawFunction string) (*Function, error) {
	f := &Function{}
	err := yaml.Unmarshal([]byte(rawFunction), f)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to parse TOSCA function %q", rawFunction)
	}
	return f, nil
}
