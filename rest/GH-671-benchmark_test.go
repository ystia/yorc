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

package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tasks/collector"
	ytestutil "github.com/ystia/yorc/v4/testutil"
)

func BenchmarkGetDeployment(b *testing.B) {
	log.SetOutput(ioutil.Discard)
	stdlog.SetOutput(ioutil.Discard)
	cfg := ytestutil.SetupTestConfig(b)
	srv, client := ytestutil.NewTestConsulInstanceWithConfigAndStore(b, func(c *testutil.TestServerConfig) {
		c.LogLevel = "ERR"
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	}, &cfg)
	defer func() {
		srv.Stop()
		os.RemoveAll(cfg.WorkingDirectory)
	}()

	deploymentID := ytestutil.BuildDeploymentID(b)
	loadTestYaml(b, deploymentID)

	benches := []struct {
		existingTasksNb int
	}{
		{1},
		{10},
		{100},
		{1000},
		{10000},
	}

	collector := collector.NewCollector(client)

	b.ResetTimer()
	for _, bb := range benches {
		b.Run(fmt.Sprintf("BenchmarkGetDeployment-%d", bb.existingTasksNb), func(b *testing.B) {
			b.StopTimer()
			taskID, err := collector.RegisterTask(deploymentID, tasks.TaskTypeCustomCommand)
			assert.NilError(b, err, "failed to register task")
			_, errGrp, store := consulutil.WithContext(context.Background())

			for i := 0; i < bb.existingTasksNb; i++ {
				taskPath := path.Join(consulutil.TasksPrefix, fmt.Sprintf("000-task-%08d", i))
				store.StoreConsulKeyAsString(path.Join(taskPath, "targetId"), fmt.Sprintf("dep-%08d", i))
				store.StoreConsulKey(path.Join(taskPath, "status"), []byte(strconv.Itoa(int(tasks.TaskStatusINITIAL))))
				store.StoreConsulKey(path.Join(taskPath, "type"), []byte(strconv.Itoa(int(tasks.TaskTypeDeploy))))
			}
			require.NoError(b, errGrp.Wait())

			defer client.KV().DeleteTree(consulutil.TasksPrefix, nil)
			defer client.KV().DeleteTree(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "tasks"), nil)

			req := httptest.NewRequest("GET", "/deployments/"+deploymentID, nil)
			req.Header.Set("Accept", mimeTypeApplicationJSON)
			var resp *http.Response
			b.StartTimer()
			for n := 0; n < b.N; n++ {
				resp = newTestHTTPRouter(client, cfg, req)
			}

			// stop here as we have delete tree in deferred function
			b.StopTimer()

			// Sanity checks
			assert.Assert(b, resp != nil, "unexpected nil response")
			assert.Equal(b, 200, resp.StatusCode, "unexpected status code")

			body, err := ioutil.ReadAll(resp.Body)
			assert.NilError(b, err, "unexpected error reading body response")

			depFound := new(Deployment)
			err = json.Unmarshal(body, depFound)
			assert.NilError(b, err, "unexpected error unmarshalling json body")
			expectedDeployment := &Deployment{
				ID:     "BenchmarkGetDeployment",
				Status: "INITIAL",
				Links: []AtomLink{
					{
						Rel:      "self",
						Href:     "/deployments/BenchmarkGetDeployment",
						LinkType: "application/json",
					},
					{
						Rel:      "node",
						Href:     "/deployments/BenchmarkGetDeployment/nodes/Compute",
						LinkType: "application/json",
					},
					{
						Rel:      "task",
						Href:     "/deployments/BenchmarkGetDeployment/tasks/" + taskID,
						LinkType: "application/json",
					},
				},
			}
			assert.DeepEqual(b, depFound, expectedDeployment)

		})

	}

}
