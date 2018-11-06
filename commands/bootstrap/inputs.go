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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/ystia/yorc/config"

	"github.com/ystia/yorc/commands"
)

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
	}

	consulDefaultInputs = map[string]defaultInputType{
		"consul.download_url": defaultInputType{
			description: "Consul download URL",
			value: fmt.Sprintf("https://releases.hashicorp.com/consul/%s/consul_%s_linux_amd64.zip",
				consulVersion, consulVersion),
		},
		"consul.port": defaultInputType{
			description: "Consul port",
			value:       8500,
		},
	}

	terraformDefaultInputs = map[string]defaultInputType{
		"terraform.version": defaultInputType{
			description: "Terraform version",
			value:       terraformVersion,
		},
		"terraform.plugins_download_urls": defaultInputType{
			description: "Terraform plugins dowload URLs",
			value:       getTerraformPluginsDownloadURLs(),
		},
	}

	jdkDefaultInputs = map[string]defaultInputType{
		"jdk.download_url": defaultInputType{
			description: "Java Development Kit download URL",
			value:       "https://edelivery.oracle.com/otn-pub/java/jdk/8u131-b11/d54c1d3a095b4ff2b6607d096fa80163/jdk-8u131-linux-x64.tar.gz",
		},
		"jdk.version": defaultInputType{
			description: "Java Development Kit version",
			value:       "1.8.0-131-b11",
		},
	}
)

// Mandatory inputs, which when missing, will have to be provided interactively
/*
var {
	infrastructureInputs = map[]
}
*/
// setDefaultInputValues sets environment variables and command line bootstrap options
// with default values
func setDefaultInputValues() error {

	defaultInputs := []map[string]defaultInputType{
		ansibleDefaultInputs,
		yorcDefaultInputs,
		yorcPluginDefaultInputs,
		alien4CloudDefaultInputs,
		consulDefaultInputs,
		terraformDefaultInputs,
		jdkDefaultInputs,
	}

	for _, defaultInput := range defaultInputs {

		for key, input := range defaultInput {
			flatKey := strings.Replace(key, ".", "_", 1)
			switch input.value.(type) {
			case string:
				bootstrapCmd.PersistentFlags().String(flatKey, input.value.(string), input.description)
			case int:
				bootstrapCmd.PersistentFlags().Int(flatKey, input.value.(int), input.description)
			case []string:
				bootstrapCmd.PersistentFlags().StringArray(flatKey, input.value.([]string), input.description)
			default:
				return fmt.Errorf("Unexpected default value type for %s value %v", key, input.value)

			}

			bootstrapViper.BindPFlag(key, bootstrapCmd.PersistentFlags().Lookup(flatKey))
			bootstrapViper.BindEnv(key,
				strings.ToUpper(fmt.Sprintf("%s_%s",
					environmentVariablePrefix, flatKey)))
			bootstrapViper.SetDefault(key, input.value)
		}
	}

	return nil
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
	}

	if err := bootstrapViper.Unmarshal(&values); err != nil {
		return values, err
	}

	// Fill in uninitialized values
	values.Location.ResourcesFile = filepath.Join("resources",
		fmt.Sprintf("ondemand_resources_%s.yaml", infrastructureType))
	switch infrastructureType {
	case "openstack":
		values.Location.Type = "OpenStack"
	case "google":
		values.Location.Type = "Google Cloud"
	case "aws":
		values.Location.Type = "AWS"
	case "hostspool":
		values.Location.Type = "HostsPool"
	default:
		return values, fmt.Errorf("Bootstrapping a location on %s not supported yet", infrastructureType)
	}

	values.Location.Name = values.Location.Type
	return values, nil
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
			"https://dl.bintray.com/ystia/yorc-engine/snapshots/develop/yorc-%s.tgz",
			yorcVersion)
	} else {
		downloadURL = fmt.Sprintf(
			"https://github.com/ystia/yorc/releases/download/v%s/yorc-%s.tgz",
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
			"https://dl.bintray.com/ystia/yorc-a4c-plugin/snapshots/develop/alien4cloud-yorc-plugin-%s.zip",
			yorcVersion)
	} else {
		downloadURL = fmt.Sprintf(
			"https://github.com/ystia/yorc-a4c-plugin/releases/download/v%s/alien4cloud-yorc-plugin-%s.zip",
			yorcVersion, yorcVersion)
	}
	return downloadURL
}
