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

package hostspool

import (
	"testing"

	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulHostsPoolPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	defer srv.Stop()
	log.SetDebug(true)
	t.Run("TestConsulManagerAdd", func(t *testing.T) {
		testConsulManagerAdd(t, client)
	})
	t.Run("TestConsulManagerRemove", func(t *testing.T) {
		testConsulManagerRemove(t, client)
	})
	t.Run("TestConsulManagerAddLabels", func(t *testing.T) {
		testConsulManagerAddLabels(t, client)
	})
	t.Run("TestConsulManagerRemoveLabels", func(t *testing.T) {
		testConsulManagerRemoveLabels(t, client)
	})
	t.Run("TestConsulManagerConcurrency", func(t *testing.T) {
		testConsulManagerConcurrency(t, client)
	})
	t.Run("TestConsulManagerUpdateConnection", func(t *testing.T) {
		testConsulManagerUpdateConn(t, client)
	})
	t.Run("TestConsulManagerList", func(t *testing.T) {
		testConsulManagerList(t, client)
	})
	t.Run("TestConsulManagerGetHost", func(t *testing.T) {
		testConsulManagerGetHost(t, client)
	})
	t.Run("TestConsulManagerApply", func(t *testing.T) {
		testConsulManagerApply(t, client)
	})
	t.Run("testConsulManagerApplyErrorNoName", func(t *testing.T) {
		testConsulManagerApplyErrorNoName(t, client)
	})
	t.Run("testConsulManagerApplyErrorDuplicateName", func(t *testing.T) {
		testConsulManagerApplyErrorDuplicateName(t, client)
	})
	t.Run("testConsulManagerApplyErrorDeleteAllocatedHost", func(t *testing.T) {
		testConsulManagerApplyErrorDeleteAllocatedHost(t, client)
	})
	t.Run("testConsulManagerApplyErrorOutdatedCheckpoint", func(t *testing.T) {
		testConsulManagerApplyErrorOutdatedCheckpoint(t, client)
	})
	t.Run("testConsulManagerApplyBadConnection", func(t *testing.T) {
		testConsulManagerApplyBadConnection(t, client)
	})
	t.Run("testConsulManagerAllocateConcurrency", func(t *testing.T) {
		testConsulManagerAllocateConcurrency(t, client)
	})
}
