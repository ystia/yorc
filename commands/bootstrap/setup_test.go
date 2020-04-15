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

package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func Test_waitForConsulReadiness(t *testing.T) {
	type endpoint struct {
		started    bool
		httpStatus int
		text       string
	}
	tests := []struct {
		name          string
		endpoint      endpoint
		shouldSucceed bool
	}{
		{"NormalStartup", endpoint{true, 200, `"127.0.0.1:8300"`}, true},
		{"ConsulStartedButNotReady", endpoint{true, 200, `""`}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.endpoint.httpStatus)
				w.Write([]byte(tt.endpoint.text))
			}))
			if !tt.endpoint.started {
				// close now!
				ts.Close()
			} else {
				defer ts.Close()
			}
			c := make(chan struct{}, 0)
			go func() {
				waitForConsulReadiness(ts.URL)
				close(c)
			}()
			var timedOut bool
			select {
			case <-c:
			case <-time.After(6 * time.Second):
				timedOut = true
			}
			if tt.shouldSucceed == timedOut {
				t.Errorf("expecting waitForConsulReadiness() to succeed but it timedout")
			} else if !tt.shouldSucceed && !timedOut {
				t.Errorf("expecting waitForConsulReadiness() to timeout but it succeeded")
			}
		})
	}
}

func Test_getConsulLeader(t *testing.T) {
	type endpoint struct {
		started    bool
		httpStatus int
		text       string
	}
	tests := []struct {
		name     string
		endpoint endpoint
		want     string
		wantErr  bool
	}{
		{"NormalStartup", endpoint{true, 200, `"127.0.0.1:8300"`}, "127.0.0.1:8300", false},
		{"ConsulStartedButNotReady", endpoint{true, 200, `""`}, "", false},
		{"ConsulStartedButBadResponse", endpoint{true, 200, `qsdhjgy@àà*ùù$ugfbdnflv`}, "", false},
		{"ConsulStartedButBadStatus", endpoint{true, 500, ``}, "", true},
		{"ConsulNotStarted", endpoint{false, 0, ``}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.endpoint.httpStatus)
				w.Write([]byte(tt.endpoint.text))
			}))
			defer ts.Close()
			got, err := getConsulLeader(ts.URL)
			if (err != nil) != tt.wantErr {
				t.Errorf("getConsulLeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.endpoint.httpStatus == 500 {
				assert.ErrorContains(t, err, "500")
			}
			if got != tt.want {
				t.Errorf("getConsulLeader() = %v, want %v", got, tt.want)
			}
		})
	}
}
