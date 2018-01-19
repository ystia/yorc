package tosca

// An CapabilityDefinition is the representation of a TOSCA Capability Definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_CAPABILITY_DEFN for more details
type CapabilityDefinition struct {
	Type             string                         `yaml:"type"`
	Description      string                         `yaml:"description,omitempty"`
	Properties       map[string]PropertyDefinition  `yaml:"properties,omitempty"`
	Attributes       map[string]AttributeDefinition `yaml:"attributes,omitempty"`
	ValidSourceTypes []string                       `yaml:"valid_source_types,omitempty,flow"`
	Occurrences      Range                          `yaml:"occurrences,omitempty"`
}

// An CapabilityAssignment is the representation of a TOSCA Capability Assignment
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_CAPABILITY_ASSIGNMENT for more details
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
		Type             string                         `yaml:"type"`
		Description      string                         `yaml:"description,omitempty"`
		Properties       map[string]PropertyDefinition  `yaml:"properties,omitempty"`
		Attributes       map[string]AttributeDefinition `yaml:"attributes,omitempty"`
		ValidSourceTypes []string                       `yaml:"valid_source_types,omitempty,flow"`
		Occurrences      Range                          `yaml:"occurrences,omitempty"`
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
