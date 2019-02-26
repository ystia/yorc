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
	"testing"

	"github.com/stretchr/testify/require"
)

/*
func filterFromString(t *testing.T, input string) *Filter {
	filter := &Filter{}
	err := filterParser.ParseString(input, filter)
	require.NoError(t, err)
	return filter
}*/

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
		{"TestEqStringQuote", `l1="v1"`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestEqStringNoMatchKey", `l3="v1"`, args{map[string]string{"l1": "v1", "m2": "v2"}}, false, false},
		{"TestEqStringNoMatchValue", `l1="v2"`, args{map[string]string{"l1": "v1", "m2": "v2"}}, false, false},
		{"TestNotEqStringQuote", `l1!="v2"`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestNotEqStringNoMatchKey", `l3!="v1"`, args{map[string]string{"l1": "v1", "m2": "v2"}}, false, false},
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

		{"TestCompFloatEqTrue", `l1 == 5`, args{map[string]string{"l1": "5", "m2": "v2"}}, true, false},
		{"TestCompFloatEqTrueDiffStr", `l1 == 5.0`, args{map[string]string{"l1": "5", "m2": "v2"}}, false, false},
		{"TestCompFloatEqFalse", `l1 == 10`, args{map[string]string{"l1": "50.0", "m2": "v2"}}, false, false},
		{"TestCompFloatEqNoKey", `l3 == 10`, args{map[string]string{"l1": "50.0", "m2": "v2"}}, false, false},
		{"TestCompFloatNeqTrue", `l1 != 1.0`, args{map[string]string{"l1": "5.0", "m2": "v2"}}, true, false},
		{"TestCompFloatNeqFalse", `l1 != 10`, args{map[string]string{"l1": "10", "m2": "v2"}}, false, false},
		{"TestCompFloatNeqFalseDiffStr", `l1 != 10.0`, args{map[string]string{"l1": "10", "m2": "v2"}}, true, false},
		{"TestCompFloatNeqNoKey", `l3 != 10`, args{map[string]string{"l1": "50.0", "m2": "v2"}}, false, false},

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

		{"TestCompSILTTrue", `l1 < 10GHz`, args{map[string]string{"l1": "5MHz", "m2": "v2"}}, true, false},
		{"TestCompSILTFalse", `l1 < 10 MHz`, args{map[string]string{"l1": "50.0THz", "m2": "v2"}}, false, false},
		{"TestCompSILTNoKey", `l3 < 10 THz`, args{map[string]string{"l1": "50.0 h", "m2": "v2"}}, false, false},
		{"TestCompSILETrue", `l1 <= 10.0Hz`, args{map[string]string{"l1": "10 Hz", "m2": "v2"}}, true, false},
		{"TestCompSILEFalse", `l1 <= 10.0MHz`, args{map[string]string{"l1": "20.0GHz", "m2": "v2"}}, false, false},
		{"TestCompSILENoKey", `l3 <= 10.0THz`, args{map[string]string{"l1": "20.0", "m2": "v2"}}, false, false},
		{"TestCompSIGTTrue", `l1 > 1.0MHz`, args{map[string]string{"l1": "0.5GHz", "m2": "v2"}}, true, false},
		{"TestCompSIGTFalse", `l1 > 10.0GHz`, args{map[string]string{"l1": "500.1Hz", "m2": "v2"}}, false, false},
		{"TestCompSIGTNoKey", `l3 > 10Hz`, args{map[string]string{"l1": "50.0", "m2": "v2"}}, false, false},
		{"TestCompSIGETrue", `l1 >= 1.8010THz`, args{map[string]string{"l1": "1.8010THz", "m2": "v2"}}, true, false},
		{"TestCompSIGEFalse", `l1 >= 10.0Hz`, args{map[string]string{"l1": "0.0Hz", "m2": "v2"}}, false, false},
		{"TestCompSIGENoKey", `l3 >= 10.0MHz`, args{map[string]string{"l1": "20.0", "m2": "v2"}}, false, false},

		{"TestCompNotNumber", `l1 >= 10.0`, args{map[string]string{"l1": "2.0.0-SNAPSHOT", "m2": "v2"}}, false, true},
		{"TestCompNotDuration", `l1 <= 10.0s`, args{map[string]string{"l1": "dev", "m2": "v2"}}, false, true},
		{"TestCompNotBytes", `l1 <= 10.0MB`, args{map[string]string{"l1": "dev", "m2": "v2"}}, false, true},
		{"TestCompUnitMismatch", `l1 <= 10.0 Ghz`, args{map[string]string{"l1": "1.0MB", "m2": "v2"}}, false, true},
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
		{"TestNotSPACEInString", `l1 not in (v2)`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestNotSPACEInString", `l1 not in ("v2","v3","v4")`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestNotSPCACEInStringQuote", `l1 not in ("v2")`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestNotSPACEInStringNoMatchKey", `l3 not in ("v1")`, args{map[string]string{"l1": "v1", "m2": "v2"}}, false, false},
		{"TestNotInString", `l1 notin (v2)`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestNotInString", `l1 notin ("v2","v3","v4")`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestNotInStringQuote", `l1 notin ("v2")`, args{map[string]string{"l1": "v1", "m2": "v2"}}, true, false},
		{"TestNotInStringNoMatchKey", `l3 notin ("v1")`, args{map[string]string{"l1": "v1", "m2": "v2"}}, false, false},
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

func TestFiltersRegexMatching(t *testing.T) {

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
		{"TestRegexp", `l1 ~= vv`, args{map[string]string{"l1": "vv", "m2": "v2"}}, true, false},
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
