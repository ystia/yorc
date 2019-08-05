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

package bootstrap

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	survey "gopkg.in/AlecAivazis/survey.v1"
	yaml "gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v4/commands"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/rest"
	"github.com/ystia/yorc/v4/tosca"
)

var inputValues TopologyValues

// Default values for inputs
type defaultInputType struct {
	description string
	value       interface{}
}

var (
	ansibleDefaultInputs = map[string]defaultInputType{
		"ansible.version": defaultInputType{
			description: "Ansible version",
			value:       ansibleVersion,
		},
		"ansible.extra_package_repository_url": defaultInputType{
			description: "URL of package indexes where to find the ansible package, instead of the default Python Package repository",
			value:       "",
		},
		"ansible.use_openssh": defaultInputType{
			description: "Prefer OpenSSH over Paramiko, python implementation of SSH",
			value:       false,
		},
	}

	yorcDefaultInputs = map[string]defaultInputType{
		"yorc.download_url": defaultInputType{
			description: "Yorc download URL",
			value:       getYorcDownloadURL(),
		},
		"yorc.port": defaultInputType{
			description: "Yorc HTTP REST API port",
			value:       config.DefaultHTTPPort,
		},
		"yorc.data_dir": defaultInputType{
			description: "Bootstrapped Yorc Home directory",
			value:       "/var/yorc",
		},
		"yorc.private_key_file": defaultInputType{
			description: "Path to ssh private key accessible locally",
			value:       "",
		},
		"yorc.ca_passphrase": defaultInputType{
			description: "Certificate authority private key passphrase",
			value:       "",
		},
		"yorc.ca_key_file": defaultInputType{
			description: "Path to Certificate Authority private key, accessible locally",
			value:       "",
		},
		"yorc.ca_pem_file": defaultInputType{
			description: "Path to PEM-encoded Certificate Authority, accessible locally",
			value:       "",
		},
		"yorc.workers_number": defaultInputType{
			description: "Number of Yorc workers handling bootstrap deployment tasks",
			value:       config.DefaultWorkersNumber,
		},
		"yorc.resources_prefix": defaultInputType{
			description: "Prefix used to create resources (like Computes and so on)",
			value:       "yorc-",
		},
	}

	yorcPluginDefaultInputs = map[string]defaultInputType{
		"yorc_plugin.download_url": defaultInputType{
			description: "Yorc plugin download URL",
			value:       getYorcPluginDownloadURL(),
		},
	}

	alien4CloudDefaultInputs = map[string]defaultInputType{
		"alien4cloud.download_url": defaultInputType{
			description: "Alien4Cloud download URL",
			value: fmt.Sprintf(
				"https://fastconnect.org/maven/content/repositories/opensource/alien4cloud/alien4cloud-dist/%s/alien4cloud-dist-%s-dist.tar.gz",
				alien4cloudVersion, alien4cloudVersion),
		},
		"alien4cloud.port": defaultInputType{
			description: "Alien4Cloud port",
			value:       8088,
		},
		"alien4cloud.user": defaultInputType{
			description: "Alien4Cloud user",
			value:       "admin",
		},
		"alien4cloud.password": defaultInputType{
			description: "Alien4Cloud password",
			value:       "admin",
		},
	}

	consulDefaultInputs = map[string]defaultInputType{
		"consul.download_url": defaultInputType{
			description: "Consul download URL",
			value: fmt.Sprintf("https://releases.hashicorp.com/consul/%s/consul_%s_linux_amd64.zip",
				consulVersion, consulVersion),
		},
		"consul.port": defaultInputType{
			description: "Consul port",
			value:       8543,
		},
		"consul.encrypt_key": defaultInputType{
			description: "16-bytes, Base64 encoded value of an encryption key used to encrypt Consul network traffic",
			value:       "",
		},
	}

	vaultDefaultInputs = map[string]defaultInputType{
		"vault.download_url": defaultInputType{
			description: "Hashicorp Vault download URL",
			value:       "https://releases.hashicorp.com/vault/1.0.3/vault_1.0.3_linux_amd64.zip",
		},
		"vault.port": defaultInputType{
			description: "Vault port",
			value:       8200,
		},
	}

	terraformDefaultInputs = map[string]defaultInputType{
		"terraform.download_url": defaultInputType{
			description: "Terraform download URL",
			value: fmt.Sprintf("https://releases.hashicorp.com/terraform/%s/terraform_%s_linux_amd64.zip",
				terraformVersion, terraformVersion),
		},
		"terraform.plugins_download_urls": defaultInputType{
			description: "Terraform plugins dowload URLs",
			value:       getTerraformPluginsDownloadURLs(),
		},
	}

	jdkDefaultInputs = map[string]defaultInputType{
		"jdk.download_url": defaultInputType{
			description: "Java Development Kit download URL",
			value:       "https://api.adoptopenjdk.net/v2/binary/releases/openjdk8?openjdk_impl=hotspot&os=linux&arch=x64&release=jdk8u212-b03&type=jdk",
		},
		"jdk.version": defaultInputType{
			description: "Java Development Kit version",
			value:       "1.8.0-212-b03",
		},
	}

	credentialsInputs = map[string]defaultInputType{
		"credentials.user": defaultInputType{
			description: "User Yorc uses to connect to Compute Nodes",
			value:       "",
		},
	}

	mandatoryHostPoolLabels = []string{"public_address", "private_address"}
)

// Infrastructure inputs, which when missing, will have to be provided interactively
type infrastructureInputType struct {
	Name         string
	Description  string
	Required     bool
	Secret       bool        `yaml:"secret,omitempty"`
	DataType     string      `yaml:"type"`
	DefaultValue interface{} `yaml:"default,omitempty"`
}

