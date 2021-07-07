// Copyright 2021 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ystia/yorc/v4/commands/deployments"
	"github.com/ystia/yorc/v4/config"
)

func Test_updateTaskStepState(t *testing.T) {

	s := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "fail") {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte("Some user error"))
			return
		}
		rw.WriteHeader(http.StatusOK)
	}))
	defer s.Close()

	deployments.ClientConfig = config.Client{
		YorcAPI: strings.Replace(s.URL, "http://", "", 1),
	}

	type args struct {
		deploymentID string
		taskID       string
		stepName     string
		statusStr    string
	}
	tests := []struct {
		name string
		args args
	}{
		{"TestOK", args{"depOK", "taskID", "step", "initial"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateTaskStepState(tt.args.deploymentID, tt.args.taskID, tt.args.stepName, tt.args.statusStr)
		})
	}
}
