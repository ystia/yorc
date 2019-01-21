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

package commands

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/config"
)

// Test the following args:
// ./yorc server --infrastructure_infra1_auth_url http://localhost:5000/v2.0 --infrastructure_infra1_tenant_name validation
func TestServerInitInfraExtraFlagsWithSpaceDelimiter(t *testing.T) {
	args := []string{"./yorc server", "--infrastructure_infra1_auth_url", "http://localhost:5000/v2.0", "--infrastructure_infra1_tenant_name", "validation"}
	serverInitExtraFlags(args)

	require.Len(t, args, 5)
	require.Len(t, resolvedServerExtraParams[0].viperNames, 2)
	require.Equal(t, resolvedServerExtraParams[0].viperNames[0], "infrastructures.infra1.auth_url")
	require.Equal(t, resolvedServerExtraParams[0].viperNames[1], "infrastructures.infra1.tenant_name")
}

// Test the following args:
// ./yorc server --infrastructure_infra2_private_network_name=mag3-yorc-network --infrastructure_infra2_region=regionOne
func TestServerInitInfraExtraFlagsWithEqualDelimiter(t *testing.T) {
	args := []string{"./yorc server", "--infrastructure_infra2_private_network_name=mag3-yorc-network", "--infrastructure_infra2_region=regionOne"}
	serverInitExtraFlags(args)
	require.Len(t, args, 3)
	require.Len(t, resolvedServerExtraParams[0].viperNames, 2)
	require.Equal(t, resolvedServerExtraParams[0].viperNames[0], "infrastructures.infra2.private_network_name")
	require.Equal(t, resolvedServerExtraParams[0].viperNames[1], "infrastructures.infra2.region")
}

// Test the following args:
// ./yorc server --infrastructure_infra3_private_network_name=mag3-yorc-network --infrastructure_infra3_region regionOne
func TestServerInitInfraExtraFlagsWithSpaceAndEqualDelimiters(t *testing.T) {
	args := []string{"./yorc server", "--infrastructure_infra3_private_network_name=mag3-yorc-network", "--infrastructure_infra3_region", "regionOne"}
	serverInitExtraFlags(args)

	require.Len(t, args, 4)
	require.Len(t, resolvedServerExtraParams[0].viperNames, 2)
	require.Equal(t, resolvedServerExtraParams[0].viperNames[0], "infrastructures.infra3.private_network_name")
	require.Equal(t, resolvedServerExtraParams[0].viperNames[1], "infrastructures.infra3.region")
}

// Test the following args:
// ./yorc server --infrastructure_infra4_auth_url http://localhost:5000/v2.0 --infrastructure_infra4_secured --infrastructure_infra4_tenant_name validation
func TestServerInitInfraExtraFlagsWithSpaceDelimiterAndBool(t *testing.T) {
	args := []string{"./yorc server", "--infrastructure_infra4_auth_url", "http://localhost:5000/v2.0", "--infrastructure_infra4_secured", "--infrastructure_infra4_tenant_name", "validation"}
	serverInitExtraFlags(args)

	require.Len(t, args, 6)
	require.Len(t, resolvedServerExtraParams[0].viperNames, 3)
	require.Equal(t, resolvedServerExtraParams[0].viperNames[0], "infrastructures.infra4.auth_url")
	require.Equal(t, resolvedServerExtraParams[0].viperNames[1], "infrastructures.infra4.secured")
	require.Equal(t, resolvedServerExtraParams[0].viperNames[2], "infrastructures.infra4.tenant_name")
}

// Test the following args:
// ./yorc server --infrastructure_infra4_auth_url http://localhost:5000/v2.0 --infrastructure_infra4_tenant_name validation --infrastructure_infra4_secured
func TestServerInitInfraExtraFlagsWithSpaceDelimiterAndBoolAtEnd(t *testing.T) {
	args := []string{"./yorc server", "--infrastructure_infra5_auth_url", "http://localhost:5000/v2.0", "--infrastructure_infra5_tenant_name", "validation", "--infrastructure_infra5_secured"}
	serverInitExtraFlags(args)

	require.Len(t, args, 6)
	require.Len(t, resolvedServerExtraParams[0].viperNames, 3)
	require.Equal(t, resolvedServerExtraParams[0].viperNames[0], "infrastructures.infra5.auth_url")
	require.Equal(t, resolvedServerExtraParams[0].viperNames[1], "infrastructures.infra5.tenant_name")
	require.Equal(t, resolvedServerExtraParams[0].viperNames[2], "infrastructures.infra5.secured")
}

