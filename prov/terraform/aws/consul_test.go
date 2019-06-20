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
	"testing"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulAWSPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()

	// AWS infrastructure config
	cfg := config.Configuration{
		Infrastructures: map[string]config.DynamicMap{
			infrastructureName: config.DynamicMap{
				"region":     "us-east-2",
				"access_key": "test",
				"secret_key": "test",
			}}}

	t.Run("groupAWS", func(t *testing.T) {
		t.Run("simpleAWSInstance", func(t *testing.T) {
			testSimpleAWSInstance(t, kv, cfg)
		})
		t.Run("simpleAWSInstanceWithPrivateKey", func(t *testing.T) {
			testSimpleAWSInstanceWithPrivateKey(t, kv, cfg)
		})
		t.Run("simpleAWSInstanceFailed", func(t *testing.T) {
			testSimpleAWSInstanceFailed(t, kv, cfg)
		})
		t.Run("simpleAWSInstanceWithEIP", func(t *testing.T) {
			testSimpleAWSInstanceWithEIP(t, kv, cfg)
		})
		t.Run("simpleAWSInstanceWithProvidedEIP", func(t *testing.T) {
			testSimpleAWSInstanceWithProvidedEIP(t, kv, cfg)
		})
		t.Run("simpleAWSInstanceWithListOfProvidedEIP", func(t *testing.T) {
			testSimpleAWSInstanceWithListOfProvidedEIP(t, kv, cfg)
		})
		t.Run("simpleAWSInstanceWithListOfProvidedEIP2", func(t *testing.T) {
			testSimpleAWSInstanceWithNotEnoughProvidedEIPS(t, kv, cfg)
		})
		t.Run("simpleAWSInstanceWithNoDeleteVolumeOnTermination", func(t *testing.T) {
			testSimpleAWSInstanceWithNoDeleteVolumeOnTermination(t, kv, cfg)
		})
		t.Run("simpleAWSInstanceWithMalformedEIP", func(t *testing.T) {
			testSimpleAWSInstanceWithMalformedEIP(t, kv, cfg)
		})

	})
}
