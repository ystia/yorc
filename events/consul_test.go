package events

import (
	"testing"

	"novaforge.bull.com/starlings-janus/janus/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulEventsPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()

	t.Run("groupEvents", func(t *testing.T) {

		t.Run("TestRegisterLogsInConsul", func(t *testing.T) {
			testRegisterLogsInConsul(t, kv)
		})
	})
}
