package tosca

type CapabilityDefinition struct {
	Type             string                         `yaml:"type"`
	Description      string                         `yaml:"description,omitempty"`
	Properties       map[string]PropertyDefinition  `yaml:"properties,omitempty"`
	Attributes       map[string]AttributeDefinition `yaml:"attributes,omitempty"`
	ValidSourceTypes []string                       `yaml:"valid_source_types,omitempty,flow"`
	Occurrences      string                         `yaml:"occurrences,omitempty"`
}

type CapabilityAssignment struct {
	Properties map[string]ValueAssignment `yaml:"properties,omitempty"`
	Attributes map[string]ValueAssignment `yaml:"attributes,omitempty"`
}
