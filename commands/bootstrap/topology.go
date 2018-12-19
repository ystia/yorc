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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/helper/ziputil"
	"github.com/ystia/yorc/rest"

	"gopkg.in/yaml.v2"
)

// AnsibleConfiguration provides Ansible user-defined settings
type AnsibleConfiguration struct {
	Version              string
	PackageRepositoryURL string `yaml:"extra_package_repository_url" mapstructure:"extra_package_repository_url"`
}

// YorcConfiguration provides Yorc user-defined settings
type YorcConfiguration struct {
	DownloadURL       string `yaml:"download_url" mapstructure:"download_url"`
	Port              int
	PrivateKeyContent string `yaml:"private_key_content" mapstructure:"private_key_content"`
	PrivateKeyFile    string `yaml:"private_key_file" mapstructure:"private_key_file"`
	DataDir           string `yaml:"data_dir" mapstructure:"data_dir"`
	WorkersNumber     int    `yaml:"workers_number" mapstructure:"workers_number"`
}

// YorcPluginConfiguration provides Yorc plugin user-defined settings
type YorcPluginConfiguration struct {
	DownloadURL string `yaml:"download_url" mapstructure:"download_url"`
}

// Alien4CloudConfiguration provides Alien4Cloud user-defined settings
type Alien4CloudConfiguration struct {
	DownloadURL string `yaml:"download_url" mapstructure:"download_url"`
	Port        int
	User        string
	Password    string
}

// ConsulConfiguration provides Consul user-defined settings
type ConsulConfiguration struct {
	DownloadURL string `yaml:"download_url" mapstructure:"download_url"`
	Port        int
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

// LocationConfiguration provides an Alien4Cloud plugin location configuration
type LocationConfiguration struct {
	Type          string
	Name          string
	ResourcesFile string
}

// CredentialsConfiguration provides a user and private key
type CredentialsConfiguration struct {
	User string
	Keys map[string]string
}

// TopologyValues provides inputs to the topology templates
type TopologyValues struct {
	Ansible         AnsibleConfiguration
	Alien4cloud     Alien4CloudConfiguration
	YorcPlugin      YorcPluginConfiguration `mapstructure:"yorc_plugin"`
	Consul          ConsulConfiguration
	Terraform       TerraformConfiguration
	Yorc            YorcConfiguration
	Infrastructures map[string]config.DynamicMap
	Compute         config.DynamicMap
	Credentials     *CredentialsConfiguration
	Address         config.DynamicMap
	Jdk             JdkConfiguration
	Location        LocationConfiguration
	Hosts           []rest.HostConfig
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

// formatAsformatOnDemandResourceCredsAsYAMLYAML is a function used in
// on-demand resources templates to output the yaml representation
// of crednetials
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

	err = createFileFromTemplates(resourcesTemplateFileNames,
		filepath.Base(resourcesTemplateFileNames[0]),
		filepath.Join(topologyDir, inputValues.Location.ResourcesFile),
		inputValues)
	if err != nil {
		return errors.Wrap(err, "Failed to create on-demand resources file from templates")
	}

	err = createFileFromTemplates(topologyTemplateFileNames, topologyTemplateFile,
		filepath.Join(topologyDir, "topology.yaml"),
		inputValues)
	if err != nil {
		return errors.Wrap(err, "Failed to create topology file from templates")
	}

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

// Creates a file from templates, substituting annotations with data
func createFileFromTemplates(templateFileNames []string, templateName, resultFilePath string, values TopologyValues) error {

	// Mapping from names to functions of functions referenced in templates
	fmap := template.FuncMap{
		"formatAsYAML":                        formatAsYAML,
		"formatOnDemandResourceCredsAsYAML":   formatOnDemandResourceCredsAsYAML,
		"indent":                              indent,
		"getFile":                             getFile,
		"getRepositoryURL":                    getRepositoryURL,
		"getAlien4CloudVersion":               getAlien4CloudVersion,
		"getAlien4CloudVersionFromTOSCATypes": getAlien4CloudVersionFromTOSCATypes,
		"getForgeVersionFromTOSCATypes":       getForgeVersionFromTOSCATypes,
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
	err = removeEmptyLines(resultFilePath)
	return err
}

func removeEmptyLines(filename string) error {
	re := regexp.MustCompile("(?m)^\\s*$[\r\n]*")
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	result := strings.Trim(re.ReplaceAllString(string(data[:]), ""), "\r\n")
	err = ioutil.WriteFile(filename, []byte(result), 0600)
	return err

}
