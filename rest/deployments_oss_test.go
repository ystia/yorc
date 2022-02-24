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

//go:build !premium
// +build !premium

package rest

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/ystia/yorc/v4/config"

	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
	ytestutil "github.com/ystia/yorc/v4/testutil"
)

func testUpdateDeployments(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	depID := ytestutil.BuildDeploymentID(t)

	type result struct {
		statusCode int
		errors     *Errors
	}

	tests := []struct {
		name         string
		deploymentID string
		want         *result
	}{
		{"updateExistingDep", depID, &result{statusCode: http.StatusForbidden, errors: &Errors{[]*Error{newForbiddenRequest(fmt.Sprintf("Trying to update deployment %q on an open source version. Updates are supported only on premium versions.", depID))}}}},
		{"updateNotExistingDep", "noDeployment", &result{statusCode: http.StatusNotFound, errors: &Errors{[]*Error{errNotFound}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareTest(t, tt.deploymentID, client, srv)

			req := httptest.NewRequest("PATCH", "/deployments/"+tt.deploymentID, nil)
			req.Header.Set("Content-Type", mimeTypeApplicationZip)
			resp := newTestHTTPRouter(client, cfg, req)
			require.NotNil(t, resp, "unexpected nil response")
			require.Equal(t, tt.want.statusCode, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, tt.want.statusCode)

			body, err := ioutil.ReadAll(resp.Body)
			require.Nil(t, err, "unexpected error reading body response")

			if tt.want.errors != nil {
				var errorsFound Errors
				err := json.Unmarshal(body, &errorsFound)
				require.Nil(t, err, "unexpected error unmarshalling json body")
				if !reflect.DeepEqual(errorsFound, *tt.want.errors) {
					t.Errorf("errors = %v, want %v", errorsFound, *tt.want.errors)
				}
			}
			cleanTest(tt.deploymentID, "")
		})
	}
}
