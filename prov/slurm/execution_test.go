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

package slurm

import (
	"regexp"
	"testing"
)

func Test_executionCommon_wrapCommand(t *testing.T) {
	type fields struct {
		Artifacts map[string]string
		jobInfo   *jobInfo
	}
	type args struct {
		innerCmd string
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantPattern *regexp.Regexp
		wantErr     bool
	}{
		{"TestBasicGeneration", fields{
			jobInfo: &jobInfo{Name: "MyJob", Nodes: 1, WorkingDir: "~"}},
			args{"ping -c 3 1.1.1.1"}, regexp.MustCompile(`cat <<'EOF' > ~/b-[-a-f0-9]+.batch\n#!/bin/bash\n\nping -c 3 1.1.1.1\nEOF\nsbatch -D ~ --job-name='MyJob' --nodes=1 ~/b-[-a-f0-9]+.batch; rm -f ~/b-[-a-f0-9]+.batch`), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &executionCommon{
				Artifacts: tt.fields.Artifacts,
				jobInfo:   tt.fields.jobInfo,
			}
			got, err := e.wrapCommand(tt.args.innerCmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("executionCommon.wrapCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantPattern.MatchString(got) {
				t.Errorf("executionCommon.wrapCommand() = %v, want %v", got, tt.wantPattern.String())
			}
		})
	}
}
