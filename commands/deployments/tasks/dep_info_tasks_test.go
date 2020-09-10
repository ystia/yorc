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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/rest"
)

type httpClientInfoTask struct {
}

func (c *httpClientInfoTask) Do(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()

	task := rest.Task{
		ID:       "task123",
		TargetID: "deployment123",
		Type:     "CustomWorkflow",
		Status:   "DONE",
		Outputs: map[string]string{
			"outputOne":   "valOne",
			"outputTwo":   "valTwo",
			"outputThree": "valThree",
		},
	}
	b, err := json.Marshal(task)
	if err != nil {
		return nil, errors.New("failed to build http client mock response")
	}
	w.Write(b)
	return w.Result(), nil
}

func (c *httpClientInfoTask) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, path, body)
}

func (c *httpClientInfoTask) Get(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientInfoTask) Head(path string) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientInfoTask) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return &http.Response{}, nil
}

func (c *httpClientInfoTask) PostForm(path string, data url.Values) (*http.Response, error) {
	return &http.Response{}, nil
}

func TestGetTaskInfo(t *testing.T) {
	err := taskInfo(&httpClientInfoTask{}, []string{"deployment123", "task123"}, false)
	require.NoError(t, err, "Failed to get info task")
}