// Test the following args:
// ./yorc server --vault_auth_url http://localhost:5000/v2.0 --vault_tenant_name validation
func TestServerInitVaultExtraFlagsWithSpaceDelimiter(t *testing.T) {
	args := []string{"./yorc server", "--vault_auth_url", "http://localhost:5000/v2.0", "--vault_tenant_name", "validation"}
	serverInitExtraFlags(args)

	require.Len(t, args, 5)
	require.Len(t, resolvedServerExtraParams[1].viperNames, 2)
	require.Equal(t, resolvedServerExtraParams[1].viperNames[0], "vault.auth_url")
	require.Equal(t, resolvedServerExtraParams[1].viperNames[1], "vault.tenant_name")
}

// Test the following args:
// ./yorc server --vault_private_network_name=mag3-yorc-network --vault_region=regionOne
func TestServerInitVaultExtraFlagsWithEqualDelimiter(t *testing.T) {
	args := []string{"./yorc server", "--vault_private_network_name=mag3-yorc-network", "--vault_region=regionOne"}
	serverInitExtraFlags(args)
	require.Len(t, args, 3)
	require.Len(t, resolvedServerExtraParams[1].viperNames, 2)
	require.Equal(t, resolvedServerExtraParams[1].viperNames[0], "vault.private_network_name")
	require.Equal(t, resolvedServerExtraParams[1].viperNames[1], "vault.region")
}

// Test the following args:
// ./yorc server --vault_public_network_name=mag3-yorc-network --vault_region2 regionOne
func TestServerInitVaultExtraFlagsWithSpaceAndEqualDelimiters(t *testing.T) {
	args := []string{"./yorc server", "--vault_public_network_name=mag3-yorc-network", "--vault_region2", "regionOne"}
	serverInitExtraFlags(args)

	require.Len(t, args, 4)
	require.Len(t, resolvedServerExtraParams[1].viperNames, 2)
	require.Equal(t, resolvedServerExtraParams[1].viperNames[0], "vault.public_network_name")
	require.Equal(t, resolvedServerExtraParams[1].viperNames[1], "vault.region2")
}

// Test the following args:
// ./yorc server --vault_auth_url2 http://localhost:5000/v2.0 --vault_secured2 --vault_tenant_name2 validation
func TestServerInitVaultExtraFlagsWithSpaceDelimiterAndBool(t *testing.T) {
	args := []string{"./yorc server", "--vault_auth_url2", "http://localhost:5000/v2.0", "--vault_secured2", "--vault_tenant_name2", "validation"}
	serverInitExtraFlags(args)

	require.Len(t, args, 6)
	require.Len(t, resolvedServerExtraParams[1].viperNames, 3)
	require.Equal(t, resolvedServerExtraParams[1].viperNames[0], "vault.auth_url2")
	require.Equal(t, resolvedServerExtraParams[1].viperNames[1], "vault.secured2")
	require.Equal(t, resolvedServerExtraParams[1].viperNames[2], "vault.tenant_name2")
}

// Test the following args:
// ./yorc server --vault_auth_url3 http://localhost:5000/v2.0 --vault_tenant_name3 validation --vault_secured
func TestServerInitVaultExtraFlagsWithSpaceDelimiterAndBoolAtEnd(t *testing.T) {
	args := []string{"./yorc server", "--vault_auth_url3", "http://localhost:5000/v2.0", "--vault_tenant_name3", "validation", "--vault_secured3"}
	serverInitExtraFlags(args)

	require.Len(t, args, 6)
	require.Len(t, resolvedServerExtraParams[1].viperNames, 3)
	require.Equal(t, resolvedServerExtraParams[1].viperNames[0], "vault.auth_url3")
	require.Equal(t, resolvedServerExtraParams[1].viperNames[1], "vault.tenant_name3")
	require.Equal(t, resolvedServerExtraParams[1].viperNames[2], "vault.secured3")
}

