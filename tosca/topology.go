package tosca

// An Topology is the representation of a TOSCA Service Template definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html#DEFN_ELEMENT_SERVICE_TEMPLATE for more details
type Topology struct {
	TOSCAVersion string `yaml:"tosca_definitions_version"`
	Description  string `yaml:"description,omitempty"`
	Name         string `yaml:"template_name"`
	Version      string `yaml:"template_version"`
	Author       string `yaml:"template_author"`

	Imports []ImportMap `yaml:"imports,omitempty"`

	Repositories map[string]Repository `yaml:"repositories,omitempty"`

	// TODO Data Types
	ArtifactTypes     map[string]ArtifactType     `yaml:"artifact_types,omitempty"`
	NodeTypes         map[string]NodeType         `yaml:"node_types,omitempty"`
	CapabilityTypes   map[string]CapabilityType   `yaml:"capability_types,omitempty"`
	RelationshipTypes map[string]RelationshipType `yaml:"relationship_types,omitempty"`
	// TODO Group Types
	// TODO Policy Types

	TopologyTemplate TopologyTemplate `yaml:"topology_template"`
}

// An TopologyTemplate is the representation of a TOSCA Topology Template
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html#DEFN_ENTITY_TOPOLOGY_TEMPLATE for more details
type TopologyTemplate struct {
	Description   string                         `yaml:"description,omitempty"`
	Inputs        map[string]ParameterDefinition `yaml:"inputs,omitempty"`
	NodeTemplates map[string]NodeTemplate        `yaml:"node_templates"`
	//RelationshipTemplates []RelationshipTemplate `yaml:"relationship_templates,omitempty"`
	//Groups                []Group `yaml:",omitempty"`
	//Policies              []Policy                 `yaml:",omitempty"`
	Outputs map[string]ParameterDefinition `yaml:"outputs,omitempty"`
	//substitution_mappings
	Workflows map[string]Workflow
}

// An NodeTemplate is the representation of a TOSCA Node Template
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html#DEFN_ENTITY_NODE_TEMPLATE for more details
type NodeTemplate struct {
	Type         string                          `yaml:"type"`
	Description  string                          `yaml:"description,omitempty"`
	Directives   []string                        `yaml:"directives,omitempty"`
	Properties   map[string]ValueAssignment      `yaml:"properties,omitempty"`
	Attributes   map[string]ValueAssignment      `yaml:"attributes,omitempty"`
	Capabilities map[string]CapabilityAssignment `yaml:"capabilities,omitempty"`
	Requirements []RequirementAssignmentMap      `yaml:"requirements,omitempty"`
	Artifacts    ArtifactDefMap                  `yaml:"artifacts,omitempty"`
}

//A Repository is representation of TOSCA Repository
//
//See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/csprd01/TOSCA-Simple-Profile-YAML-v1.0-csprd01.html#_Toc430015673 for more details
type Repository struct {
	URL         string     `yaml:"url,omitempty"`
	Type        string     `yaml:"type,omitempty"`
	Description string     `yaml:"description,omitempty"`
	Credit      Credential `yaml:"credential,omitempty"`
}

type Credential struct {
	TokenType string            `yaml:"token_type"`
	Token     string            `yaml:"token"`
	User      string            `yaml:"user,omitempty"`
	Protocol  string            `yaml:"protocol,omitempty"`
	Keys      map[string]string `yaml:"keys,omitempty"`
}
