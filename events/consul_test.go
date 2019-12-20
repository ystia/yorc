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

package events

import (
	"os"
	"testing"

	"github.com/ystia/yorc/v4/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulEventsPackageTests(t *testing.T) {
	srv, _, workingDir := testutil.NewTestConsulInstance(t)
	defer func() {
		srv.Stop()
		os.RemoveAll(workingDir)
	}()

	t.Run("groupEvents", func(t *testing.T) {
		t.Run("TestConsulPubSubStatusChange", func(t *testing.T) {
			testConsulPubSubStatusChange(t)
		})
		t.Run("TestConsulPubSubNewEvents", func(t *testing.T) {
			testConsulPubSubNewEvents(t)
		})
		t.Run("TestConsulPubSubNewEventsTimeout", func(t *testing.T) {
			testConsulPubSubNewEventsTimeout(t)
		})
		t.Run("TestConsulPubSubNewEventsWithIndex", func(t *testing.T) {
			testConsulPubSubNewEventsWithIndex(t)
		})
		t.Run("TestConsulPubSubNewNodeEvents", func(t *testing.T) {
			testConsulPubSubNewNodeEvents(t)
		})
		t.Run("TestDeploymentStatusChange", func(t *testing.T) {
			testconsulDeploymentStatusChange(t)
		})
		t.Run("TestCustomCommandStatusChange", func(t *testing.T) {
			testconsulCustomCommandStatusChange(t)
		})
		t.Run("TestScalingStatusChange", func(t *testing.T) {
			testconsulScalingStatusChange(t)
		})
		t.Run("TestWorkflowStatusChange", func(t *testing.T) {
			testconsulWorkflowStatusChange(t)
		})
		t.Run("TestWorkflowStepStatusChange", func(t *testing.T) {
			testconsulWorkflowStepStatusChange(t)
		})
		t.Run("testAlienTaskStatusChange", func(t *testing.T) {
			testconsulAlienTaskStatusChange(t)
		})
		t.Run("TestGetStatusEvents", func(t *testing.T) {
			testconsulGetStatusEvents(t)
		})
		t.Run("TestGetLogs", func(t *testing.T) {
			testconsulGetLogs(t)
		})
		t.Run("TestRegisterLogsInConsul", func(t *testing.T) {
			testRegisterLogsInConsul(t)
		})
		t.Run("TestLogsSortedByTimestamp", func(t *testing.T) {
			testLogsSortedByTimestamp(t)
		})
		t.Run("TestAttributeValueChange", func(t *testing.T) {
			testconsulAttributeValueChange(t)
		})
		t.Run("TestPurgeDeploymentEvents", func(t *testing.T) {
			testPurgeDeploymentEvents(t)
		})
		t.Run("TestPurgeDeploymentLogs", func(t *testing.T) {
			testPurgeDeploymentLogs(t)
		})
	})
}
