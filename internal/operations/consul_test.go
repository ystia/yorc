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

package operations

import (
	"context"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestConsul(t *testing.T) {
	cfg := testutil.SetupTestConfig(t)
	srv, client := testutil.NewTestConsulInstance(t, &cfg)
	defer func() {
		srv.Stop()
		os.RemoveAll(cfg.WorkingDirectory)
	}()

	t.Run("groupInternalOperations", func(t *testing.T) {
		testPurgeTasks(t, srv, client)
		testPurgeDeployment(t, cfg, srv, client)
		testPurgeDeploymentPreChecks(t, cfg, srv, client)
		testEnsurePurgeFailedStatus(t, cfg, srv, client)
	})

}

func createTaskKV(t *testing.T, taskID string) {
	t.Helper()

	var keyValue string
	keyValue = strconv.Itoa(int(tasks.TaskStatusINITIAL))
	_, err := consulutil.GetKV().Put(&api.KVPair{Key: path.Join(consulutil.TasksPrefix, taskID, "status"), Value: []byte(keyValue)}, nil)
	require.NoError(t, err)
	_, err = consulutil.GetKV().Put(&api.KVPair{Key: path.Join(consulutil.TasksPrefix, taskID, "targetId"), Value: []byte(taskID)}, nil)
	require.NoError(t, err)

	creationDate := time.Now()
	keyValue = creationDate.Format(time.RFC3339Nano)
	_, err = consulutil.GetKV().Put(&api.KVPair{Key: path.Join(consulutil.TasksPrefix, taskID, "creationDate"), Value: []byte(keyValue)}, nil)
	require.NoError(t, err)
}

func loadTestYaml(t testing.TB, deploymentID string) {
	yamlName := "testdata/topology.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), deploymentID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")
}
