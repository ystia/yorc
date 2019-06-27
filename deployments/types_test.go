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
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/log"
	"testing"
)

func testTypes(t *testing.T, kv *api.KV) {
	log.SetDebug(true)

	t.Run("testTypes", func(t *testing.T) {
		tesGetLatestCommonsTypesPaths(t, kv)
	})
}

func tesGetLatestCommonsTypesPaths(t *testing.T, kv *api.KV) {
	t.Parallel()

	commonTypes, err := getLatestCommonsTypesPaths(kv)
	require.Nil(t, err, "expected nil error ret")
	require.NotNil(t, commonTypes, "expected commons types")
	require.Contains(t, commonTypes, "_yorc/commons_types/yorc-google-types/1.0.0/types", "commons types expected containing yorc-google-types types")
	require.Contains(t, commonTypes, "_yorc/commons_types/yorc-hostspool-types/1.0.0/types", "commons types expected containing yorc-hostpool-types types")
	require.Contains(t, commonTypes, "_yorc/commons_types/yorc-kubernetes-types/2.0.0/types", "commons types expected containing yorc-kubernetes-types types")
	require.Contains(t, commonTypes, "_yorc/commons_types/yorc-openstack-types/1.1.0/types", "commons types expected containing yorc-openstack-types types")
	require.Contains(t, commonTypes, "_yorc/commons_types/yorc-slurm-types/1.2.0/types", "commons types expected containing yorc-slurm-types types")
	require.Contains(t, commonTypes, "_yorc/commons_types/yorc-types/1.1.0/types", "commons types expected containing yorc types")
	require.Contains(t, commonTypes, "_yorc/commons_types/tosca-normative-types/1.2.0/types", "commons types expected containing tosca-normative-types types")
	require.Contains(t, commonTypes, "_yorc/commons_types/yorc-aws-types/1.0.0/types", "commons types expected containing yorc-aws-types types")
}
