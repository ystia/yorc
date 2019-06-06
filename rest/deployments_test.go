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
	"github.com/ystia/yorc/v3/deployments"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/tasks"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
)

type ErrorData struct {
	ID     string `json:"id,omitempty"`
	Status int    `json:"status,omitempty"`
	Title  string `json:"title,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type BodyStruct struct {
	Errors []ErrorData `json:"errors,omitempty"`
}

func testDeploymentHandlers(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Run("testDeleteDeploymentHandlerWithStopOnErrorParam", func(t *testing.T) {
		testDeleteDeploymentHandlerWithStopOnErrorParam(t, client, srv)
	})
	t.Run("testDeleteDeploymentHandlerWithPurgeParam", func(t *testing.T) {
		testDeleteDeploymentHandlerWithPurgeParam(t, client, srv)
	})
}

func loadTestYaml(t *testing.T, testName string, kv *api.KV) string {
	deploymentID := testName
	yamlName := "testdata/testSimpleTopology.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), kv, deploymentID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")
	return deploymentID
}

func cleanTest(kv *api.KV, taskID string) {
	kv.DeleteTree(path.Join(consulutil.TasksPrefix, taskID), nil)
}

func prepareTest(t *testing.T, testName string, client *api.Client, srv *testutil.TestServer) string {
	deploymentID := loadTestYaml(t, testName, client.KV())
	srv.PopulateKV(t, map[string][]byte{
		consulutil.DeploymentKVPrefix + "/" + deploymentID + "/status": []byte("DEPLOYED"),
	})
	return deploymentID
}

func testDeleteDeploymentHandlerWithStopOnErrorParam(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()
	type result struct {
		statusCode      int
		body            *BodyStruct
		continueOnError bool
	}

	tests := []struct {
		name             string
		stopOnErrorParam string
		want             *result
	}{
		{"stopOnErrorWithBadValue", "badValue", &result{statusCode: http.StatusBadRequest, body: &BodyStruct{[]ErrorData{{ID: "bad_request", Status: 400, Title: "Bad Request", Detail: "stopOnError query parameter must be a boolean value"}}}}},
		{"stopOnErrorWithTrue", "true", &result{continueOnError: false, statusCode: http.StatusAccepted, body: nil}},
		{"stopOnErrorWithFalse", "false", &result{continueOnError: true, statusCode: http.StatusAccepted, body: nil}},
		{"stopOnErrorWithNoValue", "", &result{continueOnError: false, statusCode: http.StatusAccepted, body: nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploymentID := prepareTest(t, tt.name, client, srv)
			req := httptest.NewRequest("DELETE", "/deployments/"+deploymentID, nil)

			params := url.Values{}
			params.Add("stopOnError", tt.stopOnErrorParam)
			req.URL.RawQuery = params.Encode()
			resp := newTestHTTPRouter(client, req)
			require.NotNil(t, resp, "unexpected nil response")
			require.Equal(t, tt.want.statusCode, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, tt.want.statusCode)

			bodyb, err := ioutil.ReadAll(resp.Body)
			require.Nil(t, err, "unexpected error reading body response")
			if len(bodyb) > 0 {
				var body *BodyStruct
				err = json.Unmarshal(bodyb, &body)
				require.Nil(t, err, "unexpected error unmarshalling json body")
				fmt.Println(body)
				if !reflect.DeepEqual(body, tt.want.body) {
					t.Errorf("body = %v, want %v", body, tt.want)
				}
			} else if tt.want.body != nil {
				t.Errorf("body is empty but want %v", tt.want)
			}

			// Check task data for correct params tests
			if tt.want.body == nil {
				require.NotNil(t, resp.Header.Get("Location"), "unexpected nil location header")

				location := resp.Header.Get("Location")
				// /deployments/DEPLOYMENT_ID/tasks/TASK_ID
				locationSplit := strings.Split(location, "/")
				require.Equal(t, 5, len(locationSplit), "unexpected location format")
				taskID := strings.Split(location, "/")[4]
				continueOnErrorStr, err := tasks.GetTaskData(client.KV(), taskID, "continueOnError")
				require.Nil(t, err, "unexpected error getting continueOnError task data")
				continueOnError, err := strconv.ParseBool(continueOnErrorStr)
				require.Nil(t, err, "unexpected error parsing bool continueOnError")
				require.Equal(t, tt.want.continueOnError, continueOnError, "unexpected continueOnError value")

				cleanTest(client.KV(), taskID)
			}
		})
	}
}

func testDeleteDeploymentHandlerWithPurgeParam(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	type result struct {
		statusCode int
		body       *BodyStruct
		taskType   tasks.TaskType
	}

	tests := []struct {
		name       string
		purgeParam string
		want       *result
	}{
		{"purgeWithBadValue", "badValue", &result{statusCode: http.StatusBadRequest, body: &BodyStruct{[]ErrorData{{ID: "bad_request", Status: 400, Title: "Bad Request", Detail: "purge query parameter must be a boolean value"}}}}},
		{"purgeWithTrue", "true", &result{taskType: tasks.TaskTypePurge, statusCode: http.StatusAccepted, body: nil}},
		{"purgeWithFalse", "false", &result{taskType: tasks.TaskTypeUnDeploy, statusCode: http.StatusAccepted, body: nil}},
		{"purgeWithNoValue", "", &result{taskType: tasks.TaskTypePurge, statusCode: http.StatusAccepted, body: nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploymentID := prepareTest(t, tt.name, client, srv)
			req := httptest.NewRequest("DELETE", "/deployments/"+deploymentID, nil)

			params := url.Values{}
			params.Add("purge", tt.purgeParam)
			req.URL.RawQuery = params.Encode()
			resp := newTestHTTPRouter(client, req)
			require.NotNil(t, resp, "unexpected nil response")
			require.Equal(t, tt.want.statusCode, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, tt.want.statusCode)

			bodyb, err := ioutil.ReadAll(resp.Body)
			require.Nil(t, err, "unexpected error reading body response")
			if len(bodyb) > 0 {
				var body *BodyStruct
				err = json.Unmarshal(bodyb, &body)
				require.Nil(t, err, "unexpected error unmarshalling json body")
				fmt.Println(body)
				if !reflect.DeepEqual(body, tt.want.body) {
					t.Errorf("body = %v, want %v", body, tt.want)
				}
			} else if tt.want.body != nil {
				t.Errorf("body is empty but want %v", tt.want)
			}

			// Check task data for correct params tests
			if tt.want.body == nil {
				require.NotNil(t, resp.Header.Get("Location"), "unexpected nil location header")

				location := resp.Header.Get("Location")
				// /deployments/DEPLOYMENT_ID/tasks/TASK_ID
				locationSplit := strings.Split(location, "/")
				require.Equal(t, 5, len(locationSplit), "unexpected location format")
				taskID := strings.Split(location, "/")[4]
				taskType, err := tasks.GetTaskType(client.KV(), taskID)
				require.Nil(t, err, "unexpected error getting task type")
				require.Equal(t, tt.want.taskType, taskType, "unexpected task type")

				cleanTest(client.KV(), taskID)
			}

		})
	}
}