type bootstrapExtraParams struct {
	argPrefix   string
	envPrefix   string
	viperPrefix string
	viperNames  []string
	subSplit    int
	storeFn     bootstrapExtraParamStoreFn
	readConfFn  bootstrapExtraParamReadConf
}

type bootstrapExtraParamStoreFn func(values *TopologyValues, param string)
type bootstrapExtraParamReadConf func(values *TopologyValues)

var bootstrapExtraComputeParams bootstrapExtraParams
var bootstrapExtraAddressParams bootstrapExtraParams

func addBootstrapExtraComputeParams(values *TopologyValues, param string) {
	paramParts := strings.Split(param, ".")
	value := viper.Get(param)
	values.Compute.Set(paramParts[1], value)
}

func addBootstrapExtraAddressParams(values *TopologyValues, param string) {
	paramParts := strings.Split(param, ".")
	value := viper.Get(param)
	values.Address.Set(paramParts[1], value)
}

func readComputeViperConfig(values *TopologyValues) {
	computeConfig := viper.GetStringMap("compute")
	for k, v := range computeConfig {
		values.Compute.Set(k, v)
	}
}

func readAddressViperConfig(values *TopologyValues) {
	addressConfig := viper.GetStringMap("address")
	for k, v := range addressConfig {
		values.Address.Set(k, v)
	}
}

// setBootstrapExtraParams sets flags and default values related to bootstrap
// configuration
func setBootstrapExtraParams(args []string, cmd *cobra.Command) error {

	defaultInputs := []map[string]defaultInputType{
		ansibleDefaultInputs,
		yorcDefaultInputs,
		yorcPluginDefaultInputs,
		alien4CloudDefaultInputs,
		consulDefaultInputs,
		vaultDefaultInputs,
		terraformDefaultInputs,
		jdkDefaultInputs,
		credentialsInputs,
	}

	for _, defaultInput := range defaultInputs {

		for key, input := range defaultInput {
			flatKey := strings.Replace(key, ".", "_", 1)
			switch input.value.(type) {
			case string:
				bootstrapCmd.PersistentFlags().String(flatKey, input.value.(string), input.description)
			case int:
				bootstrapCmd.PersistentFlags().Int(flatKey, input.value.(int), input.description)
			case bool:
				bootstrapCmd.PersistentFlags().Bool(flatKey, input.value.(bool), input.description)
			case []string:
				bootstrapCmd.PersistentFlags().StringSlice(flatKey, input.value.([]string), input.description)
			default:
				return fmt.Errorf("Unexpected default value type for %s value %v", key, input.value)

			}

			bootstrapViper.BindPFlag(key, bootstrapCmd.PersistentFlags().Lookup(flatKey))
			bootstrapViper.BindEnv(key,
				strings.ToUpper(fmt.Sprintf("%s_%s",
					commands.EnvironmentVariablePrefix, flatKey)))
			bootstrapViper.SetDefault(key, input.value)
		}
	}

	bootstrapExtraComputeParams = bootstrapExtraParams{
		argPrefix:   "compute_",
		envPrefix:   "YORC_COMPUTE_",
		viperPrefix: "compute.",
		viperNames:  make([]string, 0),
		storeFn:     addBootstrapExtraComputeParams,
		readConfFn:  readComputeViperConfig,
	}

	bootstrapExtraAddressParams = bootstrapExtraParams{
		argPrefix:   "address_",
		envPrefix:   "YORC_ADDRESS_",
		viperPrefix: "address.",
		viperNames:  make([]string, 0),
		storeFn:     addBootstrapExtraAddressParams,
		readConfFn:  readAddressViperConfig,
	}

	resolvedBootstrapExtraParams := []*bootstrapExtraParams{
		&bootstrapExtraComputeParams,
		&bootstrapExtraAddressParams,
	}

	for _, extraParams := range resolvedBootstrapExtraParams {
		for i := range args {
			if strings.HasPrefix(args[i], "--"+extraParams.argPrefix) {
				var viperName, flagName string
				if strings.ContainsRune(args[i], '=') {
					// Handle the syntax --infrastructure_xxx_yyy = value
					flagParts := strings.Split(args[i], "=")
					flagName = strings.TrimLeft(flagParts[0], "-")
					viperName = strings.Replace(strings.Replace(
						flagName, extraParams.argPrefix, extraParams.viperPrefix, 1),
						"_", ".", extraParams.subSplit)
					if len(flagParts) == 1 {
						// Boolean flag
						cmd.PersistentFlags().Bool(flagName, false, "")
						viper.SetDefault(viperName, false)
					} else {
						cmd.PersistentFlags().String(flagName, "", "")
						viper.SetDefault(viperName, "")
					}
				} else {
					// Handle the syntax --infrastructure_xxx_yyy value
					flagName = strings.TrimLeft(args[i], "-")
					viperName = strings.Replace(strings.Replace(
						flagName, extraParams.argPrefix, extraParams.viperPrefix, 1),
						"_", ".", extraParams.subSplit)
					if len(args) > i+1 && !strings.HasPrefix(args[i+1], "--") {
						cmd.PersistentFlags().String(flagName, "", "")
						viper.SetDefault(viperName, "")
					} else {
						// Boolean flag
						cmd.PersistentFlags().Bool(flagName, false, "")
						viper.SetDefault(viperName, false)
					}
				}
				// Add viper flag
				viper.BindPFlag(viperName, cmd.PersistentFlags().Lookup(flagName))
				extraParams.viperNames = append(extraParams.viperNames, viperName)
			}
		}
		for _, envVar := range os.Environ() {
			if strings.HasPrefix(envVar, extraParams.envPrefix) {
				envVarParts := strings.SplitN(envVar, "=", 2)
				viperName := strings.ToLower(strings.Replace(
					strings.Replace(envVarParts[0], extraParams.envPrefix, extraParams.viperPrefix, 1),
					"_", ".", extraParams.subSplit))
				viper.BindEnv(viperName, envVarParts[0])
				if !collections.ContainsString(extraParams.viperNames, viperName) {
					extraParams.viperNames = append(extraParams.viperNames, viperName)
				}
			}
		}
	}

	return nil
}

