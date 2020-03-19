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
	"encoding/json"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/tosca"
)

func testExecutor(t *testing.T, srv *testutil.TestServer, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)

	jobInfoTest := jobInfo{
		ID: "myJobID",
	}
	jobInfoBytes, err := json.Marshal(jobInfoTest)
	require.NoError(t, err, "Unexpected error marshaling job info")

	srv.PopulateKV(t, map[string][]byte{
		consulutil.TasksPrefix + "/myTask/targetId":         []byte("Job"),
		consulutil.TasksPrefix + "/myTask/status":           []byte("0"),
		consulutil.TasksPrefix + "/myTask/type":             []byte("0"),
		consulutil.TasksPrefix + "/myTask/data/inputs/i0":   []byte("0"),
		consulutil.TasksPrefix + "/myTask/data/nodes/node1": []byte("0,1,2"),
		consulutil.TasksPrefix + "/myTask/data/Job-jobInfo": jobInfoBytes,
	})

	executor := defaultExecutor{}

	operation := prov.Operation{
		Name:                   tosca.RunnableRunOperationName,
		ImplementationArtifact: artifactBatchImplementation,
		ImplementedInType:      "yorc.nodes.slurm.Job",
	}
	_, _, err = executor.ExecAsyncOperation(context.Background(), cfg, "myTask",
		deploymentID, "Job", operation, "myStep")
	require.NoError(t, err, "Failed to prepare execution of asynchronous operation")

}
