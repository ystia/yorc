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

package rest

import (
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/helper/sshutil"
	"github.com/ystia/yorc/prov/hostspool"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/testutil"
	"golang.org/x/crypto/ssh"
	"net/http"
	"net/http/httptest"
)

type mockSSHClient struct {
	config *ssh.ClientConfig
}

func (m *mockSSHClient) RunCommand(string) (string, error) {
	if m.config != nil && m.config.User == "fail" {
		return "", errors.Errorf("Failed to connect")
	}

	return "ok", nil
}

var mockSSHClientFactory = func(config *ssh.ClientConfig, conn hostspool.Connection) sshutil.Client {
	return &mockSSHClient{config}
}

func newTestHTTPRouter(client *api.Client, req *http.Request) *http.Response {
	router := newRouter()

	httpSrv := &Server{
		router:         router,
		consulClient:   client,
		hostsPoolMgr:   hostspool.NewManagerWithSSHFactory(client, mockSSHClientFactory),
		tasksCollector: tasks_old.NewCollector(client),
		config:         config.Configuration{},
	}
	httpSrv.registerHandlers()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Result()
}

func TestRunConsulRestPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	defer srv.Stop()

	t.Run("groupRest", func(t *testing.T) {
		t.Run("testHostsPoolHandlers", func(t *testing.T) {
			testHostsPoolHandlers(t, client, srv)
		})
		t.Run("testSSLRest", func(t *testing.T) {
			testSSLREST(t, client, srv)
		})
	})
}
