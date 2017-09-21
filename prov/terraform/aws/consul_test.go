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
		t.Run("simpleAWSInstanceWithEIP", func(t *testing.T) {
			testSimpleAWSInstanceWithEIP(t, kv)
		})
		t.Run("simpleAWSInstanceWithProvidedEIP", func(t *testing.T) {
			testSimpleAWSInstanceWithProvidedEIP(t, kv)
		})
		t.Run("simpleAWSInstanceWithListOfProvidedEIP", func(t *testing.T) {
			testSimpleAWSInstanceWithListOfProvidedEIP(t, kv)
		})
		t.Run("simpleAWSInstanceWithListOfProvidedEIP2", func(t *testing.T) {
			testSimpleAWSInstanceWithNotEnoughProvidedEIPS(t, kv)
		})

	})
}
