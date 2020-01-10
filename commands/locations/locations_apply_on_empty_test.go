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

package locations

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

// Mock client implementation for the apply command

type httpMockClientApplyOnEmpty struct {
}

func (c *httpMockClientApplyOnEmpty) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, path, body)
}

func (c *httpMockClientApplyOnEmpty) Do(req *http.Request) (*http.Response, error) {

	if req.Method == "POST" || req.Method == "PUT" {
		res := httptest.ResponseRecorder{Code: 201}
		return res.Result(), nil
	}

	w := httptest.NewRecorder()

	return w.Result(), nil
}

func (c *httpMockClientApplyOnEmpty) Get(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpMockClientApplyOnEmpty) Head(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpMockClientApplyOnEmpty) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpMockClientApplyOnEmpty) PostForm(path string, data url.Values) (*http.Response, error) {
	return &http.Response{}, nil
}

func TestLocationApply(t *testing.T) {
	err := applyLocationsConfig(&httpMockClientApplyOnEmpty{}, []string{"./testdata/locations.json"}, true)
	require.NoError(t, err, "Failed to apply locations config on empty list")
}
