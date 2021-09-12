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

package maas

import (
	"context"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
)

func testGetComputeFromDeployment(t *testing.T, cfg config.Configuration) {
	t.Parallel()
	deploymentID := path.Base(t.Name())
	err := deployments.StoreDeploymentDefinition(context.Background(), deploymentID, "testdata/testSimpleMaasCompute.yaml")
	require.Nil(t, err, "Failed to parse testInstallCompute definition")

	operationParams := operationParameters{
		locationProps:     MAAS_LOCATION_PROPS,
		taskID:            "0",
		deploymentID:      deploymentID,
		nodeName:          "Compute",
		delegateOperation: "install",
		instances:         []string{"instance0"},
	}

	compute, err := getComputeFromDeployment(context.Background(), &operationParams)
	require.Nil(t, err)
	require.NotNil(t, compute)

	require.Equal(t, "bionic", compute.distro_series)
	require.Equal(t, "amd64", compute.arch)
	require.Equal(t, "true", compute.erase)
	require.Equal(t, "true", compute.quick_erase)
	require.Equal(t, "true", compute.secure_erase)
	require.Equal(t, "tagTest", compute.tags)
	require.Equal(t, "notTagTest", compute.not_tags)

	require.Equal(t, "3", compute.host.num_cpus)
	require.Equal(t, "10 GB", compute.host.mem_size)
	require.Equal(t, "200 GB", compute.host.disk_size)
}
