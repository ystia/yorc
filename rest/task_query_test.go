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

func testTaskQueryHandlers(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Run("testTaskQueryHandlers", func(t *testing.T) {
		testGetTaskQueryHandler(t, client, cfg, srv)
	})
	t.Run("testDeleteTaskQueryHandler", func(t *testing.T) {
		testDeleteTaskQueryHandler(t, client, cfg, srv)
	})
	t.Run("testListTaskQueryHandler", func(t *testing.T) {
		testListTaskQueryHandler(t, client, cfg, srv)
	})
	t.Run("testDeleteTaskQueryHandlerWithTaskNotFound", func(t *testing.T) {
		testDeleteTaskQueryHandlerWithTaskNotFound(t, client, cfg, srv)
	})
	t.Run("testGetTaskQueryHandlerWithTaskNotFound", func(t *testing.T) {
		testGetTaskQueryHandlerWithTaskNotFound(t, client, cfg, srv)
	})
	t.Run("testDeleteTaskQueryHandlerWithRunningTask", func(t *testing.T) {
		testDeleteTaskQueryHandlerWithRunningTask(t, client, cfg, srv)
	})
	t.Run("testDeleteTaskQueryHandlerWithNotTaskQuery", func(t *testing.T) {
		testDeleteTaskQueryHandlerWithNotTaskQuery(t, client, cfg, srv)
	})
}

func testGetTaskQueryHandler(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	srv.PopulateKV(t, map[string][]byte{
		consulutil.TasksPrefix + "/task123/type":              []byte("7"),
		consulutil.TasksPrefix + "/task123/targetId":          []byte("infra_usage:slurm"),
		consulutil.TasksPrefix + "/task123/status":            []byte("2"),
		consulutil.TasksPrefix + "/task123/resultSet":         []byte("{\"result\": \"success\"}"),
		consulutil.TasksPrefix + "/task123/data/locationName": []byte("slurm"),
	})

	req := httptest.NewRequest("GET", "/infra_usage/myInfraName/myLocationName/tasks/task123", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)

	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)

	_, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err, "unexpected error reading body")
	client.KV().DeleteTree(consulutil.TasksPrefix, nil)
}

func testGetTaskQueryHandlerWithTaskNotFound(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	req := httptest.NewRequest("GET", "/infra_usage/myInfraName/myLocationName/tasks/taskNotFound", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)

	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusNotFound)
}

func testDeleteTaskQueryHandler(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	srv.PopulateKV(t, map[string][]byte{
		consulutil.TasksPrefix + "/task123/type":              []byte("7"),
		consulutil.TasksPrefix + "/task123/targetId":          []byte("infra_usage:slurm"),
		consulutil.TasksPrefix + "/task123/status":            []byte("2"),
		consulutil.TasksPrefix + "/task123/resultSet":         []byte("{\"result\": \"success\"}"),
		consulutil.TasksPrefix + "/task123/data/locationName": []byte("slurm"),
	})

	req := httptest.NewRequest("DELETE", "/infra_usage/myInfraName/myLocationName/tasks/task123", nil)
	resp := newTestHTTPRouter(client, cfg, req)

	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusAccepted, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusAccepted)

	client.KV().DeleteTree(consulutil.TasksPrefix, nil)
}

func testDeleteTaskQueryHandlerWithRunningTask(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	srv.PopulateKV(t, map[string][]byte{
		consulutil.TasksPrefix + "/task123/type":              []byte("7"),
		consulutil.TasksPrefix + "/task123/targetId":          []byte("infra_usage:slurm"),
		consulutil.TasksPrefix + "/task123/status":            []byte("1"),
		consulutil.TasksPrefix + "/task123/resultSet":         []byte("{\"result\": \"success\"}"),
		consulutil.TasksPrefix + "/task123/data/locationName": []byte("slurm"),
	})

	req := httptest.NewRequest("DELETE", "/infra_usage/myInfraName/myLocationName/tasks/task123", nil)
	resp := newTestHTTPRouter(client, cfg, req)

	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)

	client.KV().DeleteTree(consulutil.TasksPrefix, nil)
}

func testDeleteTaskQueryHandlerWithNotTaskQuery(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	srv.PopulateKV(t, map[string][]byte{
		consulutil.TasksPrefix + "/task123/type":              []byte("2"),
		consulutil.TasksPrefix + "/task123/targetId":          []byte("infra_usage:slurm"),
		consulutil.TasksPrefix + "/task123/status":            []byte("2"),
		consulutil.TasksPrefix + "/task123/resultSet":         []byte("{\"result\": \"success\"}"),
		consulutil.TasksPrefix + "/task123/data/locationName": []byte("slurm"),
	})

	req := httptest.NewRequest("DELETE", "/infra_usage/myInfraName/myLocationName/tasks/task123", nil)
	resp := newTestHTTPRouter(client, cfg, req)

	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusBadRequest)

	client.KV().DeleteTree(consulutil.TasksPrefix, nil)
}

func testDeleteTaskQueryHandlerWithTaskNotFound(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	req := httptest.NewRequest("DELETE", "/infra_usage/myInfraName/myLocationName/tasks/taskNotFound", nil)
	resp := newTestHTTPRouter(client, cfg, req)

	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusNotFound)
}

func testListTaskQueryHandler(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	srv.PopulateKV(t, map[string][]byte{
		consulutil.TasksPrefix + "/task123/type":              []byte("7"),
		consulutil.TasksPrefix + "/task123/targetId":          []byte("infra_usage:slurm"),
		consulutil.TasksPrefix + "/task123/status":            []byte("2"),
		consulutil.TasksPrefix + "/task123/resultSet":         []byte("{\"result\": \"success\"}"),
		consulutil.TasksPrefix + "/task123/data/locationName": []byte("slurm"),

		consulutil.TasksPrefix + "/task124/type":      []byte("7"),
		consulutil.TasksPrefix + "/task124/targetId":  []byte("infra_usage:slurm"),
		consulutil.TasksPrefix + "/task124/status":    []byte("2"),
		consulutil.TasksPrefix + "/task124/resultSet": []byte("{\"result\": \"success\"}"),

		consulutil.TasksPrefix + "/task125/type":      []byte("7"),
		consulutil.TasksPrefix + "/task125/targetId":  []byte("bad format"),
		consulutil.TasksPrefix + "/task125/status":    []byte("2"),
		consulutil.TasksPrefix + "/task125/resultSet": []byte("{\"result\": \"success\"}"),
	})

	req := httptest.NewRequest("GET", "/infra_usage", nil)
	req.Header.Add("Accept", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)

	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)

	_, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err, "unexpected error reading body")
	client.KV().DeleteTree(consulutil.TasksPrefix, nil)
}
