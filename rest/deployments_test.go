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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/helper/ziputil"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tasks/collector"
	ytestutil "github.com/ystia/yorc/v4/testutil"
)

func testDeploymentHandlers(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Run("testDeleteDeploymentHandlerWithStopOnErrorParam", func(t *testing.T) {
		testDeleteDeploymentHandlerWithStopOnErrorParam(t, client, cfg, srv)
	})
	t.Run("testDeleteDeploymentHandlerWithPurgeParam", func(t *testing.T) {
		testDeleteDeploymentHandlerWithPurgeParam(t, client, cfg, srv)
	})
	t.Run("testGetDeploymentHandler", func(t *testing.T) {
		testGetDeploymentHandler(t, client, cfg, srv)
	})
	t.Run("testListDeploymentHandler", func(t *testing.T) {
		testListDeploymentHandler(t, client, cfg, srv)
	})
	t.Run("testUpdateDeployments", func(t *testing.T) {
		testUpdateDeployments(t, client, cfg, srv)
	})
	t.Run("testNewDeployments", func(t *testing.T) {
		testNewDeployments(t, client, cfg, srv)
	})
	t.Run("testPurgeDeployment", func(t *testing.T) {
		testPurgeDeploymentHandler(t, client, cfg, srv)
	})

	// Cleanup work
	os.RemoveAll("./testdata/work")
}

func loadTestYaml(t testing.TB, deploymentID string) {
	yamlName := "testdata/testSimpleTopology.yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), deploymentID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")
}

func cleanTest(deploymentID, taskID string) {
	if taskID != "" {
		consulutil.Delete(path.Join(consulutil.TasksPrefix, taskID), true)
	}
	if deploymentID != "" {
		consulutil.Delete(path.Join(consulutil.DeploymentKVPrefix, deploymentID), true)
		os.RemoveAll("./testdata/work/deployments/" + deploymentID)
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

func testDeleteDeploymentHandlerWithStopOnErrorParam(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
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
			resp := newTestHTTPRouter(client, cfg, req)
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

				cleanTest(deploymentID, taskID)
			}
		})
	}
}

func testDeleteDeploymentHandlerWithPurgeParam(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
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
			resp := newTestHTTPRouter(client, cfg, req)
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

				cleanTest(deploymentID, taskID)
			}
		})
	}
}

func testGetDeploymentHandler(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
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
				Links: []AtomLink{{Href: "/deployments/getDeployment", Rel: "self", LinkType: mimeTypeApplicationJSON},
					{Href: "/deployments/getDeployment/nodes/Compute", Rel: "node", LinkType: mimeTypeApplicationJSON}}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareTest(t, tt.name, client, srv)
			req := httptest.NewRequest("GET", "/deployments/"+tt.deploymentID, nil)
			req.Header.Set("Accept", mimeTypeApplicationJSON)
			resp := newTestHTTPRouter(client, cfg, req)
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
			cleanTest(tt.deploymentID, "")
		})
	}
}

func testListDeploymentHandler(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
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
				Links: []AtomLink{{Href: "/deployments/getDeployment", Rel: "deployment", LinkType: mimeTypeApplicationJSON}}}}}}},
		{"noDeployment", &result{statusCode: http.StatusNoContent, errors: nil,
			deployments: nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareTest(t, tt.name, client, srv)
			req := httptest.NewRequest("GET", "/deployments", nil)
			req.Header.Set("Accept", mimeTypeApplicationJSON)
			resp := newTestHTTPRouter(client, cfg, req)
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
			cleanTest(tt.name, "")
		})
	}
}

