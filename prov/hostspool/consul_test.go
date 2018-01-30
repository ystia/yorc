package hostspool

import (
	"testing"

	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/testutil"
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
	t.Run("TestConsulManagerAddTags", func(t *testing.T) {
		testConsulManagerAddTags(t, client)
	})
	t.Run("TestConsulManagerRemoveTags", func(t *testing.T) {
		testConsulManagerRemoveTags(t, client)
	})
	t.Run("TestConsulManagerConcurrency", func(t *testing.T) {
		testConsulManagerConcurrency(t, client)
	})
}