// initializeInputs Initializes parameters from environment variables, CLI options,
// input file in argument, and asks for user input if needed
func initializeInputs(inputFilePath, resourcesPath string, configuration config.Configuration) error {

	var err error
	inputValues, err = getInputValues(inputFilePath)
	if err != nil {
		return err
	}

	//generating a deployment name if needed.
	if deploymentID == "" {
		t := time.Now()
		deploymentID = fmt.Sprintf("bootstrap-%d-%02d-%02d--%02d-%02d-%02d",
			t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	}

	// Get infrastructure from viper configuration if not provided in inputs
	if inputValues.Infrastructures == nil {
		inputValues.Infrastructures = configuration.Infrastructures
	}

	inputValues.Insecure = insecure

	//recovering infra type if needed
	if infrastructureType == "" {
		switch inputValues.Location.Type {
		case "OpenStack":
			infrastructureType = "openstack"
		case "Google Cloud":
			infrastructureType = "google"
		case "AWS":
			infrastructureType = "aws"
		case "HostsPool":
			infrastructureType = "hostspool"
		default:
			infrastructureType = ""
		}
	}
	// Now check for missing mandatory parameters and ask them to the user

	if infrastructureType == "" {

		// if one and only one infrastructure is already defined in inputs,
		// selecting this infrastructure
		if len(inputValues.Infrastructures) == 1 && len(inputValues.Hosts) == 0 {
			for k := range inputValues.Infrastructures {
				infrastructureType = k
			}
		} else if len(inputValues.Infrastructures) == 0 && len(inputValues.Hosts) > 0 {
			infrastructureType = "hostspool"
		}

		if infrastructureType == "" {
			fmt.Println("")
			prompt := &survey.Select{
				Message: "Select an infrastructure:",
				Options: []string{"Google", "AWS", "OpenStack", "HostsPool"},
			}
			survey.AskOne(prompt, &infrastructureType, nil)
			infrastructureType = strings.ToLower(infrastructureType)
		}
	}

	// Define node types to get according to the selecting infrastructure
	var infraNodeType, networkNodeType string
	switch infrastructureType {
	case "openstack":
		infraNodeType = "org.ystia.yorc.pub.infrastructure.OpenStackConfig"
		networkNodeType = "yorc.nodes.openstack.FloatingIP"
	case "google":
		infraNodeType = "org.ystia.yorc.pub.infrastructure.GoogleConfig"
		networkNodeType = "yorc.nodes.google.Address"
	case "aws":
		infraNodeType = "org.ystia.yorc.pub.infrastructure.AWSConfig"
		networkNodeType = "yorc.nodes.aws.PublicNetwork"
	case "hostspool":
		// No infrastructure defined in case of hosts pool
		// Hosts Pool don't have network-reletad on-demand resources
	default:
		return fmt.Errorf("Infrastruture type %s is not supported by bootstrap",
			infrastructureType)
	}

	// Get the infrastructure definition from resources
	nodeTypesFilePathPattern := filepath.Join(
		resourcesPath, "topology", "org.ystia.yorc.pub", "*", "types.yaml")

	matchingPath, err := filepath.Glob(nodeTypesFilePathPattern)
	if err != nil {
		return err
	}
	if len(matchingPath) == 0 {
		return fmt.Errorf("Found no node types definition file matching pattern %s", nodeTypesFilePathPattern)
	}

	data, err := ioutil.ReadFile(matchingPath[0])
	if err != nil {
		return err
	}
	var topology tosca.Topology
	if err := yaml.Unmarshal(data, &topology); err != nil {
		return err
	}

	fmt.Println("\nGetting Yorc configuration")

	// Set yorc and Alien4Cloud protocol
	if insecure {
		inputValues.Yorc.Protocol = "http"
		inputValues.Alien4cloud.Protocol = "http"
	} else {
		inputValues.Yorc.Protocol = "https"
		inputValues.Alien4cloud.Protocol = "https"
	}

	// Get path where to store files to be used by the local Yorc
	// when bootstrapping the remote Yorc
	resourcesAbsolutePath, err := filepath.Abs(resourcesPath)
	if err != nil {
		return err
	}

	// Update Yorc private key content if needed

	if inputValues.Yorc.PrivateKeyContent == "" {

		if inputValues.Yorc.PrivateKeyFile == "" {
			answer := struct {
				Value string
			}{}

			prompt := &survey.Input{
				Message: "Path to ssh private key accessible locally (required, will be stored as .ssh/yorc.pem on bootstrapped Yorc server Home):",
			}
			question := &survey.Question{
				Name:     "value",
				Prompt:   prompt,
				Validate: survey.Required,
			}
			if err := survey.Ask([]*survey.Question{question}, &answer); err != nil {
				return err
			}

			inputValues.Yorc.PrivateKeyFile = strings.TrimSpace(answer.Value)
		}

		data, err := ioutil.ReadFile(inputValues.Yorc.PrivateKeyFile)
		if err != nil {
			return err
		}
		inputValues.Yorc.PrivateKeyContent = string(data[:])
	} else {
		inputValues.Yorc.PrivateKeyFile = filepath.Join(resourcesAbsolutePath, "yorc.pem")
		err = ioutil.WriteFile(inputValues.Yorc.PrivateKeyFile, []byte(inputValues.Yorc.PrivateKeyContent), 0700)
		if err != nil {
			return err
		}

	}

	if !insecure {
		fmt.Println("\nGetting Certificate Authority configuration")
		if err := getCAConfiguration(
			&inputValues.Yorc,
			inputFilePath != "",
			resourcesAbsolutePath); err != nil {
			return err
		}

	}

	fmt.Println("\nGetting Consul configuration")
	if insecure {
		inputValues.Consul.Port = 8500
	} else {
		inputValues.Consul.TLSEnabled = true
		inputValues.Consul.TLSForChecksEnabled = true

		// Get or generate an encrytion key
		if inputValues.Consul.EncryptKey == "" && inputFilePath == "" {

			answer := struct {
				Value string
			}{}

			prompt := &survey.Input{
				Message: "16-bytes, Base64 encoded value of an encryption key used to encrypt Consul network traffic (if none set, one will be generated):",
			}
			question := &survey.Question{
				Name:   "value",
				Prompt: prompt,
			}
			if err := survey.Ask([]*survey.Question{question}, &answer); err != nil {
				return err
			}

			inputValues.Consul.EncryptKey = strings.TrimSpace(answer.Value)
		}

		if inputValues.Consul.EncryptKey == "" {
			fmt.Println("Generating a 16-bytes, Base64 encoded encryption key used to encrypt Consul network traffic")
			inputValues.Consul.EncryptKey, err = generateConsulEncryptKey()
			if err != nil {
				return err
			}
		}
	}

	fmt.Println("\nGetting Infrastructure configuration")

	askIfNotRequired := false
	convertBooleanToString := false
	// Get infrastructure inputs, except in the Hosts Pool case as Hosts Pool
	// doesn't have any infrastructure property
	if infrastructureType != "hostspool" {
		if inputValues.Infrastructures == nil {
			askIfNotRequired = true
			inputValues.Infrastructures = make(map[string]config.DynamicMap)
		}
		configMap := inputValues.Infrastructures[infrastructureType]
		if configMap == nil {
			askIfNotRequired = true
			configMap = make(config.DynamicMap)
		}

		if err := getResourceInputs(topology, infraNodeType, askIfNotRequired,
			convertBooleanToString, &configMap); err != nil {
			return err
		}
		inputValues.Infrastructures[infrastructureType] = configMap
	} else {

		// Hosts Pool
		if inputValues.Hosts == nil {
			askIfNotRequired = true
			inputValues.Hosts = make([]rest.HostConfig, 0)
			inputValues.Hosts, err = getHostsInputs(resourcesPath)
			if err != nil {
				return err
			}
		} else {

			// The private SSH key used to connect to each host is the Yorc private key
			// Initializing this private key and checking as well mandatory
			// labels are defined
			privateKeyPath := filepath.Join(inputValues.Yorc.DataDir, ".ssh", "yorc.pem")
			for _, host := range inputValues.Hosts {
				host.Connection.PrivateKey = privateKeyPath
				// Checking labels
				if host.Labels == nil {
					return fmt.Errorf("Missing mandatory labels %s for host %s",
						strings.Join(mandatoryHostPoolLabels, ", "), host.Name)
				}
				for _, label := range mandatoryHostPoolLabels {
					if _, found := host.Labels[label]; !found {
						return fmt.Errorf("Missing mandatory label %s for host %s",
							label, host.Name)
					}
				}
			}
		}
	}

	// Get on-demand resources definition for this infrastructure type
	onDemandResourceName := fmt.Sprintf("yorc-%s-types.yml", infrastructureType)

	data, err = getTOSCADefinition(onDemandResourceName)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, &topology); err != nil {
		return err
	}

	// First get on-demand compute nodes inputs
	// If the user didn't yet provided any property value,
	// let him the ability to specify any property.
	// If the user has already provided some property values,
	// jus asking missing required values

	fmt.Println("\nGetting Compute instances configuration")

	if inputValues.Compute == nil {
		inputValues.Compute = make(config.DynamicMap)
	}
	bootstrapExtraComputeParams.readConfFn(&inputValues)
	for _, computeParam := range bootstrapExtraComputeParams.viperNames {
		bootstrapExtraComputeParams.storeFn(&inputValues, computeParam)
	}

	askIfNotRequired = false
	if len(inputValues.Compute) == 0 {
		askIfNotRequired = true
	}
	nodeType := fmt.Sprintf("yorc.nodes.%s.Compute", infrastructureType)
	// On-demand resources boolean values need to be passed to
	// Alien4Cloud REST API as strings
	convertBooleanToString = true
	if err := getResourceInputs(topology, nodeType, askIfNotRequired,
		convertBooleanToString, &inputValues.Compute); err != nil {
		return err
	}

	// Get Compute credentials, not on Hosts Pool as Hosts credentials are provided
	// in the Hosts Pool configuration
	convertBooleanToString = false
	if infrastructureType != "hostspool" {
		fmt.Println("\nGetting Compute instances credentials")

		askIfNotRequired = false
		if inputValues.Credentials == nil {
			var creds CredentialsConfiguration
			inputValues.Credentials = &creds
		}
		if inputValues.Credentials.User == "" {
			askIfNotRequired = true
		}
		if err := getComputeCredentials(askIfNotRequired, inputValues.Credentials); err != nil {
			return err
		}
	}

	// Network connection
	if networkNodeType != "" {

		fmt.Println("\nGetting Network configuration")

		if inputValues.Address == nil {
			inputValues.Address = make(config.DynamicMap)
		}
		bootstrapExtraAddressParams.readConfFn(&inputValues)
		for _, addressParam := range bootstrapExtraAddressParams.viperNames {
			bootstrapExtraAddressParams.storeFn(&inputValues, addressParam)
		}

		askIfNotRequired = false
		if len(inputValues.Address) == 0 {
			askIfNotRequired = true
		}
		if err := getResourceInputs(topology, networkNodeType, askIfNotRequired,
			convertBooleanToString, &inputValues.Address); err != nil {
			return err
		}
	}

	// Fill in uninitialized values
	inputValues.Location.ResourcesFile = filepath.Join("resources",
		fmt.Sprintf("ondemand_resources_%s.yaml", infrastructureType))
	switch infrastructureType {
	case "openstack":
		inputValues.Location.Type = "OpenStack"
	case "google":
		inputValues.Location.Type = "Google Cloud"
	case "aws":
		inputValues.Location.Type = "AWS"
	case "hostspool":
		inputValues.Location.Type = "HostsPool"
	default:
		return fmt.Errorf("Bootstrapping on %s not supported yet", infrastructureType)
	}

	inputValues.Location.Name = inputValues.Location.Type

	if reviewInputs {
		if err := reviewAndUpdateInputs(); err != nil {
			return err
		}
	}

	// Post treatment needed on Google Cloud
	if infrastructureType == "google" {
		if err := prepareGoogleInfraInputs(); err != nil {
			return err
		}
	}

	// In insecure mode, the infrastructure secrets will not be stored in vault
	// (Hosts Pool infrastructure config doesn't have this use_vault property)
	if insecure && infrastructureType != "hostspool" {
		inputValues.Infrastructures[infrastructureType].Set("use_vault", false)
	}

	exportInputs()

	if configOnly == true {
		println("config_only option is set, exiting.")
		os.Exit(0)
	}
	return nil

}

