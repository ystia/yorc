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

package aws

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
)

func testSimpleVPC(t *testing.T, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t)
	ctx := context.Background()
	infrastructure := commons.Infrastructure{}
	g := awsGenerator{}
	networkName := "simplevpc_network"

	nodeParams := nodeParams{
		deploymentID:   deploymentID,
		nodeName:       "Network",
		infrastructure: &infrastructure,
	}

	err := g.generateVPC(ctx, nodeParams, "instance0", make(map[string]string))

	require.NoError(t, err, "Unexpected error attempting to generate vpc for %s", deploymentID)
	require.Len(t, infrastructure.Resource["aws_vpc"], 1, "Expected one vpc")

	instancesMap := infrastructure.Resource["aws_vpc"].(map[string]interface{})
	require.Len(t, instancesMap, 1)
	require.Contains(t, instancesMap, networkName)

	vpc, ok := instancesMap[networkName].(*VPC)
	require.True(t, ok, "%s is not a VPC", networkName)
	assert.Equal(t, "10.0.0.0/16", vpc.CidrBlock)
	assert.Equal(t, "true", vpc.AssignGeneratedIpv6CidrBlock)

}
