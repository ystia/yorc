package openstack

import (
	"testing"

	"novaforge.bull.com/starlings-janus/janus/testutil"
)

// The aim of this function is to run all package tests with consul server dependency with only one consul server start
func TestRunConsulOpenstackPackageTests(t *testing.T) {
	srv, client := testutil.NewTestConsulInstance(t)
	kv := client.KV()
	defer srv.Stop()

	t.Run("groupOpenstack", func(t *testing.T) {
		t.Run("TestGenerateOSBSVolumeSizeConvert", func(t *testing.T) {
			testGenerateOSBSVolumeSizeConvert(t, srv, kv)
		})
		t.Run("TestGenerateOSBSVolumeSizeConvertError", func(t *testing.T) {
			testGenerateOSBSVolumeSizeConvertError(t, srv, kv)
		})
		t.Run("TestGenerateOSBSVolumeMissingSize", func(t *testing.T) {
			testGenerateOSBSVolumeMissingSize(t, srv, kv)
		})
		t.Run("TestGenerateOSBSVolumeWrongType", func(t *testing.T) {
			testGenerateOSBSVolumeWrongType(t, srv, kv)
		})
		t.Run("TestGenerateOSBSVolumeCheckOptionalValues", func(t *testing.T) {
			testGenerateOSBSVolumeCheckOptionalValues(t, srv, kv)
		})
		t.Run("TestGeneratePoolIP", func(t *testing.T) {
			testGeneratePoolIP(t, srv, kv)
		})
		t.Run("TestGenerateSingleIp", func(t *testing.T) {
			testGenerateSingleIP(t, srv, kv)
		})
		t.Run("TestGenerateMultipleIP", func(t *testing.T) {
			testGenerateMultipleIP(t, srv, kv)
		})
	})
}
