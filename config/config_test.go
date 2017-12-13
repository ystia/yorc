package config

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigMap_Get(t *testing.T) {
	t.Parallel()
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		inputs map[string]interface{}
		args   args
		want   interface{}
	}{
		{name: "TestString", inputs: map[string]interface{}{"s": "res", "S1": 1}, args: args{"s"}, want: "res"},
		{name: "TestInt", inputs: map[string]interface{}{"s": "res", "S1": 1}, args: args{"S1"}, want: 1},
		{name: "TestNil", inputs: map[string]interface{}{"s": "res", "S1": 1}, args: args{"S4"}, want: nil},
		{name: "TestTemplate", inputs: map[string]interface{}{"s": `{{print "myres"}}`, "S1": 1}, args: args{"s"}, want: "myres"},
		// When we can't resolve a template we return it as it is
		{name: "TestBadTemplate", inputs: map[string]interface{}{"s": `{{print myres |}}`, "S1": 1}, args: args{"s"}, want: `{{print myres |}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := NewDynamicMapWithPayload(tt.inputs)
			if got := dm.Get(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConfigMap.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigMap_GetString(t *testing.T) {
	t.Parallel()
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		inputs map[string]interface{}
		args   args
		want   string
	}{
		{name: "TestString", inputs: map[string]interface{}{"s": "res", "S1": 1}, args: args{"s"}, want: "res"},
		{name: "TestInt", inputs: map[string]interface{}{"s": "res", "S1": 1}, args: args{"S1"}, want: "1"},
		{name: "TestNil", inputs: map[string]interface{}{"s": "res", "S1": 1}, args: args{"S4"}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := NewDynamicMapWithPayload(tt.inputs)
			if got := dm.GetString(tt.args.name); got != tt.want {
				t.Errorf("ConfigMap.GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigMap_GetStringOrDefault(t *testing.T) {
	t.Parallel()
	type args struct {
		name       string
		defaultVal string
	}
	tests := []struct {
		name   string
		inputs map[string]interface{}
		args   args
		want   string
	}{
		{name: "TestString", inputs: map[string]interface{}{"s": "res", "S1": 1}, args: args{"s", "res2"}, want: "res"},
		{name: "TestInt", inputs: map[string]interface{}{"s": "res", "S1": 1}, args: args{"S1", "res2"}, want: "1"},
		{name: "TestNil", inputs: map[string]interface{}{"s": "res", "S1": 1}, args: args{"S4", "res2"}, want: "res2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := NewDynamicMapWithPayload(tt.inputs)
			if got := dm.GetStringOrDefault(tt.args.name, tt.args.defaultVal); got != tt.want {
				t.Errorf("ConfigMap.GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigMap_GetBool(t *testing.T) {
	t.Parallel()
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		inputs map[string]interface{}
		args   args
		want   bool
	}{
		{name: "TestStringInvalid", inputs: map[string]interface{}{"s": "res", "S1": 1}, args: args{"s"}, want: false},
		{name: "TestInt1", inputs: map[string]interface{}{"s": "res", "S1": 1}, args: args{"S1"}, want: true},
		{name: "TestInt0", inputs: map[string]interface{}{"s": "res", "S1": 0}, args: args{"S1"}, want: false},
		{name: "TestNil", inputs: map[string]interface{}{"s": "res", "S1": 1}, args: args{"S4"}, want: false},
		{name: "TestStringFalse", inputs: map[string]interface{}{"s": "false", "S1": 1}, args: args{"s"}, want: false},
		{name: "TestStringTrue1", inputs: map[string]interface{}{"s": "true", "S1": 1}, args: args{"s"}, want: true},
		{name: "TestStringTrue1", inputs: map[string]interface{}{"s": "True", "S1": 1}, args: args{"s"}, want: true},
		{name: "TestStringT", inputs: map[string]interface{}{"s": "t", "S1": 1}, args: args{"s"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := NewDynamicMapWithPayload(tt.inputs)
			if got := dm.GetBool(tt.args.name); got != tt.want {
				t.Errorf("ConfigMap.GetBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigMap_GetStringSlice(t *testing.T) {
	t.Parallel()
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		inputs map[string]interface{}
		args   args
		want   []string
	}{
		{name: "TestString", inputs: map[string]interface{}{"s": "res,2,3", "S1": []string{"1"}}, args: args{"s"}, want: []string{"res", "2", "3"}},
		{name: "TestStringSlice", inputs: map[string]interface{}{"s": "res,2,3", "S1": []string{"1"}}, args: args{"S1"}, want: []string{"1"}},
		{name: "TestNotExist", inputs: map[string]interface{}{"s": "res,2,3", "S1": []string{"1"}}, args: args{"S4"}, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := NewDynamicMapWithPayload(tt.inputs)
			if got := dm.GetStringSlice(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConfigMap.GetStringSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDynamicMap_IsSet(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		inputs map[string]interface{}
		args   args
		want   bool
	}{
		{name: "TestString", inputs: map[string]interface{}{"s": "res", "S1": []string{"1"}}, args: args{"s"}, want: true},
		{name: "TestNotExist", inputs: map[string]interface{}{"s": "res", "S1": []string{"1"}}, args: args{"x2"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := NewDynamicMapWithPayload(tt.inputs)
			if got := dm.IsSet(tt.args.name); got != tt.want {
				t.Errorf("DynamicMap.IsSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDynamicMap_Keys(t *testing.T) {
	tests := []struct {
		name   string
		inputs map[string]interface{}
		want   []string
	}{
		{name: "TestString", inputs: map[string]interface{}{"s": "res"}, want: []string{"s"}},
		{name: "TestEmpty", inputs: map[string]interface{}{}, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := NewDynamicMapWithPayload(tt.inputs)
			got := dm.Keys()
			assert.Equal(t, tt.want, got)
		})
	}
}
