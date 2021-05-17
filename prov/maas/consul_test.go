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
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/testutil"
)

var MAAS_LOCATION_PROPS config.DynamicMap

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulMaasPackageTests(t *testing.T) {
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
		"api_url": "10.0.0.0",
		"api_key": "jGhsLn7JdVHNn6JhpT:SFtuTJSs2DpFbNGvQd:PJfNarCdtXUAQ8694f529a8fLZPaDBHE",
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
			testSimpleMaasCompute(t, cfg)
		})
	})
}
