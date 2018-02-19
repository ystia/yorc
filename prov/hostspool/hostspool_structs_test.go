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
