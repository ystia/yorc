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

package ansible

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/testutil"
)

func newMockDockerClient(t *testing.T, doer func(*http.Request) (*http.Response, error)) *client.Client {
	// this is hacky due to the way NewClient behave in this version (waiting for )
	// will be improved soon https://github.com/moby/moby/blob/8bb5a28eed5eba5651c6e48eb401c03be938b4c1/client/client.go#L213 (v > v17.05.0)
	hc := &http.Client{
		Transport: &http.Transport{},
	}
	cli, err := client.NewClient("tcp://somewhere:42/api", "1.25", hc, nil)
	if err != nil {
		t.Fatal(err)
	}

	c := testutil.NewMockClient(doer)
	hc.Transport = c.Transport
	return cli
}

func newFailingDockerMockForPaths(t *testing.T, paths []string, status int) *client.Client {
	return newMockDockerClient(t, func(r *http.Request) (*http.Response, error) {
		if r == nil {
			return nil, errors.New("nil http request")
		}
		for _, p := range paths {
			if strings.Contains(r.URL.Path, p) {
				return &http.Response{StatusCode: status, Body: ioutil.NopCloser(bytes.NewReader([]byte("Failed")))}, nil
			}
		}

		return &http.Response{StatusCode: http.StatusOK}, nil
	})
}

func newJSONBodyResponseDockerMock(t *testing.T, p string, obj interface{}) *client.Client {
	return newMockDockerClient(t, func(r *http.Request) (*http.Response, error) {
		if r == nil {
			return nil, errors.New("nil http request")
		}
		resp := &http.Response{StatusCode: http.StatusOK}
		b, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		if strings.Contains(r.URL.Path, p) {
			resp.Body = ioutil.NopCloser(bytes.NewReader(b))
		}
		return resp, nil
	})
}

func Test_createSandbox(t *testing.T) {

	type args struct {
		cli          *client.Client
		sandboxCfg   *config.DockerSandbox
		deploymentID string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"FailOnPull", args{newFailingDockerMockForPaths(t, []string{"/images/create"}, 404), &config.DockerSandbox{Image: "busybox:latest"}, "d1"}, "", true},
		{"Success", args{newJSONBodyResponseDockerMock(t, "/containers/create", &container.ContainerCreateCreatedBody{ID: "myid"}), &config.DockerSandbox{Image: "busybox:latest"}, "d1"}, "myid", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cf := context.WithCancel(context.Background())
			defer cf()
			got, err := createSandbox(ctx, tt.args.cli, tt.args.sandboxCfg, tt.args.deploymentID)
			if (err != nil) != tt.wantErr {
				t.Errorf("createSandbox() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("createSandbox() = %v, want %v", got, tt.want)
			}
		})
	}
}
