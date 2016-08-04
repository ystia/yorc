package tosca

type Topology struct {
	TOSCAVersion string `yaml:"tosca_definitions_version"`
	Description  string `yaml:"description,omitempty"`
	Name         string `yaml:"template_name"`
	Version      string `yaml:"template_version"`
	Author       string `yaml:"template_author"`

	Imports []ImportMap `yaml:"imports,omitempty"`

	// TODO Data Types
	NodeTypes         map[string]NodeType         `yaml:"node_types,omitempty"`
	CapabilityTypes   map[string]CapabilityType   `yaml:"capability_types,omitempty"`
	RelationshipTypes map[string]RelationshipType `yaml:"relationship_types,omitempty"`
	// TODO Group Types
	// TODO Policy Types

	TopologyTemplate TopologyTemplate `yaml:"topology_template"`
}

type TopologyTemplate struct {
	Description string `yaml:"description,omitempty"`
	//Inputs                []ParameterDefinition  `yaml:",omitempty"`
	NodeTemplates map[string]NodeTemplate `yaml:"node_templates"`
	//RelationshipTemplates []RelationshipTemplate `yaml:"relationship_templates,omitempty"`
	//Groups                []Group `yaml:",omitempty"`
	//Policies              []Policy                 `yaml:",omitempty"`
	//Outputs               []ParameterDefinition  `yaml:",omitempty"`
	//substitution_mappings
	Workflows map[string]Workflow
}

type NodeTemplate struct {
	Type         string                          `yaml:"type"`
	Description  string                          `yaml:"description,omitempty"`
	Directives   []string                        `yaml:"directives,omitempty"`
	Properties   map[string]ValueAssignment      `yaml:"properties,omitempty"`
	Attributes   map[string]ValueAssignment      `yaml:"attributes,omitempty"`
	Capabilities map[string]CapabilityAssignment `yaml:"capabilities,omitempty"`
	Requirements []RequirementAssignmentMap      `yaml:"requirements,omitempty"`
	Artifacts    map[string]ArtifactDefinition   `yaml:"artifacts,omitempty"`
}
