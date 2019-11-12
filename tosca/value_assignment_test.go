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
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v4/log"
)

func TestValueAssignment_GetLiteral(t *testing.T) {
	t.Parallel()
	type fields struct {
		Type  ValueAssignmentType
		Value interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"TestStringLiteral", fields{ValueAssignmentLiteral, "hello"}, "hello"},
		{"TestIntLiteral", fields{ValueAssignmentLiteral, 10}, "10"},
		{"TestBoolTLiteral", fields{ValueAssignmentLiteral, true}, "true"},
		{"TestBoolFLiteral", fields{ValueAssignmentLiteral, false}, "false"},
		{"TestList", fields{ValueAssignmentList, []string{"1", "2", "3"}}, ""},
		{"TestMap", fields{ValueAssignmentMap, map[string]string{"1": "one", "2": "two", "3": "three"}}, ""},
		{"TestNil", fields{ValueAssignmentLiteral, nil}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ValueAssignment{
				Type:  tt.fields.Type,
				Value: tt.fields.Value,
			}
			if got := p.GetLiteral(); got != tt.want {
				t.Errorf("ValueAssignment.GetLiteral() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValueAssignment_GetFunction(t *testing.T) {
	t.Parallel()
	type fields struct {
		Type  ValueAssignmentType
		Value interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   *Function
	}{
		{"TestGetProp", fields{ValueAssignmentFunction,
			&Function{
				Operator: GetPropertyOperator,
				Operands: []Operand{LiteralOperand("SELF"), LiteralOperand("prop")},
			}},
			&Function{
				Operator: GetPropertyOperator,
				Operands: []Operand{LiteralOperand("SELF"), LiteralOperand("prop")},
			},
		},
		{"TestConcat", fields{ValueAssignmentFunction,
			&Function{
				Operator: ConcatOperator,
				Operands: []Operand{
					&Function{
						Operator: GetPropertyOperator,
						Operands: []Operand{LiteralOperand("SELF"), LiteralOperand("prop")},
					},
					LiteralOperand(":"),
				},
			}},
			&Function{
				Operator: ConcatOperator,
				Operands: []Operand{
					&Function{
						Operator: GetPropertyOperator,
						Operands: []Operand{LiteralOperand("SELF"), LiteralOperand("prop")},
					},
					LiteralOperand(":"),
				},
			},
		},
		{"TestFunction", fields{ValueAssignmentFunction, nil}, nil},
		{"TestLiteral", fields{ValueAssignmentLiteral, "ko"}, nil},
		{"TestList", fields{ValueAssignmentList, []string{"1", "2", "3"}}, nil},
		{"TestMap", fields{ValueAssignmentMap, map[string]string{"1": "one", "2": "two", "3": "three"}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ValueAssignment{
				Type:  tt.fields.Type,
				Value: tt.fields.Value,
			}
			if got := p.GetFunction(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValueAssignment.GetFunction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValueAssignment_GetList(t *testing.T) {
	t.Parallel()
	type fields struct {
		Type  ValueAssignmentType
		Value interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   []interface{}
	}{
		{"TestList", fields{ValueAssignmentList, []interface{}{"1", "2", "3"}}, []interface{}{"1", "2", "3"}},
		{"TestNil", fields{ValueAssignmentList, nil}, nil},
		{"TestLiteral", fields{ValueAssignmentLiteral, "ko"}, nil},
		{"TestMap", fields{ValueAssignmentMap, map[string]string{"1": "one", "2": "two", "3": "three"}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ValueAssignment{
				Type:  tt.fields.Type,
				Value: tt.fields.Value,
			}
			if got := p.GetList(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValueAssignment.GetList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValueAssignment_GetMap(t *testing.T) {
	t.Parallel()
	type fields struct {
		Type  ValueAssignmentType
		Value interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]interface{}
	}{
		{"TestNil", fields{ValueAssignmentMap, nil}, nil},
		{"TestMap", fields{ValueAssignmentMap, map[string]interface{}{"1": "one", "2": "two", "3": "three"}}, map[string]interface{}{"1": "one", "2": "two", "3": "three"}},
		{"TestLiteral", fields{ValueAssignmentLiteral, "ko"}, nil},
		{"TestList", fields{ValueAssignmentList, []interface{}{"1", "2", "3"}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ValueAssignment{
				Type:  tt.fields.Type,
				Value: tt.fields.Value,
			}
			if got := p.GetMap(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValueAssignment.GetMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValueAssignment_String(t *testing.T) {
	t.Parallel()
	type fields struct {
		Type  ValueAssignmentType
		Value interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"TestStringLiteral", fields{ValueAssignmentLiteral, "hello"}, "hello"},
		{"TestNilLiteral", fields{ValueAssignmentLiteral, nil}, ""},
		{"TestMap", fields{ValueAssignmentMap, map[string]interface{}{"1": "o:ne"}}, `{"1": "o:ne"}`},
		{"TestNilMap", fields{ValueAssignmentMap, nil}, `{}`},
		{"TestList", fields{ValueAssignmentList, []interface{}{"o:ne", "two", "th\"ree"}}, `["o:ne", "two", "th\"ree"]`},
		{"TestNilList", fields{ValueAssignmentList, nil}, `[]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ValueAssignment{
				Type:  tt.fields.Type,
				Value: tt.fields.Value,
			}
			if got := p.String(); got != tt.want {
				t.Errorf("ValueAssignment.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValueAssignment_UnmarshalYAML(t *testing.T) {

	type args struct {
		yaml string
	}
	tests := []struct {
		name      string
		args      args
		wantErr   bool
		wantType  ValueAssignmentType
		wantValue string
	}{
		{"VersionLiteral", args{`1.10`}, false, ValueAssignmentLiteral, "1.10"},
		{"StringLiteral", args{"1"}, false, ValueAssignmentLiteral, "1"},
		{"FunctionSimple", args{"{ get_property: [SELF, port] }"}, false, ValueAssignmentFunction, "get_property: [SELF, port]"},
		{"FunctionNested", args{`{concat: [get_attribute: [SELF, ip_address], ":", get_property: [SELF, port]]}`}, false, ValueAssignmentFunction, `concat: [get_attribute: [SELF, ip_address], ":", get_property: [SELF, port]]`},
		{"FunctionNestedErr", args{`{ concat: [get_attribute: [SELF, ip_address], ":", get_property: [SELF, port] }`}, true, ValueAssignmentFunction, `concat: [get_attribute: [SELF, ip_address], ":", get_property: [SELF, port]"`},
		{"ListShort", args{`["1", "two"]`}, false, ValueAssignmentList, `["1", "two"]`},
		{"ListExpend", args{`- "1"
- "two"
`}, false, ValueAssignmentList, `["1", "two"]`},
		{"MapShort", args{`{"1": "one", 2: "two"}`}, false, ValueAssignmentMap, `{"1": "one", 2: "two"}`},
		{"MapExpend", args{`"1": "one"
2: "two"`}, false, ValueAssignmentMap, `{"1": "one", 2: "two"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ValueAssignment{}

			err := yaml.Unmarshal([]byte(tt.args.yaml), p)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValueAssignment.UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)

			}
			if err == nil {
				require.Equal(t, tt.wantType, p.Type)
				//if tt.wantType != ValueAssignmentMap {
				//	// Maps can't be tested reliably as we can't ensure keys order
				//	assert.Equal(t, tt.wantValue, p.String())
				//} else {
				//	var m map[interface{}]interface{}
				//	err := yaml.Unmarshal([]byte(tt.wantValue), &m)
				//	require.NoError(t, err)
				//	assert.Equal(t, m, p.Value)
				//}
			}
		})
	}
}

func TestValueAssignmentTypeFromString(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    ValueAssignmentType
		wantErr bool
	}{
		{"Literal", args{"literal"}, ValueAssignmentLiteral, false},
		{"LiteralCase", args{"liTEral"}, ValueAssignmentLiteral, false},
		{"Function", args{"function"}, ValueAssignmentFunction, false},
		{"FunctionCase", args{"FuNction"}, ValueAssignmentFunction, false},
		{"List", args{"list"}, ValueAssignmentList, false},
		{"ListCase", args{"LisT"}, ValueAssignmentList, false},
		{"Map", args{"map"}, ValueAssignmentMap, false},
		{"MapCase", args{"MAP"}, ValueAssignmentMap, false},
		{"Empty", args{""}, ValueAssignmentLiteral, true},
		{"Wrong", args{"Something"}, ValueAssignmentLiteral, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValueAssignmentTypeFromString(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValueAssignmentTypeFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && got != tt.want {
				t.Errorf("ValueAssignmentTypeFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValueAssignmentString(t *testing.T) {
	t.Parallel()
	data := `{ concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port] ] }`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	require.Nil(t, err)

	require.Equal(t, `concat: ["http://", get_attribute: [HOST, public_ip_address], ":", get_property: [SELF, port]]`, va.String())
}

func TestValueAssignmentReadWrite(t *testing.T) {
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

func TestValueAssignmentMalformed(t *testing.T) {
	t.Parallel()
	data := `{ concat: ["http://", [get_attribute, get_property]: [HOST, public_ip_address], ":", get_property: [SELF, port] ]}`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	log.Printf("%v", err)
	require.Error(t, err)
}

func TestValueAssignmentGetInput(t *testing.T) {
	t.Parallel()
	data := `{ get_input: port }`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	require.Nil(t, err)
	require.Equal(t, ValueAssignmentFunction, va.Type)
	require.Len(t, va.GetFunction().Operands, 1)
	require.Equal(t, `get_input: port`, va.String())

	va2 := ValueAssignment{}
	err = yaml.Unmarshal([]byte(va.String()), &va2)
	require.Nil(t, err)
	require.Equal(t, ValueAssignmentFunction, va2.Type)
	require.Len(t, va2.GetFunction().Operands, 1)
}

func TestValueAssignmentSlurmResult(t *testing.T) {
	t.Parallel()
	data := `"Final Results: \"Minibatch[1-11]\": errs = 0.550%"`
	va := ValueAssignment{}

	err := yaml.Unmarshal([]byte(data), &va)
	require.Nil(t, err)
	require.Equal(t, ValueAssignmentLiteral, va.Type)
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

func TestValueAssignment_UnmarshalJSON(t *testing.T) {

	tests := []struct {
		name      string
		jsonVal   string
		wantType  ValueAssignmentType
		wantValue interface{}
	}{
		{"IntLiteral", `1.10`, ValueAssignmentLiteral, 1.10},
		{"StringLiteral", `"abc"`, ValueAssignmentLiteral, "abc"},
		{"List", `["1", "two"]`, ValueAssignmentList, []interface{}{"1", "two"}},
		{"Map", `{"one":"val1","two":"val2"}`, ValueAssignmentMap,
			map[string]interface{}{"one": "val1", "two": "val2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v ValueAssignment

			err := json.Unmarshal([]byte(tt.jsonVal), &v)
			if err != nil {
				t.Errorf("ValueAssignment.UnmarshalJSON() error = %v", err)

			} else {
				assert.Equal(t, tt.wantType.String(), v.Type.String(),
					"Unexpected type error unmarshalling %s", tt.jsonVal)
				assert.Equal(t, tt.wantValue, v.Value,
					"Unexpected value error unmarshalling %s (%t)", tt.jsonVal, v.Value)
			}
		})
	}
}

func TestValueAssignment_MarshalJSON(t *testing.T) {

	type args struct {
		json string
	}
	tests := []struct {
		name    string
		want    string
		valType ValueAssignmentType
		value   interface{}
	}{
		{"StringLiteral", `"abc"`, ValueAssignmentLiteral, "abc"},
		{"List", `["1","two"]`, ValueAssignmentList, []interface{}{"1", "two"}},
		{"Map", `{"one":"val1","two":"val2"}`, ValueAssignmentMap,
			map[string]string{"one": "val1", "two": "val2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := &ValueAssignment{Type: tt.valType, Value: tt.value}
			bytesVal, err := json.Marshal(val)
			if err != nil {
				t.Errorf("ValueAssignment.MarshalJSON() error = %v marshalling %q", err, val)
			} else {
				stringVal := string(bytesVal)
				assert.Equal(t, tt.want, stringVal, "Unexpected result marshalling %q", val)
			}
		})
	}
}
