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

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/testutil"
)

var slumTestLocationProps config.DynamicMap

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulSlurmPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()

	// Create a slurm location
	var cfg config.Configuration

	slumTestLocationProps = config.DynamicMap{
		"user_name": "root",
		"password":  "pwd",
		"name":      "slurm",
		"url":       "1.2.3.4",
		"port":      "1234",
	}

	t.Run("groupSlurm", func(t *testing.T) {
		t.Run("simpleSlurmNodeAllocation", func(t *testing.T) {
			testSimpleSlurmNodeAllocation(t, kv, cfg)
		})
		t.Run("simpleSlurmNodeAllocationWithoutProps", func(t *testing.T) {
			testSimpleSlurmNodeAllocationWithoutProps(t, kv, cfg)
		})
		t.Run("multipleSlurmNodeAllocation", func(t *testing.T) {
			testMultipleSlurmNodeAllocation(t, kv, cfg)
		})
	})
}
