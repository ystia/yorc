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
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/consul/api"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/prov/operations"
	"github.com/ystia/yorc/v4/tosca/datatypes"
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
		{"TestWithInlineOptsGeneration", fields{
			jobInfo: &jobInfo{Name: "MyJob", Nodes: 1, WorkingDir: "~", ExecutionOptions: datatypes.SlurmExecutionOptions{InScriptOptions: []string{"#BB ddd", "not dash prefixed so will not appear", "#another one"}}}},
			args{"ping -c 3 1.1.1.1"}, regexp.MustCompile(`cat <<'EOF' > ~/b-[-a-f0-9]+.batch\n#!/bin/bash\n#BB ddd\n#another one\n\nping -c 3 1.1.1.1\nEOF\nsbatch -D ~ --job-name='MyJob' --nodes=1 ~/b-[-a-f0-9]+.batch; rm -f ~/b-[-a-f0-9]+.batch`), false},
		{"TestWithSourceEnvFile", fields{
			jobInfo: &jobInfo{Name: "MyJob", Nodes: 1, WorkingDir: "~", EnvFile: "~/.bash_profile"}},
			args{"ping -c 3 1.1.1.1"}, regexp.MustCompile(`\[ -f ~/.bash_profile \] && \{ source ~/.bash_profile ; \} ;cat <<'EOF' > ~/b-[-a-f0-9]+.batch\n#!/bin/bash\n\nping -c 3 1.1.1.1\nEOF\nsbatch -D ~ --job-name='MyJob' --nodes=1 ~/b-[-a-f0-9]+.batch; rm -f ~/b-[-a-f0-9]+.batch`), false},
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

func testExecutionCommonBuildJobInfo(t *testing.T, kv *api.KV) {

	deploymentID := testutil.BuildDeploymentID(t)
	ctx := context.Background()
	err := deployments.StoreDeploymentDefinition(ctx, kv, deploymentID, "testdata/simple_job.yaml")
	require.NoError(t, err)

	deploymentIDOpts := deploymentID + "-with-opts"
	err = deployments.StoreDeploymentDefinition(ctx, kv, deploymentIDOpts, "testdata/job_with_options.yaml")
	require.NoError(t, err)

	type fields struct {
		locationProps config.DynamicMap
		deploymentID  string
		NodeName      string
		EnvInputs     []*operations.EnvInput
		Primary       string
		isSingularity bool
	}

	tests := []struct {
		name            string
		fields          fields
		wantErr         bool
		expectedJobInfo jobInfo
	}{
		{"CheckDefaultValues", fields{config.DynamicMap{}, deploymentID, "ClassificationJobUnit_Singularity", make([]*operations.EnvInput, 0), "primary", false}, false,
			jobInfo{Name: deploymentID, Tasks: 1, Nodes: 1, MonitoringTimeInterval: 5 * time.Second, Inputs: make(map[string]string), WorkingDir: home}},
		{"ChecklocationPropertiesValues", fields{config.DynamicMap{"default_job_name": "myjobname", "job_monitoring_time_interval": "1s"}, deploymentID, "ClassificationJobUnit_Singularity", make([]*operations.EnvInput, 0), "primary", false}, false,
			jobInfo{Name: "myjobname", Tasks: 1, Nodes: 1, MonitoringTimeInterval: time.Second, Inputs: make(map[string]string), WorkingDir: home}},
		{"CheckErrorIfNoCommandAndPrimary", fields{config.DynamicMap{}, deploymentID, "ClassificationJobUnit_Singularity", make([]*operations.EnvInput, 0), "", false}, true, jobInfo{}},
		{"CheckDefaultValues", fields{config.DynamicMap{}, deploymentIDOpts, "ClassificationJobUnit_Singularity", make([]*operations.EnvInput, 0), "primary", false}, false,
			jobInfo{Name: "ClassificationJobUnit_Singularity", Tasks: 1, Nodes: 1, MonitoringTimeInterval: 5 * time.Second, Inputs: make(map[string]string), WorkingDir: home,
				ExecutionOptions: datatypes.SlurmExecutionOptions{
					Args:            []string{"-c", "python3 /opt/kdetect.py ${STORAGE_PATH}"},
					InScriptOptions: []string{"#BB volume=a4b4f33c-994f-4f3f-877e-395d21bd3fb2 user=bu key=key path=/sharing lustre_path=/fs1/myuser/bu size=1"},
					Command:         "sh",
					EnvVars:         []string{"STORAGE_PATH=/mnt/data-bu/models/C56A8BCD-380E-4E6C-B265-D50208641102-4203addf-aca9-3040-271d-d2bbe0719f79"},
				}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &executionCommon{
				kv:            kv,
				locationProps: tt.fields.locationProps,
				deploymentID:  tt.fields.deploymentID,
				NodeName:      tt.fields.NodeName,
				EnvInputs:     tt.fields.EnvInputs,
				Primary:       tt.fields.Primary,
				isSingularity: tt.fields.isSingularity,
			}
			err := e.buildJobInfo(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("executionCommon.buildJobInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				assert.Equal(t, tt.expectedJobInfo, *e.jobInfo)
			}
		})
	}
}
