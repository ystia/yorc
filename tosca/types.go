package tosca

type NodeType struct {
	DerivedFrom  string                            `yaml:"derived_from,omitempty"`
	Version      string                            `yaml:"version,omitempty"`
	Description  string                            `yaml:"description,omitempty"`
	Properties   map[string]PropertyDefinition     `yaml:"properties,omitempty"`
	Attributes   map[string]AttributeDefinition    `yaml:"attributes,omitempty"`
	Requirements []RequirementDefinitionMap        `yaml:"requirements,omitempty,flow"`
	Capabilities map[string]CapabilityDefinition   `yaml:"capabilities,omitempty"`
	Interfaces   map[string]InterfaceDefinitionMap `yaml:"interfaces,omitempty"`
	Artifacts    map[string]ArtifactDefinition     `yaml:"artifacts,omitempty"`
}

type RelationshipType struct {
	DerivedFrom      string                            `yaml:"derived_from,omitempty"`
	Version          string                            `yaml:"version,omitempty"`
	Description      string                            `yaml:"description,omitempty"`
	Properties       map[string]PropertyDefinition     `yaml:"properties,omitempty"`
	Attributes       map[string]AttributeDefinition    `yaml:"attributes,omitempty"`
	Interfaces       map[string]InterfaceDefinitionMap `yaml:"interfaces,omitempty"`
	ValidTargetTypes []string                          `yaml:"valid_target_types,omitempty"`
}

type CapabilityType struct {
	DerivedFrom      string                         `yaml:"derived_from,omitempty"`
	Version          string                         `yaml:"version,omitempty"`
	Description      string                         `yaml:"description,omitempty"`
	Properties       map[string]PropertyDefinition  `yaml:"properties,omitempty"`
	Attributes       map[string]AttributeDefinition `yaml:"attributes,omitempty"`
	ValidSourceTypes []string                       `yaml:"valid_source_types,omitempty,flow"`
}
