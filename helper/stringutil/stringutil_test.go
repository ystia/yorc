package stringutil

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetLastElement(t *testing.T) {
	t.Parallel()
	type args struct {
		str       string
		separator string
	}
	tests := []struct {
		name     string
		args     args
		expected string
	}{
		{name: "TestWithSeparator", args: args{str: "tosca.interfaces.node.lifecycle.standard.create", separator: "."}, expected: "create"},
		{name: "TestWithoutSeparator", args: args{str: "tosca.interfaces.node.lifecycle.standard.create", separator: "_"}, expected: ""},
	}

	for _, tt := range tests {
		require.Equal(t, tt.expected, GetLastElement(tt.args.str, tt.args.separator))
	}
}

func TestGetAllExceptLastElement(t *testing.T) {
	t.Parallel()
	type args struct {
		str       string
		separator string
	}
	tests := []struct {
		name     string
		args     args
		expected string
	}{
		{name: "TestWithSeparator", args: args{str: "tosca.interfaces.node.lifecycle.standard.create", separator: "."}, expected: "tosca.interfaces.node.lifecycle.standard"},
		{name: "TestWithoutSeparator", args: args{str: "tosca.interfaces.node.lifecycle.standard.create", separator: "_"}, expected: ""},
	}

	for _, tt := range tests {
		require.Equal(t, tt.expected, GetAllExceptLastElement(tt.args.str, tt.args.separator))
	}
}
