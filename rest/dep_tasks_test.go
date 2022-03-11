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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/consulutil"
)

func testDeploymentTaskHandlers(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Run("testGetTaskHandlerWithTaskOutput", func(t *testing.T) {
		testGetTaskHandlerWithTaskOutput(t, client, cfg, srv)
	})
	t.Run("testGetTaskHandlerWithTaskError", func(t *testing.T) {
		testGetTaskHandlerWithTaskError(t, client, cfg, srv)
	})
	t.Run("testGetTaskHandlerWithTaskNotFound", func(t *testing.T) {
		testGetTaskHandlerWithTaskNotFound(t, client, cfg, srv)
	})
}

func testGetTaskHandlerWithTaskOutput(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
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
	resp := newTestHTTPRouter(client, cfg, req)

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

func testGetTaskHandlerWithTaskError(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	srv.PopulateKV(t, map[string][]byte{
		consulutil.TasksPrefix + "/task123/type":         []byte("0"),
		consulutil.TasksPrefix + "/task123/targetId":     []byte("myDepID"),
		consulutil.TasksPrefix + "/task123/status":       []byte("3"),
		consulutil.TasksPrefix + "/task123/errorMessage": []byte("failure"),
	})

	req := httptest.NewRequest("GET", "/deployments/myDepID/tasks/task123", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)

	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)

	body, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err, "unexpected error reading body")

	task := new(Task)
	err = json.Unmarshal(body, task)
	require.Nil(t, err, "unexpected error unmarshalling body")
	require.Equal(t, "Deploy", task.Type, "unexpected task type")
	require.Equal(t, "task123", task.ID, "unexpected task ID")
	require.Equal(t, "FAILED", task.Status, "unexpected task status")
	require.Equal(t, "failure", task.ErrorMessage, "unexpected task error message")
	require.Equal(t, "myDepID", task.TargetID, "unexpected task targetID")

	client.KV().DeleteTree(consulutil.TasksPrefix, nil)
}

func testGetTaskHandlerWithTaskNotFound(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	req := httptest.NewRequest("GET", "/deployments/myDepID/tasks/taskNotFound", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)

	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusNotFound)
}
