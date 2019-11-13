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

// An InterfaceDefinition is the representation of a TOSCA Interface Definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_INTERFACE_DEF for more details
type InterfaceDefinition struct {
	Type        string                         `yaml:"type,omitempty" json:"type,omitempty"`
	Description string                         `yaml:"description,omitempty" json:"description,omitempty"`
	Inputs      map[string]Input               `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Operations  map[string]OperationDefinition `yaml:",inline,omitempty" json:",inline,omitempty"`
}

// An OperationDefinition is the representation of a TOSCA Operation Definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_OPERATION_DEF for more details
type OperationDefinition struct {
	Inputs         map[string]Input `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Description    string           `yaml:"description,omitempty" json:"description,omitempty"`
	Implementation Implementation   `yaml:"implementation,omitempty" json:"implementation,omitempty"`
}

// UnmarshalYAML unmarshals a yaml into an InterfaceDefinition
func (i *OperationDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		i.Implementation = Implementation{Primary: s}
		return nil
	}
	var str struct {
		Inputs         map[string]Input `yaml:"inputs,omitempty"`
		Description    string           `yaml:"description,omitempty"`
		Implementation Implementation   `yaml:"implementation,omitempty"`
	}
	if err := unmarshal(&str); err != nil {
		return err
	}
	i.Inputs = str.Inputs
	i.Implementation = str.Implementation
	i.Description = str.Description
	return nil
}
