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
		t.Run("TestConsulPubSubStatusChange", func(t *testing.T) {
			testConsulPubSubStatusChange(t, kv)
		})
		t.Run("TestConsulPubSubNewEvents", func(t *testing.T) {
			testConsulPubSubNewEvents(t, kv)
		})
		t.Run("TestConsulPubSubNewEventsTimeout", func(t *testing.T) {
			testConsulPubSubNewEventsTimeout(t, kv)
		})
		t.Run("TestConsulPubSubNewEventsWithIndex", func(t *testing.T) {
			testConsulPubSubNewEventsWithIndex(t, kv)
		})
		t.Run("TestConsulPubSubNewNodeEvents", func(t *testing.T) {
			testConsulPubSubNewNodeEvents(t, kv)
		})
		t.Run("TestDeploymentStatusChange", func(t *testing.T) {
			testconsulDeploymentStatusChange(t, kv)
		})
		t.Run("TestCustomCommandStatusChange", func(t *testing.T) {
			testconsulCustomCommandStatusChange(t, kv)
		})
		t.Run("TestScalingStatusChange", func(t *testing.T) {
			testconsulScalingStatusChange(t, kv)
		})
		t.Run("TestWorkflowStatusChange", func(t *testing.T) {
			testconsulWorkflowStatusChange(t, kv)
		})
		t.Run("TestGetStatusEvents", func(t *testing.T) {
			testconsulGetStatusEvents(t, kv)
		})
		t.Run("TestGetLogs", func(t *testing.T) {
			testconsulGetLogs(t, kv)
		})
	})
}
