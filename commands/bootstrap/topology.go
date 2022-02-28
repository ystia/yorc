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
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/ziputil"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/rest"
)

// AnsibleConfiguration provides Ansible user-defined settings
type AnsibleConfiguration struct {
	Version               string
	PackageRepositoryURL  string              `yaml:"extra_package_repository_url" mapstructure:"extra_package_repository_url"`
	UseOpenSSH            bool                `yaml:"use_openssh,omitempty" mapstructure:"use_openssh" json:"use_open_ssh,omitempty"`
	Inventory             map[string][]string `yaml:"inventory,omitempty" mapstructure:"inventory"`
	HostOperationsAllowed bool                `yaml:"host_operations_allowed" mapstructure:"host_operations_allowed"`
}

const (
	regexpYamlTrue          = "(?m):\\s+(y|Y|yes|Yes|YES|true|True|TRUE|on|On|ON)$"
	regexpYamlFalse         = "(?m):\\s+(n|N|no|No|NO|false|False|FALSE|off|Off|OFF)$"
	regexpMarshalingPattern = "(?m)^(\\s+)%s:"
)

// YorcConfiguration provides Yorc user-defined settings
type YorcConfiguration struct {
	DownloadURL       string `yaml:"download_url" mapstructure:"download_url"`
	Port              int
	Protocol          string
	PrivateKeyContent string `yaml:"private_key_content" mapstructure:"private_key_content"`
	PrivateKeyFile    string `yaml:"private_key_file" mapstructure:"private_key_file"`
	CAPEMContent      string `yaml:"ca_pem" mapstructure:"ca_pem"`
	CAPEMFile         string `yaml:"ca_pem_file" mapstructure:"ca_pem_file"`
	CAKeyContent      string `yaml:"ca_key" mapstructure:"ca_key"`
	CAKeyFile         string `yaml:"ca_key_file" mapstructure:"ca_key_file"`
	CAPassPhrase      string `yaml:"ca_passphrase" mapstructure:"ca_passphrase"`
	DataDir           string `yaml:"data_dir" mapstructure:"data_dir"`
	WorkersNumber     int    `yaml:"workers_number" mapstructure:"workers_number"`
	ResourcesPrefix   string `yaml:"resources_prefix" mapstructure:"resources_prefix"`
}

// YorcPluginConfiguration provides Yorc plugin user-defined settings
type YorcPluginConfiguration struct {
	DownloadURL string `yaml:"download_url" mapstructure:"download_url"`
}

// Alien4CloudConfiguration provides Alien4Cloud user-defined settings
type Alien4CloudConfiguration struct {
	DownloadURL string `yaml:"download_url" mapstructure:"download_url"`
	Port        int
	Protocol    string
	User        string
	Password    string
	ExtraEnv    string `yaml:"extra_env" mapstructure:"extra_env"`
}

// ConsulConfiguration provides Consul user-defined settings
type ConsulConfiguration struct {
	DownloadURL         string `yaml:"download_url" mapstructure:"download_url"`
	Port                int
	TLSEnabled          bool   `yaml:"tls_enabled" mapstructure:"tls_enabled"`
	TLSForChecksEnabled bool   `yaml:"tls_for_checks_enabled" mapstructure:"tls_for_checks_enabled"`
	EncryptKey          string `yaml:"encrypt_key" mapstructure:"encrypt_key"`
}

// TerraformConfiguration provides Terraform settings
type TerraformConfiguration struct {
	DownloadURL string   `yaml:"download_url" mapstructure:"download_url"`
	PluginURLs  []string `yaml:"plugins_download_urls" mapstructure:"plugins_download_urls"`
}

// JdkConfiguration configuration provides Java settings
type JdkConfiguration struct {
	DownloadURL string `yaml:"download_url" mapstructure:"download_url"`
	Version     string
}

// LocationConfiguration provides the configuration of the location selected to
// install the orchestratror
type LocationConfiguration struct {
	Type          string
	Name          string
	ResourcesFile string            // Path to file describing on-demand resources for this location
	Properties    config.DynamicMap // Referenced in go template files used to build the topology
}

// CredentialsConfiguration provides a user, private key, and token type (required property)
type CredentialsConfiguration struct {
	User      string
	Keys      map[string]string
	TokenType string `yaml:"token_type" mapstructure:"token_type"`
}

// VaultConfiguration provides Vault user-defined settings
type VaultConfiguration struct {
	DownloadURL string `yaml:"download_url" mapstructure:"download_url"`
	Port        int
}

// TopologyValues provides inputs to the topology templates
type TopologyValues struct {
	Ansible     AnsibleConfiguration
	Alien4cloud Alien4CloudConfiguration
	YorcPlugin  YorcPluginConfiguration `mapstructure:"yorc_plugin"`
	Consul      ConsulConfiguration
	Terraform   TerraformConfiguration
	Yorc        YorcConfiguration
	Locations   []locations.LocationConfiguration
	Compute     config.DynamicMap
	Credentials *CredentialsConfiguration
	Address     config.DynamicMap
	Jdk         JdkConfiguration
	Location    LocationConfiguration
	Hosts       []rest.HostConfig
	Vault       VaultConfiguration
	Insecure    bool
}

