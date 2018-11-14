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

package sizeutil

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConvertToGB(t *testing.T) {
	var testData = []struct {
		test          string
		inputSize     string
		expectedSize  int
		expectedError bool
	}{
		{"volume1", "1", 1, false},
		{"volume10000000", "100", 1, false},
		{"volume10000000", "1500 M", 2, false},
		{"volume1GB", "1GB", 1, false},
		{"volume1GBS", "1      GB", 1, false},
		{"olume1GiB", "1 GiB", 2, false},
		{"volume2GIB", "2 GIB", 3, false},
		{"volume1TB", "1 tb", 1000, false},
		{"volume1TiB", "1 TiB", 1100, false},
		{"error", "1 deca", 0, true},
	}
	for _, tt := range testData {
		s, err := ConvertToGB(tt.inputSize)
		if !tt.expectedError {
			assert.Nil(t, err)
			assert.Equal(t, tt.expectedSize, s)
		} else {
			assert.Error(t, err, "Expected an error")
		}

	}
}
