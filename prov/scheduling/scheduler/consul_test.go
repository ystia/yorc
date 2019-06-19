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

package scheduler

import (
	"testing"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulSchedulingPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)

	cfg := config.Configuration{
		HTTPAddress: "localhost",
		ServerID:    "0",
	}

	// Register the consul service
	chStop := make(chan struct{})
	consulutil.RegisterServerAsConsulService(cfg, client, chStop)

	// Start/Stop the scheduling manager
	Start(cfg, client)
	defer func() {
		Stop()
		srv.Stop()
	}()

	t.Run("groupScheduling", func(t *testing.T) {
		t.Run("testRegisterAction", func(t *testing.T) {
			testRegisterAction(t, client)
		})
		t.Run("testProceedScheduledAction", func(t *testing.T) {
			testProceedScheduledAction(t, client)
		})
		t.Run("testProceedScheduledActionWithFirstActionStillRunning", func(t *testing.T) {
			testProceedScheduledActionWithFirstActionStillRunning(t, client)
		})
		t.Run("testProceedScheduledActionWithBadStatusError", func(t *testing.T) {
			testProceedScheduledActionWithBadStatusError(t, client)
		})
		t.Run("testUnregisterAction", func(t *testing.T) {
			testUnregisterAction(t, client)
		})
	})
}
