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

	"github.com/ystia/yorc/v4/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulDeploymentsPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	defer srv.Stop()
	t.Run("groupDeployments", func(t *testing.T) {
		//t.Run("testArtifacts", func(t *testing.T) {
		//	testArtifacts(t, srv)
		//})
		//t.Run("testCapabilities", func(t *testing.T) {
		//	testCapabilities(t, srv)
		//})
		//t.Run("testDefinitionStore", func(t *testing.T) {
		//	testDefinitionStore(t)
		//})
		//t.Run("testDeploymentNodes", func(t *testing.T) {
		//	testDeploymentNodes(t, srv)
		//})
		t.Run("testRequirements", func(t *testing.T) {
			testRequirements(t, srv)
		})
		//t.Run("testResolver", func(t *testing.T) {
		//	testResolver(t)
		//})
		//t.Run("testGetTypePropertyDataType", func(t *testing.T) {
		//	testGetTypePropertyDataType(t)
		//})
		//t.Run("testGetNestedDataType", func(t *testing.T) {
		//	testGetNestedDataType(t)
		//})
		//t.Run("testReadComplexVA", func(t *testing.T) {
		//	testReadComplexVA(t)
		//})
		//t.Run("testIssueGetEmptyPropRel", func(t *testing.T) {
		//	testIssueGetEmptyPropRel(t)
		//})
		//t.Run("testRelationshipWorkflow", func(t *testing.T) {
		//	testRelationshipWorkflow(t)
		//})
		//t.Run("testGlobalInputs", func(t *testing.T) {
		//	testGlobalInputs(t)
		//})
		//t.Run("testInlineWorkflow", func(t *testing.T) {
		//	testInlineWorkflow(t)
		//})
		//t.Run("testDeleteWorkflow", func(t *testing.T) {
		//	testDeleteWorkflow(t)
		//})
		//t.Run("testCheckCycleInNestedWorkflows", func(t *testing.T) {
		//	testCheckCycleInNestedWorkflows(t)
		//})
		//t.Run("testGetCapabilityProperties", func(t *testing.T) {
		//	testGetCapabilityProperties(t)
		//})
		//t.Run("testSubstitutionServiceCapabilityMappings", func(t *testing.T) {
		//	testSubstitutionServiceCapabilityMappings(t)
		//})
		//t.Run("testSubstitutionServiceRequirementMappings", func(t *testing.T) {
		//	testSubstitutionServiceRequirementMappings(t)
		//})
		//t.Run("testSubstitutionClientDirective", func(t *testing.T) {
		//	testSubstitutionClientDirective(t)
		//})
		//t.Run("testSubstitutionClientServiceInstance", func(t *testing.T) {
		//	testSubstitutionClientServiceInstance(t)
		//})
		//t.Run("TestOperationImplementationArtifact", func(t *testing.T) {
		//	testOperationImplementationArtifact(t)
		//})
		//t.Run("TestOperationHost", func(t *testing.T) {
		//	testOperationHost(t)
		//})
		t.Run("testIssueGetEmptyPropOnRelationship", func(t *testing.T) {
			testIssueGetEmptyPropOnRelationship(t)
		})

		t.Run("testTopologyUpdate", func(t *testing.T) {
			testTopologyUpdate(t)
		})
		t.Run("testTopologyBadUpdate", func(t *testing.T) {
			testTopologyBadUpdate(t)
		})
		t.Run("testRepositories", func(t *testing.T) {
			testRepositories(t)
		})
		t.Run("testPurgedDeployments", func(t *testing.T) {
			testPurgedDeployments(t, client)
		})
		t.Run("testDeleteDeployment", func(t *testing.T) {
			testDeleteDeployment(t)
		})
		t.Run("testDeleteInstance", func(t *testing.T) {
			testDeleteInstance(t)
		})
		t.Run("testDeleteAllInstances", func(t *testing.T) {
			testDeleteAllInstances(t)
		})
		t.Run("testDeleteRelationshipInstance", func(t *testing.T) {
			testDeleteRelationshipInstance(t)
		})
	})

	t.Run("CommonsTestsOn_test_topology.yml", func(t *testing.T) {
		t.Skip()
		deploymentID := testutil.BuildDeploymentID(t)
		err := StoreDeploymentDefinition(context.Background(), deploymentID, "testdata/test_topology.yml")
		require.NoError(t, err)

		t.Run("TestNodeHasAttribute", func(t *testing.T) {
			testNodeHasAttribute(t, deploymentID)
		})
		t.Run("TestNodeHasProperty", func(t *testing.T) {
			testNodeHasProperty(t, deploymentID)
		})
		t.Run("TestTopologyTemplateMetadata", func(t *testing.T) {
			testTopologyTemplateMetadata(t, deploymentID)
		})
		t.Run("TestAttributeNotifications", func(t *testing.T) {
			t.Skip()
			testAttributeNotifications(t, deploymentID)
		})
		t.Run("TestNotifyAttributeOnValueChange", func(t *testing.T) {
			testNotifyAttributeOnValueChange(t, deploymentID)
		})
		t.Run("TestImportTopologyTemplate", func(t *testing.T) {
			testImportTopologyTemplate(t, deploymentID)
		})
		t.Run("TestTopologyTemplateMetadata", func(t *testing.T) {
			testTopologyTemplateMetadata(t, deploymentID)
		})
	})

	t.Run("CommonsTestsOn_test_topology_substitution.yml", func(t *testing.T) {
		t.Skip()
		deploymentID := testutil.BuildDeploymentID(t)
		err := StoreDeploymentDefinition(context.Background(), deploymentID, "testdata/test_topology_substitution.yml")
		require.NoError(t, err)

		t.Run("TestAddSubstitutionMappingAttributeHostNotification", func(t *testing.T) {
			testAddSubstitutionMappingAttributeHostNotification(t, deploymentID)
		})
	})
}
