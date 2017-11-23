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

func TestTimestampedName(t *testing.T) {
	t.Parallel()

	type args struct {
		prefix string
		suffix string
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "TestWithSeparator", args: args{prefix: ".janus_", suffix: ""}},
		{name: "TestWithSeparator", args: args{prefix: ".janus_", suffix: "_ending"}},
	}

	for _, tt := range tests {
		require.Contains(t, UniqueTimestampedName(tt.args.prefix, tt.args.suffix), tt.args.prefix, tt.args.suffix)
	}
}
