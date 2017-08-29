package config

import (
	"reflect"
	"testing"
)

func TestInfrastructureConfig_Get(t *testing.T) {
	t.Parallel()
	type args struct {
		name string
	}
	tests := []struct {
		name string
		ic   InfrastructureConfig
		args args
		want interface{}
	}{
		{name: "TestString", ic: InfrastructureConfig{"s": "res", "S1": 1}, args: args{"s"}, want: "res"},
		{name: "TestInt", ic: InfrastructureConfig{"s": "res", "S1": 1}, args: args{"S1"}, want: 1},
		{name: "TestNil", ic: InfrastructureConfig{"s": "res", "S1": 1}, args: args{"S4"}, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ic.Get(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InfrastructureConfig.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInfrastructureConfig_GetString(t *testing.T) {
	t.Parallel()
	type args struct {
		name string
	}
	tests := []struct {
		name string
		ic   InfrastructureConfig
		args args
		want string
	}{
		{name: "TestString", ic: InfrastructureConfig{"s": "res", "S1": 1}, args: args{"s"}, want: "res"},
		{name: "TestInt", ic: InfrastructureConfig{"s": "res", "S1": 1}, args: args{"S1"}, want: "1"},
		{name: "TestNil", ic: InfrastructureConfig{"s": "res", "S1": 1}, args: args{"S4"}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ic.GetString(tt.args.name); got != tt.want {
				t.Errorf("InfrastructureConfig.GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInfrastructureConfig_GetStringOrDefault(t *testing.T) {
	t.Parallel()
	type args struct {
		name       string
		defaultVal string
	}
	tests := []struct {
		name string
		ic   InfrastructureConfig
		args args
		want string
	}{
		{name: "TestString", ic: InfrastructureConfig{"s": "res", "S1": 1}, args: args{"s", "res2"}, want: "res"},
		{name: "TestInt", ic: InfrastructureConfig{"s": "res", "S1": 1}, args: args{"S1", "res2"}, want: "1"},
		{name: "TestNil", ic: InfrastructureConfig{"s": "res", "S1": 1}, args: args{"S4", "res2"}, want: "res2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ic.GetStringOrDefault(tt.args.name, tt.args.defaultVal); got != tt.want {
				t.Errorf("InfrastructureConfig.GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInfrastructureConfig_GetBool(t *testing.T) {
	t.Parallel()
	type args struct {
		name string
	}
	tests := []struct {
		name string
		ic   InfrastructureConfig
		args args
		want bool
	}{
		{name: "TestStringInvalid", ic: InfrastructureConfig{"s": "res", "S1": 1}, args: args{"s"}, want: false},
		{name: "TestInt1", ic: InfrastructureConfig{"s": "res", "S1": 1}, args: args{"S1"}, want: true},
		{name: "TestInt0", ic: InfrastructureConfig{"s": "res", "S1": 0}, args: args{"S1"}, want: false},
		{name: "TestNil", ic: InfrastructureConfig{"s": "res", "S1": 1}, args: args{"S4"}, want: false},
		{name: "TestStringFalse", ic: InfrastructureConfig{"s": "false", "S1": 1}, args: args{"s"}, want: false},
		{name: "TestStringTrue1", ic: InfrastructureConfig{"s": "true", "S1": 1}, args: args{"s"}, want: true},
		{name: "TestStringTrue1", ic: InfrastructureConfig{"s": "True", "S1": 1}, args: args{"s"}, want: true},
		{name: "TestStringT", ic: InfrastructureConfig{"s": "t", "S1": 1}, args: args{"s"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ic.GetBool(tt.args.name); got != tt.want {
				t.Errorf("InfrastructureConfig.GetBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInfrastructureConfig_GetStringSlice(t *testing.T) {
	t.Parallel()
	type args struct {
		name string
	}
	tests := []struct {
		name string
		ic   InfrastructureConfig
		args args
		want []string
	}{
		{name: "TestString", ic: InfrastructureConfig{"s": "res,2,3", "S1": []string{"1"}}, args: args{"s"}, want: []string{"res", "2", "3"}},
		{name: "TestStringSlice", ic: InfrastructureConfig{"s": "res,2,3", "S1": []string{"1"}}, args: args{"S1"}, want: []string{"1"}},
		{name: "TestNotExist", ic: InfrastructureConfig{"s": "res,2,3", "S1": []string{"1"}}, args: args{"S4"}, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ic.GetStringSlice(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InfrastructureConfig.GetStringSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}
