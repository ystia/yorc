package workflow

import (
	"testing"

	"novaforge.bull.com/starlings-janus/janus/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulWorkflowPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()

	t.Run("groupWorkflow", func(t *testing.T) {
		t.Run("testReadStepWithNext", func(t *testing.T) {
			testReadStepWithNext(t, srv, kv)
		})
		t.Run("testReadStepFromConsul", func(t *testing.T) {
			testReadStepFromConsul(t, srv, kv)
		})
		t.Run("testReadWorkFlowFromConsul", func(t *testing.T) {
			testReadWorkFlowFromConsul(t, srv, kv)
		})
	})

	// Run this test after previous to avoid failing them
	t.Run("testReadStepFromConsulFailing", func(t *testing.T) {
		testReadStepFromConsulFailing(t, srv, kv)
	})
}
