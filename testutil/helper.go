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
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/ystia/yorc/v3/config"
	"github.com/ystia/yorc/v3/helper/consulutil"
)

// NewTestConsulInstance allows to :
//  - creates and returns a new Consul server and client
//  - starts a Consul Publisher
//  - stores common-types to Consul
// Warning: You need to defer the server stop command in the caller
func NewTestConsulInstance(t testing.TB) (*testutil.TestServer, *api.Client) {
	logLevel := "debug"
	if isCI, ok := os.LookupEnv("CI"); ok && isCI == "true" {
		logLevel = "warn"
	}

	cb := func(c *testutil.TestServerConfig) {
		c.Args = []string{"-ui"}
		c.LogLevel = logLevel
	}
	return NewTestConsulInstanceWithConfigAndStore(t, cb)
}

// NewTestConsulInstanceWithConfigAndStore sets up a consul instance for testing
func NewTestConsulInstanceWithConfigAndStore(t testing.TB, cb testutil.ServerConfigCallback) (*testutil.TestServer, *api.Client) {

	return NewTestConsulInstanceWithConfig(t, cb, true)
}

// NewTestConsulInstanceWithConfig sets up a consul instance for testing :
//  - creates and returns a new Consul server and client
//  - starts a Consul Publisher
//  - stores common-types to Consul only if storeCommons bool parameter is true
// Warning: You need to defer the server stop command in the caller
func NewTestConsulInstanceWithConfig(t testing.TB, cb testutil.ServerConfigCallback, storeCommons bool) (*testutil.TestServer, *api.Client) {
	srv1, err := testutil.NewTestServerConfig(cb)
	if err != nil {
		t.Fatalf("Failed to create consul server: %v", err)
	}

	cfg := config.Configuration{
		Consul: config.Consul{
			Address:        srv1.HTTPAddr,
			PubMaxRoutines: config.DefaultConsulPubMaxRoutines,
		},
	}

	client, err := cfg.GetNewConsulClient()
	assert.Nil(t, err)

	kv := client.KV()
	consulutil.InitConsulPublisher(cfg.Consul.PubMaxRoutines, kv)

	if storeCommons {
		storeCommonDefinitions()
	}

	return srv1, client
}

// BuildDeploymentID allows to create a deploymentID from the test name value
func BuildDeploymentID(t testing.TB) string {
	return strings.Replace(t.Name(), "/", "_", -1)
}
