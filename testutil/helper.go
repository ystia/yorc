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
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/storage"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/consulutil"
)

// NewTestConsulInstance allows to :
//  - creates and returns a new Consul server and client
//  - starts a Consul Publisher
//  - loads stores
//  - stores common-types to Consul
// Warning: You need to defer the server stop command in the caller
func NewTestConsulInstance(t testing.TB, cfg *config.Configuration) (*testutil.TestServer, *api.Client) {
	logLevel := "debug"
	if isCI, ok := os.LookupEnv("CI"); ok && isCI == "true" {
		logLevel = "warn"
	}

	cb := func(c *testutil.TestServerConfig) {
		c.Args = []string{"-ui"}
		c.LogLevel = logLevel
	}
	return NewTestConsulInstanceWithConfigAndStore(t, cb, cfg)
}

// NewTestConsulInstanceWithConfigAndStore sets up a consul instance for testing
func NewTestConsulInstanceWithConfigAndStore(t testing.TB, cb testutil.ServerConfigCallback, cfg *config.Configuration) (*testutil.TestServer, *api.Client) {

	return NewTestConsulInstanceWithConfig(t, cb, cfg, true)
}

// NewTestConsulInstanceWithConfig sets up a consul instance for testing :
//  - creates and returns a new Consul server and client
//  - starts a Consul Publisher
//  - stores common-types to Consul only if storeCommons bool parameter is true
// Warning: You need to defer the server stop command in the caller
func NewTestConsulInstanceWithConfig(t testing.TB, cb testutil.ServerConfigCallback, cfg *config.Configuration, storeCommons bool) (*testutil.TestServer, *api.Client) {

	// Retrying attempts to create a test consul instance and create a session
	// as they can fail intermittently on slow test servers
	var srv1 *testutil.TestServer
	var client *api.Client
	var lock *consulutil.AutoDeleteLock
	err := errors.Errorf("Failed to create consul server")
	i := 0
	for i < 10 && err != nil {
		srv1, err = testutil.NewTestServerConfigT(t, cb)
		if err != nil {
			i = i + 1
			t.Logf("Attempt %d to create a consul server failed with error: %s", i, err.Error())
			continue
		}
		cfg.Consul.Address = srv1.HTTPAddr
		cfg.Consul.PubMaxRoutines = config.DefaultConsulPubMaxRoutines
		client, err = cfg.GetNewConsulClient()
		assert.Nil(t, err, "Failed to get a consul client")

		kv := client.KV()
		consulutil.InitConsulPublisher(cfg.Consul.PubMaxRoutines, kv)

		// Checking a request to avoid intermittent session creation errors
		lock, err = consulutil.AcquireLock(client, ".lock_test", 0)
		if err != nil {
			i = i + 1
			t.Logf("Attempt %d to acquire a consul lock failed with error: %s", i, err.Error())
			continue
		}
		_ = lock.Unlock()
	}

	if err != nil {
		t.Fatalf("Failed to create a test consul server: %v", err)
	}

	// Load stores
	// Load main stores used for deployments, logs, events
	err = storage.LoadStores(*cfg)
	if err != nil {
		t.Fatalf("Failed to load stores due to error: %v", err)
	}

	if storeCommons {
		storeCommonDefinitions()
	}

	return srv1, client
}

// BuildDeploymentID allows to create a deploymentID from the test name value
func BuildDeploymentID(t testing.TB) string {
	return strings.Replace(t.Name(), "/", "_", -1)
}

// SetupTestConfig sets working directory configuration
// Warning: You need to defer the working directory removal
func SetupTestConfig(t testing.TB) config.Configuration {
	workingDir, err := ioutil.TempDir(os.TempDir(), "work")
	assert.Nil(t, err)

	return config.Configuration{
		WorkingDirectory:        workingDir,
		UpgradeConcurrencyLimit: config.DefaultUpgradesConcurrencyLimit,
	}
}
