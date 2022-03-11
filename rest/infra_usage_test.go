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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/registry"
)

type mockInfraUsageCollector struct {
	getUsageInfoCalled bool
	ctx                context.Context
	conf               config.Configuration
	taskID             string
	infraName          string
	locationName       string
	contextCancelled   bool
	lof                events.LogOptionalFields
}

func (m *mockInfraUsageCollector) GetUsageInfo(ctx context.Context, conf config.Configuration, taskID, infraName, locationName string,
	params map[string]string) (map[string]interface{}, error) {

	var err error
	if params["myparam"] == "failure" {
		err = fmt.Errorf("Expected test error")
	}
	return nil, err
}

func testPostInfraUsageHandler(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Parallel()

	mock := new(mockInfraUsageCollector)
	var reg = registry.GetRegistry()
	reg.RegisterInfraUsageCollector("myInfraName", mock, "mock")

	req := httptest.NewRequest("POST", "/infra_usage/myInfraName/myLocationName?myparam=success", nil)
	req.Header.Add("Content-Type", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)

	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusAccepted, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusAccepted)

	require.Equal(t, 1, len(resp.Header["Location"]), "unexpected len(resp.Header[\"Location\"] equal to 1")
	require.Equal(t, true, strings.HasPrefix(resp.Header["Location"][0], "/infra_usage/myInfraName/myLocationName/tasks"), "unexpected location header prefix")
}

func testPostInfraUsageHandlerError(t *testing.T, client *api.Client, cfg config.Configuration, srv *testutil.TestServer) {
	t.Parallel()

	mock := new(mockInfraUsageCollector)
	var reg = registry.GetRegistry()
	reg.RegisterInfraUsageCollector("myInfraName", mock, "mock")

	req := httptest.NewRequest("POST", "/infra_usage/myInfraError/myLocationName?myparam=failure", nil)
	req.Header.Add("Content-Type", mimeTypeApplicationJSON)
	resp := newTestHTTPRouter(client, cfg, req)

	require.NotNil(t, resp, "unexpected nil response")
	require.Equal(t, http.StatusAccepted, resp.StatusCode, "unexpected status code %d instead of %d", resp.StatusCode, http.StatusAccepted)

	require.Equal(t, 1, len(resp.Header["Location"]), "unexpected len(resp.Header[\"Location\"] equal to 1")
	require.Equal(t, true, strings.HasPrefix(resp.Header["Location"][0], "/infra_usage/myInfraName/myLocationName/tasks"), "unexpected location header prefix")
}
