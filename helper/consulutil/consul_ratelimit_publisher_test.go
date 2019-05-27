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

package consulutil

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/v3/config"
)

func TestConsulStore(t *testing.T) {
	// Can't import testutil due to import cycles so duplicate a bit of code here
	logLevel := "debug"
	if isCI, ok := os.LookupEnv("CI"); ok && isCI == "true" {
		logLevel = "warn"
	}

	cb := func(c *testutil.TestServerConfig) {
		c.Args = []string{"-ui"}
		c.LogLevel = logLevel
	}
	srv, err := testutil.NewTestServerConfig(cb)
	if err != nil {
		t.Fatalf("Failed to create consul server: %v", err)
	}

	cfg := config.Configuration{
		Consul: config.Consul{
			Address:        srv.HTTPAddr,
			PubMaxRoutines: config.DefaultConsulPubMaxRoutines,
		},
	}

	client, err := cfg.GetNewConsulClient()
	require.Nil(t, err)

	kv := client.KV()
	InitConsulPublisher(cfg.Consul.PubMaxRoutines, kv)
	defer srv.Stop()

	t.Run("StoreConsulKeyAsString", func(t *testing.T) {
		testStoreConsulKeyAsString(t, srv)
	})

	t.Run("StoreConsulKeyAsStringWithFlags", func(t *testing.T) {
		testStoreConsulKeyAsStringWithFlags(t, kv)
	})

	t.Run("StoreConsulKey", func(t *testing.T) {
		testStoreConsulKey(t, kv)
	})
	t.Run("StoreConsulKeyWithFlags", func(t *testing.T) {
		testStoreConsulKeyWithFlags(t, kv)
	})
	t.Run("consulStore_StoreConsulKeyWithFlags", func(t *testing.T) {
		testConsulStoreStoreConsulKeyWithFlags(t, kv)
	})
}

// buildDeploymentID allows to create a deploymentID from the test name value
func buildDeploymentID(t testing.TB) string {
	return strings.Replace(t.Name(), "/", "_", -1)
}

func testStoreConsulKeyAsString(t *testing.T, srv *testutil.TestServer) {
	deploymentID := buildDeploymentID(t)
	type args struct {
		key   string
		value string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"Test Simple KV Store", args{deploymentID + "/key1", "value1"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := StoreConsulKeyAsString(tt.args.key, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("StoreConsulKeyAsString() error = %v, wantErr %v", err, tt.wantErr)
			}
			actual := srv.GetKVString(t, tt.args.key)
			if actual != tt.args.value {
				t.Errorf("StoreConsulKeyAsString() key: %q, expected %q, actual %q", tt.args.key, tt.args.value, actual)
			}
		})
	}
}

func testStoreConsulKeyAsStringWithFlags(t *testing.T, kv *api.KV) {
	deploymentID := buildDeploymentID(t)
	type args struct {
		key   string
		value string
		flags uint64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"Test Simple KV Store with flag", args{deploymentID + "/key1", "value1", 45}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := StoreConsulKeyAsStringWithFlags(tt.args.key, tt.args.value, tt.args.flags); (err != nil) != tt.wantErr {
				t.Errorf("StoreConsulKeyAsStringWithFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
			actual, _, err := kv.Get(tt.args.key, nil)
			assert.NoError(t, err)
			assert.NotNil(t, actual)

			if string(actual.Value) != tt.args.value {
				t.Errorf("StoreConsulKeyAsStringWithFlags() value for key: %q, expected %q, actual %q", tt.args.key, tt.args.value, string(actual.Value))
			}

			if actual.Flags != tt.args.flags {
				t.Errorf("StoreConsulKeyAsStringWithFlags() flags for key: %q, expected %d, actual %d", tt.args.key, tt.args.flags, actual.Flags)
			}
		})
	}
}

func testStoreConsulKey(t *testing.T, kv *api.KV) {
	deploymentID := buildDeploymentID(t)
	type args struct {
		key   string
		value []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"Test Simple KV Store with flag", args{deploymentID + "/key1", []byte("value1")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := StoreConsulKey(tt.args.key, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("StoreConsulKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			actual, _, err := kv.Get(tt.args.key, nil)
			assert.NoError(t, err)
			assert.NotNil(t, actual)

			assert.Equal(t, tt.args.value, actual.Value, "StoreConsulKeyAsStringWithFlags() value missmatch")
		})
	}
}

func testStoreConsulKeyWithFlags(t *testing.T, kv *api.KV) {
	deploymentID := buildDeploymentID(t)
	type args struct {
		key   string
		value []byte
		flags uint64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"Test Simple KV Store with flag", args{deploymentID + "/key1", []byte("value1"), 42}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := StoreConsulKeyWithFlags(tt.args.key, tt.args.value, tt.args.flags); (err != nil) != tt.wantErr {
				t.Errorf("StoreConsulKeyWithFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
			actual, _, err := kv.Get(tt.args.key, nil)
			assert.NoError(t, err)
			assert.NotNil(t, actual)

			assert.Equal(t, tt.args.value, actual.Value, "StoreConsulKeyAsStringWithFlags() value missmatch")

			if actual.Flags != tt.args.flags {
				t.Errorf("StoreConsulKeyAsStringWithFlags() flags for key: %q, expected %d, actual %d", tt.args.key, tt.args.flags, actual.Flags)
			}
		})
	}
}

func testConsulStoreStoreConsulKeyWithFlags(t *testing.T, kv *api.KV) {
	deploymentID := buildDeploymentID(t)

	type args struct {
		nb    int
		key   string
		value []byte
		flags uint64
	}
	tests := []struct {
		name             string
		txPackingTimeout time.Duration
		args             args
	}{
		{"TestSimpleNoTxn", 0, args{10, "key", []byte("vlae"), 30}},
		{"TestTxnNotCompleted", 5 * time.Millisecond, args{10, "key", []byte("vlae"), 30}},
		{"TestTxnExactly64Elem", 10 * time.Millisecond, args{64, "key", []byte("v"), 0}},
		{"TestTxnMoreThan64Elem", 10 * time.Millisecond, args{65, "key", []byte("v"), 0}},
		{"TestTxnMuchMoreThan64Elem", 10 * time.Millisecond, args{650, "key", []byte("v"), 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx, errGrp, cs := withContext(ctx)
			cs.txPackingTimeout = tt.txPackingTimeout

			for i := 1; i <= tt.args.nb; i++ {
				cs.StoreConsulKeyWithFlags(fmt.Sprintf("%s/%s-%d", deploymentID, tt.args.key, i), tt.args.value, tt.args.flags)
			}

			err := errGrp.Wait()
			require.NoError(t, err)

			for i := 1; i <= tt.args.nb; i++ {
				actual, _, err := kv.Get(fmt.Sprintf("%s/%s-%d", deploymentID, tt.args.key, i), nil)
				assert.NoError(t, err)
				assert.NotNil(t, actual)

				assert.Equal(t, tt.args.value, actual.Value, "StoreConsulKeyAsStringWithFlags() value missmatch")

				if actual.Flags != tt.args.flags {
					t.Errorf("StoreConsulKeyAsStringWithFlags() flags for key: %q, expected %d, actual %d", tt.args.key, tt.args.flags, actual.Flags)
				}
			}
		})
	}
}
