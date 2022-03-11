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
	"reflect"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v4/log"
)

func TestFunctionParsing(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)
	type inputs struct {
		yml string
	}

	tests := []struct {
		name    string
		inputs  inputs
		wantErr bool
	}{
		{"TestLiteralNotFunction", inputs{yml: "myLit"}, true},
		{"TestUnknownOperatorNotFunction", inputs{yml: "myLit: [test]"}, true},
		{"TestGetPropertyFunction", inputs{yml: "get_property: [SELF, ip_address]"}, false},
		{"TestConcatFunction", inputs{yml: "concat: [get_property: [SELF, ip_address], get_attribute: [SELF, port]]"}, false},
		{"TestGetInputFunction", inputs{yml: "get_input: ip_address"}, false},
		{"TestConcatFunctionQuoting", inputs{yml: `concat: ["http://", get_property: [SELF, ip_address], get_attribute: [SELF, port], "\"ff\""]`}, false},
		{"TestComplexNestedFunctions", inputs{yml: `get_secret: [concat: [/secrets/data/credentials/, get_input: user_name]]`}, false},
		{"TestComplexNestedFunctions2", inputs{yml: `get_vault_secret: [concat: [/secrets/data/credentials/, get_input: user_name]]`}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultFn := &Function{}
			err := yaml.Unmarshal([]byte(tt.inputs.yml), resultFn)
			if (err != nil) != tt.wantErr {
				t.Errorf("Function.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				expecting := tt.inputs.yml
				// get_vault_secret is an alias to get_secret
				expecting = strings.Replace(expecting, "get_vault_secret", "get_secret", -1)
				if resultFn.String() != expecting {
					t.Errorf("Function.Unmarshal() expecting = %v, got %v", expecting, resultFn)
				}
			}
		})
	}
}

func TestOperandsImplementation(t *testing.T) {
	t.Parallel()
	// Checks that LiteralOperand and Function implement the Operand interface
	var _ Operand = (*LiteralOperand)(nil)
	var _ Operand = (*Function)(nil)
}

func generateFunctionFromYaml(t testing.TB, yml string) *Function {
	f := new(Function)
	err := yaml.Unmarshal([]byte(yml), f)
	if err != nil {
		t.Fatalf("Invalid YAML in test: %v", err)
	}
	return f
}

func TestFunction_GetFunctionsByOperator(t *testing.T) {

	type args struct {
		o Operator
	}
	tests := []struct {
		name     string
		function *Function
		args     args
		want     []*Function
	}{
		{"1stLevel", generateFunctionFromYaml(t, `{get_property: [SELF, port]}`), args{GetPropertyOperator}, []*Function{generateFunctionFromYaml(t, `{get_property: [SELF, port]}`)}},
		{"nestedLevel", generateFunctionFromYaml(t, `{concat: [get_property: [SELF, port]]}`), args{GetPropertyOperator}, []*Function{generateFunctionFromYaml(t, `{get_property: [SELF, port]}`)}},
		{"severalNestedLevel", generateFunctionFromYaml(t, `{concat: [get_property: [SELF, port], concat: [get_input: "i", get_property: [SELF, test]]]}`), args{GetPropertyOperator}, []*Function{generateFunctionFromYaml(t, `{get_property: [SELF, port]}`), generateFunctionFromYaml(t, `{get_property: [SELF, test]}`)}},
		{"notFound", generateFunctionFromYaml(t, `{concat: [get_property: [SELF, port], concat: [get_input: "i", get_property: [SELF, test]]]}`), args{GetAttributeOperator}, []*Function{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if got := tt.function.GetFunctionsByOperator(tt.args.o); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Function.GetFunctionsByOperator() = %v, want %v", got, tt.want)
			}
		})
	}
}
