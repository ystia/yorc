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

package hostspool

import (
	"encoding/json"
	"testing"
)

func TestHostStatusJSONMarshalling(t *testing.T) {

	type args struct {
		status HostStatus
	}
	tests := []struct {
		name           string
		args           args
		wantErr        bool
		expectedResult string
	}{
		{"TestDefaultHostStatus", args{HostStatus(0)}, false, `"free"`},
		{"TestUnknownHostStatus", args{HostStatus(-1)}, false, `"HostStatus(-1)"`},
		{"TestHostStatusFree", args{HostStatusFree}, false, `"free"`},
		{"TestHostStatusAllocated", args{HostStatusAllocated}, false, `"allocated"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r []byte
			var err error
			if r, err = json.Marshal(&tt.args.status); (err != nil) != tt.wantErr {
				t.Fatalf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
			}

			if string(r) != tt.expectedResult {
				t.Errorf("json.Marshal() result = %q, expected %q", string(r), tt.expectedResult)
			}
		})
	}
}

func TestHostStatusJSONUnmarshalling(t *testing.T) {

	type args struct {
		status string
	}
	tests := []struct {
		name           string
		args           args
		wantErr        bool
		expectedResult HostStatus
	}{
		{"TestUnmarshalHostStatusFree", args{`"free"`}, false, HostStatusFree},
		{"TestUnmarshalHostStatusAlloc", args{`"allocated"`}, false, HostStatusAllocated},
		{"TestUnmarshalHostStatusAllocNoCase", args{`"alLoCatEd"`}, false, HostStatusAllocated},
		{"TestUnmarshalHostStatusNotString", args{`10`}, true, HostStatus(0)},
		{"TestUnmarshalInvalidHostStatus", args{`"HostStatusFree"`}, true, HostStatus(0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r HostStatus
			if err := json.Unmarshal([]byte(tt.args.status), &r); (err != nil) != tt.wantErr {
				t.Fatalf("json.Unmarshal() error = %+v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && r != tt.expectedResult {
				t.Errorf("json.Unmarshal() result = %q, expected %q", string(r), tt.expectedResult)
			}
		})
	}
}