// Tests configuration values:
// - using a configuration file with deprecated values (backward compatibility check)
// - using a configuration file with the expected format
func TestConfigFile(t *testing.T) {

	fileToExpectedValues := []struct {
		SubTestName   string
		FileName      string
		AnsibleConfig config.Ansible
		ConsulConfig  config.Consul
	}{
		{SubTestName: "config_flat",
			FileName: "testdata/config_flat.yorc.json",
			AnsibleConfig: config.Ansible{
				UseOpenSSH:              true,
				DebugExec:               true,
				ConnectionRetries:       10,
				OperationRemoteBaseDir:  "test_base_dir",
				KeepOperationRemotePath: true,
				ArchiveArtifacts:        true,
				CacheFacts:              true,
				JobsChecksPeriod:        15 * time.Second,
			},
			ConsulConfig: config.Consul{
				Token:               "testToken",
				Datacenter:          "testDC",
				Address:             "http://127.0.0.1:8500",
				Key:                 "testKeyFile",
				Cert:                "testCertFile",
				CA:                  "testCACert",
				CAPath:              "testCAPath",
				SSL:                 true,
				SSLVerify:           false,
				PubMaxRoutines:      1234,
				TLSHandshakeTimeout: 30 * time.Second,
			},
		},
		{SubTestName: "config_structured",
			FileName: "testdata/config_structured.yorc.json",
			AnsibleConfig: config.Ansible{
				UseOpenSSH:              true,
				DebugExec:               true,
				ConnectionRetries:       11,
				OperationRemoteBaseDir:  "test_base_dir2",
				KeepOperationRemotePath: true,
				ArchiveArtifacts:        true,
				CacheFacts:              true,
				JobsChecksPeriod:        15 * time.Second,
			},
			ConsulConfig: config.Consul{
				Token:               "testToken2",
				Datacenter:          "testDC2",
				Address:             "http://127.0.0.1:8502",
				Key:                 "testKeyFile2",
				Cert:                "testCertFile2",
				CA:                  "testCACert2",
				CAPath:              "testCAPath2",
				SSL:                 true,
				SSLVerify:           false,
				PubMaxRoutines:      4321,
				TLSHandshakeTimeout: 51 * time.Second,
			},
		},
	}

	for _, fileToExpectedValue := range fileToExpectedValues {
		t.Run(fileToExpectedValue.SubTestName, func(t *testing.T) {
			testResetConfig()
			setConfig()
			viper.SetConfigFile(fileToExpectedValue.FileName)
			initConfig()
			testConfig := GetConfig()

			assert.Equal(t, fileToExpectedValue.AnsibleConfig, testConfig.Ansible, "Ansible configuration differs from value defined in config file %s", fileToExpectedValue.FileName)
			assert.Equal(t, fileToExpectedValue.ConsulConfig, testConfig.Consul, "Consul configuration differs from value defined in config file %s", fileToExpectedValue.FileName)
		})
	}
}

// Tests Ansible configuration default values
func TestAnsibleDefaultValues(t *testing.T) {

	expectedAnsibleConfig := config.Ansible{
		UseOpenSSH:              false,
		DebugExec:               false,
		ConnectionRetries:       5,
		OperationRemoteBaseDir:  ".yorc",
		KeepOperationRemotePath: false,
		JobsChecksPeriod:        15 * time.Second,
	}

	testResetConfig()
	setConfig()
	initConfig()
	testConfig := GetConfig()

	assert.Equal(t, expectedAnsibleConfig, testConfig.Ansible, "Ansible configuration differs from expected default configuration")
}

// Tests Consul configuration default values
func TestConsulDefaultValues(t *testing.T) {

	expectedConsulConfig := config.Consul{
		Token:               "anonymous",
		Datacenter:          "dc1",
		Address:             "",
		Key:                 "",
		Cert:                "",
		CA:                  "",
		CAPath:              "",
		SSL:                 false,
		SSLVerify:           true,
		PubMaxRoutines:      config.DefaultConsulPubMaxRoutines,
		TLSHandshakeTimeout: config.DefaultConsulTLSHandshakeTimeout,
	}

	testResetConfig()
	setConfig()
	initConfig()
	testConfig := GetConfig()

	assert.Equal(t, expectedConsulConfig, testConfig.Consul, "Consul configuration differs from expected default configuration")
}

