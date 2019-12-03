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
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/prov/terraform/commons"
)

func Test_addOutput(t *testing.T) {
	type args struct {
		infrastructure *commons.Infrastructure
		outputName     string
		output         *commons.Output
	}
	tests := []struct {
		name       string
		args       args
		jsonResult string
	}{
		{"OneOutput", args{&commons.Infrastructure{}, "O1", &commons.Output{Value: "V1"}}, `{"output":{"O1":{"value":"V1"}}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commons.AddOutput(tt.args.infrastructure, tt.args.outputName, tt.args.output)
			res, err := json.Marshal(tt.args.infrastructure)
			require.Nil(t, err)
			require.Equal(t, tt.jsonResult, string(res))
		})
	}
}

func testGenerateTerraformInfraForAWSNode(t *testing.T, cfg config.Configuration, locationMgr locations.Manager) {

	t.Parallel()
	deploymentID := loadTestYaml(t)

	locationProps := config.DynamicMap{
		"access_key": "ak1",
		"secret_key": "secret1",
		"region":     "europe",
	}
	err := locationMgr.CreateLocation(
		locations.LocationConfiguration{
			Name:       "testAWSLocation",
			Type:       infrastructureType,
			Properties: locationProps,
		})
	require.NoError(t, err, "Failed to create a location")
	defer func() {
		locationMgr.RemoveLocation(t.Name())
	}()

	g := awsGenerator{}
	infrastructurePath, err := ioutil.TempDir("", deploymentID)
	require.NoError(t, err, "Failed to to create temporary directory")
	defer os.RemoveAll(infrastructurePath)

	_, _, env, _, err := g.GenerateTerraformInfraForNode(
		context.Background(), cfg, deploymentID, "ComputeAWS", infrastructurePath)
	require.NoError(t, err, "Unexpected error generating terraform infra")

	assert.Equal(t, "AWS_ACCESS_KEY_ID=ak1", env[0], "Did not find expected access key in %+v", env)

}
