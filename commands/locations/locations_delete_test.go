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

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// Mock client implementation for the update command

type httpMockClientDelete struct {
}

func (c *httpMockClientDelete) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, path, body)
}

func (c *httpMockClientDelete) Do(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()

	if req.URL.Path == "/locations/locerror" {
		return nil, errors.New("Failed to do delete")
	}

	return w.Result(), nil
}

func (c *httpMockClientDelete) Get(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpMockClientDelete) Head(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpMockClientDelete) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpMockClientDelete) PostForm(path string, data url.Values) (*http.Response, error) {
	return &http.Response{}, nil
}

func TestLocationDelete(t *testing.T) {
	//err := deleteLocation(&httpMockClientDelete{}, []string{"location3"})
	err := deleteLocation(&httpMockClientDelete{}, []string{"loc3"})
	require.NoError(t, err, "Failed to delete location")
}

func TestLocationDeleteNoName(t *testing.T) {
	err := deleteLocation(&httpMockClientDelete{}, []string{})
	require.Error(t, err, "Expecting one location name (got 0 parameters)")
}
func TestLocationDeleteError(t *testing.T) {
	err := deleteLocation(&httpMockClientDelete{}, []string{"locerror"})
	require.Error(t, err, "Expecting one location name (got 0 parameters)")
}
