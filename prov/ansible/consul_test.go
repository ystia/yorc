package ansible

import (
	"testing"

	"novaforge.bull.com/starlings-janus/janus/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulAnsiblePackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()

	t.Run("groupAnsible", func(t *testing.T) {
		t.Run("TestExecutionOnNode", func(t *testing.T) {
			testExecutionOnNode(t, srv, kv)
		})
		t.Run("TestExecutionOnRelationshipSource", func(t *testing.T) {
			testExecutionOnRelationshipSource(t, srv, kv)
		})
		t.Run("TestExecutionOnRelationshipTarget", func(t *testing.T) {
			testExecutionOnRelationshipTarget(t, srv, kv)
		})
		t.Run("TestLogAnsibleOutputInConsul", func(t *testing.T) {
			testLogAnsibleOutputInConsul(t, kv)
		})
	})
}
