package tosca

type CapabilityDefinition struct {
	Type             string                         `yaml:"type"`
	Description      string                         `yaml:"description,omitempty"`
	Properties       map[string]PropertyDefinition  `yaml:"properties,omitempty"`
	Attributes       map[string]AttributeDefinition `yaml:"attributes,omitempty"`
	ValidSourceTypes []string                       `yaml:"valid_source_types,omitempty,flow"`
	Occurrences      ToscaRange                     `yaml:"occurrences,omitempty"`
}

type CapabilityAssignment struct {
	Properties map[string]ValueAssignment `yaml:"properties,omitempty"`
	Attributes map[string]ValueAssignment `yaml:"attributes,omitempty"`
}

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
		Occurrences      ToscaRange                     `yaml:"occurrences,omitempty"`
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
