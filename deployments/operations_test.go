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

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/testutil"
)

func testOperationImplementationArtifact(t *testing.T, kv *api.KV) {
	deploymentID := testutil.BuildDeploymentID(t)
	err := StoreDeploymentDefinition(context.Background(), kv, deploymentID, "testdata/operation_implementation_artifact.yaml")
	require.NoError(t, err, "Failed to store test topology deployment definition")

	t.Run("GetPrimaryImplementationForNodeType", func(t *testing.T) {
		testOperationImplementationArtifactPrimary(t, kv, deploymentID)
	})

	t.Run("testGetOperationImplementationFile", func(t *testing.T) {
		testGetOperationImplementationFile(t, kv, deploymentID)
	})
}

func testOperationImplementationArtifactPrimary(t *testing.T, kv *api.KV, deploymentID string) {

	type args struct {
		typeName  string
		operation string
	}
	type checks struct {
		implementationType string
		primary            string
	}
	oiaTests := []struct {
		name string
		args args
		want checks
	}{
		{"TestBashOnNodeType", args{"yorc.tests.nodes.OpImplementationArtifact", "standard.create"}, checks{"tosca.artifacts.Implementation.Bash", "scripts/create.sh"}},
		{"TestBashOnRelType", args{"yorc.tests.relationships.OpImplementationArtifact", "configure.pre_configure_source"}, checks{"tosca.artifacts.Implementation.Bash", "something"}},
		{"TestBashOnImportedRelType", args{"yorc.tests.relationships.imports.OpImplementationArtifact", "configure.pre_configure_source"}, checks{"tosca.artifacts.Implementation.Bash", "imports/something"}},
		{"TestBashOnImportedNodeType", args{"yorc.tests.nodes.imports.OpImplementationArtifact", "standard.create"}, checks{"tosca.artifacts.Implementation.Bash", "imports/scripts/create.sh"}},
	}

	for _, tt := range oiaTests {
		t.Run(tt.name, func(t *testing.T) {
			implType, err := GetOperationImplementationType(kv, deploymentID, tt.args.typeName, tt.args.operation)
			require.NoError(t, err)
			assert.Equal(t, tt.want.implementationType, implType)
			_, primary, err := GetOperationPathAndPrimaryImplementationForNodeType(kv, deploymentID, tt.args.typeName, tt.args.operation)
			require.NoError(t, err)
			assert.Equal(t, tt.want.primary, primary)
		})
	}

}

func testGetOperationImplementationFile(t *testing.T, kv *api.KV, deploymentID string) {
	type args struct {
		nodeType      string
		operationName string
	}
	type want struct {
		file         string
		relativeFile string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"TestOpImplemFileOnImplemArtifactNodeType", args{"yorc.tests.nodes.OpImplementationArtifact", "standard.create"}, want{"scripts/create.sh", "scripts/create.sh"}},
		{"TestOpImplemFileOnImplemArtifactRelType", args{"yorc.tests.relationships.OpImplementationArtifact", "configure.pre_configure_source"}, want{"something", "something"}},
		{"TestOpImplemFileOnImplemArtifactImportedNodeType", args{"yorc.tests.nodes.imports.OpImplementationArtifact", "standard.create"}, want{"scripts/create.sh", "imports/scripts/create.sh"}},
		{"TestOpImplemFileOnImplemArtifactImportedRelType", args{"yorc.tests.relationships.imports.OpImplementationArtifact", "configure.pre_configure_source"}, want{"something", "imports/something"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetOperationImplementationFile(kv, deploymentID, tt.args.nodeType, tt.args.operationName)
			require.NoError(t, err, "GetOperationImplementationFile() error = %v", err)
			if got != tt.want.file {
				t.Errorf("GetOperationImplementationFile() = %v, want %v", got, tt.want)
			}
			got, err = GetOperationImplementationFileWithRelativePath(kv, deploymentID, tt.args.nodeType, tt.args.operationName)
			require.NoError(t, err, "GetOperationImplementationFileWithRelativePath() error = %v", err)
			if got != tt.want.relativeFile {
				t.Errorf("GetOperationImplementationFileWithRelativePath() = %v, want %v", got, tt.want)
			}
		})
	}
}
