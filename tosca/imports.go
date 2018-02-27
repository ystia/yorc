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

import (
	"github.com/pkg/errors"
)

// An ImportDefinition is the representation of a TOSCA Import Definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_IMPORT_DEF for more details
type ImportDefinition struct {
	File            string `yaml:"file"` // Required
	Repository      string `yaml:"repository,omitempty"`
	NamespaceURI    string `yaml:"namespace_uri,omitempty"` // Deprecated
	NamespacePrefix string `yaml:"namespace_prefix,omitempty"`
}

// UnmarshalYAML unmarshals a yaml into an ImportDefinition
func (i *ImportDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string

	// First case, single-line import providing a URI.
	// Example of imports using this format:
	// imports:
	//   - path/file.yaml
	if err := unmarshal(&s); err == nil {
		i.File = s
		return nil
	}

	// Second case, multi-line import.
	// Example of imports using this format:
	// imports:
	//   - file: path/file.yaml
	//     repository: myrepository
	//     namespace_uri: http://mycompany.com/tosca/1.0/platform
	//     namespace_prefix: mycompany

	var str struct {
		File            string `yaml:"file"`
		Repository      string `yaml:"repository,omitempty"`
		NamespaceURI    string `yaml:"namespace_uri,omitempty"`
		NamespacePrefix string `yaml:"namespace_prefix,omitempty"`
	}

	var unmarshalError error
	if unmarshalError = unmarshal(&str); unmarshalError == nil {
		if str.File != "" {
			i.File = str.File
			i.Repository = str.Repository
			i.NamespaceURI = str.NamespaceURI
			i.NamespacePrefix = str.NamespacePrefix
			return nil
		} else {
			// This entry is not compatible with the latest TOSCA specification.
			// But the error is not returned yet, if ever the entry is
			// compatible with a previous TOSCA specification
			unmarshalError = errors.New("Missing required key 'file' in import definition")
		}
	}

	// Additional case for backward compatibility with TOSCA 1.0 specification:
	// Multi-line import with an import name.
	// Example of imports using this format:
	// imports:
	//   - my_import-name:
	//     file: path/file.yaml
	//     repository: myrepository
	//     namespace_uri: http://mycompany.com/tosca/1.0/platform
	//     namespace_prefix: mycompany
	var importMap map[string]ImportDefinition
	if err := unmarshal(&importMap); err == nil && len(importMap) == 1 {
		// Iterate over a map of one element
		for _, importDefinition := range importMap {
			i.File = importDefinition.File
			i.Repository = importDefinition.Repository
			i.NamespaceURI = importDefinition.NamespaceURI
			i.NamespacePrefix = importDefinition.NamespacePrefix
		}
		return nil
	}

	// Additional case for backward compatibility with TOSCA 1.0 specification:
	// Single-line import with an import name.
	// Example of imports using this format:
	// imports:
	//   - my_import-name: path/file.yaml
	var stringMap map[string]string
	if err := unmarshal(&stringMap); err == nil && len(stringMap) == 1 {
		// Iterate over a map of one element
		for _, value := range stringMap {
			i.File = value
		}
		return nil
	}

	// No match for this entry, returning the unmarshal error that was found
	// when attempting to match the latest TOSCA specification
	return unmarshalError
}
