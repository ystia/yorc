package aws

import (
	"testing"

	"novaforge.bull.com/starlings-janus/janus/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulAWSPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()

	t.Run("groupAWS", func(t *testing.T) {
		t.Run("simpleAWSInstance", func(t *testing.T) {
			testSimpleAWSInstance(t, kv)
		})
		t.Run("simpleAWSInstanceFailed", func(t *testing.T) {
			testSimpleAWSInstanceFailed(t, kv)
		})
	})
}
