package ansible

import (
	"testing"

	"novaforge.bull.com/starlings-janus/janus/log"

	"novaforge.bull.com/starlings-janus/janus/testutil"
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
}
