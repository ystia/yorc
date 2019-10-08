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

package deployments

import (
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type httpClientMockExecCustom struct {
}

func (c *httpClientMockExecCustom) Do(req *http.Request) (*http.Response, error) {
	res := &httptest.ResponseRecorder{
		Code: 202,
	}
	return res.Result(), nil
}

func (c *httpClientMockExecCustom) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, path, body)
}

func (c *httpClientMockExecCustom) Get(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockExecCustom) Head(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockExecCustom) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientMockExecCustom) PostForm(path string, data url.Values) (*http.Response, error) {
	return &http.Response{}, nil
}

func TestExecuteCustomCommand(t *testing.T) {
	err := executeCustomCommand(&httpClientMockExecCustom{}, []string{"id"}, "", "node1", "custom", "custom", []string{"key1=\"[value1, value3]\"", "key2=\"value2\""})
	require.NoError(t, err, "Failed to execute custom command")
}

func TestExecuteCustomCommandWithoutInfo(t *testing.T) {
	err := executeCustomCommand(&httpClientMockExecCustom{}, []string{"id"}, "", "", "", "", []string{"key1=value1", "key2=value2"})
	require.Error(t, err, "Expect error as no info has been provided")
}

func TestExecuteCustomCommandWithoutID(t *testing.T) {
	err := executeCustomCommand(&httpClientMockExecCustom{}, []string{}, "", "node1", "custom-command", "interface", []string{})
	require.Error(t, err, "Expect error as no ID has been provided")
}