// generateConsulEncryptKey generates a 16-bytes, Base64 encoded value of an
// encryption key used to encrypt Consul network traffic
func generateConsulEncryptKey() (string, error) {
	bKey := make([]byte, 16)
	_, err := rand.Read(bKey)
	if err != nil {
		return "", errors.Wrapf(err, "Error generating Consul encrypt key")
	}

	return base64.StdEncoding.EncodeToString(bKey), nil
}

// getCAConfiguration asks for a CA passphrase if not provided, then gets
// the content of Certificate Authority and key if provided by the user or generates
// a key and certificate authority
func getCAConfiguration(pConfig *YorcConfiguration, inputFileProvided bool, resourcesPath string) error {

	// Mandatory parameter: CA key passphrase
	if pConfig.CAPassPhrase == "" {
		answer := struct {
			Value string
		}{}

		prompt := &survey.Password{
			Message: "Certificate authority private key passphrase (required, at least 4 characters):",
		}
		question := &survey.Question{
			Name:   "value",
			Prompt: prompt,
			Validate: func(val interface{}) error {
				// Must be at least 4 characters
				str := val.(string)
				if len(str) < 4 {
					return fmt.Errorf("You entered %d characters, but should provide at least 4 characters", len(str))
				}
				return nil
			},
		}
		if err := survey.Ask([]*survey.Question{question}, &answer); err != nil {
			return err
		}

		pConfig.CAPassPhrase = strings.TrimSpace(answer.Value)
	}

	// Entering interactive mode if one of CA or Key is defined and the other
	// is not, if both are undefined they will be generated
	askForInput := !inputFileProvided ||
		((pConfig.CAKeyFile == "" || pConfig.CAPEMFile == "") &&
			pConfig.CAKeyFile != pConfig.CAPEMFile)

	// Get CA key or generate one
	if pConfig.CAKeyContent == "" {

		if pConfig.CAKeyFile == "" && askForInput {
			answer := struct {
				Value string
			}{}

			prompt := &survey.Input{
				Message: "Path to Certificate Authority private key (if none passed, a key will be generated):",
			}
			question := &survey.Question{
				Name:   "value",
				Prompt: prompt,
			}
			if err := survey.Ask([]*survey.Question{question}, &answer); err != nil {
				return err
			}

			pConfig.CAKeyFile = strings.TrimSpace(answer.Value)
		}

		if pConfig.CAKeyFile == "" {
			fmt.Println("Generating a CA key")
			pConfig.CAKeyFile = filepath.Join(resourcesPath, "ca-key.pem")
			cmdArgs := fmt.Sprintf("genrsa -aes256 -out %s -passout pass:%s 4096",
				pConfig.CAKeyFile, pConfig.CAPassPhrase)
			cmd := exec.Command("openssl", strings.Split(cmdArgs, " ")...)
			if err := cmd.Run(); err != nil {
				return errors.Wrapf(err, "Failed to generate CA key running 'openssl %s'", cmdArgs)
			}
		}

		data, err := ioutil.ReadFile(pConfig.CAKeyFile)
		if err != nil {
			return errors.Wrapf(err, "Failed to read CA key file %s", pConfig.CAKeyFile)
		}
		pConfig.CAKeyContent = string(data[:])
	}

	// Get CA or generate one
	if pConfig.CAPEMContent == "" {

		if pConfig.CAPEMFile == "" && askForInput {
			answer := struct {
				Value string
			}{}

			prompt := &survey.Input{
				Message: "Path to PEM-encoded Certificate Authority (if none passed, a Certificate Authority will be generated):",
			}
			question := &survey.Question{
				Name:   "value",
				Prompt: prompt,
			}
			if err := survey.Ask([]*survey.Question{question}, &answer); err != nil {
				return err
			}

			pConfig.CAPEMFile = strings.TrimSpace(answer.Value)
		}

		if pConfig.CAPEMFile == "" {
			pConfig.CAPEMFile = filepath.Join(resourcesPath, "ca.pem")
			fmt.Printf("\nGenerating a PEM-encoded Certificate Authority %s\n", pConfig.CAPEMFile)
			fmt.Println("This Certificate Authority should be imported in your Web brower as a trusted Certificate Authority")
			cmdArgs := fmt.Sprintf("req -new -x509 -days 3650 -key %s -sha256 -passin pass:%s -subj /CN=yorc/O=ystia/C=US -out %s",
				pConfig.CAKeyFile, pConfig.CAPassPhrase, pConfig.CAPEMFile)
			cmd := exec.Command("openssl", strings.Split(cmdArgs, " ")...)
			if err := cmd.Run(); err != nil {
				return errors.Wrapf(err,
					"Failed to generate a PEM-encoded Certificate Authority running 'openssl %s'",
					cmdArgs)
			}
		}

		data, err := ioutil.ReadFile(pConfig.CAPEMFile)
		if err != nil {
			return errors.Wrapf(err, "Failed to read CA file %s", pConfig.CAPEMFile)
		}
		pConfig.CAPEMContent = string(data[:])
	}

	return nil
}

