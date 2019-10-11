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

package hostspool

import (
	"errors"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type httpClientMockDelete struct {
	testID string
}

func (c *httpClientMockDelete) Do(req *http.Request) (*http.Response, error) {
	if strings.Contains(c.testID, "fails") {
		return nil, errors.New("a failure occurs")
	}
	return httptest.NewRecorder().Result(), nil
}

func (c *httpClientMockDelete) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, path, body)
}

func (c *httpClientMockDelete) Get(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockDelete) Head(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockDelete) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockDelete) PostForm(path string, data url.Values) (*http.Response, error) {
	return &http.Response{}, nil
}

func TestDeleteHost(t *testing.T) {
	err := deleteHost(&httpClientMockDelete{}, []string{"hostOne", "hostTwo"}, "locationOne")
	require.NoError(t, err, "Failed to add host")
}

func TestDeleteHostWithoutHostname(t *testing.T) {
	err := deleteHost(&httpClientMockDelete{}, []string{}, "locationOne")
	require.Error(t, err, "Expected error as no hostname has been provided")
}

func TestDeleteHostWithoutLocation(t *testing.T) {
	err := deleteHost(&httpClientMockDelete{}, []string{"hostOne", "hostTwo"}, "")
	require.Error(t, err, "Expected error as no location has been provided")
}

func TestDeleteHostWithHTTPFailure(t *testing.T) {
	err := deleteHost(&httpClientMockDelete{testID: "fails"}, []string{"hostOne", "hostTwo"}, "fails")
	require.Error(t, err, "Expected error due to HTTP failure")
}
