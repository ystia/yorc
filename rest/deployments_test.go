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
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/tasks"
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

func testDeploymentHandlers(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Run("testDeleteDeploymentHandlerWithStopOnErrorParam", func(t *testing.T) {
		testDeleteDeploymentHandlerWithStopOnErrorParam(t, client, srv)
	})
	t.Run("testDeleteDeploymentHandlerWithPurgeParam", func(t *testing.T) {
		testDeleteDeploymentHandlerWithPurgeParam(t, client, srv)
	})
	t.Run("testGetDeploymentHandler", func(t *testing.T) {
		testGetDeploymentHandler(t, client, srv)
	})
	t.Run("testListDeploymentHandler", func(t *testing.T) {
		testListDeploymentHandler(t, client, srv)
	})
}

func loadTestYaml(t *testing.T, deploymentID string) {
	yamlName := "testdata/testSimpleTopology.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), deploymentID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")
}

func cleanTest(kv *api.KV, deploymentID, taskID string) {
	if taskID != "" {
		kv.DeleteTree(path.Join(consulutil.TasksPrefix, taskID), nil)
	}
	if deploymentID != "" {
		kv.DeleteTree(path.Join(consulutil.DeploymentKVPrefix, deploymentID), nil)
	}
}

func prepareTest(t *testing.T, deploymentID string, client *api.Client, srv *testutil.TestServer) {
	if deploymentID != "noDeployment" {
		loadTestYaml(t, deploymentID)
		srv.PopulateKV(t, map[string][]byte{
			consulutil.DeploymentKVPrefix + "/" + deploymentID + "/status": []byte("DEPLOYED"),
		})
	}
}

func testDeleteDeploymentHandlerWithStopOnErrorParam(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()
	type result struct {
		statusCode      int
		errors          *Errors
		continueOnError bool
	}

	tests := []struct {
		name             string
		stopOnErrorParam string
		want             *result
	}{
		{"stopOnErrorWithBadValue", "badValue", &result{statusCode: http.StatusBadRequest, errors: &Errors{[]*Error{{ID: "bad_request", Status: 400, Title: "Bad Request", Detail: "stopOnError query parameter must be a boolean value"}}}}},
		{"stopOnErrorWithTrue", "true", &result{continueOnError: false, statusCode: http.StatusAccepted, errors: nil}},
		{"stopOnErrorWithFalse", "false", &result{continueOnError: true, statusCode: http.StatusAccepted, errors: nil}},
		{"stopOnErrorWithNoValue", "", &result{continueOnError: false, statusCode: http.StatusAccepted, errors: nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploymentID := tt.name
			prepareTest(t, deploymentID, client, srv)
			req := httptest.NewRequest("DELETE", "/deployments/"+deploymentID, nil)

			params := url.Values{}
			params.Add("stopOnError", tt.stopOnErrorParam)
			req.URL.RawQuery = params.Encode()
			resp := newTestHTTPRouter(client, req)
			require.NotNil(t, resp, "unexpected nil response")
			require.Equal(t, tt.want.statusCode, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, tt.want.statusCode)

			body, err := ioutil.ReadAll(resp.Body)
			require.Nil(t, err, "unexpected error reading body response")
			if len(body) > 0 {
				var errorsFound Errors
				err = json.Unmarshal(body, &errorsFound)
				require.Nil(t, err, "unexpected error unmarshalling json body")
				if !reflect.DeepEqual(errorsFound, *tt.want.errors) {
					t.Errorf("errors = %v, want %v", errorsFound, tt.want)
				}
			} else if tt.want.errors != nil {
				t.Errorf("body is empty but want %v", tt.want)
			}

			// Check task data for correct params tests
			if tt.want.errors == nil {
				require.NotNil(t, resp.Header.Get("Location"), "unexpected nil location header")

				location := resp.Header.Get("Location")
				// /deployments/DEPLOYMENT_ID/tasks/TASK_ID
				locationSplit := strings.Split(location, "/")
				require.Equal(t, 5, len(locationSplit), "unexpected location format")
				taskID := strings.Split(location, "/")[4]
				continueOnErrorStr, err := tasks.GetTaskData(taskID, "continueOnError")
				require.Nil(t, err, "unexpected error getting continueOnError task data")
				continueOnError, err := strconv.ParseBool(continueOnErrorStr)
				require.Nil(t, err, "unexpected error parsing bool continueOnError")
				require.Equal(t, tt.want.continueOnError, continueOnError, "unexpected continueOnError value")

				cleanTest(client.KV(), deploymentID, taskID)
			}
		})
	}
}

