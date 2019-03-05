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

package ansible

import (
	"testing"

	"github.com/ystia/yorc/v3/log"
	"github.com/ystia/yorc/v3/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulAnsiblePackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()
	log.SetDebug(true)
	t.Run("TestExecution", func(t *testing.T) {
		testExecution(t, srv, kv)
	})
	t.Run("TestLogAnsibleOutputInConsul", func(t *testing.T) {
		testLogAnsibleOutputInConsul(t, kv)
	})
	t.Run("TestLogAnsibleOutputInConsulFromScript", func(t *testing.T) {
		testLogAnsibleOutputInConsulFromScript(t, kv)
	})
	t.Run("TestLogAnsibleOutputInConsulFromScriptFailure", func(t *testing.T) {
		testLogAnsibleOutputInConsulFromScriptFailure(t, kv)
	})
}
