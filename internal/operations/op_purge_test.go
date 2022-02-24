// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package operations

import (
	"context"
	"path"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/tasks"
	"gotest.tools/v3/assert"
)

func testPurgeTasks(t *testing.T, srv *testutil.TestServer, client *api.Client) {

	type preTest struct {
		existingTasks []string
	}

	type checks struct {
		existingTasks    []string
		nonExistingTasks []string
	}

	type args struct {
		force       bool
		ignoreTasks []string
	}
	tests := []struct {
		name    string
		preTest preTest
		args    args
		wantErr bool
		checks  checks
	}{
		{"TestNoTasks", preTest{}, args{force: false}, false, checks{}},
		{"TestDeleteAllTasks", preTest{existingTasks: []string{"1", "2", "3"}}, args{force: false, ignoreTasks: []string{}}, false, checks{nonExistingTasks: []string{"1", "2", "3"}}},
		{"TestDeleteWithIgnoreTasks", preTest{existingTasks: []string{"1", "2", "3", "4"}}, args{force: false, ignoreTasks: []string{"1", "3"}}, false, checks{existingTasks: []string{"1", "3"}, nonExistingTasks: []string{"2", "4"}}},
		{"TestNoTasksForce", preTest{}, args{force: true}, false, checks{}},
		{"TestDeleteAllTasksForce", preTest{existingTasks: []string{"1", "2", "3"}}, args{force: true, ignoreTasks: []string{}}, false, checks{nonExistingTasks: []string{"1", "2", "3"}}},
		{"TestDeleteWithIgnoreTasksForce", preTest{existingTasks: []string{"1", "2", "3", "4"}}, args{force: true, ignoreTasks: []string{"1", "3"}}, false, checks{existingTasks: []string{"1", "3"}, nonExistingTasks: []string{"2", "4"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploymentID := tt.name
			for _, taskID := range tt.preTest.existingTasks {
				createTaskKV(t, taskID)
				_, err := client.KV().Put(&api.KVPair{Key: path.Join(consulutil.DeploymentKVPrefix, deploymentID, "tasks", taskID)}, nil)
				assert.NilError(t, err)
			}
			if err := purgeTasks(context.Background(), deploymentID, tt.args.force, tt.args.ignoreTasks...); (err != nil) != tt.wantErr {
				t.Errorf("purgeTasks() error = %v, wantErr %v", err, tt.wantErr)
			}

			for _, taskID := range tt.checks.existingTasks {
				ok, err := tasks.TaskExists(taskID)
				assert.NilError(t, err)
				assert.Assert(t, ok)
			}

			for _, taskID := range tt.checks.nonExistingTasks {
				ok, err := tasks.TaskExists(taskID)
				assert.NilError(t, err)
				assert.Assert(t, !ok)
			}
		})
	}
}

func testPurgeDeployment(t *testing.T, cfg config.Configuration, srv *testutil.TestServer, client *api.Client) {
	type args struct {
		ctx             context.Context
		force           bool
		continueOnError bool
		ignoreTasks     []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"NilContextErrorNoForce", args{ctx: nil, force: false, continueOnError: false}, true},
		{"NoErrorForce", args{ctx: context.Background(), force: true, continueOnError: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploymentID := tt.name
			if err := PurgeDeployment(tt.args.ctx, deploymentID, cfg.WorkingDirectory, tt.args.force, tt.args.continueOnError, tt.args.ignoreTasks...); (err != nil) != tt.wantErr {
				t.Errorf("PurgeDeployment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func testPurgeDeploymentPreChecks(t *testing.T, cfg config.Configuration, srv *testutil.TestServer, client *api.Client) {

	tests := []struct {
		name             string
		deploymentStatus deployments.DeploymentStatus
		wantErr          bool
		errContains      string
	}{
		{"NormalCase", deployments.UNDEPLOYED, false, ""},
		{"NotUndeployedError", deployments.DEPLOYED, true, `can't purge a deployment not in "UNDEPLOYED" state, actual status is "DEPLOYED"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploymentID := tt.name
			loadTestYaml(t, deploymentID)
			err := deployments.SetDeploymentStatus(context.Background(), deploymentID, tt.deploymentStatus)
			assert.NilError(t, err)
			err = purgePreChecks(context.Background(), deploymentID)
			if (err != nil) != tt.wantErr {
				t.Errorf("PurgeDeployment() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				assert.ErrorContains(t, err, tt.errContains)
			}

		})
	}
}

func testEnsurePurgeFailedStatus(t *testing.T, cfg config.Configuration, srv *testutil.TestServer, client *api.Client) {

	tests := []struct {
		name     string
		storeDep bool
	}{
		{"NormalCase", true},
		{"NotFoundDep", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploymentID := tt.name
			ctx := context.Background()
			if tt.storeDep {
				loadTestYaml(t, deploymentID)
			}
			ensurePurgeFailedStatus(ctx, deploymentID)

			status, err := deployments.GetDeploymentStatus(ctx, deploymentID)
			assert.NilError(t, err)
			assert.Equal(t, status, deployments.PURGE_FAILED)

		})
	}
}
