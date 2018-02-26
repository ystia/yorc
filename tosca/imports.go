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

package tosca

// ImportMap is a map of ImportDefinition indexed by name
type ImportMap map[string]ImportDefinition

// An ImportDefinition is the representation of a TOSCA Import Definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_IMPORT_DEF for more details
type ImportDefinition struct {
	File            string `yaml:"file,omitempty"`
	Repository      string `yaml:"repository,omitempty"`
	NamespaceURI    string `yaml:"namespace_uri,omitempty"`
	NamespacePrefix string `yaml:"namespace_prefix,omitempty"`
}

// UnmarshalYAML unmarshals a yaml into an ImportDefinition
func (i *ImportDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string

	if err := unmarshal(&s); err == nil {
		i.File = s
		return nil
	}
	var str struct {
		File            string `yaml:"file,omitempty"`
		Repository      string `yaml:"repository,omitempty"`
		NamespaceURI    string `yaml:"namespace_uri,omitempty"`
		NamespacePrefix string `yaml:"namespace_prefix,omitempty"`
	}
	if err := unmarshal(&str); err != nil {
		return err
	}
	i.File = str.File
	i.Repository = str.Repository
	i.NamespaceURI = str.NamespaceURI
	i.NamespacePrefix = str.NamespacePrefix
	return nil
}
