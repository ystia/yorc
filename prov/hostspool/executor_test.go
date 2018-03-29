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
	"github.com/dustin/go-humanize"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func TestUpdateHostResourcesLabels(t *testing.T) {
	Test20GiB, err := humanize.ParseBytes("20 GiB")
	Test20GB, err := humanize.ParseBytes("20 GB")
	Test50GB, err := humanize.ParseBytes("50 GB")
	Test50GiB, err := humanize.ParseBytes("50 GiB")
	if err != nil {
		t.Fatalf("Error during parsing bytes:%+v", err)
	}
	type args struct {
		host     *Host
		allocRes *hostResources
		allocate bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    map[string]string
	}{

		{"testSimpleAlloc", args{&Host{Name: "host1", Labels: map[string]string{"host.num_cpus": "16", "host.disk_size": "150 GB", "host.mem_size": "20 GB"}}, &hostResources{diskSize: int64(Test50GB), memSize: int64(Test20GB), cpus: int64(8)}, true}, false, map[string]string{"host.num_cpus": "8", "host.disk_size": "100 GB", "host.mem_size": "0 B"}},
		{"testSimpleAllocWithMiB", args{&Host{Name: "host1", Labels: map[string]string{"host.num_cpus": "16", "host.disk_size": "150 GiB", "host.mem_size": "20 GiB"}}, &hostResources{diskSize: int64(Test50GiB), memSize: int64(Test20GiB), cpus: int64(8)}, true}, false, map[string]string{"host.num_cpus": "8", "host.disk_size": "100 GiB", "host.mem_size": "0 B"}},
		{"testSimpleRelease", args{&Host{Name: "host1", Labels: map[string]string{"host.num_cpus": "8", "host.disk_size": "100 GB", "host.mem_size": "0 GB"}}, &hostResources{diskSize: int64(Test50GB), memSize: int64(Test20GB), cpus: int64(8)}, false}, false, map[string]string{"host.num_cpus": "16", "host.disk_size": "150 GB", "host.mem_size": "20 GB"}},
		{"testSimpleReleaseWithMiB", args{&Host{Name: "host1", Labels: map[string]string{"host.num_cpus": "8", "host.disk_size": "100 GiB", "host.mem_size": "0 GiB"}}, &hostResources{diskSize: int64(Test50GiB), memSize: int64(Test20GiB), cpus: int64(8)}, false}, false, map[string]string{"host.num_cpus": "16", "host.disk_size": "150 GiB", "host.mem_size": "20 GiB"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			var labels map[string]string
			labels, err = updateHostResourcesLabels(tt.args.host, tt.args.allocRes, tt.args.allocate)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetCapabilitiesOfType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(labels, tt.want) {
				t.Fatalf("GetCapabilitiesOfType() = %v, want %v", labels, tt.want)
			}
			require.Equal(t, tt.want, labels)
		})
	}
}
