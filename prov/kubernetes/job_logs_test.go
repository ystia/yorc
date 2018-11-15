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

package kubernetes

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func assertTime(t *testing.T, ti string) time.Time {
	r, err := time.Parse(time.RFC3339Nano, ti)
	require.NoError(t, err)
	return r
}

func Test_parseJobLogs(t *testing.T) {
	type args struct {
		logs        string
		podID       string
		containerID string
	}
	tests := []struct {
		name string
		args args
		want []jobLog
	}{
		{"NormalBehavior", args{
			logs: `2018-10-18T14:22:52.9826038Z 1
2018-10-18T14:22:53Z 2
2018-10-18T14:22:55.984121489Z 3
2018-10-18T14:22:58.985225967Z 4
2018-10-18T14:23:02.98646628Z 5
2018-10-18T14:23:07.987815871Z Computation done!`,
			podID:       "p1",
			containerID: "c1"},
			[]jobLog{
				jobLog{timestamp: assertTime(t, "2018-10-18T14:22:52.9826038Z"), line: "1", containerName: "c1", podName: "p1"},
				jobLog{timestamp: assertTime(t, "2018-10-18T14:22:53Z"), line: "2", containerName: "c1", podName: "p1"},
				jobLog{timestamp: assertTime(t, "2018-10-18T14:22:55.984121489Z"), line: "3", containerName: "c1", podName: "p1"},
				jobLog{timestamp: assertTime(t, "2018-10-18T14:22:58.985225967Z"), line: "4", containerName: "c1", podName: "p1"},
				jobLog{timestamp: assertTime(t, "2018-10-18T14:23:02.98646628Z"), line: "5", containerName: "c1", podName: "p1"},
				jobLog{timestamp: assertTime(t, "2018-10-18T14:23:07.987815871Z"), line: "Computation done!", containerName: "c1", podName: "p1"},
			},
		},
		{"WithMultiLines", args{
			logs: `2018-10-18T14:22:52.9826038Z 1
2018-10-18T14:22:53.983103434Z 2
21
  dfssdf
2018-10-18T14:23:07.987815871Z Computation done!`,
			podID:       "p1",
			containerID: "c1"},
			[]jobLog{
				jobLog{timestamp: assertTime(t, "2018-10-18T14:22:52.9826038Z"), line: "1", containerName: "c1", podName: "p1"},
				jobLog{timestamp: assertTime(t, "2018-10-18T14:22:53.983103434Z"), line: "2\n21\n  dfssdf", containerName: "c1", podName: "p1"},
				jobLog{timestamp: assertTime(t, "2018-10-18T14:23:07.987815871Z"), line: "Computation done!", containerName: "c1", podName: "p1"},
			},
		},
		{"WithStrangeFirstLine", args{
			logs: `fzfz sfs dsfsdf

2018-10-18T14:22:52.9826038Z 1
2018-10-18T14:22:53.983103434Z 2
21
  dfssdf
2018-10-18T14:23:07.987815871Z Computation done!`,
			podID:       "p1",
			containerID: "c1"},
			[]jobLog{
				jobLog{timestamp: assertTime(t, "2018-10-18T14:22:52.9826038Z"), line: "1", containerName: "c1", podName: "p1"},
				jobLog{timestamp: assertTime(t, "2018-10-18T14:22:53.983103434Z"), line: "2\n21\n  dfssdf", containerName: "c1", podName: "p1"},
				jobLog{timestamp: assertTime(t, "2018-10-18T14:23:07.987815871Z"), line: "Computation done!", containerName: "c1", podName: "p1"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJobLogs(tt.args.logs, tt.args.podID, tt.args.containerID)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseJobLogs() = %v, want %v", got, tt.want)
			}
		})
	}
}
