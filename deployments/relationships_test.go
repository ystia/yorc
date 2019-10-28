// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/testutil"
)

func testDeleteRelationshipInstance(t *testing.T) {
	deploymentID := testutil.BuildDeploymentID(t)
	ctx := context.Background()
	ctx, errGrp, consulStore := consulutil.WithContext(ctx)

	err := StoreDeploymentDefinition(ctx, deploymentID, "testdata/relationship_instances.yaml")
	require.NoError(t, err, "Failed to store test topology deployment definition")

	nodeAName := "NodeA"
	nodeBName := "NodeB"
	err = SetInstanceStateStringWithContextualLogs(ctx, deploymentID, nodeAName, "1", "created")
	require.NoError(t, err)
	err = SetInstanceStateStringWithContextualLogs(ctx, deploymentID, nodeAName, "11", "created")
	require.NoError(t, err)
	err = SetInstanceStateStringWithContextualLogs(ctx, deploymentID, nodeBName, "1", "created")
	require.NoError(t, err)
	err = SetInstanceStateStringWithContextualLogs(ctx, deploymentID, nodeBName, "11", "created")
	require.NoError(t, err)

	err = createRelationshipInstances(consulStore, deploymentID, nodeAName)
	require.NoError(t, err)
	err = errGrp.Wait()
	require.NoError(t, err)

	attributeName := "someAttr"
	attributeValue := "someValue"
	err = SetRelationshipAttributeForAllInstances(deploymentID, nodeAName, "0", attributeName, attributeValue)
	require.NoError(t, err)

	actualValue, err := GetRelationshipAttributeValueFromRequirement(deploymentID, nodeAName, "1", "0", attributeName)
	require.NoError(t, err)
	require.NotNil(t, actualValue)
	require.Equal(t, attributeValue, actualValue.RawString())
	actualValue, err = GetRelationshipAttributeValueFromRequirement(deploymentID, nodeAName, "11", "0", attributeName)
	require.NoError(t, err)
	require.NotNil(t, actualValue)
	require.Equal(t, attributeValue, actualValue.RawString())

	// Now we test
	err = DeleteRelationshipInstance(deploymentID, nodeAName, "1")
	require.NoError(t, err)

	actualValue, err = GetRelationshipAttributeValueFromRequirement(deploymentID, nodeAName, "1", "0", attributeName)
	require.NoError(t, err)
	require.Nil(t, actualValue)
	actualValue, err = GetRelationshipAttributeValueFromRequirement(deploymentID, nodeAName, "11", "0", attributeName)
	require.NoError(t, err)
	require.NotNil(t, actualValue)
	require.Equal(t, attributeValue, actualValue.RawString())

}
