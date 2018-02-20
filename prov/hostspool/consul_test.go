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
}