// reviewAndUpdateInputs allows the user to review and change deployment inputs
// if an editor could be found, else the review is skipped
func reviewAndUpdateInputs() error {

	bSlice, err := yaml.Marshal(inputValues)
	if err != nil {
		return err
	}

	// Avoid failures on missing editor, following survey package logic
	// checking environment variables, or if we can find a usual editor
	var editor string
	editorFound := (os.Getenv("VISUAL") != "") || (os.Getenv("EDITOR") != "")
	if !editorFound {
		editor, err = exec.LookPath("vim")
		if err != nil {
			editor, err = exec.LookPath("vi")
		}

		editorFound = (err == nil)
	}

	if !editorFound {
		fmt.Printf("Skipping review and update as no editor was found (should define EDITOR environment variable)")
		return nil
	}

	prompt := &survey.Editor{
		Message:       "Review and update inputs if needed",
		Default:       string(bSlice[:]),
		HideDefault:   true,
		AppendDefault: true,
	}

	if editor != "" {
		prompt.Editor = editor
	}

	var reply string
	err = survey.AskOne(prompt, &reply, nil)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal([]byte(reply), &inputValues)

	return nil
}

func exportInputs() {
	if inputsPath == "" {

		bSlice, err := yaml.Marshal(inputValues)

		if err != nil {
			log.Fatal("cannot marshal inputValues ", err)
		}

		inputsPathOut := deploymentID + ".yaml"

		count := 1
		for {
			if _, err := os.Stat(inputsPathOut); os.IsNotExist(err) {
				break
			}
			inputsPathOut = deploymentID + "_" + strconv.Itoa(count) + ".yaml"
			count++
		}

		if _, err := os.Stat("inputsaves/"); os.IsNotExist(err) {
			err = os.Mkdir("inputsaves/", 0700)
			if err != nil {
				log.Fatal("cannot create inputsave directory ", err)
			}
		}

		println("Exporting set configuration to inputsaves/" + inputsPathOut)

		file, err := os.OpenFile("inputsaves/"+inputsPathOut, os.O_CREATE|os.O_WRONLY, 0600)

		if err != nil {
			log.Fatal("Cannot create file to save the configuration ", err)
		}

		_, err = fmt.Fprintf(file, string(bSlice[:]))

		if err != nil {
			log.Fatal("Cannot write configuration to file", err)
		}

		file.Close()
	}
}

