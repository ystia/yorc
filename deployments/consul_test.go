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
	"testing"

	"github.com/ystia/yorc/testutil"
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
		t.Run("testInlineWorkflow", func(t *testing.T) {
			testInlineWorkflow(t, kv)
		})
		t.Run("testCheckCycleInNestedWorkflows", func(t *testing.T) {
			testCheckCycleInNestedWorkflows(t, kv)
		})
		t.Run("testGetCapabilityProperties", func(t *testing.T) {
			testGetCapabilityProperties(t, kv)
		})
	})
}
