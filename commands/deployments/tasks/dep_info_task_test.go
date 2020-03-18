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

package tasks

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
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

	task := rest.Task{
		ID:           "taskID",
		TargetID:     "deploymentID",
		Type:         "taskType",
		Status:       "Error",
		ErrorMessage: "my error message",
		ResultSet:    nil,
	}
	b, err := json.Marshal(task)
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

func TestTaskInfo(t *testing.T) {
	err := taskInfo(&httpClientMockInfo{}, []string{"deploymentID", "taskID"}, false)
	require.NoError(t, err, "Failed to get task info")
}

func TestTaskInfoWithoutTaskID(t *testing.T) {
	err := taskInfo(&httpClientMockInfo{}, []string{"deploymentID"}, false)
	require.Error(t, err, "Expected error due to missing argument")
}

func TestHostInfoWithHTTPFailure(t *testing.T) {
	err := taskInfo(&httpClientMockInfo{}, []string{"fails", "taskID"}, false)
	require.Error(t, err, "Expected error due to HTTP failure")
}

func TestHostInfoWithJSONError(t *testing.T) {
	err := taskInfo(&httpClientMockInfo{}, []string{"bad_json", "taskID"}, false)
	require.Error(t, err, "Expected error due to JSON error")
}
