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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	humanize "github.com/dustin/go-humanize"
	"github.com/pkg/errors"
)

//Filter is a general abstrac filtering type
type Filter interface {
	Matches(labels map[string]string) (bool, error)
}

// FilterFromString generates a Filter from a given input string
func FilterFromString(input string) (Filter, error) {
	pfilter := &ParsedFilter{}
	err := filterParser.ParseString(input, pfilter)
	if err != nil {
		return &CompositeFilter{}, errors.Wrap(err, "failed to parse given filter string")
	}
	filter, err := pfilter.createFilter()
	return filter, err
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

var filterParser = participle.MustBuild(&ParsedFilter{}, fLexer)

// ParsedFilter is public for use by reflexion but it should be considered as this whole package as internal and not used directly
type ParsedFilter struct {
	LabelName          string              `parser:"@(Ident|String)"`
	EqOperator         *EqOperator         `parser:"[ @@ "`
	SetOperator        *SetOperator        `parser:"| @@ "`
	ComparableOperator *ComparableOperator `parser:"| @@ ]"`
}

// Matches implementation of labelsutil.Filter.Matches()
func (f *ParsedFilter) createFilter() (Filter, error) {

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

func (o *EqOperator) createFilter(labelKey string) (Filter, error) {
	str := regexp.QuoteMeta(strings.Join(o.Values, " "))

	if o.Type == "==" {
		return &RegexFilter{labelKey, Matches, str}, nil
	}

	return &RegexFilter{labelKey, Differs, str}, nil

}

// ComparableOperator is public for use by reflexion but it should be considered as this whole package as internal and not used directly
type ComparableOperator struct {
	Type  string  `parser:"@CompOperators"`
	Value float64 `parser:"@Number"`
	Unit  *string `parser:"[@(Ident|String|Number)]"`
}

// SetOperator is public for use by reflexion but it should be considered as this whole package as internal and not used directly
type SetOperator struct {
	Type   string   `parser:"@Keyword '(' "`
	Values []string `parser:"@(Ident|String|Number) {','  @(Ident|String|Number)}')'"`
}

/* ###################################################### */
/* ###################################################### */
/* ###################################################### */

/*  NEW VERSION BEGINS HERE  */

/* NEW PARSER BEGINS HERE */

/* old grammar :
STRING :== ""
NUMBER :==
IDENT :==
ABSPRES :== "in" | "not in"
COMPOP :== "<" | ">" | "<=" | ">="
EQOP :== "=="" | "!=" | "="

FILTEREXPR :== IDENT FILTER
FILTER :== SETFILTER | COMPFILTER | EQFILTER
SETFILTER :== ABSPRES '('  (IDENT | STRING | NUMBER) {','  (Ident|String|Number)} ')'
COMPFILTER :== COMPOP NUMBER [(IDENT|STRING|NUMBER)]"
EQFILTER :== EQOP  (IDENT | STRING | NUMBER) { (Ident|String|Number)}
*/

/* new grammar :
STRING :== "..."
NUMBER :==
IDENT :==


*/

/* NEW FILTER BEGINS HERE */

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
	Strat   CompositionStrategy //combination operator, AND or OR
	filters []Filter
}

// Matches implementation of labelsutil.Filter.Matches()
func (f *CompositeFilter) Matches(labels map[string]string) (bool, error) {
	switch f.Strat {
	case And:
		return f.matchAnd(labels)
	case Or:
		return f.matchOr(labels)
	}
	//no filter is true
	return true, nil
}

func (f *CompositeFilter) matchAnd(labels map[string]string) (bool, error) {
	i := 0
	matching := true
	var err error
	err = nil
	for (i < len(f.filters)) && matching && err == nil {
		matching, err = f.filters[i].Matches(labels)
		i++
	}

	return matching, err
}

func (f *CompositeFilter) matchOr(labels map[string]string) (bool, error) {
	i := 0
	matching := false
	var err error
	err = nil
	for (i < len(f.filters)) && !matching && err == nil {
		matching, err = f.filters[i].Matches(labels)
		i++
	}

	return matching, err
}

/* ###################################################### */

//RegexStrategy is used to define the type of search used into regexp filter
type RegexStrategy int

const (
	//Contains is used to check if the label contains the regex of the filter
	Contains RegexStrategy = iota
	//Excludes is used to check wether the label does not contains the regex of the filter
	Excludes RegexStrategy = iota
	//Matches is used to check wether the label matches strictly the regex
	Matches RegexStrategy = iota
	//Differs is used to check wether the label differs from the regex
	Differs RegexStrategy = iota
)

//RegexFilter is a filter checking if a label matches a regular expression
type RegexFilter struct {
	LabelKey string
	Strat    RegexStrategy
	Regex    string
}

// Matches implementation of labelsutil.Filter.Matches()
func (f *RegexFilter) Matches(labels map[string]string) (bool, error) {
	value, exists := labels[f.LabelKey]
	if !exists {
		return false, nil
	}
	reg := f.Regex

	//removing ^ prefix and $ suffix if existing
	reg = strings.TrimPrefix(reg, "^")
	reg = strings.TrimSuffix(reg, "$")

	if (f.Strat == Matches) || (f.Strat == Differs) {
		//adding ^ prefix and $ suffix
		reg = "^" + reg + "$"
	}

	switch f.Strat {
	case Contains:
		return regexp.MatchString(reg, value)
	case Excludes:
		contains, err := regexp.MatchString(reg, value)
		return !contains, err
	case Matches:
		return regexp.MatchString(reg, value)
	case Differs:
		matches, err := regexp.MatchString(reg, value)
		return !matches, err
	}

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
	LabelKey string
	Cop      ComparisonOperator
	Value    float64
	Unit     string
}

// Matches implementation of labelsutil.Filter.Matches()
func (f *ComparisonFilter) Matches(labels map[string]string) (bool, error) {
	//first we cast values to float
	lvalS, exists := labels[f.LabelKey]
	var filterValue float64
	var labelValue float64
	var err error
	if !exists {
		return false, nil
	}

	//getting checking unit type
	switch checkUnitType(f.Unit) {
	case utRaw:
		filterValue, labelValue, err = rawToFloat64(f.Value, lvalS)
	case utDuration:
		filterValue, labelValue, err = durationToFloat64(f.Value, f.Unit, lvalS)
	case utBytes:
		filterValue, labelValue, err = bytesToFloat64(f.Value, f.Unit, lvalS)
	case utSI:
		filterValue, labelValue, err = bytesToFloat64(f.Value, f.Unit, lvalS)
	default:
		return false, errors.New("Comparison filter found an unkwnown unit type : " + f.Unit)
	}

	if err != nil {
		return false, errors.Wrap(err, "Something went wrong when Comparison filtering")
	}

	//then we compare values
	switch f.Cop {
	case Inf:
		return (filterValue < labelValue), nil
	case Sup:
		return (filterValue > labelValue), nil
	case Infeq:
		return (filterValue <= labelValue), nil
	case Supeq:
		return (filterValue >= labelValue), nil
	case Eq:
		return (filterValue == labelValue), nil
	case Neq:
		return (filterValue != labelValue), nil
	}

	return false, errors.New("Comparison Filter has wrong comparison operator")
}

type unitType int

const (
	utRaw unitType = iota
	utDuration
	utBytes
	utSI
	utError
)

func checkUnitType(unit string) unitType {
	if unit == "" {
		return utRaw
	} else if _, err := time.ParseDuration("0" + unit); err == nil {
		return utDuration
	} else if _, err := humanize.ParseBytes("0" + unit); err == nil {
		return utBytes
	} else if _, _, err := humanize.ParseSI("0" + unit); err == nil {
		return utSI
	} else {
		return utError
	}
}

func rawToFloat64(fval float64, lvalS string) (float64, float64, error) {
	lval, err := strconv.ParseFloat(lvalS, 64)
	if err != nil {
		return 0.0, 0.0, errors.Wrap(err, "Filtering expected raw number, found "+lvalS)
	}

	return fval, lval, nil
}

func durationToFloat64(fval float64, funit string, lvalS string) (float64, float64, error) {
	fvalStr := strconv.FormatFloat(fval, 'f', -1, 64)
	//no error possible, if we enter this function we already know the filter unit is duration
	fDuration, _ := time.ParseDuration(fvalStr + funit)

	lDuration, err := time.ParseDuration(lvalS)
	if err != nil {
		return 0.0, 0.0, errors.Wrap(err, "Filtering expected duration type, found "+lvalS)
	}

	return float64(fDuration), float64(lDuration), nil
}

func bytesToFloat64(fval float64, funit string, lvalS string) (float64, float64, error) {
	fvalStr := strconv.FormatFloat(fval, 'f', -1, 64)
	//no error possible, if we enter this function we already know the filter unit is bytes
	fBytes, _ := humanize.ParseBytes(fvalStr + funit)

	vBytes, err := humanize.ParseBytes(lvalS)
	if err != nil {
		return 0.0, 0.0, errors.Wrap(err, "Filtering expected bytes types, found "+lvalS)
	}
	return float64(fBytes), float64(vBytes), nil
}

func siToFloat64(fval float64, funit string, lvalS string) (float64, float64, error) {
	fvalStr := strconv.FormatFloat(fval, 'f', -1, 64)
	//no error possible, if we enter this function we already know the filter unit is bytes
	fSI, funitRes, err := humanize.ParseSI(fvalStr + funit)

	lSI, lunitRes, err := humanize.ParseSI(lvalS)
	if err != nil {
		return 0.0, 0.0, errors.Wrap(err, "Filtering expected a International System unit value type, found "+lvalS)
	}
	if funitRes != lunitRes {
		return 0.0, 0.0, errors.Errorf("Units mismatch for comparison filter (can't compare %q with %q", funitRes, lunitRes)
	}
	return float64(fSI), float64(lSI), nil
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
	LabelKey string
	Strat    KeyFilterStrat
}

// Matches implementation of labelsutil.Filter.Matches()
func (f *KeyFilter) Matches(labels map[string]string) (bool, error) {
	_, exists := labels[f.LabelKey]

	if (f.Strat == Present && !exists) || (f.Strat == Absent && exists) {
		return false, nil
	}

	return true, nil
}
