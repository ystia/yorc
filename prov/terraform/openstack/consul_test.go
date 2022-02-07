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

package openstack

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulOpenstackPackageTests(t *testing.T) {
	cfg := testutil.SetupTestConfig(t)
	srv, _ := testutil.NewTestConsulInstance(t, &cfg)
	defer func() {
		srv.Stop()
		os.RemoveAll(cfg.WorkingDirectory)
	}()

	locationMgr, err := locations.GetManager(cfg)
	require.NoError(t, err, "Error initializing locations")

	t.Run("groupOpenstack", func(t *testing.T) {
		t.Run("simpleOSInstance", func(t *testing.T) {
			testSimpleOSInstance(t)
		})

		t.Run("OSInstanceWithBootVolume", func(t *testing.T) {
			testOSInstanceWithBootVolume(t)
		})
		t.Run("fipOSInstance", func(t *testing.T) {
			testFipOSInstance(t, srv)
		})
		t.Run("fipMissingOSInstance", func(t *testing.T) {
			testFipMissingOSInstance(t, srv)
		})
		t.Run("fipOSInstanceNotAllowed", func(t *testing.T) {
			testFipOSInstanceNotAllowed(t, srv)
		})
		t.Run("TestGenerateOSBSVolumeSizeConvert", func(t *testing.T) {
			testGenerateOSBSVolumeSizeConvert(t, srv)
		})
		t.Run("TestGenerateOSBSVolumeSizeConvertError", func(t *testing.T) {
			testGenerateOSBSVolumeSizeConvertError(t, srv)
		})
		t.Run("TestGenerateOSBSVolumeMissingSize", func(t *testing.T) {
			testGenerateOSBSVolumeMissingSize(t, srv)
		})
		t.Run("TestGenerateOSBSVolumeWrongType", func(t *testing.T) {
			testGenerateOSBSVolumeWrongType(t, srv)
		})
		t.Run("TestGenerateOSBSVolumeCheckOptionalValues", func(t *testing.T) {
			testGenerateOSBSVolumeCheckOptionalValues(t, srv)
		})
		t.Run("TestGeneratePoolIP", func(t *testing.T) {
			testGeneratePoolIP(t, srv)
		})
		t.Run("TestGenerateSingleIp", func(t *testing.T) {
			testGenerateSingleIP(t, srv)
		})
		t.Run("TestGenerateMultipleIP", func(t *testing.T) {
			testGenerateMultipleIP(t, srv)
		})
		t.Run("simpleServerGroup", func(t *testing.T) {
			testSimpleServerGroup(t)
		})
		t.Run("OSInstanceWithServerGroup", func(t *testing.T) {
			testOSInstanceWithServerGroup(t, srv)
		})
		t.Run("TestGenerateTerraformInfo", func(t *testing.T) {
			testGenerateTerraformInfo(t, srv, locationMgr)
		})
		t.Run("TestAppCredentials", func(t *testing.T) {
			testAppCredentials(t, srv, locationMgr)
		})
		t.Run("TestComputeBootVolumeWrongSize", func(t *testing.T) {
			testComputeBootVolumeWrongSize(t, srv)
		})
		t.Run("TestComputeBootVolumeWrongType", func(t *testing.T) {
			testComputeBootVolumeWrongType(t, srv)
		})
		t.Run("ComputeNetworkAttributes", func(t *testing.T) {
			testComputeNetworkAttributes(t, srv)
		})
	})
}
