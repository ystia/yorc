package internal

import (
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/participle"

	"github.com/dustin/go-humanize"

	"novaforge.bull.com/starlings-janus/janus/helper/collections"

	"github.com/alecthomas/participle/lexer"
	"github.com/pkg/errors"
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

// Filter --> label (= | > | <) value

type Filter struct {
	LabelName          string              `parser:"@(Ident|String)"`
	EqOperator         *EqOperator         `parser:"[ @@ "`
	SetOperator        *SetOperator        `parser:"| @@ "`
	ComparableOperator *ComparableOperator `parser:"| @@ ]"`
}

func (f *Filter) Matches(labels map[string]string) (bool, error) {
	val, ok := labels[f.LabelName]
	if !ok {
		if f.EqOperator == nil && f.SetOperator == nil && f.ComparableOperator == nil {
			return false, nil
		}
		return false, errors.Errorf("label %q not found", f.LabelName)
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

type ComparableOperator struct {
	Type  string  `parser:"@CompOperators"`
	Value string  `parser:"@Number"`
	Unit  *string `parser:"[@(Ident|String|Number)]"`
}

func (o *ComparableOperator) matches(value string) (bool, error) {
	if o.Unit == nil {
		fValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return false, errors.Wrap(err, "expecting a number for a comparison filter")
		}
		oValue, err := strconv.ParseFloat(o.Value, 64)
		if err != nil {
			return false, errors.Wrap(err, "expecting a number for a comparison filter")
		}
		switch o.Type {
		case ">":
			return fValue > oValue, nil
		case ">=":
			return fValue >= oValue, nil
		case "<":
			return fValue < oValue, nil
		case "<=":
			return fValue <= oValue, nil
		default:
			return false, errors.Errorf("Unsupported comparator %q", o.Type)
		}
	}

	oDuration, err := time.ParseDuration(o.Value + *o.Unit)
	if err == nil {
		vDuration, err := time.ParseDuration(value)
		if err != nil {
			return false, errors.Wrap(err, "expecting a duration for a comparison filter")
		}
		switch o.Type {
		case ">":
			return vDuration > oDuration, nil
		case ">=":
			return vDuration >= oDuration, nil
		case "<":
			return vDuration < oDuration, nil
		case "<=":
			return vDuration <= oDuration, nil
		default:
			return false, errors.Errorf("Unsupported comparator %q", o.Type)
		}
	}
	oBytes, err := humanize.ParseBytes(o.Value + *o.Unit)
	if err != nil {
		return false, errors.Errorf("Unsupported unit %q for comparison filter", o.Type)
	}
	vBytes, err := humanize.ParseBytes(value)
	if err != nil {
		return false, errors.Wrap(err, "expecting a bytes notation for a comparison filter")
	}
	switch o.Type {
	case ">":
		return vBytes > oBytes, nil
	case ">=":
		return vBytes >= oBytes, nil
	case "<":
		return vBytes < oBytes, nil
	case "<=":
		return vBytes <= oBytes, nil
	default:
		return false, errors.Errorf("Unsupported comparator %q", o.Type)
	}
}

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
