// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deployments

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v3/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulDeploymentsPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()

	t.Run("groupDeployments", func(t *testing.T) {
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
		t.Run("testInlineWorkflow", func(t *testing.T) {
			testInlineWorkflow(t, kv)
		})
		t.Run("testCheckCycleInNestedWorkflows", func(t *testing.T) {
			testCheckCycleInNestedWorkflows(t, kv)
		})
		t.Run("testGetCapabilityProperties", func(t *testing.T) {
			testGetCapabilityProperties(t, kv)
		})
		t.Run("testSubstitutionServiceCapabilityMappings", func(t *testing.T) {
			testSubstitutionServiceCapabilityMappings(t, kv)
		})
		t.Run("testSubstitutionServiceRequirementMappings", func(t *testing.T) {
			testSubstitutionServiceRequirementMappings(t, kv)
		})
		t.Run("testSubstitutionClientDirective", func(t *testing.T) {
			testSubstitutionClientDirective(t, kv)
		})
		t.Run("testSubstitutionClientServiceInstance", func(t *testing.T) {
			testSubstitutionClientServiceInstance(t, kv)
		})
		t.Run("TestOperationImplementationArtifact", func(t *testing.T) {
			testOperationImplementationArtifact(t, kv)
		})
		t.Run("TestOperationHost", func(t *testing.T) {
			testOperationHost(t, kv)
		})
		t.Run("testIssueGetEmptyPropOnRelationship", func(t *testing.T) {
			testIssueGetEmptyPropOnRelationship(t, kv)
		})

		t.Run("testTopologyUpdate", func(t *testing.T) {
			testTopologyUpdate(t, kv)
		})
		t.Run("testRepositories", func(t *testing.T) {
			testRepositories(t, kv)
		})
		t.Run("testPurgedDeployments", func(t *testing.T) {
			testPurgedDeployments(t, client)
		})
	})

	t.Run("CommonsTestsOn_test_topology.yml", func(t *testing.T) {
		deploymentID := testutil.BuildDeploymentID(t)
		err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/test_topology.yml")
		require.NoError(t, err)

		t.Run("TestNodeHasAttribute", func(t *testing.T) {
			testNodeHasAttribute(t, kv, deploymentID)
		})
		t.Run("TestNodeHasProperty", func(t *testing.T) {
			testNodeHasProperty(t, kv, deploymentID)
		})
		t.Run("TestTopologyTemplateMetadata", func(t *testing.T) {
			testTopologyTemplateMetadata(t, kv, deploymentID)
		})
		t.Run("TestAttributeNotifications", func(t *testing.T) {
			testAttributeNotifications(t, kv, deploymentID)
		})
		t.Run("TestImportTopologyTemplate", func(t *testing.T) {
			testImportTopologyTemplate(t, kv, deploymentID)
		})
		t.Run("TestTopologyTemplateMetadata", func(t *testing.T) {
			testTopologyTemplateMetadata(t, kv, deploymentID)
		})
	})

	t.Run("CommonsTestsOn_test_topology_substitution.yml", func(t *testing.T) {
		deploymentID := testutil.BuildDeploymentID(t)
		err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/test_topology_substitution.yml")
		require.NoError(t, err)

		t.Run("TestAddSubstitutionMappingAttributeHostNotification", func(t *testing.T) {
			testAddSubstitutionMappingAttributeHostNotification(t, kv, deploymentID)
		})
	})
}
