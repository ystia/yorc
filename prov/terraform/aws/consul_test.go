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
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulAWSPackageTests(t *testing.T) {
	cfg := testutil.SetupTestConfig(t)
	srv, _ := testutil.NewTestConsulInstance(t, &cfg)
	defer func() {
		srv.Stop()
		os.RemoveAll(cfg.WorkingDirectory)
	}()

	locationMgr, err := locations.GetManager(cfg)
	require.NoError(t, err, "Error initializing locations")

	t.Run("groupAWS", func(t *testing.T) {
		t.Run("simpleAWSInstance", func(t *testing.T) {
			testSimpleAWSInstance(t, cfg)
		})
		t.Run("simpleAWSInstanceWithPrivateKey", func(t *testing.T) {
			testSimpleAWSInstanceWithPrivateKey(t, cfg)
		})
		t.Run("simpleAWSInstanceFailed", func(t *testing.T) {
			testSimpleAWSInstanceFailed(t, cfg)
		})
		t.Run("simpleAWSInstanceWithEIP", func(t *testing.T) {
			testSimpleAWSInstanceWithEIP(t, cfg)
		})
		t.Run("simpleAWSInstanceWithProvidedEIP", func(t *testing.T) {
			testSimpleAWSInstanceWithProvidedEIP(t, cfg)
		})
		t.Run("simpleAWSInstanceWithListOfProvidedEIP", func(t *testing.T) {
			testSimpleAWSInstanceWithListOfProvidedEIP(t, cfg)
		})
		t.Run("simpleAWSInstanceWithListOfProvidedEIP2", func(t *testing.T) {
			testSimpleAWSInstanceWithNotEnoughProvidedEIPS(t, cfg)
		})
		t.Run("simpleAWSInstanceWithNoDeleteVolumeOnTermination", func(t *testing.T) {
			testSimpleAWSInstanceWithNoDeleteVolumeOnTermination(t, cfg)
		})
		t.Run("simpleAWSInstanceWithMalformedEIP", func(t *testing.T) {
			testSimpleAWSInstanceWithMalformedEIP(t, cfg)
		})
		t.Run("simpleAWSInstanceWitthVPC", func(t *testing.T) {
			testSimpleAWSInstanceWitthVPC(t, cfg, srv)
		})
		t.Run("generateTerraformInfraForAWSNode", func(t *testing.T) {
			testGenerateTerraformInfraForAWSNode(t, cfg, locationMgr)
		})
		t.Run("simpleEBS", func(t *testing.T) {
			testSimpleEBS(t, cfg)
		})
		t.Run("simpleEBSWithVolumeID", func(t *testing.T) {
			testSimpleEBSWithVolumeID(t, cfg)
		})
		t.Run("simpleAWSInstanceWithPersistentDisk", func(t *testing.T) {
			testSimpleAWSInstanceWithPersistentDisk(t, cfg, srv)
		})
		t.Run("VPC", func(t *testing.T) {
			testVPC(t, cfg)
		})
		t.Run("Subnet", func(t *testing.T) {
			testSubnet(t, srv, cfg)
		})
		t.Run("VPCWithNestedSubnetAndSG", func(t *testing.T) {
			testVPCWithNestedSubnetAndSG(t, srv, cfg)
		})
	})
}
