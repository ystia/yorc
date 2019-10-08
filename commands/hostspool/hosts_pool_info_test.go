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
	"strings"
	"testing"
)

type httpClientMockInfo struct {
}

func (c *httpClientMockInfo) Do(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.String(), "fails") {
		return nil, errors.New("a failure occurs")
	}

	w := httptest.NewRecorder()

	host := rest.Host{Host: hostspool.Host{Name: "hostOne", Connection: hostspool.Connection{Host: "1.2.3.4", Password: "pass", User: "user1"}}, Links: []rest.AtomLink{{Href: "", LinkType: rest.LinkRelHost}}}
	b, err := json.Marshal(host)
	if err != nil {
		return nil, errors.New("failed to build http client mock response")
	}
	if strings.Contains(req.URL.String(), "bad_json") {
		w.WriteString("This is not json !!!")
	} else {
		w.Write(b)
	}
	return w.Result(), nil
}

func (c *httpClientMockInfo) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, path, body)
}

func (c *httpClientMockInfo) Get(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockInfo) Head(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockInfo) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockInfo) PostForm(path string, data url.Values) (*http.Response, error) {
	return &http.Response{}, nil
}

func TestHostInfo(t *testing.T) {
	err := hostInfo(&httpClientMockInfo{}, []string{"hostOne"}, "locationOne")
	require.NoError(t, err, "Failed to get locations")
}

func TestHostInfoWithoutHostname(t *testing.T) {
	err := hostInfo(&httpClientMockInfo{}, []string{}, "locationOne")
	require.Error(t, err, "Expected error as no hostname has been provided")
}

func TestHostInfoWithoutLocation(t *testing.T) {
	err := hostInfo(&httpClientMockInfo{}, []string{"hostOne"}, "")
	require.Error(t, err, "Expected error as no location has been provided")
}

func TestHostInfoWithHTTPFailure(t *testing.T) {
	err := hostInfo(&httpClientMockInfo{}, []string{"hostOne"}, "fails")
	require.Error(t, err, "Expected error due to HTTP failure")
}

func TestHostInfoWithJSONError(t *testing.T) {
	err := hostInfo(&httpClientMockInfo{}, []string{"hostOne"}, "bad_json")
	require.Error(t, err, "Expected error due to JSON error")
}
