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
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"

	"github.com/ystia/yorc/v4/config"
)

func testMetrics(t *testing.T, client *api.Client) {
	shutdownCh := make(chan struct{})
	defer close(shutdownCh)

	cfg := config.Configuration{WorkersNumber: 1}
	wg := &sync.WaitGroup{}
	dispatcher := NewDispatcher(cfg, shutdownCh, client, wg)

	// emit workers metrics
	dispatcher.emitWorkersMetrics()

	// emit taskExecution metrics with no tasks
	lastWarn := time.Now().Add(-6 * time.Minute)
	dispatcher.emitTaskExecutionsMetrics(client, &lastWarn)

	// create a task and emit taskExecutions metrics
	execID := "2c6a9f86-a63d-4774-9f2b-ed53f96349d7"
	taskID := "taskM"
	stepName := "stepM"
	createTaskExecutionKVWithKey(t, execID, "step", stepName)
	createWfStepStatusInitial(t, taskID, stepName)
	createTaskKVWithExecution(t, taskID)
	dispatcher.emitTaskExecutionsMetrics(client, &lastWarn)
}
