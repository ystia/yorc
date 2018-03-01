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
		{name: "TestWithSeparator", args: args{prefix: ".yorc_", suffix: ""}},
		{name: "TestWithSeparator", args: args{prefix: ".yorc_", suffix: "_ending"}},
	}

	for _, tt := range tests {
		require.Contains(t, UniqueTimestampedName(tt.args.prefix, tt.args.suffix), tt.args.prefix, tt.args.suffix)
	}
}
