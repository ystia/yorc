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

package collector

import (
	"testing"

	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulCollectorPackageTests(t *testing.T) {
	t.Parallel()
	log.SetDebug(true)

	srv, client := testutil.NewTestConsulInstance(t)
	defer srv.Stop()

	populateKV(t, srv)

	t.Run("groupCollector", func(t *testing.T) {
		t.Run("testResumeTask", func(t *testing.T) {
			testResumeTask(t, client)
		})
		t.Run("testRegisterTaskWithBigWorkflow", func(t *testing.T) {
			testRegisterTaskWithBigWorkflow(t, client)
		})
	})
}
