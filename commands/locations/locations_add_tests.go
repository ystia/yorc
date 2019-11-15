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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/rest"
)

// Mock client implementation for the add location command

type httpMockClientAdd struct {
}

func (c *httpMockClientAdd) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, path, body)
}

func (c *httpMockClientAdd) Do(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()

	return w.Result(), nil
}

func (c *httpMockClientAdd) Get(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpMockClientAdd) Head(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpMockClientAdd) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpMockClientAdd) PostForm(path string, data url.Values) (*http.Response, error) {
	return &http.Response{}, nil
}

func TestLocationAdd(t *testing.T) {
	locationConfigProps := make(map[string]interface{})
	locationConfigProps["region"] = "us-east-2"
	locationConfig := &rest.LocationConfiguration{Name: "location3", Type: "aws", Properties: locationConfigProps}
	tmp, err := json.Marshal(locationConfig)
	jsonParam := string(tmp)
	err = addLocation(&httpMockClientAdd{}, jsonParam)
	require.NoError(t, err, "Failed to add location")
}

func TestLocationAddNoName(t *testing.T) {
	locationConfigProps := make(map[string]interface{})
	locationConfigProps["region"] = "us-east-2"
	locationConfig := &rest.LocationConfiguration{Type: "aws", Properties: locationConfigProps}
	tmp, err := json.Marshal(locationConfig)
	jsonParam := string(tmp)
	err = addLocation(&httpMockClientAdd{}, jsonParam)
	require.Error(t, err, "Expected error due to missing location name")
}

func TestLocationAddWithFailure(t *testing.T) {
	var jsonParam string
	jsonParam = "test"
	err := addLocation(&httpMockClientAdd{}, jsonParam)
	require.Error(t, err, "Expected error due to malformed JSON")
}
