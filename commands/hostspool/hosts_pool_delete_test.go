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
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type httpClientMockDelete struct {
}

func (c *httpClientMockDelete) Do(req *http.Request) (*http.Response, error) {
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