// Tests Ansible configuration using environment variables
func TestAnsibleEnvVariables(t *testing.T) {
	expectedAnsibleConfig := config.Ansible{
		UseOpenSSH:              true,
		DebugExec:               true,
		ConnectionRetries:       12,
		OperationRemoteBaseDir:  "testEnvBaseDir",
		KeepOperationRemotePath: true,
		JobsChecksPeriod:        15 * time.Second,
	}

	// Set Ansible configuration environment ariables
	os.Setenv("YORC_ANSIBLE_USE_OPENSSH", strconv.FormatBool(expectedAnsibleConfig.UseOpenSSH))
	os.Setenv("YORC_ANSIBLE_DEBUG", strconv.FormatBool(expectedAnsibleConfig.DebugExec))
	os.Setenv("YORC_ANSIBLE_CONNECTION_RETRIES", strconv.Itoa(expectedAnsibleConfig.ConnectionRetries))
	os.Setenv("YORC_OPERATION_REMOTE_BASE_DIR", expectedAnsibleConfig.OperationRemoteBaseDir)
	os.Setenv("YORC_KEEP_OPERATION_REMOTE_PATH", strconv.FormatBool(expectedAnsibleConfig.KeepOperationRemotePath))

	testResetConfig()
	setConfig()
	initConfig()
	testConfig := GetConfig()

	assert.Equal(t, expectedAnsibleConfig, testConfig.Ansible, "Ansible configuration differs from expected environment configuration")

	// cleanup env
	os.Unsetenv("YORC_ANSIBLE_USE_OPENSSH")
	os.Unsetenv("YORC_ANSIBLE_DEBUG")
	os.Unsetenv("YORC_ANSIBLE_CONNECTION_RETRIES")
	os.Unsetenv("YORC_OPERATION_REMOTE_BASE_DIR")
	os.Unsetenv("YORC_KEEP_OPERATION_REMOTE_PATH")
}

// Tests Consul configuration using environment variables
func TestConsulEnvVariables(t *testing.T) {

	expectedConsulConfig := config.Consul{
		Token:               "testEnvToken",
		Datacenter:          "testEnvDC",
		Address:             "testEnvAddress",
		Key:                 "testEnvKey",
		Cert:                "testEnvCert",
		CA:                  "testEnvCA",
		CAPath:              "testEnvCAPath",
		SSL:                 true,
		SSLVerify:           false,
		PubMaxRoutines:      125,
		TLSHandshakeTimeout: 11 * time.Second,
	}

	// Set Consul configuration environment variables
	os.Setenv("YORC_CONSUL_ADDRESS", expectedConsulConfig.Address)
	os.Setenv("YORC_CONSUL_TOKEN", expectedConsulConfig.Token)
	os.Setenv("YORC_CONSUL_DATACENTER", expectedConsulConfig.Datacenter)
	os.Setenv("YORC_CONSUL_KEY_FILE", expectedConsulConfig.Key)
	os.Setenv("YORC_CONSUL_CERT_FILE", expectedConsulConfig.Cert)
	os.Setenv("YORC_CONSUL_CA_CERT", expectedConsulConfig.CA)
	os.Setenv("YORC_CONSUL_CA_PATH", expectedConsulConfig.CAPath)
	os.Setenv("YORC_CONSUL_SSL", strconv.FormatBool(expectedConsulConfig.SSL))
	os.Setenv("YORC_CONSUL_SSL_VERIFY", strconv.FormatBool(expectedConsulConfig.SSLVerify))
	os.Setenv("YORC_CONSUL_PUBLISHER_MAX_ROUTINES", strconv.Itoa(expectedConsulConfig.PubMaxRoutines))
	os.Setenv("YORC_CONSUL_TLS_HANDSHAKE_TIMEOUT", expectedConsulConfig.TLSHandshakeTimeout.String())

	testResetConfig()
	setConfig()
	initConfig()
	testConfig := GetConfig()

	assert.Equal(t, expectedConsulConfig, testConfig.Consul, "Consul configuration differs from expected environment configuration")

	// cleanup env
	os.Unsetenv("YORC_CONSUL_ADDRESS")
	os.Unsetenv("YORC_CONSUL_TOKEN")
	os.Unsetenv("YORC_CONSUL_DATACENTER")
	os.Unsetenv("YORC_CONSUL_KEY_FILE")
	os.Unsetenv("YORC_CONSUL_CERT_FILE")
	os.Unsetenv("YORC_CONSUL_CA_CERT")
	os.Unsetenv("YORC_CONSUL_CA_PATH")
	os.Unsetenv("YORC_CONSUL_SSL")
	os.Unsetenv("YORC_CONSUL_SSL_VERIFY")
	os.Unsetenv("YORC_CONSUL_PUBLISHER_MAX_ROUTINES")
	os.Unsetenv("YORC_CONSUL_TLS_HANDSHAKE_TIMEOUT")
}

