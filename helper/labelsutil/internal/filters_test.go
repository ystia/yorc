package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func filterFromString(t *testing.T, input string) *Filter {
	filter := &Filter{}
	err := filterParser.ParseString(input, filter)
	require.NoError(t, err)
	return filter
}

func TestFiltersExistsMatching(t *testing.T) {

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
		{"TestExists", `l1`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestExistsFalse", `l3`, args{map[string]string{"l1": "v1", "m2": "v2"}}, false, false},
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
		{"TestEqStringNoMatchKey", `l3=v1`, args{map[string]string{"l1": "v1", "m2": "v2"}}, false, false},
		{"TestEqStringNoMatchValue", `l1=v2`, args{map[string]string{"l1": "v1", "m2": "v2"}}, false, false},
		{"TestNotEqString", `l1!=v2`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestNotEqStringQuote", `l1!="v2"`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestNotEqStringNoMatchKey", `l3!=v1`, args{map[string]string{"l1": "v1", "m2": "v2"}}, false, false},
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

func TestFiltersCompareMatching(t *testing.T) {

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
		{"TestCompFloatLTTrue", `l1 < 10`, args{map[string]string{"l1": "5", "m2": "v2"}}, true, false},
		{"TestCompFloatLTFalse", `l1 < 10`, args{map[string]string{"l1": "50.0", "m2": "v2"}}, false, false},
		{"TestCompFloatLTNoKey", `l3 < 10`, args{map[string]string{"l1": "50.0", "m2": "v2"}}, false, false},
		{"TestCompFloatLETrue", `l1 <= 10.0`, args{map[string]string{"l1": "10", "m2": "v2"}}, true, false},
		{"TestCompFloatLEFalse", `l1 <= 10.0`, args{map[string]string{"l1": "20.0", "m2": "v2"}}, false, false},
		{"TestCompFloatLENoKey", `l3 <= 10.0`, args{map[string]string{"l1": "20.0", "m2": "v2"}}, false, false},
		{"TestCompFloatGTTrue", `l1 > 1.0`, args{map[string]string{"l1": "5", "m2": "v2"}}, true, false},
		{"TestCompFloatGTFalse", `l1 > 10.0`, args{map[string]string{"l1": "5.001", "m2": "v2"}}, false, false},
		{"TestCompFloatGTNoKey", `l3 > 10`, args{map[string]string{"l1": "50.0", "m2": "v2"}}, false, false},
		{"TestCompFloatGETrue", `l1 >= 1.80e-10`, args{map[string]string{"l1": "1.80e-10", "m2": "v2"}}, true, false},
		{"TestCompFloatGEFalse", `l1 >= 10.0`, args{map[string]string{"l1": "-20.0", "m2": "v2"}}, false, false},
		{"TestCompFloatGENoKey", `l3 >= 10.0`, args{map[string]string{"l1": "20.0", "m2": "v2"}}, false, false},

		{"TestCompDurationLTTrue", `l1 < 10s`, args{map[string]string{"l1": "5s", "m2": "v2"}}, true, false},
		{"TestCompDurationLTFalse", `l1 < 10ms`, args{map[string]string{"l1": "50.0ms", "m2": "v2"}}, false, false},
		{"TestCompDurationLTNoKey", `l3 < 10h`, args{map[string]string{"l1": "50.0 h", "m2": "v2"}}, false, false},
		{"TestCompDurationLETrue", `l1 <= 10.0us`, args{map[string]string{"l1": "10us", "m2": "v2"}}, true, false},
		{"TestCompDurationLEFalse", `l1 <= 10.0ms`, args{map[string]string{"l1": "20.0s", "m2": "v2"}}, false, false},
		{"TestCompDurationLENoKey", `l3 <= 10.0h`, args{map[string]string{"l1": "20.0", "m2": "v2"}}, false, false},
		{"TestCompDurationGTTrue", `l1 > 1.0ms`, args{map[string]string{"l1": "0.5h", "m2": "v2"}}, true, false},
		{"TestCompDurationGTFalse", `l1 > 10.0h`, args{map[string]string{"l1": "500.1ms", "m2": "v2"}}, false, false},
		{"TestCompDurationGTNoKey", `l3 > 10h`, args{map[string]string{"l1": "50.0", "m2": "v2"}}, false, false},
		{"TestCompDurationGETrue", `l1 >= 1.8010m`, args{map[string]string{"l1": "1.8010m", "m2": "v2"}}, true, false},
		{"TestCompDurationGEFalse", `l1 >= 10.0s`, args{map[string]string{"l1": "-20.0s", "m2": "v2"}}, false, false},
		{"TestCompDurationGENoKey", `l3 >= 10.0ms`, args{map[string]string{"l1": "20.0", "m2": "v2"}}, false, false},

		{"TestCompBytesLTTrue", `l1 < 10GB`, args{map[string]string{"l1": "5MiB", "m2": "v2"}}, true, false},
		{"TestCompBytesLTFalse", `l1 < 10 MiB`, args{map[string]string{"l1": "50.0TB", "m2": "v2"}}, false, false},
		{"TestCompBytesLTNoKey", `l3 < 10 TB`, args{map[string]string{"l1": "50.0 h", "m2": "v2"}}, false, false},
		{"TestCompBytesLETrue", `l1 <= 10.0B`, args{map[string]string{"l1": "10 B", "m2": "v2"}}, true, false},
		{"TestCompBytesLEFalse", `l1 <= 10.0MB`, args{map[string]string{"l1": "20.0GiB", "m2": "v2"}}, false, false},
		{"TestCompBytesLENoKey", `l3 <= 10.0TB`, args{map[string]string{"l1": "20.0", "m2": "v2"}}, false, false},
		{"TestCompBytesGTTrue", `l1 > 1.0MB`, args{map[string]string{"l1": "0.5GiB", "m2": "v2"}}, true, false},
		{"TestCompBytesGTFalse", `l1 > 10.0GB`, args{map[string]string{"l1": "500.1B", "m2": "v2"}}, false, false},
		{"TestCompBytesGTNoKey", `l3 > 10B`, args{map[string]string{"l1": "50.0", "m2": "v2"}}, false, false},
		{"TestCompBytesGETrue", `l1 >= 1.8010TB`, args{map[string]string{"l1": "1.8010TB", "m2": "v2"}}, true, false},
		{"TestCompBytesGEFalse", `l1 >= 10.0B`, args{map[string]string{"l1": "0.0B", "m2": "v2"}}, false, false},
		{"TestCompBytesGENoKey", `l3 >= 10.0MB`, args{map[string]string{"l1": "20.0", "m2": "v2"}}, false, false},

		{"TestCompNotNumber", `l1 >= 10.0`, args{map[string]string{"l1": "2.0.0-SNAPSHOT", "m2": "v2"}}, false, true},
		{"TestCompNotDuration", `l1 <= 10.0s`, args{map[string]string{"l1": "dev", "m2": "v2"}}, false, true},
		{"TestCompNotBytes", `l1 <= 10.0MB`, args{map[string]string{"l1": "dev", "m2": "v2"}}, false, true},
		{"TestCompNotSupportedUnit", `l1 <= 10.0 SomeThing`, args{map[string]string{"l1": "1.0MB", "m2": "v2"}}, false, true},
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

func TestFiltersSetMatching(t *testing.T) {

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
		{"TestSetInString", `l1 in (v1)`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestSetInString2", `l1 in (v3, v1)`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestSetInStringQuote", `l1 in ("v1")`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestSetInStringNoMatchKey", `l3 in ("v1")`, args{map[string]string{"l1": "v1", "m2": "v2"}}, false, false},
		{"TestSetInStringNoMatchValue", `l1 in ("v2", "v4")`, args{map[string]string{"l1": "v1", "m2": "v2"}}, false, false},
		{"TestNotInString", `l1 not in (v2)`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestNotInString", `l1 not in ("v2","v3","v4")`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestNotInStringQuote", `l1 not in ("v2")`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestNotInStringNoMatchKey", `l3 not in ("v1")`, args{map[string]string{"l1": "v1", "m2": "v2"}}, false, false},
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
