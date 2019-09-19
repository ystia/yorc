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

package slurm

import (
	"context"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/prov"
)

func testNewExecution(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)

	newExec, err := newExecution(kv, cfg, "myTask", deploymentID, "Compute", "myStep",
		prov.Operation{Name: "myOperation"})
	require.NoError(t, err, "Unexpected error calling newExecution")

	err = newExec.execute(context.Background())
	require.NoError(t, err, "Unexpected error calling execute")

}
