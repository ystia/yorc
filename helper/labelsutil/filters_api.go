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

package labelsutil

import "github.com/ystia/yorc/helper/labelsutil/internal"

// A Warning is an error during the matching process.
// But it doesn't have the standard Go error semantic as it should be reported but not stop any process.
// It basically means that the label set doesn't match but for an unexpected reason.
type Warning error

// Filter defines the Label filter interface
type Filter interface {
	Matches(labels map[string]string) (bool, error)
}

// MatchesAll checks if all of the given filters match a set of labels
//
// Providing no filters is not considered as an error and will return true.
func MatchesAll(labels map[string]string, filters ...Filter) (bool, Warning) {
	for _, filter := range filters {
		m, err := filter.Matches(labels)
		if err != nil || !m {
			return m, err
		}
	}
	return true, nil
}

// CreateFilter creates a Filter from a given input string
func CreateFilter(filter string) (Filter, error) {
	return internal.FilterFromString(filter)
}
