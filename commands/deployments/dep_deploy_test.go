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

package deployments

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type httpClientMockDeploy struct {
	testID string
}

func (c *httpClientMockDeploy) Do(req *http.Request) (*http.Response, error) {
	if strings.Contains(c.testID, "fails") {
		return nil, errors.New("a failure occurs")
	}
	res := &httptest.ResponseRecorder{
		Code: 200,
	}
	res.Header().Set("Location", "myLocation")
	return res.Result(), nil
}

func (c *httpClientMockDeploy) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, path, body)
}

func (c *httpClientMockDeploy) Get(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockDeploy) Head(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockDeploy) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockDeploy) PostForm(path string, data url.Values) (*http.Response, error) {
	return &http.Response{}, nil
}

func TestDeploy(t *testing.T) {
	err := deploy(&httpClientMockDeploy{}, []string{"./testdata/deployment.zip"}, false, false, "myDeploymentID")
	require.NoError(t, err, "Failed to deploy")
}

func TestDeployWithoutFilePath(t *testing.T) {
	err := deploy(&httpClientMockDeploy{}, []string{}, false, false, "myDeploymentID")
	require.Error(t, err, "Expect error as no file path has been provided")
}

func TestDeployWithBadFilePath(t *testing.T) {
	err := deploy(&httpClientMockDeploy{}, []string{"fake.zip"}, false, false, "myDeploymentID")
	require.Error(t, err, "Expect error as file doesn't exist")
}

func TestDeployWithBadZip(t *testing.T) {
	err := deploy(&httpClientMockDeploy{}, []string{"./testdata/badzip.zip"}, false, false, "myDeploymentID")
	require.Error(t, err, "Expect error as file is not a correct zip")
}

func TestDeployWithHTTPFailure(t *testing.T) {
	err := deploy(&httpClientMockDeploy{testID: "fails"}, []string{"./testdata/deployment.zip"}, false, false, "myDeploymentID")
	require.Error(t, err, "Expected error due to HTTP failure")
}
