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

	nodeParams := nodeParams{
		deploymentID:   deploymentID,
		nodeName:       "Network",
		infrastructure: &infrastructure,
	}

	err := g.generateVPC(ctx, nodeParams)

	require.NoError(t, err, "Unexpected error attempting to generate vpc for %s", deploymentID)
	require.Len(t, infrastructure.Resource["aws_vpc"], 1, "Expected one vpc")

}