// getComputeCredentials asks for the user used to connect to an on-demand compute node
func getComputeCredentials(askIfNotRequired bool, creds *CredentialsConfiguration) error {

	answer := struct {
		Value string
	}{}

	if creds.User == "" {
		prompt := &survey.Input{
			Message: "User used to connect to Compute instances (required):",
		}
		question := &survey.Question{
			Name:     "value",
			Prompt:   prompt,
			Validate: survey.Required,
		}
		if err := survey.Ask([]*survey.Question{question}, &answer); err != nil {
			return err
		}
		creds.User = answer.Value
	}

	creds.Keys = make(map[string]string)
	creds.Keys["0"] = inputValues.Yorc.PrivateKeyFile
	/*

		if askIfNotRequired {
			prompt := &survey.Input{
				Message: "Private key:",
			}
			question := &survey.Question{
				Name:   "value",
				Prompt: prompt,
			}
			if err := survey.Ask([]*survey.Question{question}, &answer); err != nil {
				return err
			}
			creds.Keys = make(map[string]string)
			creds.Keys["0"] = answer.Value

		}
	*/

	return nil
}

// getHostsInputs asks for input parameters of Hosts to add in pool
func getHostsInputs(resourcesPath string) ([]rest.HostConfig, error) {

	fmt.Println("\nDefining hosts to add in the Pool, SSH connections to these hosts will be done using Yorc private key")

	answer := struct {
		Value string
	}{}

	var result []rest.HostConfig
	finished := false
	for !finished {
		prompt := &survey.Input{
			Message: "Name identifying the new host to add in pool (required):"}

		question := &survey.Question{
			Name:     "value",
			Prompt:   prompt,
			Validate: survey.Required,
		}
		if err := survey.Ask([]*survey.Question{question}, &answer); err != nil {
			return nil, err
		}
		var hostConfig rest.HostConfig
		hostConfig.Name = answer.Value

		prompt = &survey.Input{
			Message: fmt.Sprintf("Hostname or ip address used to connect to the host (default: %s):",
				hostConfig.Name)}

		question = &survey.Question{
			Name:   "value",
			Prompt: prompt,
		}
		if err := survey.Ask([]*survey.Question{question}, &answer); err != nil {
			return nil, err
		}

		if answer.Value != "" {
			hostConfig.Connection.Host = answer.Value
		} else {
			hostConfig.Connection.Host = hostConfig.Name
		}

		prompt = &survey.Input{
			Message: "User used to connect to the host (default: root):"}

		question = &survey.Question{
			Name:   "value",
			Prompt: prompt,
		}
		if err := survey.Ask([]*survey.Question{question}, &answer); err != nil {
			return nil, err
		}

		if answer.Value != "" {
			hostConfig.Connection.User = answer.Value
		} else {
			hostConfig.Connection.User = "root"
		}

		prompt = &survey.Input{
			Message: "SSH port used to connect to the host (default: 22):"}

		question = &survey.Question{
			Name:   "value",
			Prompt: prompt,
			Validate: func(val interface{}) error {
				// if the input matches the expectation
				str := val.(string)
				if str != "" {
					if _, err := strconv.ParseUint(answer.Value, 10, 64); err != nil {
						return fmt.Errorf("You entered %s, not an unsigned integer", str)
					}
				}
				return nil
			},
		}
		if err := survey.Ask([]*survey.Question{question}, &answer); err != nil {
			return nil, err
		}

		if answer.Value != "" {
			port, _ := strconv.ParseUint(answer.Value, 10, 64)
			hostConfig.Connection.Port = port
		} else {
			hostConfig.Connection.Port = 22
		}

		//asking public/private address first
		fmt.Println("Defining key/value labels for this host")
		fmt.Printf("Inputs for labels %s are mandatory\n",
			strings.Join(mandatoryHostPoolLabels, ", "))
		hostConfig.Labels = make(map[string]string)

		for _, label := range mandatoryHostPoolLabels {
			prompt = &survey.Input{
				Message: label + " value:"}

			question = &survey.Question{
				Name:     "value",
				Prompt:   prompt,
				Validate: survey.Required,
			}
			if err := survey.Ask([]*survey.Question{question}, &answer); err != nil {
				return nil, err
			}

			hostConfig.Labels[label] = answer.Value
		}

		//asking for additional key values labels
		for {

			// asking if a new label must be defined
			promptEnd := &survey.Select{
				Message: "Add another key/value label for this host ?:",
				Options: []string{"yes", "no"},
				Default: "no",
			}
			var reply string
			survey.AskOne(promptEnd, &reply, nil)
			if reply == "no" {
				break
			}

			prompt := &survey.Input{
				Message: "Label key (required):"}

			question := &survey.Question{
				Name:     "value",
				Prompt:   prompt,
				Validate: survey.Required,
			}
			if err := survey.Ask([]*survey.Question{question}, &answer); err != nil {
				return nil, err
			}
			labelKey := answer.Value

			prompt = &survey.Input{
				Message: "Label value:"}

			question = &survey.Question{
				Name:   "value",
				Prompt: prompt,
			}
			if err := survey.Ask([]*survey.Question{question}, &answer); err != nil {
				return nil, err
			}

			hostConfig.Labels[labelKey] = answer.Value
		}

		// The private SSH key used to connect to the host is the Yorc private key
		hostConfig.Connection.PrivateKey = filepath.Join(inputValues.Yorc.DataDir, ".ssh", "yorc.pem")

		// Forge component expect the label os.type to be linux,
		// if no such label is defined, adding it
		if _, ok := hostConfig.Labels["os.type"]; !ok {
			hostConfig.Labels["os.type"] = "linux"
		}

		result = append(result, hostConfig)

		fmt.Println("Host added to the Pool.")
		// Ask if another host has to be defined
		promptEnd := &survey.Select{
			Message: "Add another host in pool:",
			Options: []string{"yes", "no"},
			Default: "no",
		}
		var reply string
		survey.AskOne(promptEnd, &reply, nil)
		if reply == "no" {
			finished = true
		}
	}
	fmt.Printf("\n%d hosts added to the pool\n", len(result))
	return result, nil

}

