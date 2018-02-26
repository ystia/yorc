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

// An AttributeDefinition is the representation of a TOSCA Attribute Definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_ATTRIBUTE_DEFN for more details
type AttributeDefinition struct {
	Type        string           `yaml:"type"`
	Description string           `yaml:"description,omitempty"`
	Default     *ValueAssignment `yaml:"default,omitempty"`
	Status      string           `yaml:"status,omitempty"`
	EntrySchema EntrySchema      `yaml:"entry_schema,omitempty"`
}

// UnmarshalYAML unmarshals a yaml into an AttributeDefinition
func (r *AttributeDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var ra struct {
		Type        string           `yaml:"type"`
		Description string           `yaml:"description,omitempty"`
		Default     *ValueAssignment `yaml:"default,omitempty"`
		Status      string           `yaml:"status,omitempty"`
		EntrySchema EntrySchema      `yaml:"entry_schema,omitempty"`
	}

	if err := unmarshal(&ra); err == nil && ra.Type != "" {
		r.Description = ra.Description
		r.Type = ra.Type
		r.Default = ra.Default
		r.Status = ra.Status
		r.EntrySchema = ra.EntrySchema
		return nil
	}

	var ras ValueAssignment
	if err := unmarshal(&ras); err != nil {
		return err
	}
	r.Type = ras.Type.String()
	r.Default = &ras
	return nil
}
