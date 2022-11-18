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

// Filter is a general abstract filtering type
type Filter interface {
	Matches(labels map[string]string) (bool, error)
}

// FilterFromString generates a Filter from a given input string
func FilterFromString(input string) (Filter, error) {
	pfilter := &ParsedFilter{}
	err := filterParser.ParseString(input, pfilter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse given filter string : "+input)
	}
	filter, err := pfilter.createFilter()
	return filter, err
}

const (
	in      string = "IN"
	notIn   string = "NOT IN"
	notin   string = "NOTIN"
	eqOp    string = "="
	deqOp   string = "=="
	neqOp   string = "!="
	reqOp   string = "~="
	nreqOp  string = "!~"
	supOp   string = ">"
	infOp   string = "<"
	supeqOp string = ">="
	infeqOp string = "<="
	nexists string = `!`
)

var myLexer = lexer.Must(lexer.Regexp(
	`(\s+)` +
		`|(?P<Keyword>(?i)` + in + `|` + notIn + `|` + notin + `)` +
		`|(?P<Number>[-+]?\d*\.?\d+([eE][-+]?\d+)?)` +
		`|(?P<String>'[^']*'|"[^"]*")` +
		`|(?P<EqOperator>` + neqOp + `|` + deqOp + `|` + eqOp + `)` +
		`|(?P<RegexOperator>` + reqOp + `|` + nreqOp + `)` +

		`|(?P<NotExists>(?i)` + nexists + `)` +
		//`|(?P<Regex>Ll*)`+
		`|(?P<CompOperators>` + infeqOp + `|` + supeqOp + `|[` + infOp + supOp + `])` +
		`|(?P<Ident>[-\d\w_\./\\]+)` +
		`|(?P<SetMarks>[(),])`))

var filterParser = participle.MustBuild(&ParsedFilter{},
	participle.Lexer(myLexer),
	participle.Upper("Keyword"),
	participle.Unquote("String"),
)

//`|(?P<Regex>^.+$)`

// ParsedFilter is public for use by reflexion but it should be considered as this whole package as internal and not used directly
type ParsedFilter struct {
	NotExists          *string             `parser:"[@NotExists]"`
	LabelName          string              `parser:"@(Ident|String)"`
	EqOperator         *EqOperator         `parser:"[ @@ "`
	SetOperator        *SetOperator        `parser:"| @@ "`
	ComparableOperator *ComparableOperator `parser:"| @@ "`
	RegexOperator      *RegexOperator      `parser:"| @@ ]"`
}

// Matches implementation of labelsutil.Filter.Matches()
func (f *ParsedFilter) createFilter() (Filter, error) {
	if f.EqOperator != nil {
		return f.EqOperator.createFilter(f.LabelName)
	} else if f.SetOperator != nil {
		return f.SetOperator.createFilter(f.LabelName)
	} else if f.ComparableOperator != nil {
		return f.ComparableOperator.createFilter(f.LabelName)
	} else if f.RegexOperator != nil {
		return f.RegexOperator.createFilter(f.LabelName)
	} else {
		if f.NotExists != nil {
			return &KeyFilter{f.LabelName, Absent}, nil
		}
		return &KeyFilter{f.LabelName, Present}, nil
	}
}

// EqOperator is public for use by reflexion but it should be considered as this whole package as internal and not used directly
type EqOperator struct {
	Type   string   `parser:"@EqOperator"`
	ValueS *string  `parser:"@(String"`
	ValueN *float64 `parser:"| Number"`
	ValueU *string  `parser:"[(Ident|String|Number)])"`
}

func (o *EqOperator) createFilter(labelKey string) (Filter, error) {

	if o.ValueS != nil {
		str := regexp.QuoteMeta(*o.ValueS)

		if o.Type == deqOp || o.Type == eqOp {
			return &RegexFilter{labelKey, Matches, str}, nil
		}

		return &RegexFilter{labelKey, Differs, str}, nil
	}

	var cop ComparisonOperator
	switch o.Type {
	case deqOp, eqOp:
		cop = Eq
	case neqOp:
		cop = Neq
	}

	unit := ""
	if o.ValueU != nil {
		unit = *o.ValueU
	}

	return &ComparisonFilter{labelKey, cop, *o.ValueN, unit}, nil
}

// RegexOperator is public for use by reflexion but it should be considered as this whole package as internal and not used directly
type RegexOperator struct {
	Type  string `parser:"@RegexOperator"`
	Value string `parser:"@String"`
}

func (o *RegexOperator) createFilter(labelKey string) (Filter, error) {
	if o.Type == nreqOp {
		return &RegexFilter{labelKey, Excludes, o.Value}, nil
	}

	return &RegexFilter{labelKey, Contains, o.Value}, nil
}

