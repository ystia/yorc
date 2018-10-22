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
	"io/ioutil"
	"os"
	"path/filepath"
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

// TopologyValues provides inputs to the topology templates
type TopologyValues struct {
	Ansible        AnsibleConfiguration
	Yorc           YorcConfiguration
	Consul         ConsulConfiguration
	Terraform      TerraformConfiguration
	Infrastructure config.DynamicMap
	Compute        config.DynamicMap
	Address        config.DynamicMap
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

/*
func initMissingValues(values *TopologyValues) error {
	var err error

	switch values.Infrastructure.Type {
	case "openstack":
		values.Address.Type = "yorc.nodes.openstack.FloatingIP"
	case "google":
		values.Address.Type = "yorc.nodes.google.Address"
	default:
		err = fmt.Errorf("Unknown infrastructure type %s", values.Infrastructure.Type)
	}

	return err
}
*/

// createTopology creates under destinationPath, a topology from a zip file at topologyPath
// by executing its template files using inputs passed in inputsPath file
func createTopology(topologyPath, destinationPath, inputsPath string) error {

	// First retrieve template files from the zip file provided
	// These files are expected to have the extension tmpl at the root of the directory
	_, err := ziputil.Unzip(topologyPath, destinationPath)
	if err != nil {
		return err
	}

	var templateFileNames []string
	err = filepath.Walk(destinationPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if filepath.Ext(path) == ".tmpl" {
				templateFileNames = append(templateFileNames, path)
			}

			return err
		})

	if err != nil {
		return errors.Wrapf(err, "Failed to browse unzip of %s under %s", topologyPath, destinationPath)
	}

	if len(templateFileNames) == 0 {
		// No template found, the topology is ready
		return nil
	}

	// The template created when parsing these files has the base name of the last
	// template according to golang template.ParseFiles documentation
	templateName := filepath.Base(templateFileNames[len(templateFileNames)-1])

	// Mapping from names to functions of functions referenced in templates
	fmap := template.FuncMap{
		"formatAsYAML": formatAsYAML,
	}

	tmpl := template.Must(template.New(templateName).Funcs(fmap).ParseFiles(templateFileNames...))

	// Now read inputs for the inputsPath provided in argument
	if inputsPath != "" {

		data, err := ioutil.ReadFile(inputsPath)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(data, &inputValues)
		if err != nil {
			return errors.Wrapf(err, "Failed to unmarshall inputs from %s", inputsPath)
		}
	}

	topologyFilePath := filepath.Join(destinationPath, "topology.yaml")
	resultFile, err := os.Create(topologyFilePath)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(resultFile)

	err = tmpl.Execute(writer, inputValues)
	resultFile.Close()
	if err != nil {
		return errors.Wrap(err, "Failed to create topoly from template")
	}

	return err
}
