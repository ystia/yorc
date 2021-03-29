// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ystia/yorc/v4/rest"
)

type mockHTTPClient struct {
	GetFunc        func(path string) (*http.Response, error)
	HeadFunc       func(path string) (*http.Response, error)
	PostFunc       func(path string, contentType string, body io.Reader) (*http.Response, error)
	PostFormFunc   func(path string, data url.Values) (*http.Response, error)
	NewRequestFunc func(method, path string, body io.Reader) (*http.Request, error)
	DoFunc         func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Get(path string) (*http.Response, error) {
	if m.GetFunc == nil {
		panic("missing mock test function GetFunc")
	}
	return m.GetFunc(path)
}
func (m *mockHTTPClient) Head(path string) (*http.Response, error) {
	if m.HeadFunc == nil {
		panic("missing mock test function HeadFunc")
	}
	return m.HeadFunc(path)
}
func (m *mockHTTPClient) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	if m.PostFunc == nil {
		panic("missing mock test function PostFunc")
	}
	return m.PostFunc(path, contentType, body)
}
func (m *mockHTTPClient) PostForm(path string, data url.Values) (*http.Response, error) {
	if m.PostFormFunc == nil {
		panic("missing mock test function PostFormFunc")
	}
	return m.PostFormFunc(path, data)
}

func (m *mockHTTPClient) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	if m.NewRequestFunc != nil {
		return m.NewRequestFunc(method, path, body)
	}
	return httptest.NewRequest(method, path, body), nil
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc == nil {
		panic("missing mock test function DoFunc")
	}
	return m.DoFunc(req)
}

func Test_postPurgeRequest(t *testing.T) {

	type args struct {
		doRequest func(req *http.Request) (*http.Response, error)
		force     bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"OkNoForce", args{doRequest: func(req *http.Request) (*http.Response, error) {
			errs := new(rest.Errors)
			b, err := json.Marshal(errs)
			if err != nil {
				return nil, err
			}
			res := &httptest.ResponseRecorder{
				Code: 200,
				Body: bytes.NewBuffer(b),
			}

			return res.Result(), nil
		}, force: false}, false},
		{"ForceWithErrors", args{doRequest: func(req *http.Request) (*http.Response, error) {
			errs := &rest.Errors{
				Errors: []*rest.Error{
					{ID: "internal_server_error", Status: 500, Title: "Internal Server Error", Detail: "error 1"},
					{ID: "internal_server_error", Status: 500, Title: "Internal Server Error", Detail: "error 2"},
					{ID: "internal_server_error", Status: 500, Title: "Internal Server Error", Detail: "error 3"},
				},
			}
			b, err := json.Marshal(errs)
			if err != nil {
				return nil, err
			}
			res := &httptest.ResponseRecorder{
				Code: 500,
				Body: bytes.NewBuffer(b),
			}

			return res.Result(), nil
		}, force: true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockHTTPClient{
				DoFunc: tt.args.doRequest,
			}
			if err := postPurgeRequest(mockClient, "deploymentID", tt.args.force); (err != nil) != tt.wantErr {
				t.Errorf("postPurgeRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
