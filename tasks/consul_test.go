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

package tasks

import (
	"testing"

	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulTasksPackageTests(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)

	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()

	populateKV(t, srv)

	t.Run("groupTasks", func(t *testing.T) {
		t.Run("TestGetTasksIdsForTarget", func(t *testing.T) {
			testGetTasksIdsForTarget(t, kv)
		})
		t.Run("TestGetTaskStatus", func(t *testing.T) {
			testGetTaskStatus(t, kv)
		})
		t.Run("TestGetTaskType", func(t *testing.T) {
			testGetTaskType(t, kv)
		})
		t.Run("TestGetTaskTarget", func(t *testing.T) {
			testGetTaskTarget(t, kv)
		})
		t.Run("TestTaskExists", func(t *testing.T) {
			testTaskExists(t, kv)
		})
		t.Run("TestCancelTask", func(t *testing.T) {
			testCancelTask(t, kv)
		})
		t.Run("TestTargetHasLivingTasks", func(t *testing.T) {
			testTargetHasLivingTasks(t, kv)
		})
		t.Run("TestGetTaskInput", func(t *testing.T) {
			testGetTaskInput(t, kv)
		})
		t.Run("TestGetInstances", func(t *testing.T) {
			testGetInstances(t, kv)
		})
		t.Run("TestGetTaskRelatedNodes", func(t *testing.T) {
			testGetTaskRelatedNodes(t, kv)
		})
		t.Run("TestIsTaskRelatedNode", func(t *testing.T) {
			testIsTaskRelatedNode(t, kv)
		})
		t.Run("testGetTaskRelatedWFSteps", func(t *testing.T) {
			testGetTaskRelatedWFSteps(t, kv)
		})
		t.Run("testUpdateTaskStepStatus", func(t *testing.T) {
			testUpdateTaskStepStatus(t, kv)
		})
		t.Run("testTaskStepExists", func(t *testing.T) {
			testTaskStepExists(t, kv)
		})
		t.Run("testGetTaskResultSet", func(t *testing.T) {
			testGetTaskResultSet(t, kv)
		})
		t.Run("testDeleteTask", func(t *testing.T) {
			testDeleteTask(t, kv)
		})
		t.Run("testGetQueryTaskIDs", func(t *testing.T) {
			testGetQueryTaskIDs(t, kv)
		})
		t.Run("TestIsStepRegistrationInProgress", func(t *testing.T) {
			testIsStepRegistrationInProgress(t, kv)
		})
	})
}