// Tests Ansible configuration using persistent flags
func TestAnsiblePersistentFlags(t *testing.T) {

	expectedAnsibleConfig := config.Ansible{
		UseOpenSSH:              true,
		DebugExec:               true,
		ConnectionRetries:       15,
		OperationRemoteBaseDir:  "testPFlagBaseDir",
		KeepOperationRemotePath: true,
		JobsChecksPeriod:        15 * time.Second,
	}

	ansiblePFlagConfiguration := map[string]string{
		"ansible_use_openssh":        strconv.FormatBool(expectedAnsibleConfig.UseOpenSSH),
		"ansible_debug":              strconv.FormatBool(expectedAnsibleConfig.DebugExec),
		"ansible_connection_retries": strconv.Itoa(expectedAnsibleConfig.ConnectionRetries),
		"operation_remote_base_dir":  expectedAnsibleConfig.OperationRemoteBaseDir,
		"keep_operation_remote_path": strconv.FormatBool(expectedAnsibleConfig.KeepOperationRemotePath),
	}

	testResetConfig()
	// No Configuration file specified here. Just checking environment variables.
	setConfig()
	initConfig()

	// Set persistent flags
	for key, value := range ansiblePFlagConfiguration {
		err := serverCmd.PersistentFlags().Set(key, value)
		require.NoError(t, err, "Could not set persistent flag %s", key)
	}

	testConfig := GetConfig()

	assert.Equal(t, expectedAnsibleConfig, testConfig.Ansible, "Ansible configuration differs from persistent flags settings")
}

// Tests Consul configuration using persistent flags
func TestConsulPersistentFlags(t *testing.T) {

	expectedConsulConfig := config.Consul{
		Token:               "testPFlagToken",
		Datacenter:          "testPFlagDC",
		Address:             "testPFlagAddress",
		Key:                 "testPFlagKey",
		Cert:                "testPFlagCert",
		CA:                  "testPFlagCA",
		CAPath:              "testEnvCAPath",
		SSL:                 true,
		SSLVerify:           false,
		PubMaxRoutines:      123,
		TLSHandshakeTimeout: 12 * time.Second,
	}

	consulPFlagConfiguration := map[string]string{
		"consul_token":                  expectedConsulConfig.Token,
		"consul_datacenter":             expectedConsulConfig.Datacenter,
		"consul_address":                expectedConsulConfig.Address,
		"consul_key_file":               expectedConsulConfig.Key,
		"consul_cert_file":              expectedConsulConfig.Cert,
		"consul_ca_cert":                expectedConsulConfig.CA,
		"consul_ca_path":                expectedConsulConfig.CAPath,
		"consul_ssl":                    strconv.FormatBool(expectedConsulConfig.SSL),
		"consul_ssl_verify":             strconv.FormatBool(expectedConsulConfig.SSLVerify),
		"consul_publisher_max_routines": strconv.Itoa(expectedConsulConfig.PubMaxRoutines),
		"consul_tls_handshake_timeout":  expectedConsulConfig.TLSHandshakeTimeout.String(),
	}

	testResetConfig()
	// No Configuration file specified here. Just checking environment variables.
	setConfig()
	initConfig()

	// Set persistent flags
	for key, value := range consulPFlagConfiguration {
		err := serverCmd.PersistentFlags().Set(key, value)
		require.NoError(t, err, "Could not set persistent flag %s", key)
	}

	testConfig := GetConfig()

	assert.Equal(t, expectedConsulConfig, testConfig.Consul, "Consul configuration differs from persistent flags settings")
}

// Test utility resetting the Orchestrator config
func testResetConfig() {
	viper.Reset()
	serverCmd.ResetFlags()
}

func TestVersionToConstraint(t *testing.T) {
	t.Parallel()

	type args struct {
		constraint string
		version    string
		level      string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "MinorConstraint", args: args{level: "minor", version: "1.2.3", constraint: "~>"}, want: "~> 1.2"},
		{name: "PatchConstraint", args: args{level: "patch", version: "1.2.3", constraint: "->"}, want: "-> 1.2.3"},
		{name: "MajorConstraint", args: args{level: "major", version: "1.2.3", constraint: ">="}, want: ">= 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionToConstraint(tt.args.constraint, tt.args.version, tt.args.level); got != tt.want {
				t.Errorf("versionToConstraint got:%q, want:%q", got, tt.want)
			}
		})
	}
}