// getResourceInputs asks for input parameters of an infrastructure or on-demand
// resource
func getResourceInputs(topology tosca.Topology, resourceName string,
	askIfNotRequired bool,
	convertBooleanToString bool,
	resultMap *config.DynamicMap) error {

	nodeType, found := topology.NodeTypes[resourceName]
	if !found {
		return fmt.Errorf("Unknown node type %s", resourceName)
	}

	err := getPropertiesInput(topology, nodeType.Properties, askIfNotRequired,
		convertBooleanToString, "", resultMap)

	return err

}

func getPropertiesInput(topology tosca.Topology, properties map[string]tosca.PropertyDefinition,
	askIfNotRequired bool, convertBooleanToString bool, msgPrefix string, resultMap *config.DynamicMap) error {

	for propName, definition := range properties {

		// Check if a value is already provided before asking for user input
		// ot if the value is not required
		required := definition.Required != nil && *definition.Required
		if resultMap.IsSet(propName) ||
			(!askIfNotRequired && !required) {
			continue
		}

		description := getFormattedDescription(definition.Description)
		if msgPrefix != "" {
			description = fmt.Sprintf("%s - %s", msgPrefix, description)
		}

		isList := (definition.Type == "list")
		if definition.Type == "boolean" {
			defaultValue := "false"
			if definition.Default != nil {
				defaultValue = definition.Default.GetLiteral()
			}
			prompt := &survey.Select{
				Message: fmt.Sprintf("%s:", description),
				Options: []string{"true", "false"},
				Default: defaultValue,
			}
			var answer string
			survey.AskOne(prompt, &answer, nil)
			if convertBooleanToString {
				resultMap.Set(propName, answer)
			} else {
				value := false
				if answer == "true" {
					value = true
				}
				resultMap.Set(propName, value)
			}

		} else if isDatatype(topology, definition.Type) {
			propValueMap := make(config.DynamicMap)
			if !required && askIfNotRequired {
				prompt := &survey.Select{
					Message: fmt.Sprintf("Do you want to define property %q", description),
					Options: []string{"yes", "no"},
					Default: "no",
				}
				var answer string
				survey.AskOne(prompt, &answer, nil)

				if answer == "no" {
					continue
				}

				if err := getPropertiesInput(topology, topology.DataTypes[definition.Type].Properties,
					askIfNotRequired, convertBooleanToString, description, &propValueMap); err != nil {
					return err
				}
				resultMap.Set(propName, propValueMap)
			}
		} else {

			answer := struct {
				Value string
			}{}

			var additionalMsg string
			if required {
				additionalMsg = "(required"
			}
			if isList {
				if additionalMsg == "" {
					additionalMsg = " (comma-separated list"
				} else {
					additionalMsg += ", comma-separated list"
				}

			}
			if definition.Default != nil {
				if additionalMsg == "" {
					additionalMsg = fmt.Sprintf(" (default: %v", definition.Default.GetLiteral())
				} else {
					additionalMsg = fmt.Sprintf(", default: %v", definition.Default.GetLiteral())
				}
			}
			if additionalMsg != "" {
				additionalMsg += ")"
			}

			var prompt survey.Prompt
			loweredProp := strings.ToLower(propName)
			if strings.Contains(loweredProp, "password") ||
				strings.Contains(loweredProp, "secret") {
				prompt = &survey.Password{
					Message: fmt.Sprintf("%s%s:",
						description,
						additionalMsg)}
			} else {
				prompt = &survey.Input{
					Message: fmt.Sprintf("%s%s:",
						description,
						additionalMsg)}
			}
			question := &survey.Question{
				Name:   "value",
				Prompt: prompt,
			}
			if required {
				question.Validate = survey.Required
			}
			if err := survey.Ask([]*survey.Question{question}, &answer); err != nil {
				return err
			}

			if answer.Value != "" {
				if !isList {
					resultMap.Set(propName, answer.Value)
				} else {
					value := strings.Split(answer.Value, ",")
					for i, val := range value {
						value[i] = strings.TrimSpace(val)
					}
					resultMap.Set(propName, value)
				}
			} else if definition.Default != nil {
				resultMap.Set(propName, definition.Default.GetLiteral())
			}
		}
	}

	return nil
}

