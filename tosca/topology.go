package tosca

type Topology struct {
	TOSCAVersion string `yaml:"tosca_definitions_version"`
	Description  string `yaml:",omitempty"`
	Name         string `yaml:"template_name"`
	Version      string `yaml:"template_version"`
	Author       string `yaml:"template_author"`

	Imports []string

	TopologyTemplate TopologyTemplate `yaml:"topology_template"`
}

type TopologyTemplate struct {
	Description string `yaml:",omitempty"`
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
	Type         string
	Description  string                          `yaml:",omitempty"`
	Directives   []string                        `yaml:",omitempty"`
	Properties   map[string]string               `yaml:",omitempty"`
	Attributes   map[string]string               `yaml:",omitempty"`
	Capabilities map[string]CapabilityAssignment `yaml:",omitempty"`
}

type CapabilityAssignment struct {
	Properties map[string]string `yaml:",omitempty"`
	Attributes map[string]string `yaml:",omitempty"`
}
