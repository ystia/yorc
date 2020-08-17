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
	"io/ioutil"
	"path/filepath"
	"testing"

	ctu "github.com/hashicorp/consul/testutil"
	"gotest.tools/v3/assert"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/sshutil"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/testutil"
)

/*


 */

func testActionOperatorAnalyzeJob(t *testing.T, srv *ctu.TestServer, cfg config.Configuration) {

	deploymentID := testutil.BuildDeploymentID(t)
	ctx := context.Background()
	err := deployments.StoreDeploymentDefinition(ctx, deploymentID, "testdata/jobMonitoringTest.yaml")
	assert.NilError(t, err)

	cc, err := cfg.GetConsulClient()
	assert.NilError(t, err)

	type args struct {
		deploymentID  string
		nodeName      string
		action        *prov.Action
		keepArtifacts bool
	}
	tests := []struct {
		name        string
		args        args
		jobInfoFile string
		want        bool
		wantErr     bool
	}{
		{"MonitorRunningJob", args{deploymentID: deploymentID, nodeName: "Job", action: &prov.Action{ActionType: "job-monitoring", Data: map[string]string{
			"nodeName":   "Job",
			"jobID":      "6260",
			"stepName":   "run",
			"taskID":     "t1",
			"workingDir": filepath.Join(cfg.WorkingDirectory, t.Name()),
		}}, keepArtifacts: false}, "scontrol.txt", false, false},
		{"MonitorCompletedJob", args{deploymentID: deploymentID, nodeName: "Job", action: &prov.Action{ActionType: "job-monitoring", Data: map[string]string{
			"nodeName":   "Job",
			"jobID":      "6260",
			"stepName":   "run",
			"taskID":     "t1",
			"workingDir": filepath.Join(cfg.WorkingDirectory, t.Name()),
		}}, keepArtifacts: false}, "scontrol_show_job_completed.txt", true, false},
		{"MonitorFailedJob", args{deploymentID: deploymentID, nodeName: "Job", action: &prov.Action{ActionType: "job-monitoring", Data: map[string]string{
			"nodeName":   "Job",
			"jobID":      "6260",
			"stepName":   "run",
			"taskID":     "t1",
			"workingDir": filepath.Join(cfg.WorkingDirectory, t.Name()),
		}}, keepArtifacts: false}, "scontrol_show_job_failed.txt", true, true},
		{"JobNotFound", args{deploymentID: deploymentID, nodeName: "Job", action: &prov.Action{ActionType: "job-monitoring", Data: map[string]string{
			"nodeName":   "Job",
			"jobID":      "6260",
			"stepName":   "run",
			"taskID":     "t1",
			"workingDir": filepath.Join(cfg.WorkingDirectory, t.Name()),
		}}, keepArtifacts: false}, "", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &actionOperator{}

			sshClient := &sshutil.MockSSHClient{
				MockRunCommand: func(input string) (string, error) {
					if tt.jobInfoFile != "" {
						testdataFile := filepath.Join("testdata", tt.jobInfoFile)
						testdataFileContent, err := ioutil.ReadFile(testdataFile)
						assert.NilError(t, err, "error parsing sacctResultFile %q", testdataFile)
						return string(testdataFileContent), nil
					}
					return "", nil
				},
			}

			got, err := o.analyzeJob(context.Background(), cc, sshClient, tt.args.deploymentID, tt.args.nodeName, tt.args.action, tt.args.keepArtifacts)
			if (err != nil) != tt.wantErr {
				t.Errorf("actionOperator.analyzeJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("actionOperator.analyzeJob() = %v, want %v", got, tt.want)
			}
		})
	}
}
