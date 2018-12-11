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
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestGetFilePath(t *testing.T) {
	t.Parallel()

	f, err := ioutil.TempFile("", "yorctest")
	require.NoError(t, err, "Error creating a temp file")
	fname := f.Name()
	f.Close()

	path, wasPath, err := GetFilePath(fname)
	require.NoError(t, err, "Error calling GetFilePath(%s)", fname)
	defer os.Remove(fname)
	assert.True(t, wasPath, "Unexpected boolean value returned by GetFilePath()")
	assert.Equal(t, fname, path, "Unexpected path returned by GetFilePath()")

	content := "A string value"
	path, wasPath, err = GetFilePath(content)
	require.NoError(t, err, "Error calling GetFilePath(%s)", content)
	defer os.Remove(path)
	assert.False(t, wasPath, "Unexpected boolean value returned by GetFilePath()")

	read, err := ioutil.ReadFile(path)
	require.NoError(t, err, "Error calattempting to read %s", path)
	result := string(read)
	assert.Equal(t, content, result, "Unexpected content of file returned by GetFilePath()")
}

func TestTruncate(t *testing.T) {
	type args struct {
		str string
		l   int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"TruncateToFiveChar", args{"abcdefghijklm", 5}, "ab..."},
		{"TruncateTo-1", args{"abcdefghijklm", -1}, "abcdefghijklm"},
		{"TruncateTo2", args{"abcdefghijklm", 2}, "abcdefghijklm"},
		{"NoTruncate", args{"abcdefg", 20}, "abcdefg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Truncate(tt.args.str, tt.args.l); got != tt.want {
				t.Errorf("Truncate() = %v, want %v", got, tt.want)
			}
		})
	}
}
