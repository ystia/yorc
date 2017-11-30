package slurm

import (
	"testing"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulSlurmPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()

	// AWS infrastructure config
	cfg := config.Configuration{
		Infrastructures: map[string]config.InfrastructureConfig{
			infrastructureName: {
				"user_name": "root",
				"password":  "pwd",
				"name":      "slurm",
				"url":       "1.2.3.4",
				"port":      "1234",
			}}}

	t.Run("groupAWS", func(t *testing.T) {
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
