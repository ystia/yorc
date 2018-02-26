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

package tosca

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// UNBOUNDED is the maximum value of a Range
// Max uint64 as per https://golang.org/ref/spec#Numeric_types
const UNBOUNDED uint64 = 18446744073709551615

// An Range is the representation of a TOSCA Range Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#TYPE_TOSCA_RANGE
// for more details
type Range struct {
	LowerBound uint64
	UpperBound uint64
}

// UnmarshalYAML unmarshals a yaml into a Range
func (r *Range) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v []string
	if err := unmarshal(&v); err != nil {
		return err
	}
	if len(v) != 2 {
		return errors.Errorf("Invalid range definition expected %d elements, actually found %d", 2, len(v))
	}

	bound, err := strconv.ParseUint(v[0], 10, 0)
	if err != nil {
		return errors.Errorf("Expecting a unsigned integer as lower bound of the range")
	}
	r.LowerBound = bound
	if bound, err := strconv.ParseUint(v[1], 10, 0); err != nil {
		if strings.ToUpper(v[1]) != "UNBOUNDED" {
			return errors.Errorf("Expecting a unsigned integer or the 'UNBOUNDED' keyword as upper bound of the range")
		}
		r.UpperBound = UNBOUNDED
	} else {
		r.UpperBound = bound
	}

	return nil
}

func shouldQuoteYamlString(s string) bool {
	return strings.ContainsAny(s, ":[],\"{}#") ||
		strings.HasPrefix(strings.TrimSpace(s), "- ") ||
		strings.HasPrefix(strings.TrimSpace(s), "*") ||
		strings.HasPrefix(strings.TrimSpace(s), "?") ||
		strings.HasPrefix(strings.TrimSpace(s), "|") ||
		strings.HasPrefix(strings.TrimSpace(s), "!") ||
		strings.HasPrefix(strings.TrimSpace(s), "%") ||
		strings.HasPrefix(strings.TrimSpace(s), "@") ||
		strings.HasPrefix(strings.TrimSpace(s), "&")
}