func testDeleteDeploymentHandlerWithPurgeParam(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	type result struct {
		statusCode int
		errors     *Errors
		taskType   tasks.TaskType
	}

	tests := []struct {
		name       string
		purgeParam string
		want       *result
	}{
		{"purgeWithBadValue", "badValue", &result{statusCode: http.StatusBadRequest, errors: &Errors{[]*Error{{ID: "bad_request", Status: 400, Title: "Bad Request", Detail: "purge query parameter must be a boolean value"}}}}},
		{"purgeWithTrue", "true", &result{taskType: tasks.TaskTypePurge, statusCode: http.StatusAccepted, errors: nil}},
		{"purgeWithFalse", "false", &result{taskType: tasks.TaskTypeUnDeploy, statusCode: http.StatusAccepted, errors: nil}},
		{"purgeWithNoValue", "", &result{taskType: tasks.TaskTypePurge, statusCode: http.StatusAccepted, errors: nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploymentID := tt.name
			prepareTest(t, deploymentID, client, srv)
			req := httptest.NewRequest("DELETE", "/deployments/"+deploymentID, nil)

			params := url.Values{}
			params.Add("purge", tt.purgeParam)
			req.URL.RawQuery = params.Encode()
			resp := newTestHTTPRouter(client, req)
			require.NotNil(t, resp, "unexpected nil response")
			require.Equal(t, tt.want.statusCode, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, tt.want.statusCode)

			body, err := ioutil.ReadAll(resp.Body)
			require.Nil(t, err, "unexpected error reading body response")
			if len(body) > 0 {
				var errorsFound Errors
				err = json.Unmarshal(body, &errorsFound)
				require.Nil(t, err, "unexpected error unmarshalling json body")
				if !reflect.DeepEqual(errorsFound, *tt.want.errors) {
					t.Errorf("errors = %v, want %v", errorsFound, tt.want)
				}
			} else if tt.want.errors != nil {
				t.Errorf("errors is empty but want %v", tt.want)
			}

			// Check task data for correct params tests
			if tt.want.errors == nil {
				require.NotNil(t, resp.Header.Get("Location"), "unexpected nil location header")

				location := resp.Header.Get("Location")
				// /deployments/DEPLOYMENT_ID/tasks/TASK_ID
				locationSplit := strings.Split(location, "/")
				require.Equal(t, 5, len(locationSplit), "unexpected location format")
				taskID := strings.Split(location, "/")[4]
				taskType, err := tasks.GetTaskType(taskID)
				require.Nil(t, err, "unexpected error getting task type")
				require.Equal(t, tt.want.taskType, taskType, "unexpected task type")

				cleanTest(client.KV(), deploymentID, taskID)
			}
		})
	}
}

func testGetDeploymentHandler(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Parallel()

	type result struct {
		statusCode int
		errors     *Errors
		deployment *Deployment
	}

	tests := []struct {
		name         string
		deploymentID string
		want         *result
	}{
		//	{"getDeploymentWithBadID", "badDeploymentID", &result{statusCode: http.StatusNotFound, errors: &Errors{[]*Error{errNotFound}}}},
		{"getDeployment", "getDeployment", &result{statusCode: http.StatusOK, errors: nil,
			deployment: &Deployment{ID: "getDeployment", Status: "DEPLOYED",
				Links: []AtomLink{{Href: "/deployments/getDeployment", Rel: "self", LinkType: "application/json"},
					{Href: "/deployments/getDeployment/nodes/Compute", Rel: "node", LinkType: "application/json"}}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareTest(t, tt.name, client, srv)
			req := httptest.NewRequest("GET", "/deployments/"+tt.deploymentID, nil)
			req.Header.Set("Accept", "application/json")
			resp := newTestHTTPRouter(client, req)
			require.NotNil(t, resp, "unexpected nil response")
			require.Equal(t, tt.want.statusCode, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, tt.want.statusCode)

			body, err := ioutil.ReadAll(resp.Body)
			require.Nil(t, err, "unexpected error reading body response")

			if tt.want.errors != nil {
				var errorsFound Errors
				err = json.Unmarshal(body, &errorsFound)
				require.Nil(t, err, "unexpected error unmarshalling json body")
				if !reflect.DeepEqual(errorsFound, *tt.want.errors) {
					t.Errorf("errors = %v, want %v", errorsFound, *tt.want.errors)
				}
			}

			if tt.want.deployment != nil {
				var depFound Deployment
				err = json.Unmarshal(body, &depFound)
				require.Nil(t, err, "unexpected error unmarshalling json body")
				if !reflect.DeepEqual(depFound, *tt.want.deployment) {
					t.Errorf("deployment = %v, want %v", body, *tt.want.deployment)
				}
			}
			cleanTest(client.KV(), tt.deploymentID, "")
		})
	}
}

func testListDeploymentHandler(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	//t.Parallel() because conflicts can occurs with other tests

	type result struct {
		statusCode  int
		errors      *Errors
		deployments *DeploymentsCollection
	}

	tests := []struct {
		name string
		want *result
	}{
		{"getDeployment", &result{statusCode: http.StatusOK, errors: nil,
			deployments: &DeploymentsCollection{[]Deployment{{ID: "getDeployment", Status: "DEPLOYED",
				Links: []AtomLink{{Href: "/deployments/getDeployment", Rel: "deployment", LinkType: "application/json"}}}}}}},
		{"noDeployment", &result{statusCode: http.StatusNoContent, errors: nil,
			deployments: nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareTest(t, tt.name, client, srv)
			req := httptest.NewRequest("GET", "/deployments", nil)
			req.Header.Set("Accept", "application/json")
			resp := newTestHTTPRouter(client, req)
			require.NotNil(t, resp, "unexpected nil response")
			require.Equal(t, tt.want.statusCode, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, tt.want.statusCode)

			body, err := ioutil.ReadAll(resp.Body)
			require.Nil(t, err, "unexpected error reading body response")

			if tt.want.errors != nil {
				var errorsFound Errors
				err = json.Unmarshal(body, &errorsFound)
				require.Nil(t, err, "unexpected error unmarshalling json body")
				if !reflect.DeepEqual(errorsFound, *tt.want.errors) {
					t.Errorf("errors = %v, want %v", errorsFound, *tt.want.errors)
				}
			}

			if tt.want.deployments != nil {
				var depFound DeploymentsCollection
				err = json.Unmarshal(body, &depFound)
				require.Nil(t, err, "unexpected error unmarshalling json body")
				if !reflect.DeepEqual(depFound, *tt.want.deployments) {
					t.Errorf("deployment = %v, want %v", depFound, *tt.want.deployments)
				}
			}
			cleanTest(client.KV(), tt.name, "")
		})
	}
}