func isDatatype(topology tosca.Topology, nodeType string) bool {
	_, ok := topology.DataTypes[nodeType]
	return ok
}

// prepareGoogleInfraInputs updates inputs for a Google Infrastructure if needed
// to use the content of service account key file instead of the path to this file
func prepareGoogleInfraInputs() error {

	if !inputValues.Infrastructures["google"].IsSet("application_credentials") {
		return nil
	}

	credsPath := inputValues.Infrastructures["google"].GetString("application_credentials")
	if credsPath == "" {
		return nil
	}
	data, err := ioutil.ReadFile(credsPath)
	if err != nil {
		return err
	}
	// Using file content instead of file path
	inputValues.Infrastructures["google"].Set("credentials", string(data[:]))
	inputValues.Infrastructures["google"].Set("application_credentials", "")
	return nil
}

// getFormattedDescription reformats the description found in yaml node types
// definitions
func getFormattedDescription(description string) string {
	result := strings.TrimSpace(description)
	result = strings.TrimSuffix(result, ".")
	return result
}

// getInputValues initializes topology values from an input file
func getInputValues(inputFilePath string) (TopologyValues, error) {

	var values TopologyValues

	if inputFilePath != "" {

		// A file was provided, merge its values definitions
		bootstrapViper.SetConfigFile(inputFilePath)

		if err := bootstrapViper.MergeInConfig(); err != nil {
			_, ok := err.(viper.ConfigFileNotFoundError)
			if inputFilePath != "" && !ok {
				fmt.Println("Can't use config file:", err)
			}
		}

		var infratype = bootstrapViper.GetString("InfrastructureType")
		infrastructureType = infratype
	}

	err := bootstrapViper.Unmarshal(&values)
	return values, err
}

// getTerraformPluginsDownloadURLs return URLs where to download Terraform plugins
// sed by the orchestrator
func getTerraformPluginsDownloadURLs() []string {
	urlFormat := "https://releases.hashicorp.com/terraform-provider-%s/%s/terraform-provider-%s_%s_linux_amd64.zip"

	pluginVersionMap := map[string]string{
		"null":      "1.0.0",
		"consul":    commands.TfConsulPluginVersion,
		"google":    commands.TfGooglePluginVersion,
		"openstack": commands.TfOpenStackPluginVersion,
		"aws":       commands.TfAWSPluginVersion,
	}

	var pluginURLs []string
	for plugin, version := range pluginVersionMap {
		pluginURLs = append(pluginURLs, fmt.Sprintf(urlFormat, plugin, version, plugin, version))
	}

	return pluginURLs
}

// getYorcDownloadURL returns Yorc download URL,
// either a snapshot download or a release/milestone URL
func getYorcDownloadURL() string {
	var downloadURL string
	if strings.Contains(yorcVersion, "SNAPSHOT") {
		downloadURL = fmt.Sprintf(
			"https://ystia.jfrog.io/ystia/binaries/ystia/yorc/dist/develop/yorc-%s.tgz",
			yorcVersion)
	} else {
		downloadURL = fmt.Sprintf(
			"https://dl.bintray.com/ystia/yorc-engine/%s/yorc-%s.tgz",
			yorcVersion, yorcVersion)
	}
	return downloadURL
}

// getYorcPluginDownloadURL returns Yorc plugin download URL,
// either a snapshot download or a release/milestone URL
func getYorcPluginDownloadURL() string {
	var downloadURL string
	if strings.Contains(yorcVersion, "SNAPSHOT") {
		downloadURL = fmt.Sprintf(
			"https://ystia.jfrog.io/ystia/binaries/ystia/yorc-a4c-plugin/dist/develop/alien4cloud-yorc-plugin-%s.zip",
			yorcVersion)
	} else {
		downloadURL = fmt.Sprintf(
			"https://dl.bintray.com/ystia/yorc-a4c-plugin/%s/alien4cloud-yorc-plugin-%s.zip",
			yorcVersion, yorcVersion)
	}
	return downloadURL
}
