package deployments

import (
	"testing"

	"novaforge.bull.com/starlings-janus/janus/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulDeploymentsPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()

	t.Run("groupDeploymentsArtifacts", func(t *testing.T) {
		t.Run("testArtifacts", func(t *testing.T) {
			testArtifacts(t, srv, kv)
		})
		t.Run("testCapabilities", func(t *testing.T) {
			testCapabilities(t, srv, kv)
		})
		t.Run("testDefinitionStore", func(t *testing.T) {
			testDefinitionStore(t, kv)
		})
		t.Run("testDeploymentNodes", func(t *testing.T) {
			testDeploymentNodes(t, srv, kv)
		})
		t.Run("testRequirements", func(t *testing.T) {
			testRequirements(t, srv, kv)
		})
		t.Run("testResolver", func(t *testing.T) {
			testResolver(t, kv)
		})
		t.Run("testGetTypePropertyDataType", func(t *testing.T) {
			testGetTypePropertyDataType(t, kv)
		})
		t.Run("testGetNestedDataType", func(t *testing.T) {
			testGetNestedDataType(t, kv)
		})
		t.Run("testReadComplexVA", func(t *testing.T) {
			testReadComplexVA(t, kv)
		})
		t.Run("testIssueGetEmptyPropRel", func(t *testing.T) {
			testIssueGetEmptyPropRel(t, kv)
		})
		t.Run("testRelationshipWorkflow", func(t *testing.T) {
			testRelationshipWorkflow(t, kv)
		})
		t.Run("testGlobalInputs", func(t *testing.T) {
			testGlobalInputs(t, kv)
		})
	})
}
