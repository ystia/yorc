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

const (
	// EndpointCapability is the default TOSCA type that should be used or
	// extended to define a network endpoint capability.
	EndpointCapability = "tosca.capabilities.Endpoint"

	// PublicEndpointCapability represents a public endpoint.
	PublicEndpointCapability = "tosca.capabilities.Endpoint.Public"
)

// An CapabilityDefinition is the representation of a TOSCA Capability Definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_CAPABILITY_DEFN for more details
// NOTE: Here is Alien specific difference with Tosca Specification about Properties/Attributes maps of ValueAssignment instead of maps of PropertyDefinition in Tosca spec
type CapabilityDefinition struct {
	Type             string                      `yaml:"type"`
	Description      string                      `yaml:"description,omitempty"`
	Properties       map[string]*ValueAssignment `yaml:"properties,omitempty"`
	Attributes       map[string]*ValueAssignment `yaml:"attributes,omitempty"`
	ValidSourceTypes []string                    `yaml:"valid_source_types,omitempty,flow"`
	Occurrences      Range                       `yaml:"occurrences,omitempty"`
}

// An CapabilityAssignment is the representation of a TOSCA Capability Assignment
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html#DEFN_ELEMENT_CAPABILITY_ASSIGNMENT for more details
type CapabilityAssignment struct {
	Properties map[string]*ValueAssignment `yaml:"properties,omitempty"`
	Attributes map[string]*ValueAssignment `yaml:"attributes,omitempty"`
}

// UnmarshalYAML unmarshals a yaml into an CapabilityDefinition
func (c *CapabilityDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		c.Type = s
		return nil
	}
	var str struct {
		Type             string                      `yaml:"type"`
		Description      string                      `yaml:"description,omitempty"`
		Properties       map[string]*ValueAssignment `yaml:"properties,omitempty"`
		Attributes       map[string]*ValueAssignment `yaml:"attributes,omitempty"`
		ValidSourceTypes []string                    `yaml:"valid_source_types,omitempty,flow"`
		Occurrences      Range                       `yaml:"occurrences,omitempty"`
	}
	if err := unmarshal(&str); err != nil {
		return err
	}
	c.Type = str.Type
	c.Description = str.Description
	c.Properties = str.Properties
	c.Attributes = str.Attributes
	c.ValidSourceTypes = str.ValidSourceTypes
	c.Occurrences = str.Occurrences
	return nil
}
