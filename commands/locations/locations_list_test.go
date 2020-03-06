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

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/rest"
)

// Mock client implementation for the list command

type httpClientMockList struct {
}

func (c *httpClientMockList) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, path, body)
}

func (c *httpClientMockList) Do(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()

	if req.URL.Path == "/locations" {
		locations := &rest.LocationCollection{Locations: []rest.AtomLink{{Rel: "location", Href: "/locations/locationOne", LinkType: rest.LinkRelHost}}}
		b, err := json.Marshal(locations)
		if err != nil {
			return nil, errors.New("Failed to build MockList http client response")
		}
		w.Write(b)
	}

	if req.URL.Path == "/locations/locationOne" {
		locationConfigProps := make(map[string]interface{})
		locationConfigProps["region"] = "us-east-2"
		locationConfigProps["regionbis"] = "us-east-3"

		locationConfigProps["security_groups"] = []string{"sg1", "sg2"}

		locationConfigProps["hosts"] = []map[string]interface{}{
			{
				"name": "host1",
				"connection": map[string]interface{}{
					"user": "john",
					"key":  "mykey.pem",
					"host": "1.2.3.4",
					"port": 22,
				},
			},
			{
				"name": "host2",
				"connection": map[string]interface{}{
					"user": "john",
					"key":  "mykey.pem",
					"host": "1.2.3.4",
					"port": 22,
				},
			},
		}
		locationConfig := &rest.LocationConfiguration{Name: "location3", Type: "aws", Properties: locationConfigProps}
		b, err := json.Marshal(locationConfig)
		if err != nil {
			return nil, errors.New("Failed to build Mock http client response")
		}
		w.Write(b)
	}

	return w.Result(), nil
}

func (c *httpClientMockList) Get(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockList) Head(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockList) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockList) PostForm(path string, data url.Values) (*http.Response, error) {
	return &http.Response{}, nil
}

func TestListLocations(t *testing.T) {
	err := listLocations(&httpClientMockList{})
	require.NoError(t, err, "Failed to get locations")
}
