package tosca

// An NodeType is the representation of a TOSCA Node Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html#DEFN_ENTITY_NODE_TYPE
// for more details
type NodeType struct {
	DerivedFrom  string                            `yaml:"derived_from,omitempty"`
	Version      string                            `yaml:"version,omitempty"`
	Description  string                            `yaml:"description,omitempty"`
	Properties   map[string]PropertyDefinition     `yaml:"properties,omitempty"`
	Attributes   map[string]AttributeDefinition    `yaml:"attributes,omitempty"`
	Requirements []RequirementDefinitionMap        `yaml:"requirements,omitempty,flow"`
	Capabilities map[string]CapabilityDefinition   `yaml:"capabilities,omitempty"`
	Interfaces   map[string]InterfaceDefinitionMap `yaml:"interfaces,omitempty"`
	Artifacts    ArtifactDefMap                    `yaml:"artifacts,omitempty"`
}

// An RelationshipType is the representation of a TOSCA Relationship Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html#DEFN_ENTITY_RELATIONSHIP_TYPE
// for more details
type RelationshipType struct {
	DerivedFrom      string                            `yaml:"derived_from,omitempty"`
	Version          string                            `yaml:"version,omitempty"`
	Description      string                            `yaml:"description,omitempty"`
	Properties       map[string]PropertyDefinition     `yaml:"properties,omitempty"`
	Attributes       map[string]AttributeDefinition    `yaml:"attributes,omitempty"`
	Interfaces       map[string]InterfaceDefinitionMap `yaml:"interfaces,omitempty"`
	Artifacts        ArtifactDefMap                    `yaml:"artifacts,omitempty"`
	ValidTargetTypes []string                          `yaml:"valid_target_types,omitempty"`
}

// An CapabilityType is the representation of a TOSCA Capability Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html#DEFN_ENTITY_CAPABILITY_TYPE
// for more details
type CapabilityType struct {
	DerivedFrom      string                         `yaml:"derived_from,omitempty"`
	Version          string                         `yaml:"version,omitempty"`
	Description      string                         `yaml:"description,omitempty"`
	Properties       map[string]PropertyDefinition  `yaml:"properties,omitempty"`
	Attributes       map[string]AttributeDefinition `yaml:"attributes,omitempty"`
	ValidSourceTypes []string                       `yaml:"valid_source_types,omitempty,flow"`
}
