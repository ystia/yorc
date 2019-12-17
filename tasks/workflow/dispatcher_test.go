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
	"github.com/stretchr/testify/assert"

	"github.com/ystia/yorc/v4/config"
)

func testDeleteExecutionTreeSamePrefix(t *testing.T, client *api.Client) {
	shutdownCh := make(chan struct{})
	defer close(shutdownCh)

	cfg := config.Configuration{WorkersNumber: 1}
	wg := &sync.WaitGroup{}
	dispatcher := NewDispatcher(cfg, shutdownCh, client, wg)

	createTaskExecutionKVWithKey(t, "testDeleteExecutionTreeSamePrefixExecID1", "somekey", "val")
	createTaskExecutionKVWithKey(t, "testDeleteExecutionTreeSamePrefixExecID11", "somekey", "val")

	time.Sleep(2 * time.Second)

	dispatcher.deleteExecutionTree("testDeleteExecutionTreeSamePrefixExecID1")

	actualValue, err := getExecutionKeyValue("testDeleteExecutionTreeSamePrefixExecID11", "somekey")
	assert.NoError(t, err)
	assert.Equal(t, "val", actualValue)

}
