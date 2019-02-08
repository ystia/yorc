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

package internal

import (
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	humanize "github.com/dustin/go-humanize"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/helper/collections"
)

// FilterFromString generates a Filter from a given input string
func FilterFromString(input string) (*Filter, error) {
	filter := &Filter{}
	err := filterParser.ParseString(input, filter)
	return filter, errors.Wrap(err, "failed to parse given filter string")
}

var fLexer = lexer.Unquote(lexer.Upper(lexer.Must(lexer.Regexp(
	`(\s+)`+
		`|(?P<Keyword>(?i)IN|NOT IN)`+
		`|(?P<Number>[-+]?\d*\.?\d+([eE][-+]?\d+)?)`+
		`|(?P<String>'[^']*'|"[^"]*")`+
		`|(?P<EqOperators>!=|==|=)`+
		`|(?P<CompOperators><=|>=|[<>])`+
		`|(?P<Ident>[-\d\w_\./\\]+)`+
		`|(?P<SetMarks>[(),])`)), "Keyword"), "String")

var filterParser = participle.MustBuild(&Filter{}, fLexer)

// Filter is public for use by reflexion but it should be considered as this whole package as internal and not used directly
type Filter struct {
	LabelName          string              `parser:"@(Ident|String)"`
	EqOperator         *EqOperator         `parser:"[ @@ "`
	SetOperator        *SetOperator        `parser:"| @@ "`
	ComparableOperator *ComparableOperator `parser:"| @@ ]"`
}

// Matches implementation of labelsutil.Filter.Matches()
func (f *Filter) Matches(labels map[string]string) (bool, error) {
	val, ok := labels[f.LabelName]
	if !ok {
		return false, nil
	}
	if f.EqOperator != nil {
		return f.EqOperator.matches(val)
	}
	if f.SetOperator != nil {
		return f.SetOperator.matches(val)
	}
	if f.ComparableOperator != nil {
		return f.ComparableOperator.matches(val)
	}
	return true, nil
}

// EqOperator is public for use by reflexion but it should be considered as this whole package as internal and not used directly
type EqOperator struct {
	Type   string   `parser:"@EqOperators"`
	Values []string `parser:"@(Ident|String|Number) {@(Ident|String|Number)}"`
}

func (o *EqOperator) matches(value string) (bool, error) {
	if o.Type == "!=" {
		return strings.Join(o.Values, " ") != value, nil
	}
	return strings.Join(o.Values, " ") == value, nil
}

// ComparableOperator is public for use by reflexion but it should be considered as this whole package as internal and not used directly
type ComparableOperator struct {
	Type  string  `parser:"@CompOperators"`
	Value float64 `parser:"@Number"`
	Unit  *string `parser:"[@(Ident|String|Number)]"`
}

func compareFloats(op string, v1, v2 float64) (bool, error) {
	switch op {
	case ">":
		return v1 > v2, nil
	case ">=":
		return v1 >= v2, nil
	case "<":
		return v1 < v2, nil
	case "<=":
		return v1 <= v2, nil
	default:
		return false, errors.Errorf("Unsupported comparator %q", op)
	}
}

func compareDurations(op string, v1, v2 time.Duration) (bool, error) {
	return compareFloats(op, float64(v1), float64(v2))
}

func compareUInts(op string, v1, v2 uint64) (bool, error) {
	return compareFloats(op, float64(v1), float64(v2))
}

func (o *ComparableOperator) matches(value string) (bool, error) {
	if o.Unit == nil {
		fValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return false, errors.Wrap(err, "expecting a number for a comparison filter")
		}
		return compareFloats(o.Type, fValue, o.Value)
	}
	oValueAsString := strconv.FormatFloat(o.Value, 'f', -1, 64)
	oDuration, err := time.ParseDuration(oValueAsString + *o.Unit)
	if err == nil {
		vDuration, err := time.ParseDuration(value)
		if err != nil {
			return false, errors.Wrap(err, "expecting a duration for a comparison filter")
		}
		return compareDurations(o.Type, vDuration, oDuration)
	}
	oBytes, err := humanize.ParseBytes(oValueAsString + *o.Unit)
	if err == nil {
		vBytes, err := humanize.ParseBytes(value)
		if err != nil {
			return false, errors.Wrap(err, "expecting a bytes notation for a comparison filter")
		}
		return compareUInts(o.Type, vBytes, oBytes)
	}
	oSI, oUnit, err := humanize.ParseSI(oValueAsString + *o.Unit)
	if err == nil {
		vSI, vUnit, err := humanize.ParseSI(value)
		if err != nil {
			return false, errors.Wrap(err, "expecting a international system of unit notation for a comparison filter")
		}
		if oUnit != vUnit {
			return false, errors.Errorf("Units mismatch for comparison filter (can't compare %q with %q", oUnit, vUnit)
		}
		return compareFloats(o.Type, vSI, oSI)
	}
	return false, errors.Errorf("Unsupported unit %q for comparison filter", o.Type)
}

