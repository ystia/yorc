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

	"gopkg.in/yaml.v2"
)

// AnsibleConfiguration provides Ansible user-defined settings
type AnsibleConfiguration struct {
	Version string
}

// YorcConfiguration provides Yorc user-defined settings
type YorcConfiguration struct {
	DownloadURL string `yaml:"download_url"`
	Port        int
}

// YorcPluginConfiguration provides Yorc plugin user-defined settings
type YorcPluginConfiguration struct {
	DownloadURL string `yaml:"download_url"`
}

// Alien4CloudConfiguration provides Alien4Cloud user-defined settings
type Alien4CloudConfiguration struct {
	DownloadURL string `yaml:"download_url"`
	Port        int
}

// ConsulConfiguration provides Consul user-defined settings
type ConsulConfiguration struct {
	DownloadURL string `yaml:"download_url"`
	Port        int
}

// TerraformConfiguration provides Terraform settings
type TerraformConfiguration struct {
	Version    string   `yaml:"component_version"`
	PluginURLs []string `yaml:"plugins_download_urls"`
}

// LocationConfiguration provides an Alien4Cloud plugin location configuration
type LocationConfiguration struct {
	Type          string
	Name          string
	ResourcesFile string
}

// TopologyValues provides inputs to the topology templates
type TopologyValues struct {
	Ansible        AnsibleConfiguration
	Alien4cloud    Alien4CloudConfiguration
	Yorc           YorcConfiguration
	YorcPlugin     YorcPluginConfiguration
	Consul         ConsulConfiguration
	Terraform      TerraformConfiguration
	Infrastructure config.DynamicMap
	Compute        config.DynamicMap
	Address        config.DynamicMap
	Location       LocationConfiguration
}

var inputValues TopologyValues

// formatAsYAML is a function used in templates to output the yaml representation
// of a variable
func formatAsYAML(data interface{}, indentations int) (string, error) {

	result := ""
	bSlice, err := yaml.Marshal(data)
	if err == nil {
		bSlice = append([]byte("\n"), bSlice...)
		if indentations > 0 {
			indentString := strings.Repeat(" ", indentations)
			result = strings.Replace(string(bSlice), "\n", "\n"+indentString, -1)
			result = strings.TrimRight(result, " \n")
		} else {
			result = string(bSlice)
		}
	}
	return result, err
}

// getAlien4CloudVersion extracts an Alien4cloud version form its download URL
func getAlien4CloudVersion(url string) (string, error) {
	match := regexp.MustCompile(`-dist-([0-9a-zA-Z.-]+)-dist.tar.gz`).FindStringSubmatch(url)
	version := ""
	var err error
	if match != nil {
		version = match[1]
	} else {
		err = fmt.Errorf("Failed to retrieve Alien4Cloud version from URL %s", url)
	}

	return version, err
}

// getFile returns the file par of a URL
func getFile(url string) string {
	_, file := filepath.Split(url)
	return file
}

// createTopology creates under destinationPath, a topology from a zip file at topologyPath
// by executing its template files using inputs passed in inputsPath file
func createTopology(topologyPath, destinationPath, inputsPath string) error {

	var err error
	inputValues, err = getInputValues(inputsPath)
	if err != nil {
		return err
	}

	// First retrieve template files from the zip file provided
	// These files are expected to have the extension tmpl at the root of the directory
	_, err = ziputil.Unzip(topologyPath, destinationPath)
	if err != nil {
		return err
	}

	var topologyTemplateFileNames []string
	var resourcesTemplateFileNames []string
	resourcesTemplatePrefix := "ondemand_resources"
	topologyTemplatePrefix := "topology"
	infrastructureTemplateSuffix := infrastructureType + ".tmpl"
	err = filepath.Walk(destinationPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			_, file := filepath.Split(path)
			if file == "topology.tmpl" ||
				strings.HasSuffix(file, infrastructureTemplateSuffix) {

				if strings.HasPrefix(file, resourcesTemplatePrefix) {
					resourcesTemplateFileNames = append(resourcesTemplateFileNames, path)
				} else if strings.HasPrefix(file, topologyTemplatePrefix) {
					topologyTemplateFileNames = append(topologyTemplateFileNames, path)
				}
			}

			return err
		})

	if err != nil {
		return errors.Wrapf(err, "Failed to browse unzip of %s under %s", topologyPath, destinationPath)
	}

	if len(topologyTemplateFileNames) == 0 {
		// No topology template found, the topology is ready
		return nil
	}

	if len(resourcesTemplateFileNames) == 0 {
		// Need an on-demand resource template file name
		return fmt.Errorf("Found no on-demand resources template in %s", destinationPath)
	}

	err = createFileFromTemplates(resourcesTemplateFileNames,
		filepath.Join(destinationPath, inputValues.Location.ResourcesFile),
		inputValues)
	if err != nil {
		return errors.Wrap(err, "Failed to create on-demand resources file from templates")
	}

	err = createFileFromTemplates(topologyTemplateFileNames,
		filepath.Join(destinationPath, "topology.yaml"),
		inputValues)
	if err != nil {
		return errors.Wrap(err, "Failed to create topology file from templates")
	}

	return err
}

// Creates a file from templates, substituting annotations with data
func createFileFromTemplates(templateFileNames []string, resultFilePath string, values TopologyValues) error {

	templateName := filepath.Base(templateFileNames[0])

	// Mapping from names to functions of functions referenced in templates
	fmap := template.FuncMap{
		"formatAsYAML":          formatAsYAML,
		"getFile":               getFile,
		"getAlien4CloudVersion": getAlien4CloudVersion,
	}

	parsedTemplate, err := template.New(templateName).Funcs(fmap).ParseFiles(templateFileNames...)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(resultFilePath), os.ModePerm); err != nil {
		return err
	}

	resultFile, err := os.Create(resultFilePath)
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
	return err
}

// getInputValues initializes topology values from an input file
func getInputValues(inputFilePath string) (TopologyValues, error) {

	var values TopologyValues

	// Read inputs for the inputsPath provided in argument
	if inputFilePath != "" {

		data, err := ioutil.ReadFile(inputFilePath)
		if err != nil {
			return values, err
		}
		err = yaml.Unmarshal(data, &values)
		if err != nil {
			return values, errors.Wrapf(err, "Failed to unmarshall inputs from %s", inputFilePath)
		}
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
