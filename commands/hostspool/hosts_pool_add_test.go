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

type httpClientMockAdd struct {
}

func (c *httpClientMockAdd) Do(req *http.Request) (*http.Response, error) {
	res := &httptest.ResponseRecorder{
		Code: 201,
	}
	return res.Result(), nil
}

func (c *httpClientMockAdd) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, path, body)
}

func (c *httpClientMockAdd) Get(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockAdd) Head(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockAdd) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockAdd) PostForm(path string, data url.Values) (*http.Response, error) {
	return &http.Response{}, nil
}

func TestAddHost(t *testing.T) {
	err := addHost(&httpClientMockAdd{}, []string{"hostOne"}, "locationOne", "", "", "pass", "userOne", "1.2.3.1", 22, []string{"label1=value1", "label2=value2", "label3=value3"})
	require.NoError(t, err, "Failed to add host")
}

func TestAddHostWithoutHostname(t *testing.T) {
	err := addHost(&httpClientMockAdd{}, []string{}, "locationOne", "", "", "pass", "userOne", "1.2.3.1", 22, []string{"label1=value1", "label2=value2", "label3=value3"})
	require.Error(t, err, "Expected error as no hostname has been provided")
}

func TestAddHostWithoutLocation(t *testing.T) {
	err := addHost(&httpClientMockAdd{}, []string{"hostOne"}, "", "", "", "pass", "userOne", "1.2.3.1", 22, []string{"label1=value1", "label2=value2", "label3=value3"})
	require.Error(t, err, "Expected error as no location has been provided")
}

func TestAddHostWithoutPassowrd(t *testing.T) {
	err := addHost(&httpClientMockAdd{}, []string{"hostOne"}, "locationOne", "", "", "", "userOne", "1.2.3.1", 22, []string{"label1=value1", "label2=value2", "label3=value3"})
	require.Error(t, err, "Expected error as no location has been provided")
}
