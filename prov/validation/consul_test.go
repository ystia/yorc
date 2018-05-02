package validation

import (
	"testing"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulValidationPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	defer srv.Stop()
	kv := client.KV()
	cfg := config.Configuration{
		Consul: config.Consul{
			Address:        srv.HTTPAddr,
			PubMaxRoutines: config.DefaultConsulPubMaxRoutines,
		},
	}

	t.Run("groupValidation", func(t *testing.T) {
		t.Run("testPostComputeCreationHook", func(t *testing.T) {
			testPostComputeCreationHook(t, kv, cfg)
		})
	})
}
