package aws

import (
	"testing"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulAWSPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()

	// AWS infrastructure config
	cfg := config.Configuration{
		Infrastructures: map[string]config.InfrastructureConfig{
			infrastructureName: {
				"region":     "us-east-2",
				"access_key": "test",
				"secret_key": "test",
			}}}

	t.Run("groupAWS", func(t *testing.T) {
		t.Run("simpleAWSInstance", func(t *testing.T) {
			testSimpleAWSInstance(t, kv, cfg)
		})
		t.Run("simpleAWSInstanceFailed", func(t *testing.T) {
			testSimpleAWSInstanceFailed(t, kv, cfg)
		})
		t.Run("simpleAWSInstanceWithEIP", func(t *testing.T) {
			testSimpleAWSInstanceWithEIP(t, kv, cfg)
		})
		t.Run("simpleAWSInstanceWithProvidedEIP", func(t *testing.T) {
			testSimpleAWSInstanceWithProvidedEIP(t, kv, cfg)
		})
		t.Run("simpleAWSInstanceWithListOfProvidedEIP", func(t *testing.T) {
			testSimpleAWSInstanceWithListOfProvidedEIP(t, kv, cfg)
		})
		t.Run("simpleAWSInstanceWithListOfProvidedEIP2", func(t *testing.T) {
			testSimpleAWSInstanceWithNotEnoughProvidedEIPS(t, kv, cfg)
		})
		t.Run("simpleAWSInstanceWithNoDeleteVolumeOnTermination", func(t *testing.T) {
			testSimpleAWSInstanceWithNoDeleteVolumeOnTermination(t, kv, cfg)
		})
		t.Run("simpleAWSInstanceWithMalformedEIP", func(t *testing.T) {
			testSimpleAWSInstanceWithMalformedEIP(t, kv, cfg)
		})

	})
}
