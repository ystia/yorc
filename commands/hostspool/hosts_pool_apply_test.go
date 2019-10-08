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
	"github.com/ystia/yorc/v4/prov/hostspool"
	"github.com/ystia/yorc/v4/rest"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type httpClientMockApply struct {
}

func (c *httpClientMockApply) Do(req *http.Request) (*http.Response, error) {
	var res httptest.ResponseRecorder
	if req.Method == "POST" {
		res = httptest.ResponseRecorder{
			Code: 201,
		}
	} else if req.Method == "GET" {
		if req.URL.String() == "/hosts_pool/locationOne" {
			res = httptest.ResponseRecorder{
				Code: 204,
			}
		} else {
			w := httptest.NewRecorder()
			host := rest.Host{Host: hostspool.Host{Name: "hostOne", Connection: hostspool.Connection{Host: "1.2.3.4", Password: "pass", User: "user1"}}, Links: []rest.AtomLink{{Href: "", LinkType: rest.LinkRelHost}}}
			b, err := json.Marshal(host)
			if err != nil {
				return nil, errors.New("failed to build http client mock response")
			}
			w.Write(b)
			return w.Result(), nil
		}
	}

	return res.Result(), nil
}

func (c *httpClientMockApply) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, path, body)
}

func (c *httpClientMockApply) Get(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockApply) Head(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockApply) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockApply) PostForm(path string, data url.Values) (*http.Response, error) {
	return &http.Response{}, nil
}

func TestApplyHostsPoolConfig(t *testing.T) {
	err := applyHostsPoolConfig(&httpClientMockApply{}, []string{"./testdata/hosts_pool.yaml"}, "locationOne", true)
	require.NoError(t, err, "Failed to apply hosts pool config")
}

func TestApplyHostsPoolConfigWithoutLocation(t *testing.T) {
	err := applyHostsPoolConfig(&httpClientMockApply{}, []string{"./testdata/hosts_pool.yaml"}, "", true)
	require.Error(t, err, "Expected error as no location has been provided")
}

func TestApplyHostsPoolConfigWithNoFile(t *testing.T) {
	err := applyHostsPoolConfig(&httpClientMockApply{}, []string{}, "", true)
	require.Error(t, err, "Expected error as no file path has been provided")
}

func TestApplyHostsPoolConfigWithBadFilePath(t *testing.T) {
	err := applyHostsPoolConfig(&httpClientMockApply{}, []string{"./testdata/fake.yaml"}, "", true)
	require.Error(t, err, "Expected error as a bad file path has been provided")
}

func TestApplyHostsPoolConfigWithBadFile(t *testing.T) {
	err := applyHostsPoolConfig(&httpClientMockApply{}, []string{"./testdata/bad_hosts_pool.yaml"}, "", true)
	require.Error(t, err, "Expected error as host has no name")
}