// SetOperator is public for use by reflexion but it should be considered as this whole package as internal and not used directly
type SetOperator struct {
	Type   string   `parser:"@Keyword '(' "`
	Values []string `parser:"@(Ident|String|Number) {','  @(Ident|String|Number)}')'"`
}

func (o *SetOperator) matches(value string) (bool, error) {
	if o.Type == "NOT IN" {
		return !collections.ContainsString(o.Values, value), nil
	}
	return collections.ContainsString(o.Values, value), nil
}

/*  NEW VERSION BEGINS HERE  */

/* ###################################################### */

//CompositionStrategy is used to define the type of combination used into composite filter
type CompositionStrategy int

const (
	//And represents the AND combination, all component filters have to match for the composite filter to match.
	And CompositionStrategy = iota
	//Or represents the OR combination, at least one  component filter have to match for the composite filter to match.
	Or CompositionStrategy = iota
)

//CompositeFilter is a filter made of other filters, combined using AND method
type CompositeFilter struct {
	Cop     CompositionStrategy //combination operator, AND or OR
	filters []Filter
}

// Matches implementation of labelsutil.Filter.Matches()
func (f *CompositeFilter) Matches(labels map[string]string) (bool, error) {
	return true, nil
}

/* ###################################################### */

//RegexStrategy is used to define the type of search used into regexp filter
type RegexStrategy int

const (
	//Match is used to check if the label matches the filter
	Match RegexStrategy = iota
	//Diff is used to check wether the label does not matches the filter
	Diff RegexStrategy = iota
)

//RegexFilter is a filter checking if a label matches a regular expression
type RegexFilter struct {
	Mop      RegexStrategy
	LabelKey string
	Values   []string
}

// Matches implementation of labelsutil.Filter.Matches()
func (f *RegexFilter) Matches(labels map[string]string) (bool, error) {
	return true, nil
}

/* ###################################################### */

//ComparisonOperator is used to define the type of comparison used into comparison filter
type ComparisonOperator int

const (
	//Inf represents the less than operator <
	Inf ComparisonOperator = iota
	//Sup represents the sup than operator >
	Sup ComparisonOperator = iota
	//Infeq represents the inferior or equal operator =<
	Infeq ComparisonOperator = iota
	//Supeq represents the superior or equal operator >=
	Supeq ComparisonOperator = iota
	//Eq represents the equal operator ==
	Eq ComparisonOperator = iota
	//Neq represents the not equal operator !=
	Neq ComparisonOperator = iota
)

//ComparisonFilter is a filter checking if a label is higher than a certain value
type ComparisonFilter struct {
	Cop      ComparisonOperator
	LabelKey string
	Value    float64
	Unit     *string
}

// Matches implementation of labelsutil.Filter.Matches()
func (f *ComparisonFilter) Matches(labels map[string]string) (bool, error) {
	return false, nil
}

/* ###################################################### */

//KeyFilterStrat is used to define the type of combination used into LabelExists filter
type KeyFilterStrat int

const (
	//Present is used to check if the label exists
	Present KeyFilterStrat = iota
	//Absent is used to check if the label does not exists
	Absent KeyFilterStrat = iota
)

//KeyFilter is a filter checking if a label is existing or not
type KeyFilter struct {
	Eop      KeyFilterStrat
	LabelKey string
}

// Matches implementation of labelsutil.Filter.Matches()
func (f *KeyFilter) Matches(labels map[string]string) (bool, error) {
	_, exists := labels[f.LabelKey]

	if (f.Eop == Present && !exists) || (f.Eop == Absent && exists) {
		return false, nil
	}

	return true, nil
}
