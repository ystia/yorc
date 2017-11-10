package tosca

// Type is the base type for all TOSCA types (like node types, relationship types, ...)
type Type struct {
	DerivedFrom string            `yaml:"derived_from,omitempty"`
	Version     string            `yaml:"version,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
}

// An NodeType is the representation of a TOSCA Node Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html#DEFN_ENTITY_NODE_TYPE
// for more details
type NodeType struct {
	Type         `yaml:",inline"`
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
	Type             `yaml:",inline"`
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
	Type             `yaml:",inline"`
	Properties       map[string]PropertyDefinition  `yaml:"properties,omitempty"`
	Attributes       map[string]AttributeDefinition `yaml:"attributes,omitempty"`
	ValidSourceTypes []string                       `yaml:"valid_source_types,omitempty,flow"`
}

// An ArtifactType is the representation of a TOSCA Artifact Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html#DEFN_ENTITY_ARTIFACT_TYPE
// for more details
type ArtifactType struct {
	Type       `yaml:",inline"`
	MimeType   string                        `yaml:"mime_type,omitempty"`
	FileExt    []string                      `yaml:"file_ext,omitempty"`
	Properties map[string]PropertyDefinition `yaml:"properties,omitempty"`
}

// An DataType is the representation of a TOSCA Data Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_DATA_TYPE
// for more details
type DataType struct {
	Type       `yaml:",inline"`
	Properties map[string]PropertyDefinition `yaml:"properties,omitempty"`
	// Constraints not enforced in Janus so we don't parse them
	// Constraints []ConstraintClause
}
