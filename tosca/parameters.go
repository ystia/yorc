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

// An ParameterDefinition is the representation of a TOSCA Parameter Definition
//
// See https://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.3/cos01/TOSCA-Simple-Profile-YAML-v1.3-cos01.html#DEFN_ELEMENT_PARAMETER_DEF for more details
type ParameterDefinition struct {
	Type        string           `yaml:"type,omitempty" json:"type,omitempty"`
	Description string           `yaml:"description,omitempty" json:"description,omitempty"`
	Required    *bool            `yaml:"required,omitempty" json:"required,omitempty"`
	Default     *ValueAssignment `yaml:"default,omitempty" json:"default,omitempty"`
	Status      string           `yaml:"status,omitempty" json:"status,omitempty"`
	//Constraints []ConstraintClause `yaml:"constraints,omitempty"`
	EntrySchema EntrySchema      `yaml:"entry_schema,omitempty" json:"entry_schema,omitempty"`
	Value       *ValueAssignment `yaml:"value,omitempty" json:"value,omitempty"`
}

// UnmarshalYAML unmarshals a yaml into a ParameterDefinition
func (p *ParameterDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var err error
	var str struct {
		Type        string           `yaml:"type,omitempty" json:"type,omitempty"`
		Description string           `yaml:"description,omitempty" json:"description,omitempty"`
		Required    *bool            `yaml:"required,omitempty" json:"required,omitempty"`
		Default     *ValueAssignment `yaml:"default,omitempty" json:"default,omitempty"`
		Status      string           `yaml:"status,omitempty" json:"status,omitempty"`
		EntrySchema EntrySchema      `yaml:"entry_schema,omitempty" json:"entry_schema,omitempty"`
		Value       *ValueAssignment `yaml:"value,omitempty" json:"value,omitempty"`
	}
	if err = unmarshal(&str); err == nil {
		p.Type = str.Type
		p.Description = str.Description
		p.Required = str.Required
		p.Default = str.Default
		p.Status = str.Status
		p.EntrySchema = str.EntrySchema
		p.Value = str.Value
		return nil
	}

	// Other case:, single-line grammar
	// param_name: param_value
	var v ValueAssignment
	if err = unmarshal(&v); err == nil {
		p.Value = &v
		return nil
	}

	return err
}