// formatAsYAML is a function used in templates to output the yaml representation
// of a variable
func formatAsYAML(data interface{}, indentations int) (string, error) {

	result := ""
	bSlice, err := yaml.Marshal(data)
	if err == nil {
		result = indent(string(bSlice), indentations)
	}
	return result, err
}

// formatAsJSON is a function used in templates to output the json representation
// of a variable
func formatAsJSON(data interface{}) (string, error) {

	result := ""
	bSlice, err := json.Marshal(data)
	if err == nil {
		result = string(bSlice)
	}
	return result, err
}

// formatOnDemandResourceCredsAsYAML is a function used in
// on-demand resources templates to output the yaml representation
// of credentials
func formatOnDemandResourceCredsAsYAML(creds *CredentialsConfiguration, indentations int) (string, error) {

	var onDemandCreds CredentialsConfiguration
	onDemandCreds.User = creds.User
	if len(creds.Keys) > 0 {
		// Credentials keys contains the path to a private key used by the
		// local Yorc Server. Defining here the path to private key available
		// on the remote bootsrtrapped Yorc Server
		onDemandCreds.Keys = make(map[string]string)
		onDemandCreds.Keys["0"] = filepath.Join(inputValues.Yorc.DataDir, ".ssh", "yorc.pem")

	}
	result := ""
	bSlice, err := yaml.Marshal(onDemandCreds)
	if err == nil {
		result = indent(string(bSlice), indentations)
	}
	return result, err
}

// indent adds a new line and indents the string in argument
func indent(data string, indentations int) string {
	result := "\n" + data
	if indentations > 0 {
		indentString := strings.Repeat(" ", indentations)
		result = strings.Replace(result, "\n", "\n"+indentString, -1)
		result = strings.TrimRight(result, " \n")
	}

	return result
}

// getAlien4CloudVersion extracts an Alien4cloud version form its download URL
func getAlien4CloudVersion(url string) (string, error) {
	match := regexp.MustCompile(`-dist-([0-9a-zA-Z.-]+)-dist.tar.gz`).FindStringSubmatch(url)
	version := ""
	if match != nil {
		version = match[1]
	} else {
		fmt.Printf("Failed to retrieve Alien4Cloud version from URL %s, we will use a default value.\n", url)
		return getAlien4CloudVersionFromTOSCATypes(), nil
	}

	return version, nil
}

// getFile returns the file part of a URL
func getFile(url string) string {
	_, file := filepath.Split(url)
	return file
}

// getFile returns the repository part of a URL of the form repository/file
func getRepositoryURL(url string) string {
	dir, _ := filepath.Split(url)
	return dir
}

// createTopology creates a topology from template files under topologyDir,
// by executing these template files against input values
func createTopology(topologyDir string) error {

	// First unzip tosca types available in the resources directory
	toscaTypesZipFilePath := filepath.Join(topologyDir, "tosca_types.zip")
	if _, err := ziputil.Unzip(toscaTypesZipFilePath, topologyDir); err != nil {
		return err
	}

	var topologyTemplateFileNames []string
	var resourcesTemplateFileNames []string
	resourcesTemplatePrefix := "ondemand_resources"
	topologyTemplatesPrefix := "topology"
	infrastructureTemplateSuffix := infrastructureType + ".tmpl"
	topologyTemplateFile := fmt.Sprintf("%s_%s.tmpl", topologyTemplatesPrefix, deploymentType)
	err := filepath.Walk(topologyDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			_, file := filepath.Split(path)
			if file == topologyTemplateFile ||
				strings.HasSuffix(file, infrastructureTemplateSuffix) {

				if strings.HasPrefix(file, resourcesTemplatePrefix) {
					resourcesTemplateFileNames = append(resourcesTemplateFileNames, path)
				} else if strings.HasPrefix(file, topologyTemplatesPrefix) {
					topologyTemplateFileNames = append(topologyTemplateFileNames, path)
				}
			}

			return err
		})

	if err != nil {
		return errors.Wrapf(err, "Failed to browse topology files under %s", topologyDir)
	}

	if len(topologyTemplateFileNames) == 0 {
		// No topology template found, the topology is ready
		return nil
	}

	if len(resourcesTemplateFileNames) == 0 {
		// Need an on-demand resource template file name
		return fmt.Errorf("Found no on-demand resources template in %s", topologyDir)
	}

	onDemandResourcesFilePath := filepath.Join(topologyDir, inputValues.Location.ResourcesFile)
	err = createFileFromTemplates(resourcesTemplateFileNames,
		filepath.Base(resourcesTemplateFileNames[0]),
		onDemandResourcesFilePath,
		inputValues)
	if err != nil {
		return errors.Wrap(err, "Failed to create on-demand resources file from templates")
	}

	err = postProcessOnDemandResourceFile(onDemandResourcesFilePath)
	if err != nil {
		return errors.Wrap(err, "Failed to post-process on-demand resources file")
	}

	topologyFilePath := filepath.Join(topologyDir, "topology.yaml")
	err = createFileFromTemplates(topologyTemplateFileNames, topologyTemplateFile,
		topologyFilePath,
		inputValues)
	if err != nil {
		return errors.Wrap(err, "Failed to create topology file from templates")
	}

	err = postProcessYamlMarshaling(topologyFilePath)

	// If a Hosts Pool infrastructure was specified, an additional resources file
	// containing the hosts pool description has to be created
	if infrastructureType == "hostspool" {

		hostsPool := rest.HostsPoolRequest{
			Hosts: inputValues.Hosts,
		}

		bSlice, err := yaml.Marshal(hostsPool)
		if err != nil {
			return err
		}

		cfgFileName := filepath.Join(topologyDir, "resources", "hostspool.yaml")
		err = ioutil.WriteFile(cfgFileName, bSlice, 0700)
		if err != nil {
			return err
		}

	}

	return err
}

