// Copyright 2020 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
)

func testDeploymentWorkflowHandlers(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Run("testExecuteWorkflow", func(t *testing.T) {
		testExecuteWorkflow(t, client, cfg, srv)
	})
}

func testExecuteWorkflow(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Parallel()

	tests := []struct {
		name                 string
		request              WorkflowRequest
		wantErr              bool
		wantStatus           int
		wantLocationLength   int
		expectedErrorMessage string
	}{
		{"execWithInput",
			WorkflowRequest{
				Inputs: map[string]interface{}{
					"param1": "value1"}},
			false,
			http.StatusCreated,
			1,
			""},
		{"execWithoutInput",
			WorkflowRequest{
				Inputs: map[string]interface{}{}},
			false,
			http.StatusBadRequest,
			0,
			"Missing value for required workflow input"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploymentID := tt.name
			prepareTest(t, deploymentID, client, srv)
			wfEndpoint := fmt.Sprintf("/deployments/%s/workflows/%s", deploymentID, "testWorkflow")

			body, err := json.Marshal(tt.request)
			require.NoError(t, err, "unexpected error marshalling data to provide body request")
			req := httptest.NewRequest("POST", wfEndpoint, bytes.NewBuffer(body))
			req.Header.Add("Content-Type", mimeTypeApplicationJSON)
			resp := newTestHTTPRouter(client, cfg, req)
			require.NotNil(t, resp, "unexpected nil response")
			body, err = ioutil.ReadAll(resp.Body)
			if (err != nil) != tt.wantErr {
				// t.Errorf("%s error = %v, wantErr %v", tt.name, err, tt.wantErr)
				t.Errorf("%s error = %v, wantErr %s", tt.name, err, string(body))
				return
			}
			require.Equal(t, tt.wantStatus, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusOK)

			require.Equal(t, tt.wantLocationLength, len(resp.Header["Location"]), "unexpected len(resp.Header[\"Location\"] equal to 1")

			if tt.expectedErrorMessage != "" {
				var errorsFound Errors
				err = json.Unmarshal(body, &errorsFound)
				require.NoError(t, err, "unexpected error unmarshalling json body")
				require.Equal(t, 1, len(errorsFound.Errors), "Unexpected number of errors found")
				require.Contains(t, errorsFound.Errors[0].Error(), tt.expectedErrorMessage, "Unexpected error message")
			}
		})
	}
}
