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

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/helper/consulutil"
)

// NewTestConsulInstance allows to :
//  - creates and returns a new Consul server and client
//  - starts a Consul Publisher
// Warning: You need to defer the server stop command in the caller
func NewTestConsulInstance(t *testing.T) (*testutil.TestServer, *api.Client) {
	logLevel := "debug"
	if isCI, ok := os.LookupEnv("CI"); ok && isCI == "true" {
		logLevel = "warn"
	}

	srv1, err := testutil.NewTestServerConfig(func(c *testutil.TestServerConfig) {
		c.Args = []string{"-ui"}
		c.LogLevel = logLevel
	})
	if err != nil {
		t.Fatalf("Failed to create consul server: %v", err)
	}

	consulConfig := api.DefaultConfig()
	consulConfig.Address = srv1.HTTPAddr

	client, err := api.NewClient(consulConfig)
	assert.Nil(t, err)

	kv := client.KV()
	consulutil.InitConsulPublisher(config.DefaultConsulPubMaxRoutines, kv)
	return srv1, client
}

// BuildDeploymentID allows to create a deploymentID from the test name value
func BuildDeploymentID(t *testing.T) string {
	return strings.Replace(t.Name(), "/", "_", -1)
}
