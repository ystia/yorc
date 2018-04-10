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

package sliceutil

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPadSlices(t *testing.T) {
	s1 := []string{"a", "b", "c"}
	s2 := []string{"a", "b", "c", "d", "e"}
	s3 := []string{"a", "b", "c", "d", "e", "f", "g"}

	PadSlices("x", &s1, &s2, &s3)
	require.Equal(t, 7, len(s1))
	require.Equal(t, 7, len(s2))
	require.Equal(t, 7, len(s3))
	for i := 3; i < 7; i++ {
		require.Equal(t, "x", s1[i])
	}
	for i := 5; i < 7; i++ {
		require.Equal(t, "x", s2[i])
	}
}