// postProcessOnDemandResourceFile will replace booleans by strings in the on-demand
// resources file as Alien4Cloud REST API fails to manage such boolean values
// when processing the request of configuring an on-demand resource boolean attribute value
func postProcessOnDemandResourceFile(filepath string) error {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}

	tmpContent := regexp.MustCompile(regexpYamlTrue).ReplaceAllString(string(content), `: "true"`)
	newContent := regexp.MustCompile(regexpYamlFalse).ReplaceAllString(tmpContent, `: "false"`)

	err = ioutil.WriteFile(filepath, []byte(newContent), 0)
	if err != nil {
		return err
	}

	err = postProcessYamlMarshaling(filepath)
	return err
}

// postProcessYamlMarshaling fixes the yaml marshaling of map keys which
// are converted to lower case, while some yaml attributes are not exepcted to
// be lower cased
func postProcessYamlMarshaling(filepath string) error {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}

	newContent := string(content)
	attrNames := []string{"imageName", "flavorName"}
	for _, attr := range attrNames {
		regexpVal := fmt.Sprintf(regexpMarshalingPattern, strings.ToLower(attr))
		newContent = regexp.MustCompile(regexpVal).ReplaceAllString(newContent,
			fmt.Sprintf("${1}%s:", attr))
	}

	err = ioutil.WriteFile(filepath, []byte(newContent), 0)
	if err != nil {
		return err
	}
	return err
}

// Creates a file from templates, substituting annotations with data
func createFileFromTemplates(templateFileNames []string, templateName, resultFilePath string, values TopologyValues) error {

	// Mapping from names to functions of functions referenced in templates
	fmap := template.FuncMap{
		"formatAsYAML":                             formatAsYAML,
		"formatAsJSON":                             formatAsJSON,
		"formatOnDemandResourceCredsAsYAML":        formatOnDemandResourceCredsAsYAML,
		"indent":                                   indent,
		"getFile":                                  getFile,
		"getRepositoryURL":                         getRepositoryURL,
		"getAlien4CloudVersion":                    getAlien4CloudVersion,
		"getAlien4CloudVersionFromTOSCATypes":      getAlien4CloudVersionFromTOSCATypes,
		"getAlien4CloudForgeVersionFromTOSCATypes": getAlien4CloudForgeVersionFromTOSCATypes,
		"getForgeVersionFromTOSCATypes":            getForgeVersionFromTOSCATypes,
	}

	parsedTemplate, err := template.New(templateName).Funcs(fmap).ParseFiles(templateFileNames...)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(resultFilePath), 0700); err != nil {
		return err
	}

	resultFile, err := os.OpenFile(resultFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0700)
	if err != nil {
		return err
	}
	defer resultFile.Close()

	writer := bufio.NewWriter(resultFile)

	err = parsedTemplate.Execute(writer, values)
	if err != nil {
		return err
	}
	err = writer.Flush()

	if err != nil {
		return err
	}
	// Remove empty lines than may appear in conditional template code parsed
	// as it would cause issues in Alien4Cloud Workflows parsing
	err = removeWorkflowsEmptyLines(resultFilePath)
	return err
}

func removeWorkflowsEmptyLines(filename string) error {
	re := regexp.MustCompile("(?m)^\\s*$[\r\n]*")
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	workflowStartIndex := strings.Index(string(data), "workflows:")
	if workflowStartIndex == -1 {
		return nil
	}
	workflowSection := strings.Trim(re.ReplaceAllString(string(data[workflowStartIndex:]), ""), "\r\n")
	err = ioutil.WriteFile(filename, []byte(fmt.Sprintf("%s%s", string(data[:workflowStartIndex]), workflowSection)), 0600)
	return err

}
