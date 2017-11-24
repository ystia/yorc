package tosca

import (
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v2"
	"novaforge.bull.com/starlings-janus/janus/log"
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
				if resultFn.String() != tt.inputs.yml {
					t.Errorf("Function.Unmarshal() expecting = %v, got %v", tt.inputs.yml, resultFn)
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