// ComparableOperator is public for use by reflexion but it should be considered as this whole package as internal and not used directly
type ComparableOperator struct {
	Type  string  `parser:"@CompOperators"`
	Value float64 `parser:"@Number"`
	Unit  *string `parser:"[@(Ident|String|Number)]"`
}

func (o *ComparableOperator) createFilter(labelKey string) (Filter, error) {
	var cop ComparisonOperator
	switch o.Type {
	case infeqOp:
		cop = Infeq
	case supeqOp:
		cop = Supeq
	case infOp:
		cop = Inf
	case supOp:
		cop = Sup
	}

	unit := ""
	if o.Unit != nil {
		unit = *o.Unit
	}

	return &ComparisonFilter{labelKey, cop, o.Value, unit}, nil
}

// SetOperator is public for use by reflexion but it should be considered as this whole package as internal and not used directly
type SetOperator struct {
	Type   string   `parser:"@Keyword '(' "`
	Values []string `parser:"@(Ident|String|Number) {','  @(Ident|String|Number)}')'"`
}

func (o *SetOperator) createFilter(labelKey string) (Filter, error) {

	rstrat := Matches
	cstrat := Or
	if o.Type == notIn || o.Type == notin {
		rstrat = Differs
		cstrat = And
	}

	var filters []Filter
	for _, val := range o.Values {
		str := regexp.QuoteMeta(val)
		filters = append(filters, &RegexFilter{labelKey, rstrat, str})
	}

	return &CompositeFilter{cstrat, filters}, nil
}

/* ###################################################### */
/* ###################################################### */
/* ###################################################### */

// CompositionStrategy is used to define the type of combination used into composite filter
type CompositionStrategy int

const (
	//And represents the AND combination, all component filters have to match for the composite filter to match.
	And CompositionStrategy = iota
	//Or represents the OR combination, at least one  component filter have to match for the composite filter to match.
	Or CompositionStrategy = iota
)

// CompositeFilter is a filter made of other filters, combined using AND method
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

// RegexStrategy is used to define the type of search used into regexp filter
type RegexStrategy int

const (
	//Contains is used to check if the label contains the regex of the filter
	Contains RegexStrategy = iota
	//Excludes is used to check whether the label does not contains the regex of the filter
	Excludes RegexStrategy = iota
	//Matches is used to check whether the label matches strictly the regex
	Matches RegexStrategy = iota
	//Differs is used to check whether the label differs from the regex
	Differs RegexStrategy = iota
)

// RegexFilter is a filter checking if a label matches a regular expression
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

	if (f.Strat == Matches) || (f.Strat == Differs) {
		//removing ^ prefix and $ suffix if existing
		//adding ^ prefix and $ suffix
		reg = strings.TrimPrefix(reg, "^")
		reg = strings.TrimSuffix(reg, "$")
		reg = "^" + reg + "$"
	}

	contains, err := regexp.MatchString(reg, value)
	switch f.Strat {
	case Contains, Matches:
		return contains, err
	case Excludes, Differs:
		return !contains, err
	}

	return true, nil
}

/* ###################################################### */

// ComparisonOperator is used to define the type of comparison used into comparison filter
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

// ComparisonFilter is a filter checking if a label is higher than a certain value
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
		filterValue, labelValue, err = siToFloat64(f.Value, f.Unit, lvalS)
	default:
		return false, errors.New("Comparison filter found an unkwnown unit type : " + f.Unit)
	}

	if err != nil {
		return false, errors.Wrap(err, "Something went wrong when Comparison filtering")
	}

	//then we compare values
	switch f.Cop {
	case Inf:
		return (labelValue < filterValue), nil
	case Sup:
		return (labelValue > filterValue), nil
	case Infeq:
		return (labelValue <= filterValue), nil
	case Supeq:
		return (labelValue >= filterValue), nil
	case Eq:
		return (labelValue == filterValue), nil
	case Neq:
		return (labelValue != filterValue), nil
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

func f64ToStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func durationToFloat64(fval float64, funit string, lvalS string) (float64, float64, error) {
	fvalStr := f64ToStr(fval)
	//no error possible, if we enter this function we already know the filter unit is duration
	fDuration, _ := time.ParseDuration(fvalStr + funit)

	lDuration, err := time.ParseDuration(lvalS)
	if err != nil {
		return 0.0, 0.0, errors.Wrap(err, "Filtering expected duration type, found "+lvalS)
	}

	return float64(fDuration), float64(lDuration), nil
}

func bytesToFloat64(fval float64, funit string, lvalS string) (float64, float64, error) {
	fvalStr := f64ToStr(fval)
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

// KeyFilterStrat is used to define the type of combination used into LabelExists filter
type KeyFilterStrat int

const (
	//Present is used to check if the label exists
	Present KeyFilterStrat = iota
	//Absent is used to check if the label does not exists
	Absent KeyFilterStrat = iota
)

// KeyFilter is a filter checking if a label is existing or not
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
