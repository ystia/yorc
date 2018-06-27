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
	// TemplateName is the optional descriptive name for the template provided
	// in the template metadata
	TemplateName = "template_name"
	// TemplateVersion is the optional version for the template provided in the
	// template metadata
	TemplateVersion = "template_version"
	// TemplateAuthor is the optional declaration of the author of the template
	// provided in the template metadata
	TemplateAuthor = "template_author"
)

// An Topology is the representation of a TOSCA Service Template definition
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_SERVICE_TEMPLATE for more details
type Topology struct {
	TOSCAVersion string            `yaml:"tosca_definitions_version"`
	Description  string            `yaml:"description,omitempty"`
	Metadata     map[string]string `yaml:"metadata,omitempty"`

	Imports []ImportDefinition `yaml:"imports,omitempty"`

	Repositories map[string]Repository `yaml:"repositories,omitempty"`

	DataTypes         map[string]DataType         `yaml:"data_types,omitempty"`
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
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_TOPOLOGY_TEMPLATE for more details
type TopologyTemplate struct {
	Description        string                         `yaml:"description,omitempty"`
	Inputs             map[string]ParameterDefinition `yaml:"inputs,omitempty"`
	NodeTemplates      map[string]NodeTemplate        `yaml:"node_templates"`
	Outputs            map[string]ParameterDefinition `yaml:"outputs,omitempty"`
	SubstitionMappings *SubstitutionMapping           `yaml:"substitution_mappings,omitempty"`
	Workflows          map[string]Workflow
	//RelationshipTemplates []RelationshipTemplate `yaml:"relationship_templates,omitempty"`
	//Groups                []Group `yaml:",omitempty"`
	//Policies              []Policy                 `yaml:",omitempty"`
}

// An NodeTemplate is the representation of a TOSCA Node Template
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_NODE_TEMPLATE for more details
type NodeTemplate struct {
	Type         string                          `yaml:"type"`
	Description  string                          `yaml:"description,omitempty"`
	Directives   []string                        `yaml:"directives,omitempty"`
	Properties   map[string]*ValueAssignment     `yaml:"properties,omitempty"`
	Attributes   map[string]*ValueAssignment     `yaml:"attributes,omitempty"`
	Capabilities map[string]CapabilityAssignment `yaml:"capabilities,omitempty"`
	Requirements []RequirementAssignmentMap      `yaml:"requirements,omitempty"`
	Artifacts    ArtifactDefMap                  `yaml:"artifacts,omitempty"`
	Metadata     map[string]string               `yaml:"metadata,omitempty"`
	Interfaces   map[string]InterfaceDefinition  `yaml:"interfaces,omitempty"`
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

// A Credential is a representation of TOSCA Credential
type Credential struct {
	TokenType string            `yaml:"token_type"`
	Token     string            `yaml:"token"`
	User      string            `yaml:"user,omitempty"`
	Protocol  string            `yaml:"protocol,omitempty"`
	Keys      map[string]string `yaml:"keys,omitempty"`
}
