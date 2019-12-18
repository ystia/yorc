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

package collections

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestContainsString(t *testing.T) {
	type args struct {
		s []string
		e string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"EmptySlice", args{[]string{}, "Test"}, false},
		{"NilSlice", args{nil, "Test"}, false},
		{"NilSliceEmptyString", args{nil, ""}, false},
		{"EmptySliceEmptyString", args{[]string{}, ""}, false},
		{"EmptyString", args{[]string{""}, ""}, true},
		{"EmptyStringAtFirst", args{[]string{"", "Test"}, ""}, true},
		{"EmptyStringAtMiddle", args{[]string{"A", "", "Test"}, ""}, true},
		{"EmptyStringAtLast", args{[]string{"A", "Test", ""}, ""}, true},
		{"SimpleString", args{[]string{"Test"}, "Test"}, true},
		{"SimpleStringAtFirst", args{[]string{"Test", "A"}, "Test"}, true},
		{"SimpleStringAtMiddle", args{[]string{"A", "Test", "B"}, "Test"}, true},
		{"SimpleStringAtLast", args{[]string{"A", "B", "Test"}, "Test"}, true},
		{"TestCase", args{[]string{"A", "B", "Test"}, "b"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsString(tt.args.s, tt.args.e); got != tt.want {
				t.Errorf("ContainsString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveDuplicates(t *testing.T) {
	type args struct {
		s []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"EmptySlice", args{[]string{}}, []string{}},
		{"Nillice", args{nil}, nil},
		{"SliceWithOneDuplicate", args{[]string{"john", "tara", "harry", "tara"}}, []string{"john", "tara", "harry"}},
		{"SliceWithManyDuplicates", args{[]string{"tara", "tara", "tara", "tara"}}, []string{"tara"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveDuplicates(tt.args.s)
			require.Equalf(t, got, tt.want, "RemoveDuplicates() = %v, want %v", got, tt.want)
		})
	}
}
