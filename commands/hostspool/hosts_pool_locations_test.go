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
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/rest"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type httpClientMockLocations struct {
	testID string
}

func (c *httpClientMockLocations) Do(req *http.Request) (*http.Response, error) {
	if strings.Contains(c.testID, "fails") {
		return nil, errors.New("a failure occurs")
	}

	w := httptest.NewRecorder()
	locations := &rest.HostsPoolLocations{Locations: []string{"locationOne", "locationTwo", "locationThree"}}
	b, err := json.Marshal(locations)
	if err != nil {
		return nil, errors.New("failed to build http client mock response")
	}
	if strings.Contains(c.testID, "bad_json") {
		w.WriteString("This is not json !!!")
	} else {
		w.Write(b)
	}
	return w.Result(), nil
}

func (c *httpClientMockLocations) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, path, body)
}

func (c *httpClientMockLocations) Get(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockLocations) Head(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockLocations) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockLocations) PostForm(path string, data url.Values) (*http.Response, error) {
	return &http.Response{}, nil
}

func TestGetLocations(t *testing.T) {
	err := getLocations(&httpClientMockLocations{})
	require.NoError(t, err, "Failed to get locations")
}

func TestGetLocationsWithHTTPFailure(t *testing.T) {
	err := getLocations(&httpClientMockLocations{testID: "fails"})
	require.Error(t, err, "Expected error due to HTTP failure")
}

func TestGetLocationsWithJSONError(t *testing.T) {
	err := getLocations(&httpClientMockLocations{testID: "bad_json"})
	require.Error(t, err, "Expected error due to JSON error")
}
