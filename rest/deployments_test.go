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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v3/helper/consulutil"
	ytu "github.com/ystia/yorc/v3/testutil"
)

func testDeploymentHandlers(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	t.Run("testDeleteDeploymentHandlerWithParamValues", func(t *testing.T) {
		testDeleteDeploymentHandlerWithParamValues(t, client, srv)
	})
}

func testDeleteDeploymentHandlerWithParamValues(t *testing.T, client *api.Client, srv *testutil.TestServer) {
	type ErrorStruct struct {
		Id     string `json:"id,omitempty"`
		Status int    `json:"status,omitempty"`
		Title  string `json:"title,omitempty"`
		Detail string `json:"detail,omitempty"`
	}

	type BodyStruct struct {
		Errors []ErrorStruct `json:"errors,omitempty"`
	}

	type result struct {
		statusCode int
		body       *BodyStruct
	}

	deploymentID := ytu.BuildDeploymentID(t)
	t.Parallel()
	srv.PopulateKV(t, map[string][]byte{
		consulutil.DeploymentKVPrefix + "/" + deploymentID + "/status": []byte("DEPLOYED"),
	})

	req := httptest.NewRequest("DELETE", "/deployments/"+deploymentID, nil)

	tests := []struct {
		name        string
		stopOnError string
		want        *result
	}{
		{"stopOnErrorWithBadValue", "badValue", &result{statusCode: http.StatusBadRequest, body: &BodyStruct{[]ErrorStruct{ErrorStruct{Id: "bad_request", Status: 400, Title: "Bad Request", Detail: "stopOnError URL parameter must be a boolean value"}}}}},
		{"stopOnErrorWithCorrectValue", "true", &result{statusCode: http.StatusAccepted, body: nil}},
		{"stopOnErrorWithNoValue", "", &result{statusCode: http.StatusAccepted, body: nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Add("stopOnError", tt.stopOnError)
			req.URL.RawQuery = params.Encode()
			resp := newTestHTTPRouter(client, req)
			require.NotNil(t, resp, "unexpected nil response")
			require.Equal(t, tt.want.statusCode, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, tt.want.statusCode)

			bodyb, err := ioutil.ReadAll(resp.Body)
			fmt.Println("body=" + string(bodyb))
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

		})
	}
}
