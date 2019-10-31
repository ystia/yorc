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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/testutil"
)

var slumTestLocationProps config.DynamicMap

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulSlurmPackageTests(t *testing.T) {
	srv, _ := testutil.NewTestConsulInstance(t)
	defer srv.Stop()

	// Create a slurm location
	cfg := config.Configuration{
		Consul: config.Consul{
			Address:        srv.HTTPAddr,
			PubMaxRoutines: config.DefaultConsulPubMaxRoutines,
		},
	}

	locationMgr, err := locations.GetManager(cfg)
	require.NoError(t, err, "Error initializing locations")

	slumTestLocationProps = config.DynamicMap{
		"user_name": "root",
		"password":  "pwd",
		"name":      "slurm",
		"url":       "1.2.3.4",
		"port":      "1234",
	}
	err = locationMgr.CreateLocation(
		locations.LocationConfiguration{
			Name:       "testSlurmLocation",
			Type:       infrastructureType,
			Properties: slumTestLocationProps,
		})
	require.NoError(t, err, "Failed to create a location")
	defer func() {
		locationMgr.RemoveLocation(t.Name(), infrastructureType)
	}()

	t.Run("groupSlurm", func(t *testing.T) {
		t.Run("simpleSlurmNodeAllocation", func(t *testing.T) {
			testSimpleSlurmNodeAllocation(t, cfg)
		})
		t.Run("simpleSlurmNodeAllocationWithoutProps", func(t *testing.T) {
			testSimpleSlurmNodeAllocationWithoutProps(t, cfg)
		})
		t.Run("multipleSlurmNodeAllocation", func(t *testing.T) {
			testMultipleSlurmNodeAllocation(t, cfg)
		})
		t.Run("executorTest", func(t *testing.T) {
			testExecutor(t, srv, cfg)
		})
		t.Run("ExecutionCommonBuildJobInfo", func(t *testing.T) {
			testExecutionCommonBuildJobInfo(t)
		})
		t.Run("ExecutionCommonPrepareAndSubmitJob", func(t *testing.T) {
			testExecutionCommonPrepareAndSubmitJob(t)
		})
	})
}
