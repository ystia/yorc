package slurm

import (
	"testing"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulSlurmPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()

	// Slurm infrastructure config
	cfg := config.Configuration{
		Infrastructures: map[string]config.DynamicMap{
			infrastructureName: config.DynamicMap{
				"user_name": "root",
				"password":  "pwd",
				"name":      "slurm",
				"url":       "1.2.3.4",
				"port":      "1234",
			}}}

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
