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

package maas

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jarcoal/httpmock"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/testutil"
)

var MAAS_LOCATION_PROPS config.DynamicMap

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestMainMaas(t *testing.T) {
	// Create consul instance
	cfg := testutil.SetupTestConfig(t)
	srv, _ := testutil.NewTestConsulInstance(t, &cfg)
	defer func() {
		srv.Stop()
		os.RemoveAll(cfg.WorkingDirectory)
	}()

	// Create a maas location
	locationMgr, err := locations.GetManager(cfg)
	require.NoError(t, err, "Error initializing locations")

	MAAS_LOCATION_PROPS = config.DynamicMap{
		"api_url":                 "10.0.0.0",
		"api_key":                 "jGhsLn7JdVHNn6JhpT:SFtuTJSs2DpFbNGvQd:PJfNarCdtXUAQ8694f529a8fLZPaDBHE",
		"delayBetweenStatusCheck": "1ms",
	}
	err = locationMgr.CreateLocation(
		locations.LocationConfiguration{
			Name:       "testMaasLocation",
			Type:       infrastructureType,
			Properties: MAAS_LOCATION_PROPS,
		})
	require.NoError(t, err, "Failed to create a location")
	defer func() {
		locationMgr.RemoveLocation(t.Name())
	}()

	t.Run("groupMaas", func(t *testing.T) {
		t.Run("testSimpleMaasCompute", func(t *testing.T) {
			testInstallCompute(t, cfg)
		})
		t.Run("testSimpleMaasCompute", func(t *testing.T) {
			testGetComputeFromDeployment(t, cfg)
		})

	})
}

func testInstallCompute(t *testing.T, cfg config.Configuration) {
	executor := &defaultExecutor{}
	ctx := context.Background()

	deploymentID := path.Base(t.Name())
	err := deployments.StoreDeploymentDefinition(context.Background(), deploymentID, "testdata/testSimpleMaasCompute.yaml")
	require.Nil(t, err, "Failed to parse testInstallCompute definition")

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "/10.0.0.0/api/2.0/machines/?op=allocate",
		httpmock.NewStringResponder(200, allocateResponse))

	httpmock.RegisterResponder("POST", "/10.0.0.0/api/2.0/machines/tdgqkw/?op=deploy",
		httpmock.NewStringResponder(200, deployResponse))

	httpmock.RegisterResponder("GET", "/10.0.0.0/api/2.0/machines/tdgqkw/",
		httpmock.NewStringResponder(200, machineDeployed))

	err = executor.ExecDelegate(ctx, cfg, "taskID", deploymentID, "Compute", "install")
	require.Nil(t, err)
}
