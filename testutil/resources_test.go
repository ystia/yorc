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

package testutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ystia/yorc/v4/deployments/store"
)

func TestAssets(t *testing.T) {

	srv, _ := NewTestConsulInstance(t)
	defer srv.Stop()
	t.Run("TestAssets", func(t *testing.T) {
		t.Run("NormativeTypes", testAssetNormativeParsing)
		t.Run("OpenstackTypes", testAssetYorcOpenStackParsing)
		t.Run("AWSTypes", testAssetYorcAwsParsing)
		t.Run("GoogleTypes", testAssetYorcGoogleParsing)
		t.Run("YorcTypes", testAssetYorcParsing)
		t.Run("SlurmTypes", testAssetYorcSlurmParsing)
		t.Run("HostsPoolTypes", testAssetYorcHostsPoolParsing)
	})
}

func checkBuiltinTypesPath(t *testing.T, defName string) {
	paths := store.GetCommonsTypesKeyPaths()

	for _, p := range paths {
		if strings.Contains(p, fmt.Sprintf("/%s/", defName)) {
			return
		}
	}
	t.Errorf("%s Types missing", defName)
}

func testAssetNormativeParsing(t *testing.T) {
	t.Parallel()
	checkBuiltinTypesPath(t, "tosca-normative-types")
}

func testAssetYorcOpenStackParsing(t *testing.T) {
	t.Parallel()
	checkBuiltinTypesPath(t, "yorc-openstack-types")
}

func testAssetYorcAwsParsing(t *testing.T) {
	t.Parallel()
	checkBuiltinTypesPath(t, "yorc-aws-types")
}

func testAssetYorcGoogleParsing(t *testing.T) {
	t.Parallel()
	checkBuiltinTypesPath(t, "yorc-google-types")
}

func testAssetYorcParsing(t *testing.T) {
	t.Parallel()
	checkBuiltinTypesPath(t, "yorc-types")
}

func testAssetYorcSlurmParsing(t *testing.T) {
	t.Parallel()
	checkBuiltinTypesPath(t, "yorc-slurm-types")
}

func testAssetYorcHostsPoolParsing(t *testing.T) {
	t.Parallel()
	checkBuiltinTypesPath(t, "yorc-hostspool-types")
}
