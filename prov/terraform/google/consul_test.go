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

package google

import (
	"testing"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulGooglePackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()

	// AWS infrastructure config
	cfg := config.Configuration{
		DisableSSHAgent: false,
		Infrastructures: map[string]config.DynamicMap{
			infrastructureName: config.DynamicMap{
				"credentials": "/tmp/creds.json",
				"region":      "europe-west-1",
			}}}
	t.Run("googleProvider", func(t *testing.T) {
		t.Run("simpleComputeInstance", func(t *testing.T) {
			testSimpleComputeInstance(t, kv, cfg)
		})
		t.Run("simpleComputeInstanceMissingMandatoryParameter", func(t *testing.T) {
			testSimpleComputeInstanceMissingMandatoryParameter(t, kv, cfg)
		})
		t.Run("simpleComputeAddress", func(t *testing.T) {
			testSimpleComputeAddress(t, kv, cfg)
		})
		t.Run("simpleComputeInstanceWithAddress", func(t *testing.T) {
			testSimpleComputeInstanceWithAddress(t, kv, srv, cfg)
		})
		t.Run("simplePersistentDisk", func(t *testing.T) {
			testSimplePersistentDisk(t, kv, cfg)
		})
		t.Run("simpleComputeInstanceWithPersistentDisk", func(t *testing.T) {
			testSimpleComputeInstanceWithPersistentDisk(t, kv, srv, cfg)
		})
		t.Run("simplePrivateNetwork", func(t *testing.T) {
			testSimplePrivateNetwork(t, kv, cfg)
		})
		t.Run("simpleSubnet", func(t *testing.T) {
			testSimpleSubnet(t, kv, srv, cfg)
		})
		t.Run("simpleComputeInstanceWithAutoCreationModeNetwork", func(t *testing.T) {
			testSimpleComputeInstanceWithAutoCreationModeNetwork(t, kv, srv, cfg)
		})
		t.Run("simpleComputeInstanceWithSimpleNetwork", func(t *testing.T) {
			testSimpleComputeInstanceWithSimpleNetwork(t, kv, srv, cfg)
		})
	})
}