func testNewDeployments(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {

	tests := []struct {
		name           string
		toscaFile      string
		preDeployHook  func(*testing.T, string)
		wantStatusCode int
		wantErrors     func(*testing.T, string) *Errors
	}{
		{"deployWorks", "testdata/testSimpleTopology.yaml", nil, http.StatusCreated, nil},
		{"deployMalformedCSAR", "testdata/ca-cert.pem", nil, http.StatusBadRequest,
			func(t *testing.T, deploymentID string) *Errors {
				return &Errors{[]*Error{
					newBadRequestError(fmt.Errorf(
						"one and only one YAML (.yml or .yaml) file should be present at the root of archive for deployment %s",
						deploymentID)),
				},
				}
			}},
		{"deployWithAlreadyBlockingTask", "testdata/testSimpleTopology.yaml", func(t *testing.T, deploymentID string) {
			err := deployments.AddBlockingOperationOnDeploymentFlag(context.Background(), deploymentID)
			require.NoError(t, err)
		}, http.StatusBadRequest,
			func(t *testing.T, deploymentID string) *Errors {
				return &Errors{[]*Error{
					newBadRequestError(fmt.Errorf("deployment %q, is currently processing a blocking operation we can't proceed with your request until this operation finish", deploymentID)),
				},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			depID := ytestutil.BuildDeploymentID(t)

			if tt.preDeployHook != nil {
				tt.preDeployHook(t, depID)
			}
			var reqBody io.Reader
			if tt.toscaFile != "" {
				b, err := ziputil.ZipPath(tt.toscaFile)
				require.NoError(t, err)
				reqBody = bytes.NewReader(b)
			}

			req := httptest.NewRequest("PUT", "/deployments/"+depID, reqBody)
			req.Header.Set("Content-Type", mimeTypeApplicationZip)

			resp := newTestHTTPRouter(client, cfg, req)
			require.NotNil(t, resp, "unexpected nil response")
			require.Equal(t, tt.wantStatusCode, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, tt.wantStatusCode)

			body, err := ioutil.ReadAll(resp.Body)
			require.Nil(t, err, "unexpected error reading body response")

			if tt.wantErrors != nil {
				var errorsFound Errors
				err := json.Unmarshal(body, &errorsFound)
				require.Nil(t, err, "unexpected error unmarshalling json body")
				wantErrors := tt.wantErrors(t, depID)
				if !reflect.DeepEqual(errorsFound, *wantErrors) {
					t.Errorf("errors = %v, want %v", errorsFound, *wantErrors)
				}
			} else {
				url, err := resp.Location()
				require.NoError(t, err)
				require.NotNil(t, url)

				taskList, err := deployments.GetDeploymentTaskList(req.Context(), depID)
				require.NoError(t, err)

				has, _, status, err := tasks.HasLivingTasks(taskList, nil)
				require.True(t, has, "%s should have a registered task", depID)
				require.Equal(t, tasks.TaskStatusINITIAL.String(), status)
			}
			cleanTest(depID, "")
		})
	}
}

func newStringPointer(input string) *string {
	r := new(string)
	*r = input
	return r
}

func testPurgeDeploymentHandler(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Parallel()

	type result struct {
		statusCode int
		errors     *Errors
	}

	tests := []struct {
		name      string
		force     *string
		depStatus deployments.DeploymentStatus
		want      *result
	}{
		{"purgeWithForceBadValue", newStringPointer("badValue"), deployments.INITIAL, &result{statusCode: http.StatusBadRequest, errors: &Errors{[]*Error{{ID: "bad_request", Status: 400, Title: "Bad Request", Detail: "force query parameter must be a boolean value"}}}}},
		{"purgeWithForceTrue", newStringPointer("true"), deployments.INITIAL, &result{statusCode: http.StatusOK, errors: &Errors{}}},
		{"purgeWithForceFalse", newStringPointer("false"), deployments.UNDEPLOYED, &result{statusCode: http.StatusOK, errors: &Errors{}}},
		{"purgeWithForceNoValue", nil, deployments.UNDEPLOYED, &result{statusCode: http.StatusOK, errors: &Errors{}}},
		{"purgeWithForceEmptyValue", new(string), deployments.DEPLOYED, &result{statusCode: http.StatusOK, errors: &Errors{}}},
		{"purgeWithForceFalseWithError", newStringPointer("false"), deployments.DEPLOYED, &result{statusCode: http.StatusInternalServerError, errors: &Errors{Errors: []*Error{{ID: "internal_server_error", Status: 500, Title: "Internal Server Error", Detail: "can't purge a deployment not in \"UNDEPLOYED\" state, actual status is \"DEPLOYED\""}}}}},
		{"purgeWithForceNoValueWithError", nil, deployments.DEPLOYED, &result{statusCode: http.StatusInternalServerError, errors: &Errors{Errors: []*Error{{ID: "internal_server_error", Status: 500, Title: "Internal Server Error", Detail: "can't purge a deployment not in \"UNDEPLOYED\" state, actual status is \"DEPLOYED\""}}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploymentID := tt.name
			prepareTest(t, deploymentID, client, srv)
			err := deployments.SetDeploymentStatus(context.Background(), deploymentID, tt.depStatus)
			require.NoError(t, err)
			_, err = collector.NewCollector(client).RegisterTaskWithData(deploymentID, tasks.TaskTypeDeploy, map[string]string{"workflowName": "install"})
			require.NoError(t, err)
			req := httptest.NewRequest("POST", "/deployments/"+deploymentID+"/purge", nil)
			req.Header.Add("Accept", "application/json")
			params := url.Values{}
			if tt.force != nil {
				params.Add("force", *tt.force)
			}
			req.URL.RawQuery = params.Encode()
			resp := newTestHTTPRouter(client, cfg, req)
			require.NotNil(t, resp, "unexpected nil response")
			assert.Equal(t, tt.want.statusCode, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, tt.want.statusCode)

			body, err := ioutil.ReadAll(resp.Body)
			require.Nil(t, err, "unexpected error reading body response")
			if len(body) > 0 {
				var errorsFound Errors
				err = json.Unmarshal(body, &errorsFound)
				require.Nil(t, err, "unexpected error unmarshalling json body")
				if !reflect.DeepEqual(errorsFound, *tt.want.errors) {
					t.Errorf("errors = %v, want %v", errorsFound, tt.want)
				}
			} else {
				t.Error("Expecting a response body")
			}
		})
	}
}
