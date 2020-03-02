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
	"encoding/json"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testDeploymentTaskHandlers(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Run("testGetTaskHandler", func(t *testing.T) {
		testGetTaskHandler(t, client, srv)
	})
}

func testGetTaskHandler(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	srv.PopulateKV(t, map[string][]byte{
		consulutil.TasksPrefix + "/task123/type":                     []byte("6"),
		consulutil.TasksPrefix + "/task123/targetId":                 []byte("MyDeploymentID"),
		consulutil.TasksPrefix + "/task123/status":                   []byte("2"),
		consulutil.TasksPrefix + "/task123/data/outputs/outputOne":   []byte("valueOne"),
		consulutil.TasksPrefix + "/task123/data/outputs/outputTwo":   []byte("valueTwo"),
		consulutil.TasksPrefix + "/task123/data/outputs/outputThree": []byte("valueThree"),
	})

	req := httptest.NewRequest("GET", "/deployments/MyDeploymentID/tasks/task123", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, req)

	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)

	body, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err, "unexpected error reading body")

	var task Task
	err = json.Unmarshal(body, &task)
	require.Nil(t, err, "unexpected error unmarshaling json body")
	require.NotNil(t, task, "unexpected nil task information")
	require.Equal(t, "CustomWorkflow", task.Type, "unexpected task type")
	require.Equal(t, "MyDeploymentID", task.TargetID, "unexpected task targetID")
	require.Equal(t, "DONE", task.Status, "unexpected task status")
	require.Len(t, task.Outputs, 3, "expected 3 outputs")
	require.Equal(t, "valueOne", task.Outputs["outputOne"])
	require.Equal(t, "valueTwo", task.Outputs["outputTwo"])
	require.Equal(t, "valueThree", task.Outputs["outputThree"])

	client.KV().DeleteTree(consulutil.TasksPrefix, nil)
}
