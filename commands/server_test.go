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

// Tests Ansible and Consul Config Values:
// - using a configuration file with flat data (backwrd compatibility)
// - using a configuration file with structured data
func TestConfigFile(t *testing.T) {

	fileToExpectedValues := []struct {
		FileName string
		AnsibleConfig config.Ansible
		ConsulConfig config.Consul
	}{
		{FileName: "testData/config_flat.yorc.json",
			AnsibleConfig: config.Ansible{
				UseOpenSSH: 				true,
				DebugExec: 					true,
				ConnectionRetries: 			10,
				OperationRemoteBaseDir:		"test_base_dir",
				KeepOperationRemotePath:	true},
			ConsulConfig: config.Consul{
				Token:			"testToken",
				Datacenter:		"testDC",
				Address:		"http://127.0.0.1:8500",
				Key:			"testKeyFile",
				Cert:			"testCertFile",
				CA:				"testCACert",
				CAPath:			"testCAPath",
				SSL:			true,
				SSLVerify:		false,
				PubMaxRoutines:	1234},
		},
		{FileName: "testData/config_structured.yorc.json",
			AnsibleConfig: config.Ansible{
			   UseOpenSSH: 				true,
			   DebugExec:				true,
			   ConnectionRetries:		11,
			   OperationRemoteBaseDir:	"test_base_dir2",
			   KeepOperationRemotePath:	true,
			},
			ConsulConfig: config.Consul{
			   Token:			"testToken2",
			   Datacenter:		"testDC2",
			   Address:			"http://127.0.0.1:8502",
			   Key:				"testKeyFile2",
			   Cert:			"testCertFile2",
			   CA:				"testCACert2",
			   CAPath:			"testCAPath2",
			   SSL:				true,
			   SSLVerify:		false,
			   PubMaxRoutines:	4321,
			},
		},
	}
	
	for _, fileToExpectedValue := range fileToExpectedValues {
		
		testResetConfig()
		setConfig()
		testLoadConfigFile(t, fileToExpectedValue.FileName)
		testConfig := getConfig()
	
		assert.Equal(t, fileToExpectedValue.AnsibleConfig, testConfig.Ansible, "Ansible configuration differs from value defined in config file %s", fileToExpectedValue.FileName) 
		assert.Equal(t, fileToExpectedValue.ConsulConfig, testConfig.Consul, "Consul configuration differs from value defined in config file %s", fileToExpectedValue.FileName)
	}

}

// Tests Ansible and Consul default values
func TestDefaultValues(t *testing.T){

	// TODO: uncomment this block
/*
	expectedAnsibleConfig := config.Ansible {
		UseOpenSSH: 				false,
		DebugExec: 					false,
		ConnectionRetries: 			5,
		OperationRemoteBaseDir:		".yorc",
		KeepOperationRemotePath:	false,
	}
	
	expectedConsulConfig := config.Consul {
		Token: 			"anonymous",
		Datacenter: 	"dc1",
		Address: 		"",
		Key: 			"",
		Cert: 			"",
		CA: 			"",
		CAPath: 		"",
		SSL: 			false,
		SSLVerify: 		true,
		PubMaxRoutines:	config.DefaultConsulPubMaxRoutines,
	}

	testResetConfig()
	// No Configuration file specified here to check default values
	setConfig()
	initConfig()
	TODO : uncomment this block
	testConfig := getConfig()
	
	assert.Equal(t, expectedAnsibleConfig, testConfig.Ansible, "Ansible configuration differs from expected default configuration") 
	assert.Equal(t, expectedConsulConfig, testConfig.Consul, "Consul configuration differs from expected default configuration")
	viper.Debug()
*/

}

// Tests Ansible and Consul configuration using environment variables
func TestEnvVariables(t *testing.T){
	expectedAnsibleConfig := config.Ansible {
		UseOpenSSH: 				true,
		DebugExec: 					true,
		ConnectionRetries: 			12,
		OperationRemoteBaseDir:		".yorctestEnv",
		KeepOperationRemotePath:	true,
	}
	
	expectedConsulConfig := config.Consul {
		Token: 			"testTokenEnv",
		Datacenter: 	"testEnvDC",
		Address: 		"testEnvAddress",
		Key: 			"testEnvKey",
		Cert: 			"testEnvCert",
		CA: 			"testEnvCA",
		CAPath: 		"testEnvCAPath",
		SSL: 			true,
		SSLVerify: 		false,
		PubMaxRoutines:	125,
	}

	// Set Ansible configuration environment ariables
	os.Setenv("YORC_ANSIBLE_USE_OPENSSH", strconv.FormatBool(expectedAnsibleConfig.UseOpenSSH))
	os.Setenv("YORC_ANSIBLE_DEBUG", strconv.FormatBool(expectedAnsibleConfig.DebugExec))
	os.Setenv("YORC_ANSIBLE_CONNECTION_RETRIES", strconv.Itoa(expectedAnsibleConfig.ConnectionRetries))
	os.Setenv("YORC_OPERATION_REMOTE_BASE_DIR", expectedAnsibleConfig.OperationRemoteBaseDir)
	os.Setenv("YORC_KEEP_OPERATION_REMOTE_PATH", strconv.FormatBool(expectedAnsibleConfig.KeepOperationRemotePath))

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

	testResetConfig()	
	// No Configuration file specified here. Just checking environment variables.
	setConfig()
	initConfig()
	testConfig := getConfig()
	
	assert.Equal(t, expectedAnsibleConfig, testConfig.Ansible, "Ansible configuration differs from expected environment configuration") 
	assert.Equal(t, expectedConsulConfig, testConfig.Consul, "Consul configuration differs from expected environment configuration")
}

// Test utility resetting the Orchestrator config
func testResetConfig(){
	viper.Reset()
	serverCmd.ResetFlags()
}

// Test utility loading the content of a config file
func testLoadConfigFile(t *testing.T, fileName string){

	viper.SetConfigFile(fileName)
	err := viper.ReadInConfig()
	require.NoError(t, err, "Failure reading Config file %s", fileName)

	initConfig()
} 