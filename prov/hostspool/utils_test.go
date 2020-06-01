// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package hostspool

import (
	"testing"
)

func TestIsAllowedID(t *testing.T) {
	tests := []struct {
		name string
		str  string
		want bool
	}{
		{name: "alphaNumerical", str: "abc90", want: true},
		{name: "alphaNumericalWithSlash", str: "/abc90", want: true},
		{name: "alphaNumericalWithUnderscore", str: "abc_90", want: true},
		{name: "alphaNumericalWithDash", str: "abc-90", want: true},
		{name: "alphaNumericalWithQuote", str: "abc\"90", want: false},
		{name: "alphaNumericalWithWithSpace", str: "abc 90", want: false},
		{name: "alphaNumericalWithWithComma", str: "abc,90", want: false},
		{name: "alphaNumericalWithWithColon", str: "abc:90", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAllowedID(tt.str)
			if got != tt.want {
				t.Errorf("isAllowedID() = %v, want %v", got, tt.want)
			}
		})
	}
}
