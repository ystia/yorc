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

// Mock client implementation for the apply command

type httpMockClientApply struct {
}

func (c *httpMockClientApply) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, path, body)
}

func (c *httpMockClientApply) Do(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()

	if req.URL.Path == "/locations" {
		locations := &rest.LocationCollection{Locations: []rest.AtomLink{
			{Rel: "location", Href: "/locations/locationOne", LinkType: rest.LinkRelHost},
			{Rel: "location", Href: "/locations/location1", LinkType: rest.LinkRelHost},
		}}
		b, err := json.Marshal(locations)
		if err != nil {
			return nil, errors.New("Failed to build MockList http client response")
		}
		w.Write(b)
	}

	if req.URL.Path == "/locations/locationOne" {
		locationConfigProps := make(map[string]interface{})
		locationConfigProps["region"] = "us-east-2"
		locationConfig := &rest.LocationConfiguration{Name: "locationOne", Type: "aws", Properties: locationConfigProps}
		b, err := json.Marshal(locationConfig)
		if err != nil {
			return nil, errors.New("Failed to build Mock http client response")
		}
		w.Write(b)
	}

	if req.URL.Path == "/locations/location1" {
		locationConfigProps := make(map[string]interface{})
		locationConfigProps["p1"] = "v2"
		locationConfig := &rest.LocationConfiguration{Name: "location1", Type: "openstack1", Properties: locationConfigProps}
		b, err := json.Marshal(locationConfig)
		if err != nil {
			return nil, errors.New("Failed to build Mock http client response")
		}
		w.Write(b)
	}

	return w.Result(), nil
}

func (c *httpMockClientApply) Get(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpMockClientApply) Head(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpMockClientApply) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpMockClientApply) PostForm(path string, data url.Values) (*http.Response, error) {
	return &http.Response{}, nil
}

func TestLocationApplyWithNoPath(t *testing.T) {
	err := applyLocationsConfig(&httpMockClientApply{}, []string{}, true)
	require.Error(t, err, "Expecting a path to a file (got 0 parameters)")
}

func TestLocationApplyWithWrongPath(t *testing.T) {
	err := applyLocationsConfig(&httpMockClientApply{}, []string{"./testdata/fake.json"}, true)
	require.Error(t, err, "no such file or directory")
}
func TestLocationApplyWithDirPath(t *testing.T) {
	err := applyLocationsConfig(&httpMockClientApply{}, []string{"./testdata"}, true)
	require.Error(t, err, "Expecting a path to a file")
}

func TestLocationApply(t *testing.T) {
	err := applyLocationsConfig(&httpMockClientApply{}, []string{"./testdata/locations.json"}, true)
	require.NoError(t, err, "Failed to apply locations config")
}
