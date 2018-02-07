package internal

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func filterFromString(t *testing.T, input string) *Filter {
	filter := &Filter{}
	err := filterParser.ParseString(input, filter)
	require.NoError(t, err)
	return filter
}

func TestFilterParsing(t *testing.T) {

	f := `"l/a.bé&_c" > -2 `

	filter, err := FilterFromString(f)
	require.NoError(t, err)
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(filter)
	// t.Fatal(filter.LabelName)
	// t.Fatal(b.String())
}

func TestFiltersEqMatching(t *testing.T) {

	type args struct {
		labels map[string]string
	}
	tests := []struct {
		name    string
		filter  string
		args    args
		want    bool
		wantErr bool
	}{
		{"TestEqString", `l1=v1`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestEqStringQuote", `l1="v1"`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestEqStringNoMatchKey", `l3=v1`, args{map[string]string{"l1": "v1", "m2": "v2"}}, false, true},
		{"TestEqStringNoMatchValue", `l1=v2`, args{map[string]string{"l1": "v1", "m2": "v2"}}, false, false},
		{"TestNotEqString", `l1!=v2`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestNotEqStringQuote", `l1!="v2"`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestNotEqStringNoMatchKey", `l3!=v1`, args{map[string]string{"l1": "v1", "m2": "v2"}}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := FilterFromString(tt.filter)
			require.NoError(t, err)
			got, err := f.Matches(tt.args.labels)
			if (err != nil) != tt.wantErr {
				t.Errorf("Filter.Matches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Filter.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}
