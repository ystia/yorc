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

package workflow

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/consulutil"
)

func testDeleteExecutionTreeSamePrefix(t *testing.T, client *api.Client) {
	shutdownCh := make(chan struct{})
	defer close(shutdownCh)

	cfg := config.Configuration{WorkersNumber: 1}
	wg := &sync.WaitGroup{}
	dispatcher := NewDispatcher(cfg, shutdownCh, client, wg)

	createTaskExecutionKVWithKey(t, "testDeleteExecutionTreeSamePrefixExecID1", "somekey", "val")
	createTaskExecutionKVWithKey(t, "testDeleteExecutionTreeSamePrefixExecID11", "somekey", "val")

	dispatcher.deleteExecutionTree("testDeleteExecutionTreeSamePrefixExecID1")

	actualValue, err := getExecutionKeyValue("testDeleteExecutionTreeSamePrefixExecID11", "somekey")
	assert.NoError(t, err)
	assert.Equal(t, "val", actualValue)

}

type workerMock struct{}

func createWorkerFuncMock(d *Dispatcher, tchan chan *taskExecution) {

	d.createWorkerFunc = func(d2 *Dispatcher) {
		if tchan != nil {
			d2.WorkerPool <- tchan
		}
	}

}

func testDispatcherRun(t *testing.T, srv *testutil.TestServer, client *api.Client) {

	shutdownCh := make(chan struct{})
	defer close(shutdownCh)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)
	cfg := config.Configuration{
		WorkersNumber: 1,
		HTTPAddress:   u.Hostname(),
		HTTPPort:      port,
		ServerID:      "0",
		Consul: config.Consul{
			Address:        srv.HTTPAddr,
			PubMaxRoutines: config.DefaultConsulPubMaxRoutines,
		},
		Tasks: config.Tasks{
			Dispatcher: config.Dispatcher{
				LongPollWaitTime:   50 * time.Millisecond,
				LockWaitTime:       5 * time.Millisecond,
				MetricsRefreshTime: 50 * time.Millisecond,
			},
		},
	}
	// Register the consul service
	err = consulutil.RegisterServerAsConsulService(cfg, client, shutdownCh)
	if err != nil {
		t.Fatalf("failed to setup Yorc Consul service %v", err)
	}
	consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, "t1", "status"), "0")
	consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, "t1", "targetId"), "d1")
	consulutil.StoreConsulKeyAsString(path.Join(consulutil.TasksPrefix, "t1", "type"), "5")
	creationDate, err := time.Now().MarshalBinary()
	if err != nil {
		t.Fatalf("failed to generate task creation date %v", err)
	}
	consulutil.StoreConsulKey(path.Join(consulutil.TasksPrefix, "t1", "creationDate"), creationDate)

	createTaskExecutionKVWithKey(t, "testDispatcherExec", "taskID", "t1")

	wg := &sync.WaitGroup{}
	dispatcher := NewDispatcher(cfg, shutdownCh, client, wg)
	tChan := make(chan *taskExecution)
	createWorkerFuncMock(dispatcher, tChan)

	go dispatcher.Run()

	select {
	case tExec := <-tChan:
		t.Logf("Got taskexecution %+v", tExec)

	case <-time.After(15 * time.Second):
		require.Fail(t, "timeout awaiting dispatcher to take execution")
	}
}
